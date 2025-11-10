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
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/handlers"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/render"
)

const (
	defaultFrameHistory  = 64
	defaultSessionTTL    = 90 * time.Second
	defaultDOMGetTimeout = 5 * time.Second
)

var (
	errDOMGetNoTransport = errors.New("runtime: domget requires active transport")
	errDOMGetTimeout     = errors.New("runtime: domget timed out")
)

// Transport delivers messages to the client connection backing a session.
type Transport interface {
	SendInit(protocol.Init) error
	SendResume(protocol.ResumeOK) error
	SendFrame(protocol.Frame) error
	SendServerError(protocol.ServerError) error
	SendPubsubControl(protocol.PubsubControl) error
	SendUploadControl(protocol.UploadControl) error
	SendDOMRequest(protocol.DOMRequest) error
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

	domGetCounter atomic.Uint64
	domGetMu      sync.Mutex
	domGetPending map[string]chan domGetResult
	domGetTimeout time.Duration

	transport Transport
	devMode   bool

	pubsubCounts map[string]int

	diagnostics []Diagnostic

	hasInit bool

	clientConfig *protocol.ClientConfig
}

type domGetResult struct {
	values map[string]any
	err    error
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
		nextSeq:       1,
		pubsubCounts:  make(map[string]int),
		domGetTimeout: defaultDOMGetTimeout,
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
	session.snapshot = session.buildSnapshot(structured, session.loc, meta)
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
	s.failDOMRequests(errDOMGetNoTransport)
}

func (s *LiveSession) failDOMRequests(err error) {
	if s == nil {
		return
	}
	s.domGetMu.Lock()
	pending := s.domGetPending
	if len(pending) > 0 {
		s.domGetPending = nil
	}
	s.domGetMu.Unlock()
	if len(pending) == 0 {
		return
	}
	for _, ch := range pending {
		if ch == nil {
			continue
		}
		select {
		case ch <- domGetResult{err: err}:
		default:
		}
	}
}

