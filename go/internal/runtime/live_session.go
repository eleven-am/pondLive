package runtime

import (
	"errors"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eleven-am/go/pondlive/internal/diff"
	handlers "github.com/eleven-am/go/pondlive/internal/handlers"
	"github.com/eleven-am/go/pondlive/internal/protocol"
	render "github.com/eleven-am/go/pondlive/internal/render"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
)

const (
	defaultFrameHistory = 64
	defaultSessionTTL   = 90 * time.Second
)

// Transport delivers messages to the client connection backing a session.
type Transport interface {
	SendInit(protocol.Init) error
	SendResume(protocol.ResumeOK) error
	SendFrame(protocol.Frame) error
	SendServerError(protocol.ServerError) error
	SendPubsubControl(protocol.PubsubControl) error
}

// LiveSession wires a component tree to the PondSocket protocol and tracks
// sufficient state to resume clients after reconnects.
type LiveSession struct {
	id      SessionID
	version int

	component *ComponentSession

	mu  sync.Mutex
	now func() time.Time
	ttl atomic.Int64
	loc Location

	frameCap int

	nextSeq     int
	lastInitSeq int
	lastAck     int
	clientSeq   int

	frames []protocol.Frame

	snapshot snapshot

	pendingEffects []any

	transport Transport
	devMode   bool

	pubsubCounts map[string]int

	diagnostics []Diagnostic

	updatedAt      time.Time
	hasInit        bool
	touchObservers []func(time.Time)

	clientConfig *protocol.ClientConfig
}

// LiveSessionConfig captures the optional configuration applied when constructing a live session.
type LiveSessionConfig struct {
	Transport      Transport
	FrameHistory   int
	TTL            time.Duration
	Clock          func() time.Time
	DevMode        *bool
	PubsubProvider PubsubProvider
	ClientConfig   *protocol.ClientConfig
}

func defaultLiveSessionConfig() LiveSessionConfig {
	return LiveSessionConfig{
		FrameHistory: defaultFrameHistory,
		TTL:          defaultSessionTTL,
		Clock:        time.Now,
	}
}

func mergeLiveSessionConfig(base LiveSessionConfig, cfg *LiveSessionConfig) LiveSessionConfig {
	out := base
	if cfg == nil {
		return out
	}
	if cfg.Transport != nil {
		out.Transport = cfg.Transport
	}
	if cfg.FrameHistory > 0 {
		out.FrameHistory = cfg.FrameHistory
	}
	if cfg.TTL > 0 {
		out.TTL = cfg.TTL
	}
	if cfg.Clock != nil {
		out.Clock = cfg.Clock
	}
	if cfg.PubsubProvider != nil {
		out.PubsubProvider = cfg.PubsubProvider
	}
	if cfg.DevMode != nil {
		value := *cfg.DevMode
		out.DevMode = &value
	}
	if cfg.ClientConfig != nil {
		clone := cloneClientConfig(cfg.ClientConfig)
		out.ClientConfig = clone
	}
	return out
}

// NewLiveSession constructs a session runtime for the given component tree.
func NewLiveSession[P any](sid SessionID, version int, root Component[P], props P, cfg *LiveSessionConfig) *LiveSession {
	component := NewSession(root, props)
	component.SetRegistry(handlers.NewRegistry())

	session := &LiveSession{
		id:        sid,
		version:   version,
		component: component,
		frameCap:  defaultFrameHistory,
		now:       time.Now,
		loc: Location{
			Path:   "/",
			Query:  "",
			Params: map[string]string{},
		},
		nextSeq:      1,
		pubsubCounts: make(map[string]int),
	}

	session.ttl.Store(int64(defaultSessionTTL))
	component.setOwner(session)

	effectiveConfig := mergeLiveSessionConfig(defaultLiveSessionConfig(), cfg)
	session.transport = effectiveConfig.Transport
	session.frameCap = effectiveConfig.FrameHistory
	session.now = effectiveConfig.Clock
	if effectiveConfig.DevMode != nil {
		session.devMode = *effectiveConfig.DevMode
	}
	if effectiveConfig.TTL > 0 {
		session.ttl.Store(int64(effectiveConfig.TTL))
	}
	if effectiveConfig.PubsubProvider != nil {
		component.SetPubsubProvider(effectiveConfig.PubsubProvider)
	}

	session.clientConfig = cloneClientConfig(effectiveConfig.ClientConfig)

	component.SetDiagnosticReporter(session)
	component.SetPatchSender(session.onPatch)

	structured := component.InitialStructured()
	meta := component.Metadata()
	session.mu.Lock()
	session.snapshot = buildSnapshot(structured, session.loc, meta)
	session.touchLocked()
	session.mu.Unlock()

	return session
}

