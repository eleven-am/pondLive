package runtime

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/go/internal/diff"
	"github.com/eleven-am/pondlive/go/internal/handlers"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/render"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
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
	SendUploadControl(protocol.UploadControl) error
}

// LiveSession wires a component tree to the PondSocket protocol and tracks
// sufficient state to resume clients after reconnects.
type LiveSession struct {
	id      SessionID
	version int

	component *ComponentSession

	header *headerState

	mu        sync.Mutex
	lifecycle *sessionLifecycle
	loc       SessionLocation

	frameCap int

	nextSeq     int
	lastInitSeq int
	lastAck     int
	clientSeq   int

	frames []protocol.Frame

	snapshot snapshot

	pendingEffects []any

	cookieBatches map[string]cookieBatch
	cookieCounter atomic.Uint64

	transport Transport
	devMode   bool

	pubsubCounts map[string]int

	diagnostics []Diagnostic

	hasInit bool

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
		lifecycle: newSessionLifecycle(time.Now, defaultSessionTTL),
		loc: SessionLocation{
			Path:   "/",
			Query:  "",
			Params: map[string]string{},
		},
		nextSeq:      1,
		pubsubCounts: make(map[string]int),
	}

	component.setOwner(session)

	header := newHeaderState()
	session.header = header
	provideHeaderState(component, header)

	effectiveConfig := mergeLiveSessionConfig(defaultLiveSessionConfig(), cfg)
	session.transport = effectiveConfig.Transport
	session.frameCap = effectiveConfig.FrameHistory
	session.lifecycle.setClock(effectiveConfig.Clock)
	if effectiveConfig.DevMode != nil {
		session.devMode = *effectiveConfig.DevMode
	}
	if effectiveConfig.TTL > 0 {
		session.lifecycle.setTTL(effectiveConfig.TTL)
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

func (s *LiveSession) headerState() *headerState {
	if s == nil {
		return newHeaderState()
	}
	if s.header == nil {
		s.header = newHeaderState()
		if comp := s.ComponentSession(); comp != nil {
			provideHeaderState(comp, s.header)
		}
	}
	return s.header
}

func (s *LiveSession) hasPendingCookieMutations() bool {
	if s == nil {
		return false
	}
	state := s.headerState()
	if state != nil && state.hasCookieMutations() {
		return true
	}
	s.mu.Lock()
	pending := len(s.cookieBatches) > 0
	s.mu.Unlock()
	return pending
}

// HeaderState exposes the header state tracked for the session.
func (s *LiveSession) HeaderState() HeaderState {
	if s == nil {
		return noopHeaderState{}
	}
	state := s.headerState()
	if state == nil {
		return noopHeaderState{}
	}
	return state
}

// MergeHTTPRequest records header and cookie information from the initial HTTP request.
func (s *LiveSession) MergeHTTPRequest(r *http.Request) {
	if s == nil || r == nil {
		return
	}
	state := s.headerState()
	state.mergeRequest(r)
	if comp := s.ComponentSession(); comp != nil {
		provideHeaderState(comp, state)
	}
}

// MergeConnectionState updates the tracked header state with information from the websocket connection.
func (s *LiveSession) MergeConnectionState(headers http.Header, cookies []*http.Cookie) {
	if s == nil {
		return
	}
	state := s.headerState()
	state.mergeHeaders(headers)
	state.mergeCookies(cookies)
	if comp := s.ComponentSession(); comp != nil {
		provideHeaderState(comp, state)
	}
}

// TTL returns the inactivity timeout configured for the session.
func (s *LiveSession) TTL() time.Duration {
	if s == nil || s.lifecycle == nil {
		return 0
	}
	return s.lifecycle.ttlDuration()
}

// AddTouchObserver registers a callback invoked when the session refreshes its last touched timestamp.
// The returned function removes the observer.
func (s *LiveSession) AddTouchObserver(cb func(time.Time)) func() {
	if cb == nil {
		return func() {}
	}
	s.mu.Lock()
	var idx int
	if s.lifecycle == nil {
		s.lifecycle = newSessionLifecycle(time.Now, defaultSessionTTL)
	}
	idx = s.lifecycle.addObserver(cb)
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		if s.lifecycle != nil {
			s.lifecycle.removeObserver(idx)
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
		if errors.Is(err, ErrFlushInProgress) {
			return nil
		}
		return err
	}
	s.refreshSnapshot()
	return nil
}

// Flush applies pending state changes and updates the session snapshot.
func (s *LiveSession) Flush() error {
	if err := s.component.Flush(); err != nil {
		if errors.Is(err, ErrFlushInProgress) {
			return nil
		}
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

func (s *LiveSession) flushAsync() {
	if s == nil {
		return
	}
	go func() {
		if err := s.Flush(); err != nil {
			if diag, ok := AsDiagnosticError(err); ok {
				s.ReportDiagnostic(diag)
				return
			}
			s.ReportDiagnostic(Diagnostic{
				Phase:      "async_flush",
				Message:    fmt.Sprintf("live: async flush failed: %v", err),
				Metadata:   map[string]any{"error": err.Error()},
				CapturedAt: time.Now(),
			})
		}
	}()
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

// HandleUploadMessage processes lifecycle updates emitted by the client uploader.
func (s *LiveSession) HandleUploadMessage(msg protocol.UploadClient) error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	comp := s.ComponentSession()
	if comp == nil {
		return errors.New("runtime: session has no component")
	}
	switch msg.Op {
	case "change":
		if msg.Meta != nil {
			meta := FileMeta{Name: msg.Meta.Name, Size: msg.Meta.Size, Type: msg.Meta.Type}
			comp.HandleUploadChange(msg.ID, meta)
		}
	case "progress":
		comp.HandleUploadProgress(msg.ID, msg.Loaded, msg.Total)
	case "error":
		errMsg := msg.Error
		if strings.TrimSpace(errMsg) == "" {
			errMsg = "upload failed"
		}
		comp.HandleUploadError(msg.ID, errors.New(errMsg))
	case "cancelled":
		comp.HandleUploadCancelled(msg.ID)
	default:
		return nil
	}
	if !comp.Dirty() {
		return nil
	}
	return s.Flush()
}

// CancelUpload requests the client abort an in-flight upload and updates local state.
func (s *LiveSession) CancelUpload(id string) error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	comp := s.ComponentSession()
	if comp != nil {
		comp.HandleUploadCancelled(id)
	}
	if comp != nil && comp.Dirty() {
		if err := s.Flush(); err != nil {
			return err
		}
	}
	if s.transport == nil || id == "" {
		return nil
	}
	ctrl := protocol.UploadControl{T: "upload", SID: string(s.id), ID: id, Op: "cancel"}
	return s.transport.SendUploadControl(ctrl)
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
func (s *LiveSession) Location() SessionLocation {
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
	if s.lifecycle == nil {
		return false
	}
	if s.lifecycle.ttlDuration() <= 0 {
		return false
	}
	return s.lifecycle.expired()
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

func (s *LiveSession) dequeueCookieEffect() *CookieEffect {
	state := s.headerState()
	if state == nil {
		return nil
	}
	batch := state.drainCookieMutations()
	if batch.Empty() {
		return nil
	}
	return s.registerCookieBatch(batch)
}

func (s *LiveSession) onPatch(ops []diff.Op) error {
	var template *templateUpdate
	if s.component != nil {
		template = s.component.consumeTemplateUpdate()
	}
	frame := protocol.Frame{
		Delta:   protocol.FrameDelta{Statics: false},
		Patch:   append([]diff.Op(nil), ops...),
		Metrics: protocol.FrameMetrics{Ops: len(ops)},
	}
	if effects := s.dequeueFrameEffects(); len(effects) > 0 {
		frame.Effects = append(frame.Effects, effects...)
	}
	if cookieEffect := s.dequeueCookieEffect(); cookieEffect != nil {
		frame.Effects = append(frame.Effects, cookieEffect)
	}
	if s.component != nil && s.component.pendingNav != nil {
		frame.Nav = s.component.pendingNav
		s.component.pendingNav = nil
	}
	if s.component != nil && s.component.pendingMetrics != nil {
		frame.Metrics = *s.component.pendingMetrics
		s.component.pendingMetrics = nil
	}
	if template != nil {
		snap := buildSnapshot(template.structured, s.loc, s.component.Metadata())
		s.snapshot = snap
		boot := protocol.Boot{
			T:        "boot",
			SID:      string(s.id),
			Ver:      s.version,
			Seq:      s.nextSeq,
			HTML:     template.html,
			S:        append([]string(nil), snap.Statics...),
			D:        append([]protocol.DynamicSlot(nil), snap.Dynamics...),
			Slots:    cloneSlots(snap.Slots),
			Handlers: cloneHandlers(snap.Handlers),
			Location: snap.Location,
		}
		frame.Delta.Statics = true
		frame.Delta.Slots = cloneSlots(snap.Slots)
		frame.Patch = nil
		frame.Effects = append(frame.Effects, map[string]any{"type": "boot", "boot": boot})
	}
	return s.SendFrame(frame)
}

func (s *LiveSession) registerCookieBatch(batch CookieBatch) *CookieEffect {
	if s == nil || batch.Empty() {
		return nil
	}
	token := s.nextCookieToken()
	if token == "" {
		return nil
	}
	effect := newCookieEffect(CookieEndpointPath, string(s.id), token)
	if effect == nil {
		return nil
	}
	clone := cloneCookieBatch(batch)
	s.mu.Lock()
	if s.cookieBatches == nil {
		s.cookieBatches = make(map[string]cookieBatch)
	}
	s.cookieBatches[token] = cookieBatch{Mutations: clone}
	s.mu.Unlock()
	return effect
}

func (s *LiveSession) nextCookieToken() string {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err == nil {
		return base64.RawURLEncoding.EncodeToString(buf)
	}
	fallback := s.cookieCounter.Add(1)
	ts := time.Now().UnixNano()
	if s.lifecycle != nil {
		ts = s.lifecycle.now().UnixNano()
	}
	return strconv.FormatInt(ts, 36) + strconv.FormatUint(fallback, 36)
}

// ConsumeCookieBatch retrieves and clears a pending cookie batch identified by the provided token.
func (s *LiveSession) ConsumeCookieBatch(token string) (CookieBatch, bool) {
	if s == nil {
		return CookieBatch{}, false
	}
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return CookieBatch{}, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.cookieBatches) == 0 {
		return CookieBatch{}, false
	}
	batch, ok := s.cookieBatches[trimmed]
	if !ok {
		return CookieBatch{}, false
	}
	delete(s.cookieBatches, trimmed)
	return cloneCookieBatch(batch.Mutations), true
}

func cloneCookieBatch(in CookieBatch) CookieBatch {
	out := CookieBatch{}
	if len(in.Set) > 0 {
		out.Set = make([]*http.Cookie, 0, len(in.Set))
		for _, ck := range in.Set {
			out.Set = append(out.Set, cloneCookie(ck))
		}
	}
	if len(in.Delete) > 0 {
		out.Delete = append(out.Delete, in.Delete...)
	}
	return out
}

type cookieBatch struct {
	Mutations CookieBatch
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
	if s.lifecycle == nil {
		s.lifecycle = newSessionLifecycle(time.Now, defaultSessionTTL)
	}
	s.lifecycle.touch()
}

func copyLocation(loc SessionLocation) SessionLocation {
	out := SessionLocation{Path: loc.Path, Query: loc.Query}
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

func buildSnapshot(structured render.Structured, loc SessionLocation, meta *Meta) snapshot {
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
			suffix := strings.TrimPrefix(attr, "data-on")
			if suffix == "" || strings.HasSuffix(suffix, "-listen") || strings.HasSuffix(suffix, "-props") {
				continue
			}
			event := suffix
			if event == "" {
				continue
			}
			id := val
			if id == "" {
				continue
			}
			meta := handlers[id]
			if meta.Event == "" {
				meta.Event = event
			} else if meta.Event != event {
				meta.Listen = appendIfMissing(meta.Listen, event)
			}
			handlers[id] = meta
		}

		for attr, raw := range dyn.Attrs {
			if raw == "" || !strings.HasPrefix(attr, "data-on") {
				continue
			}
			suffix := strings.TrimPrefix(attr, "data-on")
			if suffix == "" {
				continue
			}
			idx := strings.LastIndex(suffix, "-")
			if idx == -1 {
				continue
			}
			event := suffix[:idx]
			metaType := suffix[idx+1:]
			if event == "" || metaType == "" {
				continue
			}
			id := dyn.Attrs["data-on"+event]
			if id == "" {
				continue
			}
			meta := handlers[id]
			switch metaType {
			case "listen":
				meta.Listen = appendFields(meta.Listen, raw, meta.Event)
			case "props":
				meta.Props = appendFields(meta.Props, raw, "")
			}
			handlers[id] = meta
		}
	}
	if len(handlers) == 0 {
		return nil
	}
	return handlers
}

func appendIfMissing(list []string, value string) []string {
	if value == "" {
		return list
	}
	for _, existing := range list {
		if existing == value {
			return list
		}
	}
	return append(list, value)
}

func appendFields(list []string, raw string, skip string) []string {
	if raw == "" {
		return list
	}
	for _, token := range strings.Fields(raw) {
		if token == "" || token == skip {
			continue
		}
		list = appendIfMissing(list, token)
	}
	if len(list) == 0 {
		return nil
	}
	return list
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
		meta := v
		if len(v.Listen) > 0 {
			meta.Listen = append([]string(nil), v.Listen...)
		}
		if len(v.Props) > 0 {
			meta.Props = append([]string(nil), v.Props...)
		}
		out[k] = meta
	}
	return out
}