func sanitizeSelectors(selectors []string) []string {
	if len(selectors) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(selectors))
	out := make([]string, 0, len(selectors))
	for _, sel := range selectors {
		trimmed := strings.TrimSpace(sel)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// DOMGet requests the provided selectors from the client for the given element ref.
func (s *LiveSession) DOMGet(ref string, selectors ...string) (map[string]any, error) {
	if s == nil {
		return nil, errors.New("runtime: session is nil")
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, errors.New("runtime: domget requires element ref")
	}
	props := sanitizeSelectors(selectors)
	if len(props) == 0 {
		return map[string]any{}, nil
	}

	s.mu.Lock()
	transport := s.transport
	timeout := s.domGetTimeout
	if timeout <= 0 {
		timeout = defaultDOMGetTimeout
	}
	s.mu.Unlock()

	if transport == nil {
		return nil, errDOMGetNoTransport
	}

	id := fmt.Sprintf("domget:%d", s.domGetCounter.Add(1))
	ch := make(chan domGetResult, 1)

	s.domGetMu.Lock()
	if s.domGetPending == nil {
		s.domGetPending = make(map[string]chan domGetResult)
	}
	s.domGetPending[id] = ch
	s.domGetMu.Unlock()

	req := protocol.DOMRequest{
		ID:    id,
		Ref:   ref,
		Props: append([]string(nil), props...),
	}

	if err := transport.SendDOMRequest(req); err != nil {
		s.domGetMu.Lock()
		delete(s.domGetPending, id)
		s.domGetMu.Unlock()
		return nil, err
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-ch:
		if result.err != nil {
			return nil, result.err
		}
		if result.values == nil {
			return map[string]any{}, nil
		}
		return result.values, nil
	case <-timer.C:
		s.domGetMu.Lock()
		delete(s.domGetPending, id)
		s.domGetMu.Unlock()
		return nil, errDOMGetTimeout
	}
}

// HandleDOMResponse resolves a pending DOMGet request with data from the client.
func (s *LiveSession) HandleDOMResponse(resp protocol.DOMResponse) {
	if s == nil {
		return
	}
	if resp.ID == "" {
		return
	}

	s.domGetMu.Lock()
	ch, ok := s.domGetPending[resp.ID]
	if ok {
		delete(s.domGetPending, resp.ID)
	}
	s.domGetMu.Unlock()

	if !ok || ch == nil {
		return
	}

	result := domGetResult{}
	if resp.Error != "" {
		result.err = errors.New(resp.Error)
	} else {
		if len(resp.Values) > 0 {
			clone := make(map[string]any, len(resp.Values))
			for k, v := range resp.Values {
				clone[k] = v
			}
			result.values = clone
		} else {
			result.values = map[string]any{}
		}
	}

	select {
	case ch <- result:
	default:
	}
}

// ID returns the session identifier.
func (s *LiveSession) ID() SessionID { return s.id }

// Version returns the current session epoch.
func (s *LiveSession) Version() int { return s.version }

// SetVersion updates the session epoch.
func (s *LiveSession) SetVersion(v int) { s.version = v }

// RenderRoot renders the root component tree.
func (s *LiveSession) RenderRoot() dom.Node {
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
	s.snapshot = s.buildSnapshot(structured, s.loc, meta)
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
	var componentBoots []componentTemplateUpdate
	if s.component != nil {
		template = s.component.consumeTemplateUpdate()
		componentBoots = s.component.consumeComponentBoots()
	}
	frame := protocol.Frame{
		Delta:   protocol.FrameDelta{Statics: false},
		Patch:   append([]diff.Op(nil), ops...),
		Metrics: protocol.FrameMetrics{Ops: len(ops)},
	}
	var refDelta protocol.RefDelta
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
		previousRefs := s.snapshot.Refs
		snap := s.buildSnapshot(template.structured, s.loc, s.component.Metadata())
		refDelta = diffRefs(previousRefs, snap.Refs)
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
			Bindings: cloneBindingTable(snap.Bindings),
			Markers:  cloneMarkers(snap.Markers),
			Refs:     cloneRefs(snap.Refs),
			Location: snap.Location,
		}
		frame.Delta.Statics = true
		frame.Delta.Slots = cloneSlots(snap.Slots)
		frame.Patch = nil
		frame.Effects = append(frame.Effects, map[string]any{"type": "boot", "boot": boot})
	} else if s.component != nil {
		pending := s.component.pendingRefs
		s.component.pendingRefs = nil
		if pending != nil || len(s.snapshot.Refs) > 0 {
			refDelta = diffRefs(s.snapshot.Refs, pending)
		}
	}
	if hasRefDelta(refDelta) {
		frame.Refs = refDelta
	}
	if len(componentBoots) > 0 {
		if s.snapshot.Statics == nil {
			s.snapshot = s.buildSnapshot(s.component.prev, s.loc, s.component.Metadata())
		}
		s.ensureSnapshotStaticsOwned()
		for _, update := range componentBoots {
			if update.staticsRange.end <= len(s.snapshot.Statics) && update.staticsRange.start >= 0 && update.staticsRange.end-update.staticsRange.start == len(update.statics) {
				copy(s.snapshot.Statics[update.staticsRange.start:update.staticsRange.end], update.statics)
			}
			for idx, slot := range update.slots {
				anchor := slot.AnchorID
				if anchor < 0 {
					continue
				}
				if anchor >= len(s.snapshot.Dynamics) {
					extended := make([]protocol.DynamicSlot, anchor+1)
					copy(extended, s.snapshot.Dynamics)
					s.snapshot.Dynamics = extended
				}
				if anchor >= len(s.snapshot.Slots) {
					extended := make([]protocol.SlotMeta, anchor+1)
					copy(extended, s.snapshot.Slots)
					for i := len(s.snapshot.Slots); i < len(extended); i++ {
						extended[i].AnchorID = i
					}
					s.snapshot.Slots = extended
				}
				if idx < len(update.dynamics) {
					s.snapshot.Dynamics[anchor] = update.dynamics[idx]
				}
			}
			if len(update.handlersDel) > 0 && s.snapshot.Handlers != nil {
				for _, id := range update.handlersDel {
					delete(s.snapshot.Handlers, id)
				}
			}
			if len(update.handlersAdd) > 0 {
				if s.snapshot.Handlers == nil {
					s.snapshot.Handlers = make(map[string]protocol.HandlerMeta)
				}
				for id, meta := range update.handlersAdd {
					s.snapshot.Handlers[id] = meta
				}
			}
			if len(update.handlersAdd) > 0 {
				if frame.Handlers.Add == nil {
					frame.Handlers.Add = make(map[string]protocol.HandlerMeta)
				}
				for id, meta := range update.handlersAdd {
					frame.Handlers.Add[id] = meta
				}
			}
			if len(update.handlersDel) > 0 {
				frame.Handlers.Del = append(frame.Handlers.Del, update.handlersDel...)
			}
			if len(update.slots) > 0 {
				if s.snapshot.Bindings == nil && (len(update.bindings) > 0) {
					s.snapshot.Bindings = make(protocol.BindingTable)
				}
				for _, slotMeta := range update.slots {
					slot := slotMeta.AnchorID
					entries, ok := update.bindings[slot]
					if !ok || len(entries) == 0 {
						if s.snapshot.Bindings != nil {
							delete(s.snapshot.Bindings, slot)
						}
						continue
					}
					if s.snapshot.Bindings == nil {
						s.snapshot.Bindings = make(protocol.BindingTable)
					}
					s.snapshot.Bindings[slot] = copySlotBindings(entries)
				}
			}
			if len(update.markers) > 0 {
				if s.snapshot.Markers == nil {
					s.snapshot.Markers = make(map[string]protocol.ComponentMarker)
				}
				for id, marker := range update.markers {
					s.snapshot.Markers[id] = marker
				}
			}
			effect := map[string]any{
				"type":        "componentBoot",
				"componentId": update.id,
				"html":        update.html,
				"slots":       update.slots,
			}
			if len(update.listSlots) > 0 {
				effect["listSlots"] = update.listSlots
			}
			if update.bindings != nil {
				effect["bindings"] = update.bindings
			}
			if update.markers != nil {
				effect["markers"] = update.markers
			}
			frame.Effects = append(frame.Effects, effect)
		}
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
		Bindings: cloneBindingTable(s.snapshot.Bindings),
		Markers:  cloneMarkers(s.snapshot.Markers),
		Refs:     cloneRefs(s.snapshot.Refs),
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
		Bindings: cloneBindingTable(init.Bindings),
		Markers:  cloneMarkers(init.Markers),
		Refs:     cloneRefs(init.Refs),
		Location: init.Location,
	}
	if boot.T == "" {
		boot.T = "boot"
	}
	clientCfg := cloneClientConfig(s.clientConfig)
	if s.devMode {
		if clientCfg == nil {
			clientCfg = &protocol.ClientConfig{}
		}
		if clientCfg.Debug == nil {
			value := true
			clientCfg.Debug = &value
		}
	}
	if clientCfg != nil {
		boot.Client = clientCfg
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
	if src.Debug != nil {
		value := *src.Debug
		clone.Debug = &value
	}
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
	Statics      []string
	staticsOwned bool
	Dynamics     []protocol.DynamicSlot
	Slots        []protocol.SlotMeta
	Handlers     map[string]protocol.HandlerMeta
	Bindings     protocol.BindingTable
	Markers      map[string]protocol.ComponentMarker
	Refs         map[string]protocol.RefMeta
	Location     protocol.Location
	Metadata     *Meta
}