// TTL returns the inactivity timeout configured for the session.
func (s *LiveSession) TTL() time.Duration {

	return time.Duration(s.ttl.Load())
}

// AddTouchObserver registers a callback invoked when the session refreshes its last touched timestamp.
// The returned function removes the observer.
func (s *LiveSession) AddTouchObserver(cb func(time.Time)) func() {
	if cb == nil {
		return func() {}
	}
	s.mu.Lock()
	s.touchObservers = append(s.touchObservers, cb)
	idx := len(s.touchObservers) - 1
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		if idx >= 0 && idx < len(s.touchObservers) {
			s.touchObservers[idx] = nil
		}
		s.mu.Unlock()
	}
}

// JoinResult captures the work required to satisfy a resume attempt.
type JoinResult struct {
	Init   *protocol.Init
	Resume *protocol.ResumeOK
	Frames []protocol.Frame
}

// Join reconciles a client resume attempt and decides between a full init or
// replaying frames from the local history.
func (s *LiveSession) Join(clientVersion int, ack int) JoinResult {
	s.mu.Lock()

	s.touchLocked()

	diagSnapshot := s.diagnosticsSnapshotLocked()

	if ack < 0 {
		ack = 0
	}

	latest := s.nextSeq - 1
	if ack > latest {
		ack = latest
	}

	if clientVersion != s.version || !s.hasInit || ack < s.lastInitSeq {
		init := s.buildInitLocked(diagSnapshot)
		s.mu.Unlock()
		return JoinResult{Init: &init}
	}

	if ack > s.lastAck {
		s.lastAck = ack
		s.pruneAckedLocked()
	}

	from := ack + 1
	if from < s.lastInitSeq+1 {
		from = s.lastInitSeq + 1
	}

	if from > s.nextSeq {
		from = s.nextSeq
	}

	oldest := 0
	if len(s.frames) > 0 {
		oldest = s.frames[0].Seq
	}

	if oldest > 0 && from < oldest {
		init := s.buildInitLocked(diagSnapshot)
		s.mu.Unlock()
		return JoinResult{Init: &init}
	}

	frames := s.framesFromLocked(from)
	resume := protocol.ResumeOK{
		T:    "resume",
		SID:  string(s.id),
		From: from,
		To:   s.nextSeq,
	}
	if len(diagSnapshot) > 0 {
		resume.Errors = cloneServerErrors(diagSnapshot)
	}
	s.mu.Unlock()

	return JoinResult{Resume: &resume, Frames: frames}
}

// Ack updates the session with the latest acknowledged sequence and prunes
// replay buffers when possible.
func (s *LiveSession) Ack(seq int) {
	s.mu.Lock()
	if seq > s.lastAck {
		s.lastAck = seq
		s.pruneAckedLocked()
		s.touchLocked()
	}
	s.mu.Unlock()
}

// DispatchEvent routes a client event through the component tree, performing
// idempotency checks on the provided client sequence.
func (s *LiveSession) DispatchEvent(id handlers.ID, ev handlers.Event, clientSeq int) error {
	if !s.acceptClientEvent(clientSeq) {
		return nil
	}
	if err := s.component.DispatchEvent(id, ev); err != nil {
		return err
	}
	s.refreshSnapshot()
	return nil
}

// Flush applies pending state changes and updates the session snapshot.
func (s *LiveSession) Flush() error {
	if err := s.component.Flush(); err != nil {
		return err
	}
	s.refreshSnapshot()
	return nil
}

// ReportDiagnostic satisfies the DiagnosticReporter interface and records runtime issues.
func (s *LiveSession) ReportDiagnostic(diag Diagnostic) {
	if s == nil {
		return
	}
	var (
		transport Transport
		payload   protocol.ServerError
		send      bool
	)
	s.mu.Lock()
	s.diagnostics = append(s.diagnostics, diag)
	if len(s.diagnostics) > defaultDiagnosticHistory {
		s.diagnostics = s.diagnostics[len(s.diagnostics)-defaultDiagnosticHistory:]
	}
	if s.devMode && s.transport != nil {
		transport = s.transport
		payload = diag.ToServerError(s.id)
		send = true
	}
	s.mu.Unlock()

	if send && transport != nil {
		_ = transport.SendServerError(payload)
	}
}

// SendFrame records and delivers a frame to the client transport.
func (s *LiveSession) SendFrame(frame protocol.Frame) error {
	s.mu.Lock()

	if frame.Seq <= 0 {
		frame.Seq = s.nextSeq
		s.nextSeq++
	} else if frame.Seq >= s.nextSeq {
		s.nextSeq = frame.Seq + 1
	}
	if frame.T == "" {
		frame.T = "frame"
	}
	if frame.SID == "" {
		frame.SID = string(s.id)
	}
	if frame.Ver == 0 {
		frame.Ver = s.version
	}

	renderDuration := time.Duration(frame.Metrics.RenderMs * 1e6)
	effectDuration := time.Duration(frame.Metrics.EffectsMs * 1e6)
	maxEffectDuration := time.Duration(frame.Metrics.MaxEffectMs * 1e6)
	recordFrameMetrics(FrameRecord{
		SessionID:         s.id,
		Sequence:          frame.Seq,
		Ops:               frame.Metrics.Ops,
		Effects:           len(frame.Effects),
		Nav:               frame.Nav != nil,
		RenderDuration:    renderDuration,
		EffectDuration:    effectDuration,
		MaxEffectDuration: maxEffectDuration,
		SlowEffects:       frame.Metrics.SlowEffects,
	})
	s.appendFrameLocked(frame)
	s.touchLocked()

	transport := s.transport
	s.mu.Unlock()

	if transport != nil {
		return transport.SendFrame(frame)
	}
	return nil
}

// AttachTransport installs or replaces the outbound transport for this session.
func (s *LiveSession) AttachTransport(t Transport) {
	var joins []protocol.PubsubControl

	s.mu.Lock()
	s.transport = t
	if t != nil {
		s.touchLocked()
		if len(s.pubsubCounts) > 0 {
			topics := make([]protocol.PubsubControl, 0, len(s.pubsubCounts))
			for topic, count := range s.pubsubCounts {
				if count > 0 {
					topics = append(topics, protocol.PubsubControl{Op: "join", Topic: topic})
				}
			}
			joins = topics
		}
	}
	s.mu.Unlock()

	for _, ctrl := range joins {
		_ = t.SendPubsubControl(ctrl)
	}
}

// DetachTransport clears the active transport if it matches the provided one.
func (s *LiveSession) DetachTransport(t Transport) {
	s.mu.Lock()
	if s.transport == t {
		s.transport = nil
	}
	s.mu.Unlock()
}

// ID returns the session identifier.
func (s *LiveSession) ID() SessionID { return s.id }

// Version returns the current session epoch.
func (s *LiveSession) Version() int { return s.version }

// SetVersion updates the session epoch.
func (s *LiveSession) SetVersion(v int) { s.version = v }