func (s *LiveSession) buildSnapshot(structured render.Structured, loc SessionLocation, meta *Meta) snapshot {
	statics := globalTemplateIntern.InternStatics(structured.S)
	dynamics := encodeDynamics(structured.D)
	slots := encodeSlotMeta(structured)
	handlers := extractHandlerMeta(structured)
	var refs map[string]protocol.RefMeta
	if ids := extractRefIDs(structured); len(ids) > 0 {
		if comp := s.ComponentSession(); comp != nil {
			refs = comp.snapshotRefs(ids)
		}
	}
	protoLoc := protocol.Location{Path: loc.Path, Query: loc.Query}
	return snapshot{
		Statics:      statics,
		Dynamics:     dynamics,
		Slots:        slots,
		Handlers:     handlers,
		Bindings:     encodeBindingTable(structured.Bindings),
		Markers:      encodeComponentMarkers(structured.Components),
		Refs:         refs,
		Location:     protoLoc,
		Metadata:     CloneMeta(meta),
		staticsOwned: false,
	}
}

func (s *LiveSession) ensureSnapshotStaticsOwned() {
	if s == nil {
		return
	}
	if s.snapshot.staticsOwned {
		return
	}
	if len(s.snapshot.Statics) == 0 {
		s.snapshot.staticsOwned = true
		return
	}
	s.snapshot.Statics = append([]string(nil), s.snapshot.Statics...)
	s.snapshot.staticsOwned = true
}