// RenderRoot renders the root component tree.
func (s *LiveSession) RenderRoot() h.Node {
	if s.component == nil {
		return nil
	}
	return s.component.RenderNode()
}

// Metadata returns the metadata captured during the last render.
func (s *LiveSession) Metadata() *Meta {
	if s.component == nil {
		return nil
	}
	return s.component.Metadata()
}

// Prev exposes the previous structured render.
func (s *LiveSession) Prev() render.Structured { return s.component.prev }

// SetPrev overrides the previous structured render state.
func (s *LiveSession) SetPrev(prev render.Structured) { s.component.prev = prev }

// Registry returns the handler registry backing this session.
func (s *LiveSession) Registry() handlers.Registry { return s.component.Registry() }

// MarkDirty schedules the root component for re-render.
func (s *LiveSession) MarkDirty() {
	if s.component == nil || s.component.root == nil {
		return
	}
	s.component.markDirty(s.component.root)
}

// ComponentSession exposes the underlying component session for integrations.
func (s *LiveSession) ComponentSession() *ComponentSession {
	if s == nil {
		return nil
	}
	return s.component
}

// Dirty reports whether the component session has pending work.
func (s *LiveSession) Dirty() bool {
	if s.component == nil {
		return false
	}
	return s.component.Dirty()
}

// DeliverPubsub injects a pub/sub message into the component session.
func (s *LiveSession) DeliverPubsub(topic string, payload []byte, meta map[string]string) {
	if s == nil || s.component == nil {
		return
	}
	s.component.deliverPubsub(topic, payload, meta)
}

func (s *LiveSession) pubsubSubscribed(topic string) {
	if s == nil || topic == "" {
		return
	}

	s.mu.Lock()
	if s.pubsubCounts == nil {
		s.pubsubCounts = make(map[string]int)
	}
	prev := s.pubsubCounts[topic]
	s.pubsubCounts[topic] = prev + 1
	transport := s.transport
	s.mu.Unlock()

	if prev == 0 && transport != nil {
		_ = transport.SendPubsubControl(protocol.PubsubControl{Op: "join", Topic: topic})
	}
}

func (s *LiveSession) pubsubUnsubscribed(topic string) {
	if s == nil || topic == "" {
		return
	}

	s.mu.Lock()
	if s.pubsubCounts == nil {
		s.mu.Unlock()
		return
	}
	count := s.pubsubCounts[topic]
	shouldLeave := count == 1
	if count <= 1 {
		delete(s.pubsubCounts, topic)
	} else {
		s.pubsubCounts[topic] = count - 1
	}
	transport := s.transport
	s.mu.Unlock()

	if shouldLeave && transport != nil {
		_ = transport.SendPubsubControl(protocol.PubsubControl{Op: "leave", Topic: topic})
	}
}

// Location returns the current router location tracked for the session.
func (s *LiveSession) Location() Location {
	s.mu.Lock()
	defer s.mu.Unlock()
	return copyLocation(s.loc)
}

// SetLocation updates the tracked location and snapshot metadata.
func (s *LiveSession) SetLocation(path, query string) bool {
	return s.SetRoute(path, query, nil)
}

// SetRoute updates the tracked location, query string, and matched params.
func (s *LiveSession) SetRoute(path, query string, params map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.setRouteLocked(path, query, params)
}

// Expired reports whether the session surpassed its inactivity TTL.
func (s *LiveSession) Expired() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if time.Duration(s.ttl.Load()) <= 0 {
		return false
	}
	return s.now().Sub(s.updatedAt) > time.Duration(s.ttl.Load())
}

// SnapshotSeq returns the sequence assigned to the most recent init payload.
func (s *LiveSession) SnapshotSeq() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastInitSeq
}

func (s *LiveSession) refreshSnapshot() {
	structured := s.component.prev
	meta := s.component.Metadata()
	s.mu.Lock()
	s.snapshot = buildSnapshot(structured, s.loc, meta)
	s.touchLocked()
	s.mu.Unlock()
}

func (s *LiveSession) currentMetadata() *Meta {
	s.mu.Lock()
	defer s.mu.Unlock()
	return CloneMeta(s.snapshot.Metadata)
}

func (s *LiveSession) setRouteLocked(path, query string, params map[string]string) bool {
	copied := cloneParams(params)
	if s.loc.Path == path && s.loc.Query == query && paramsEqual(s.loc.Params, copied) {
		return false
	}
	s.loc.Path = path
	s.loc.Query = query
	s.loc.Params = copied
	s.snapshot.Location = protocol.Location{Path: path, Query: query}
	s.touchLocked()
	return true
}

func (s *LiveSession) acceptClientEvent(seq int) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if seq <= 0 {
		s.touchLocked()
		return true
	}
	if seq <= s.clientSeq {
		s.touchLocked()
		return false
	}
	s.clientSeq = seq
	s.touchLocked()
	return true
}

func (s *LiveSession) enqueueFrameEffect(effect any) {
	if s == nil || effect == nil {
		return
	}
	s.mu.Lock()
	s.pendingEffects = append(s.pendingEffects, effect)
	s.mu.Unlock()
}

func (s *LiveSession) dequeueFrameEffects() []any {
	s.mu.Lock()
	effects := s.pendingEffects
	if len(effects) > 0 {
		s.pendingEffects = nil
	}
	s.mu.Unlock()
	if len(effects) == 0 {
		return nil
	}
	return append([]any(nil), effects...)
}

func (s *LiveSession) enqueueMetadataEffect(effect *MetadataEffect) {
	if effect == nil {
		return
	}
	s.enqueueFrameEffect(effect)
}

func (s *LiveSession) onPatch(ops []diff.Op) error {
	frame := protocol.Frame{
		Delta:   protocol.FrameDelta{Statics: false},
		Patch:   append([]diff.Op(nil), ops...),
		Metrics: protocol.FrameMetrics{Ops: len(ops)},
	}
	if effects := s.dequeueFrameEffects(); len(effects) > 0 {
		frame.Effects = append(frame.Effects, effects...)
	}
	if s.component != nil && s.component.pendingNav != nil {
		frame.Nav = s.component.pendingNav
		s.component.pendingNav = nil
	}
	if s.component != nil && s.component.pendingMetrics != nil {
		frame.Metrics = *s.component.pendingMetrics
		s.component.pendingMetrics = nil
	}
	return s.SendFrame(frame)
}

func (s *LiveSession) appendFrameLocked(frame protocol.Frame) {
	s.frames = append(s.frames, frame)
	if s.frameCap > 0 && len(s.frames) > s.frameCap {
		drop := len(s.frames) - s.frameCap
		newLen := copy(s.frames, s.frames[drop:])
		for i := newLen; i < len(s.frames); i++ {
			s.frames[i] = protocol.Frame{}
		}
		s.frames = s.frames[:newLen]
	}
}

func (s *LiveSession) pruneAckedLocked() {
	if len(s.frames) == 0 {
		return
	}
	idx := sort.Search(len(s.frames), func(i int) bool {
		return s.frames[i].Seq > s.lastAck
	})
	if idx <= 0 {
		return
	}
	if idx >= len(s.frames) {
		for i := range s.frames {
			s.frames[i] = protocol.Frame{}
		}
		s.frames = s.frames[:0]
		return
	}
	newLen := copy(s.frames, s.frames[idx:])
	for i := newLen; i < len(s.frames); i++ {
		s.frames[i] = protocol.Frame{}
	}
	s.frames = s.frames[:newLen]
}

func (s *LiveSession) framesFromLocked(start int) []protocol.Frame {
	if len(s.frames) == 0 {
		return nil
	}
	out := make([]protocol.Frame, 0, len(s.frames))
	for _, frame := range s.frames {
		if frame.Seq < start {
			continue
		}
		out = append(out, frame)
	}
	return out
}