func encodeSlotMeta(structured render.Structured) []protocol.SlotMeta {
	if len(structured.D) == 0 {
		return nil
	}
	slots := make([]protocol.SlotMeta, len(structured.D))
	for i := range slots {
		slots[i].AnchorID = i
	}
	for _, anchor := range structured.Anchors {
		if anchor.Slot < 0 || anchor.Slot >= len(slots) {
			continue
		}
		entry := &slots[anchor.Slot]
		applyAnchorMetadata(entry, anchor)
	}
	for _, list := range structured.ListAnchors {
		if list.Slot < 0 || list.Slot >= len(slots) {
			continue
		}
		meta := protocol.ListAnchor{}
		if list.ComponentID != "" {
			meta.Component = list.ComponentID
		}
		switch {
		case len(list.ComponentPath) > 0:
			meta.Path = append([]int(nil), list.ComponentPath...)
		case list.ComponentID != "":
			meta.Path = []int{}
		case len(list.NodePath) > 0:
			meta.Path = append([]int(nil), list.NodePath...)
		case meta.Path == nil:
			meta.Path = []int{}
		}
		slots[list.Slot].List = &meta
	}
	return slots
}

func applyAnchorMetadata(dest *protocol.SlotMeta, anchor render.SlotAnchor) {
	if dest == nil {
		return
	}
	if anchor.ComponentID != "" {
		dest.Component = anchor.ComponentID
	}
	switch {
	case len(anchor.ComponentPath) > 0:
		dest.Path = append([]int(nil), anchor.ComponentPath...)
	case anchor.ComponentID != "":
		dest.Path = []int{}
	case len(anchor.NodePath) > 0:
		dest.Path = append([]int(nil), anchor.NodePath...)
	case dest.Path == nil:
		dest.Path = []int{}
	}
	if anchor.ChildIndex >= 0 {
		idx := anchor.ChildIndex
		dest.TextIndex = &idx
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
					if len(row.Anchors) > 0 {
						metas := make([]protocol.SlotMeta, len(row.Anchors))
						for idx, anchor := range row.Anchors {
							meta := protocol.SlotMeta{AnchorID: anchor.Slot}
							applyAnchorMetadata(&meta, anchor)
							metas[idx] = meta
						}
						rows[j].Slots = metas
					}
					if bindings := encodeRowBindingTable(row.Bindings); len(bindings) > 0 {
						rows[j].Bindings = bindings
					}
					if markers := encodeRowMarkers(row.Markers); len(markers) > 0 {
						rows[j].Markers = markers
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

func encodeBindingTable(bindings []render.HandlerBinding) protocol.BindingTable {
	if len(bindings) == 0 {
		return nil
	}
	table := make(protocol.BindingTable)
	for _, binding := range bindings {
		if binding.Slot < 0 || binding.Handler == "" {
			continue
		}
		entry := protocol.SlotBinding{
			Event:   binding.Event,
			Handler: binding.Handler,
		}
		if len(binding.Listen) > 0 {
			entry.Listen = append([]string(nil), binding.Listen...)
		}
		if len(binding.Props) > 0 {
			entry.Props = append([]string(nil), binding.Props...)
		}
		table[binding.Slot] = append(table[binding.Slot], entry)
	}
	if len(table) == 0 {
		return nil
	}
	return table
}

func encodeComponentMarkers(components map[string]render.ComponentSpan) map[string]protocol.ComponentMarker {
	if len(components) == 0 {
		return nil
	}
	out := make(map[string]protocol.ComponentMarker, len(components))
	for id, span := range components {
		marker := protocol.ComponentMarker{
			Start: span.StartIndex,
			End:   span.EndIndex,
		}
		if len(span.ContainerPath) > 0 {
			marker.Container = append([]int(nil), span.ContainerPath...)
		}
		out[id] = marker
	}
	return out
}

func encodeBindings(bindings []render.HandlerBinding) []protocol.SlotBinding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]protocol.SlotBinding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.Handler == "" {
			continue
		}
		entry := protocol.SlotBinding{
			Event:   binding.Event,
			Handler: binding.Handler,
		}
		if len(binding.Listen) > 0 {
			entry.Listen = append([]string(nil), binding.Listen...)
		}
		if len(binding.Props) > 0 {
			entry.Props = append([]string(nil), binding.Props...)
		}
		out = append(out, entry)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func encodeRowBindingTable(bindings []render.HandlerBinding) protocol.BindingTable {
	if len(bindings) == 0 {
		return nil
	}
	table := make(protocol.BindingTable)
	for _, binding := range bindings {
		if binding.Slot < 0 || binding.Handler == "" {
			continue
		}
		entry := protocol.SlotBinding{
			Event:   binding.Event,
			Handler: binding.Handler,
		}
		if len(binding.Listen) > 0 {
			entry.Listen = append([]string(nil), binding.Listen...)
		}
		if len(binding.Props) > 0 {
			entry.Props = append([]string(nil), binding.Props...)
		}
		table[binding.Slot] = append(table[binding.Slot], entry)
	}
	if len(table) == 0 {
		return nil
	}
	return table
}

func encodeRowMarkers(markers map[string]render.ComponentMarker) map[string]protocol.ComponentMarker {
	if len(markers) == 0 {
		return nil
	}
	out := make(map[string]protocol.ComponentMarker, len(markers))
	for id, marker := range markers {
		entry := protocol.ComponentMarker{
			Start: marker.StartIndex,
			End:   marker.EndIndex,
		}
		if len(marker.ContainerPath) > 0 {
			entry.Container = append([]int(nil), marker.ContainerPath...)
		}
		out[id] = entry
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

func extractRefIDs(structured render.Structured) []string {
	refs := map[string]struct{}{}
	for _, dyn := range structured.D {
		if dyn.Kind != render.DynAttrs {
			continue
		}
		if dyn.Attrs == nil {
			continue
		}
		id := strings.TrimSpace(dyn.Attrs["data-live-ref"])
		if id == "" {
			continue
		}
		refs[id] = struct{}{}
	}
	for _, attrs := range staticAttrMaps(structured.S) {
		id := strings.TrimSpace(attrs["data-live-ref"])
		if id == "" {
			continue
		}
		refs[id] = struct{}{}
	}
	if len(refs) == 0 {
		return nil
	}
	out := make([]string, 0, len(refs))
	for id := range refs {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func extractHandlerMeta(structured render.Structured) map[string]protocol.HandlerMeta {
	handlers := map[string]protocol.HandlerMeta{}

	if len(structured.Bindings) > 0 {
		for _, binding := range structured.Bindings {
			if binding.Handler == "" {
				continue
			}
			meta := handlers[binding.Handler]
			if event := binding.Event; event != "" {
				if meta.Event == "" {
					meta.Event = event
				} else if meta.Event != event {
					meta.Listen = appendIfMissing(meta.Listen, event)
				}
			}
			for _, evt := range binding.Listen {
				meta.Listen = appendIfMissing(meta.Listen, evt)
			}
			for _, prop := range binding.Props {
				meta.Props = appendIfMissing(meta.Props, prop)
			}
			handlers[binding.Handler] = meta
		}
		if len(handlers) > 0 {
			return handlers
		}
	}

	handlers = map[string]protocol.HandlerMeta{}
	for _, dyn := range structured.D {
		if dyn.Kind != render.DynAttrs {
			continue
		}
		mergeHandlerAttrs(handlers, dyn.Attrs)
	}
	for _, attrs := range staticAttrMaps(structured.S) {
		mergeHandlerAttrs(handlers, attrs)
	}
	if len(handlers) == 0 {
		return nil
	}
	return handlers
}

func mergeHandlerAttrs(handlers map[string]protocol.HandlerMeta, attrs map[string]string) {
	if len(attrs) == 0 {
		return
	}
	for attr, val := range attrs {
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
	for attr, raw := range attrs {
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
		id := attrs["data-on"+event]
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

func staticAttrMaps(statics []string) []map[string]string {
	if len(statics) == 0 {
		return nil
	}
	combined := strings.Join(statics, "")
	out := make([]map[string]string, 0)
	i := 0
	for i < len(combined) {
		if combined[i] != '<' {
			i++
			continue
		}
		if i+1 < len(combined) {
			next := combined[i+1]
			if next == '/' || next == '!' {
				i++
				continue
			}
		}
		j := i + 1
		for j < len(combined) && combined[j] != '>' && !isSpace(combined[j]) {
			j++
		}
		attrs := map[string]string{}
		k := j
		for k < len(combined) && combined[k] != '>' {
			for k < len(combined) && isSpace(combined[k]) {
				k++
			}
			if k >= len(combined) || combined[k] == '>' {
				break
			}
			nameStart := k
			for k < len(combined) && combined[k] != '=' && !isSpace(combined[k]) && combined[k] != '>' {
				k++
			}
			if k >= len(combined) || combined[k] != '=' {
				for k < len(combined) && combined[k] != '>' {
					k++
				}
				break
			}
			name := combined[nameStart:k]
			k++
			for k < len(combined) && isSpace(combined[k]) {
				k++
			}
			if k >= len(combined) || combined[k] != '"' {
				break
			}
			k++
			valueStart := k
			for k < len(combined) && combined[k] != '"' {
				k++
			}
			if k >= len(combined) {
				break
			}
			value := combined[valueStart:k]
			attrs[name] = value
			k++
		}
		if len(attrs) > 0 {
			out = append(out, attrs)
		}
		for i < len(combined) && combined[i] != '>' {
			i++
		}
		i++
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func isSpace(ch byte) bool {
	switch ch {
	case ' ', '\n', '\r', '\t':
		return true
	default:
		return false
	}
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
					rows[j].Slots = cloneSlots(row.Slots)
				}
				if len(row.Bindings) > 0 {
					copied := make(protocol.BindingTable, len(row.Bindings))
					for slot, bindings := range row.Bindings {
						copied[slot] = copySlotBindings(bindings)
					}
					if len(copied) > 0 {
						rows[j].Bindings = copied
					}
				}
				if len(row.Markers) > 0 {
					copiedMarkers := make(map[string]protocol.ComponentMarker, len(row.Markers))
					for id, marker := range row.Markers {
						copiedMarkers[id] = marker
					}
					if len(copiedMarkers) > 0 {
						rows[j].Markers = copiedMarkers
					}
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
	for i, slot := range slots {
		copied := slot
		switch {
		case len(slot.Path) > 0:
			copied.Path = append([]int(nil), slot.Path...)
		case slot.Path != nil:
			copied.Path = []int{}
		default:
			copied.Path = nil
		}
		if slot.TextIndex != nil {
			idx := *slot.TextIndex
			copied.TextIndex = &idx
		}
		if slot.List != nil {
			listCopy := *slot.List
			switch {
			case len(slot.List.Path) > 0:
				listCopy.Path = append([]int(nil), slot.List.Path...)
			case slot.List.Path != nil:
				listCopy.Path = []int{}
			default:
				listCopy.Path = nil
			}
			copied.List = &listCopy
		}
		out[i] = copied
	}
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

func cloneBindingTable(bindings protocol.BindingTable) protocol.BindingTable {
	if len(bindings) == 0 {
		return nil
	}
	out := make(protocol.BindingTable, len(bindings))
	for slot, entries := range bindings {
		out[slot] = copySlotBindings(entries)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneMarkers(markers map[string]protocol.ComponentMarker) map[string]protocol.ComponentMarker {
	if len(markers) == 0 {
		return nil
	}
	out := make(map[string]protocol.ComponentMarker, len(markers))
	for id, marker := range markers {
		clone := marker
		if len(marker.Container) > 0 {
			clone.Container = append([]int(nil), marker.Container...)
		}
		out[id] = clone
	}
	return out
}

func copySlotBindings(entries []protocol.SlotBinding) []protocol.SlotBinding {
	if entries == nil {
		return nil
	}
	if len(entries) == 0 {
		return []protocol.SlotBinding{}
	}
	out := make([]protocol.SlotBinding, len(entries))
	for i, entry := range entries {
		clone := entry
		if len(entry.Listen) > 0 {
			clone.Listen = append([]string(nil), entry.Listen...)
		}
		if len(entry.Props) > 0 {
			clone.Props = append([]string(nil), entry.Props...)
		}
		out[i] = clone
	}
	return out
}

func diffRefs(prev, next map[string]protocol.RefMeta) protocol.RefDelta {
	delta := protocol.RefDelta{}
	if len(prev) == 0 && len(next) == 0 {
		return delta
	}
	if len(next) > 0 {
		for id, meta := range next {
			if prevMeta, ok := prev[id]; !ok || !refMetaEqual(prevMeta, meta) {
				if delta.Add == nil {
					delta.Add = make(map[string]protocol.RefMeta)
				}
				delta.Add[id] = cloneRefMeta(meta)
			}
		}
	}
	if len(prev) > 0 {
		for id := range prev {
			if len(next) == 0 {
				delta.Del = append(delta.Del, id)
				continue
			}
			if _, ok := next[id]; !ok {
				delta.Del = append(delta.Del, id)
			}
		}
	}
	if len(delta.Del) > 1 {
		sort.Strings(delta.Del)
	}
	return delta
}

func hasRefDelta(delta protocol.RefDelta) bool {
	return len(delta.Add) > 0 || len(delta.Del) > 0
}

func cloneRefs(refs map[string]protocol.RefMeta) map[string]protocol.RefMeta {
	if len(refs) == 0 {
		return nil
	}
	out := make(map[string]protocol.RefMeta, len(refs))
	for id, meta := range refs {
		out[id] = cloneRefMeta(meta)
	}
	return out
}

func cloneRefMeta(meta protocol.RefMeta) protocol.RefMeta {
	clone := meta
	if len(meta.Events) > 0 {
		events := make(map[string]protocol.RefEventMeta, len(meta.Events))
		for event, evMeta := range meta.Events {
			events[event] = cloneRefEventMeta(evMeta)
		}
		clone.Events = events
	}
	return clone
}

func cloneRefEventMeta(meta protocol.RefEventMeta) protocol.RefEventMeta {
	clone := meta
	if len(meta.Listen) > 0 {
		clone.Listen = append([]string(nil), meta.Listen...)
	}
	if len(meta.Props) > 0 {
		clone.Props = append([]string(nil), meta.Props...)
	}
	return clone
}

func refMetaEqual(a, b protocol.RefMeta) bool {
	if a.Tag != b.Tag {
		return false
	}
	if len(a.Events) != len(b.Events) {
		return false
	}
	if len(a.Events) == 0 {
		return true
	}
	for event, metaA := range a.Events {
		metaB, ok := b.Events[event]
		if !ok {
			return false
		}
		if !refEventMetaEqual(metaA, metaB) {
			return false
		}
	}
	return true
}

func refEventMetaEqual(a, b protocol.RefEventMeta) bool {
	if len(a.Listen) != len(b.Listen) {
		return false
	}
	if len(a.Listen) > 0 {
		for i, v := range a.Listen {
			if b.Listen[i] != v {
				return false
			}
		}
	}
	if len(a.Props) != len(b.Props) {
		return false
	}
	if len(a.Props) > 0 {
		for i, v := range a.Props {
			if b.Props[i] != v {
				return false
			}
		}
	}
	return true
}