func (s *LiveSession) buildInitLocked(errors []protocol.ServerError) protocol.Init {
	init := protocol.Init{
		T:        "init",
		SID:      string(s.id),
		Ver:      s.version,
		S:        append([]string(nil), s.snapshot.Statics...),
		D:        cloneDynamics(s.snapshot.Dynamics),
		Slots:    cloneSlots(s.snapshot.Slots),
		Handlers: cloneHandlers(s.snapshot.Handlers),
		Location: s.snapshot.Location,
		Seq:      s.nextSeq,
	}
	if len(errors) > 0 {
		init.Errors = cloneServerErrors(errors)
	}
	s.lastInitSeq = init.Seq
	s.nextSeq++
	s.hasInit = true
	return init
}

// BuildBoot constructs a boot payload using the current snapshot and rendered HTML.
func (s *LiveSession) BuildBoot(html string) protocol.Boot {
	s.mu.Lock()
	diagSnapshot := s.diagnosticsSnapshotLocked()
	init := s.buildInitLocked(diagSnapshot)
	s.mu.Unlock()

	boot := protocol.Boot{
		T:        "boot",
		SID:      init.SID,
		Ver:      init.Ver,
		Seq:      init.Seq,
		HTML:     html,
		S:        append([]string(nil), init.S...),
		D:        cloneDynamics(init.D),
		Slots:    cloneSlots(init.Slots),
		Handlers: cloneHandlers(init.Handlers),
		Location: init.Location,
	}
	if boot.T == "" {
		boot.T = "boot"
	}
	if s.clientConfig != nil {
		boot.Client = cloneClientConfig(s.clientConfig)
	}
	if len(diagSnapshot) > 0 {
		boot.Errors = cloneServerErrors(diagSnapshot)
	}
	return boot
}

// Recover attempts to clear runtime error state and resume rendering.
func (s *LiveSession) Recover() error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	if !s.devMode {
		return errors.New("runtime: recovery only available in dev mode")
	}
	if s.component == nil {
		return errors.New("runtime: session has no component")
	}
	if !s.component.Reset() {
		return nil
	}
	return s.Flush()
}

func (s *LiveSession) touchLocked() {
	s.updatedAt = s.now()
	observers := append([]func(time.Time){}, s.touchObservers...)
	for _, cb := range observers {
		if cb != nil {
			cb(s.updatedAt)
		}
	}
}

func copyLocation(loc Location) Location {
	out := Location{Path: loc.Path, Query: loc.Query}
	out.Params = cloneParams(loc.Params)
	return out
}

func cloneParams(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneClientConfig(src *protocol.ClientConfig) *protocol.ClientConfig {
	if src == nil {
		return nil
	}
	clone := *src
	return &clone
}

func paramsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			return false
		}
	}
	return true
}

type snapshot struct {
	Statics  []string
	Dynamics []protocol.DynamicSlot
	Slots    []protocol.SlotMeta
	Handlers map[string]protocol.HandlerMeta
	Location protocol.Location
	Metadata *Meta
}

func buildSnapshot(structured render.Structured, loc Location, meta *Meta) snapshot {
	statics := append([]string(nil), structured.S...)
	dynamics := encodeDynamics(structured.D)
	slots := make([]protocol.SlotMeta, len(dynamics))
	for i := range slots {
		slots[i] = protocol.SlotMeta{AnchorID: i}
	}
	handlers := extractHandlerMeta(structured)
	protoLoc := protocol.Location{Path: loc.Path, Query: loc.Query}
	return snapshot{
		Statics:  statics,
		Dynamics: dynamics,
		Slots:    slots,
		Handlers: handlers,
		Location: protoLoc,
		Metadata: CloneMeta(meta),
	}
}

func encodeDynamics(dynamics []render.Dyn) []protocol.DynamicSlot {
	if len(dynamics) == 0 {
		return nil
	}
	out := make([]protocol.DynamicSlot, len(dynamics))
	for i, dyn := range dynamics {
		slot := protocol.DynamicSlot{}
		switch dyn.Kind {
		case render.DynText:
			slot.Kind = "text"
			slot.Text = dyn.Text
		case render.DynAttrs:
			slot.Kind = "attrs"
			if len(dyn.Attrs) > 0 {
				attrs := make(map[string]string, len(dyn.Attrs))
				for k, v := range dyn.Attrs {
					attrs[k] = v
				}
				slot.Attrs = attrs
			}
		case render.DynList:
			slot.Kind = "list"
			if len(dyn.List) > 0 {
				rows := make([]protocol.ListRow, len(dyn.List))
				for j, row := range dyn.List {
					rows[j] = protocol.ListRow{Key: row.Key}
					if len(row.Slots) > 0 {
						rows[j].Slots = append([]int(nil), row.Slots...)
					}
				}
				slot.List = rows
			}
		default:
			slot.Kind = "unknown"
		}
		out[i] = slot
	}
	return out
}

func (s *LiveSession) diagnosticsSnapshotLocked() []protocol.ServerError {
	if !s.devMode || len(s.diagnostics) == 0 {
		return nil
	}
	out := make([]protocol.ServerError, 0, len(s.diagnostics))
	for _, diag := range s.diagnostics {
		out = append(out, diag.ToServerError(s.id))
	}
	return out
}

func cloneServerErrors(src []protocol.ServerError) []protocol.ServerError {
	if len(src) == 0 {
		return nil
	}
	out := make([]protocol.ServerError, len(src))
	for i, err := range src {
		out[i] = err
		if err.Details != nil {
			details := *err.Details
			if err.Details.Metadata != nil {
				details.Metadata = cloneDiagnosticMetadata(err.Details.Metadata)
			}
			out[i].Details = &details
		}
	}
	return out
}

func extractHandlerMeta(structured render.Structured) map[string]protocol.HandlerMeta {
	handlers := map[string]protocol.HandlerMeta{}
	for _, dyn := range structured.D {
		if dyn.Kind != render.DynAttrs {
			continue
		}
		for attr, val := range dyn.Attrs {
			if val == "" || !strings.HasPrefix(attr, "data-on") {
				continue
			}
			event := strings.TrimPrefix(attr, "data-on")
			handlers[val] = protocol.HandlerMeta{Event: event}
		}
	}
	if len(handlers) == 0 {
		return nil
	}
	return handlers
}

func cloneDynamics(dynamics []protocol.DynamicSlot) []protocol.DynamicSlot {
	if len(dynamics) == 0 {
		return nil
	}
	out := make([]protocol.DynamicSlot, len(dynamics))
	for i, dyn := range dynamics {
		slot := dyn
		if len(dyn.Attrs) > 0 {
			attrs := make(map[string]string, len(dyn.Attrs))
			for k, v := range dyn.Attrs {
				attrs[k] = v
			}
			slot.Attrs = attrs
		}
		if len(dyn.List) > 0 {
			rows := make([]protocol.ListRow, len(dyn.List))
			for j, row := range dyn.List {
				rows[j] = protocol.ListRow{Key: row.Key}
				if len(row.Slots) > 0 {
					rows[j].Slots = append([]int(nil), row.Slots...)
				}
			}
			slot.List = rows
		}
		out[i] = slot
	}
	return out
}

func cloneSlots(slots []protocol.SlotMeta) []protocol.SlotMeta {
	if len(slots) == 0 {
		return nil
	}
	out := make([]protocol.SlotMeta, len(slots))
	copy(out, slots)
	return out
}

func cloneHandlers(handlers map[string]protocol.HandlerMeta) map[string]protocol.HandlerMeta {
	if len(handlers) == 0 {
		return nil
	}
	out := make(map[string]protocol.HandlerMeta, len(handlers))
	for k, v := range handlers {
		out[k] = v
	}
	return out
}
