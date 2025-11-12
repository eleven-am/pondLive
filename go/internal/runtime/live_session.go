package runtime

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
	defaultDOMGetTimeout = 15 * time.Second
)

var (
	errDOMGetNoTransport = errors.New("runtime: domget requires active transport")
	errDOMGetTimeout     = errors.New("runtime: domget timed out")
)

// Transport delivers messages to the client connection backing a session.
type Transport interface {
	SendInit(protocol.Init) error
	SendResume(protocol.ResumeOK) error
	SendTemplate(protocol.TemplateFrame) error
	SendFrame(protocol.Frame) error
	SendServerError(protocol.ServerError) error
	SendDiagnostic(protocol.Diagnostic) error
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

	templateFrames []queuedTemplateFrame

	snapshot snapshot

	pendingEffects []any

	cookieBatches map[string]cookieBatch
	cookieCounter atomic.Uint64

	domGetCounter atomic.Uint64
	domGetMu      sync.Mutex
	domGetPending map[string]chan domGetResult
	domGetTimeout time.Duration

	domCallCounter atomic.Uint64
	domCallMu      sync.Mutex
	domCallPending map[string]chan domCallResult

	transport Transport
	devMode   bool

	pubsubCounts map[string]int

	diagnostics []Diagnostic

	hasInit bool

	clientConfig *protocol.ClientConfig
}

type queuedTemplateFrame struct {
	seq   int
	frame protocol.TemplateFrame
}

type domGetResult struct {
	values map[string]any
	err    error
}

type domCallResult struct {
	result any
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
	Init      *protocol.Init
	Resume    *protocol.ResumeOK
	Templates []protocol.TemplateFrame
	Frames    []protocol.Frame
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

	templates := s.templateFramesFromLocked(from)
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

	return JoinResult{Resume: &resume, Templates: templates, Frames: frames}
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
		diagMsg   protocol.Diagnostic
		errMsg    protocol.ServerError
		send      bool
	)
	s.mu.Lock()
	diag = s.enrichDiagnosticLocked(diag)
	s.diagnostics = append(s.diagnostics, diag)
	if len(s.diagnostics) > defaultDiagnosticHistory {
		s.diagnostics = s.diagnostics[len(s.diagnostics)-defaultDiagnosticHistory:]
	}
	if s.devMode && s.transport != nil {
		transport = s.transport
		diagMsg = diag.ToProtocolDiagnostic(s.id)
		errMsg = diag.ToServerError(s.id)
		send = true
	}
	s.mu.Unlock()

	if send && transport != nil {
		_ = transport.SendDiagnostic(diagMsg)
		_ = transport.SendServerError(errMsg)
	}
}

func (s *LiveSession) enrichDiagnosticLocked(diag Diagnostic) Diagnostic {
	if s == nil {
		return diag
	}

	if diag.ComponentID != "" {
		if diag.Metadata == nil {
			diag.Metadata = make(map[string]any)
		}
		if _, exists := diag.Metadata["componentId"]; !exists {
			diag.Metadata["componentId"] = diag.ComponentID
		}
		if _, scoped := diag.Metadata["componentScope"]; !scoped {
			if scope := s.snapshotComponentPathLocked(diag.ComponentID); scope != nil {
				diag.Metadata["componentScope"] = buildComponentScopeMetadata(*scope)
			}
		}
	}
	return diag
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

// SendTemplate delivers a template frame to the client transport.
func (s *LiveSession) SendTemplate(frame protocol.TemplateFrame) error {
	s.mu.Lock()
	if frame.T == "" {
		frame.T = "template"
	}
	if frame.SID == "" {
		frame.SID = string(s.id)
	}
	if frame.Ver == 0 {
		frame.Ver = s.version
	}
	seq := s.nextSeq
	s.appendTemplateFrameLocked(seq, frame)
	transport := s.transport
	s.touchLocked()
	s.mu.Unlock()

	if transport != nil {
		return transport.SendTemplate(frame)
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
	pendingGet := s.domGetPending
	if len(pendingGet) > 0 {
		s.domGetPending = nil
	}
	s.domGetMu.Unlock()

	s.domCallMu.Lock()
	pendingCall := s.domCallPending
	if len(pendingCall) > 0 {
		s.domCallPending = nil
	}
	s.domCallMu.Unlock()

	for _, ch := range pendingGet {
		if ch == nil {
			continue
		}
		select {
		case ch <- domGetResult{err: err}:
		default:
		}
	}

	for _, ch := range pendingCall {
		if ch == nil {
			continue
		}
		select {
		case ch <- domCallResult{err: err}:
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

// DOMAsyncCall calls a method on the given element ref with the provided arguments
// and returns the result from the client.
func (s *LiveSession) DOMAsyncCall(ref string, method string, args ...any) (any, error) {
	if s == nil {
		return nil, errors.New("runtime: session is nil")
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, errors.New("runtime: domcall requires element ref")
	}
	method = strings.TrimSpace(method)
	if method == "" {
		return nil, errors.New("runtime: domcall requires method name")
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

	id := fmt.Sprintf("domcall:%d", s.domCallCounter.Add(1))
	ch := make(chan domCallResult, 1)

	s.domCallMu.Lock()
	if s.domCallPending == nil {
		s.domCallPending = make(map[string]chan domCallResult)
	}
	s.domCallPending[id] = ch
	s.domCallMu.Unlock()

	req := protocol.DOMRequest{
		ID:     id,
		Ref:    ref,
		Method: method,
		Args:   args,
	}

	if err := transport.SendDOMRequest(req); err != nil {
		s.domCallMu.Lock()
		delete(s.domCallPending, id)
		s.domCallMu.Unlock()
		return nil, err
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-ch:
		if result.err != nil {
			return nil, result.err
		}
		return result.result, nil
	case <-timer.C:
		s.domCallMu.Lock()
		delete(s.domCallPending, id)
		s.domCallMu.Unlock()
		return nil, errDOMGetTimeout
	}
}

// HandleDOMResponse resolves a pending DOMGet or DOMAsyncCall request with data from the client.
func (s *LiveSession) HandleDOMResponse(resp protocol.DOMResponse) {
	if s == nil {
		return
	}
	if resp.ID == "" {
		return
	}

	if strings.HasPrefix(resp.ID, "domget:") {
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
		return
	}

	if strings.HasPrefix(resp.ID, "domcall:") {
		s.domCallMu.Lock()
		ch, ok := s.domCallPending[resp.ID]
		if ok {
			delete(s.domCallPending, resp.ID)
		}
		s.domCallMu.Unlock()

		if !ok || ch == nil {
			return
		}

		result := domCallResult{}
		if resp.Error != "" {
			result.err = errors.New(resp.Error)
		} else {
			result.result = resp.Result
		}

		select {
		case ch <- result:
		default:
		}
		return
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
	release := comp.suspendFlushScheduling()
	defer release()
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

// HandleRouterReset refreshes the template for a router component when requested by the client.
func (s *LiveSession) HandleRouterReset(componentID string) error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	trimmed := strings.TrimSpace(componentID)
	if trimmed == "" {
		return errors.New("runtime: router reset requires component identifier")
	}
	comp := s.ComponentSession()
	if comp == nil {
		return errors.New("runtime: session has no component")
	}

	requestedID := trimmed
	target := comp.componentByID(requestedID)
	if target == nil {
		metadata := map[string]any{"componentId": requestedID}
		if scope := s.snapshotComponentPath(requestedID); scope != nil {
			if scope.ParentID != "" {
				metadata["parentId"] = scope.ParentID
			}
			if len(scope.ParentPath) > 0 {
				metadata["parentPath"] = append([]int(nil), scope.ParentPath...)
			}
			if len(scope.FirstChild) > 0 {
				metadata["firstChild"] = append([]int(nil), scope.FirstChild...)
			}
			if len(scope.LastChild) > 0 {
				metadata["lastChild"] = append([]int(nil), scope.LastChild...)
			}
		}

		diag := Diagnostic{
			Code:       "router_reset_failed",
			Phase:      "router:reset",
			Message:    fmt.Sprintf("router: component %s not found", requestedID),
			Suggestion: "Reload the page to recover.",
			Metadata:   metadata,
			CapturedAt: time.Now(),
		}

		s.ReportDiagnostic(diag)
		return diag.AsError()
	}

	release := comp.suspendFlushScheduling()
	defer release()

	boot := comp.requestComponentBootInternal(requestedID)
	if boot == nil {
		metadata := map[string]any{"componentId": requestedID}
		diag := Diagnostic{
			Code:       "router_reset_failed",
			Phase:      "router:reset",
			Message:    fmt.Sprintf("router: component %s unavailable", requestedID),
			Suggestion: "Reload the page to recover.",
			Metadata:   metadata,
			CapturedAt: time.Now(),
		}
		s.ReportDiagnostic(diag)
		return diag.AsError()
	}
	if comp.currentComponent() == nil {
		comp.markDirty(boot)
	}
	if !comp.Dirty() {
		return nil
	}
	return s.Flush()
}

func componentParentID(paths []protocol.ComponentPath, id string) string {
	if id == "" {
		return ""
	}
	for _, path := range paths {
		if path.ComponentID == id {
			return path.ParentID
		}
	}
	return ""
}

func (s *LiveSession) snapshotComponentPath(id string) *protocol.ComponentPath {
	if s == nil || id == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotComponentPathLocked(id)
}

func (s *LiveSession) snapshotComponentPathLocked(id string) *protocol.ComponentPath {
	if s == nil || id == "" {
		return nil
	}
	for _, path := range s.snapshot.ComponentPaths {
		if path.ComponentID != id {
			continue
		}
		copy := path
		if len(copy.ParentPath) > 0 {
			copy.ParentPath = append([]int(nil), copy.ParentPath...)
		}
		if len(copy.FirstChild) > 0 {
			copy.FirstChild = append([]int(nil), copy.FirstChild...)
		}
		if len(copy.LastChild) > 0 {
			copy.LastChild = append([]int(nil), copy.LastChild...)
		}
		return &copy
	}
	return nil
}

func buildComponentScopeMetadata(path protocol.ComponentPath) map[string]any {
	scope := map[string]any{
		"componentId": path.ComponentID,
	}
	if path.ParentID != "" {
		scope["parentId"] = path.ParentID
	}
	if len(path.ParentPath) > 0 {
		scope["parentPath"] = append([]int(nil), path.ParentPath...)
	}
	if len(path.FirstChild) > 0 {
		scope["firstChild"] = append([]int(nil), path.FirstChild...)
	}
	if len(path.LastChild) > 0 {
		scope["lastChild"] = append([]int(nil), path.LastChild...)
	}
	return scope
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
	var (
		refDelta       protocol.RefDelta
		templateFrames []protocol.TemplateFrame
	)
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
		payload := s.templatePayloadFromSnapshot(snap)
		payload.HTML = template.html
		payload.Refs = refDelta
		templateFrames = append(templateFrames, protocol.TemplateFrame{
			TemplatePayload: payload,
			T:               "template",
			SID:             string(s.id),
			Ver:             s.version,
		})
		frame.Delta = protocol.FrameDelta{}
		frame.Patch = nil
		refDelta = protocol.RefDelta{}
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
				if slot < 0 {
					continue
				}
				if slot >= len(s.snapshot.Dynamics) {
					extended := make([]protocol.DynamicSlot, slot+1)
					copy(extended, s.snapshot.Dynamics)
					s.snapshot.Dynamics = extended
				}
				if slot >= len(s.snapshot.Slots) {
					extended := make([]protocol.SlotMeta, slot+1)
					copy(extended, s.snapshot.Slots)
					for i := len(s.snapshot.Slots); i < len(extended); i++ {
						extended[i] = protocol.SlotMeta{AnchorID: i}
					}
					s.snapshot.Slots = extended
				}
				if idx < len(update.dynamics) {
					s.snapshot.Dynamics[slot] = update.dynamics[idx]
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
				if len(update.bindings.Slots) == 0 {
					if len(s.snapshot.Bindings.Slots) > 0 {
						for _, slot := range update.slots {
							delete(s.snapshot.Bindings.Slots, slot)
						}
					}
				} else {
					if s.snapshot.Bindings.Slots == nil {
						s.snapshot.Bindings.Slots = make(protocol.BindingTable)
					}
					for _, slot := range update.slots {
						entries, ok := update.bindings.Slots[slot]
						if !ok || len(entries) == 0 {
							delete(s.snapshot.Bindings.Slots, slot)
							continue
						}
						s.snapshot.Bindings.Slots[slot] = copySlotBindings(entries)
					}
				}
			}
			if len(update.slotPaths) > 0 {
				s.snapshot.SlotPaths = replaceSlotPaths(s.snapshot.SlotPaths, update.slotPaths, update.slots)
			}
			if len(update.listPaths) > 0 {
				s.snapshot.ListPaths = replaceListPaths(s.snapshot.ListPaths, update.listPaths, update.listSlots)
			}
			if len(update.componentPaths) > 0 {
				s.snapshot.ComponentPaths = replaceComponentPaths(s.snapshot.ComponentPaths, update.componentPaths)
			}
			payload := templatePayloadFromComponentUpdate(update)
			scope := templateScopeFromUpdate(update)
			templateFrames = append(templateFrames, protocol.TemplateFrame{
				TemplatePayload: payload,
				T:               "template",
				SID:             string(s.id),
				Ver:             s.version,
				Scope:           scope,
			})
		}
	}
	for _, tpl := range templateFrames {
		if err := s.SendTemplate(tpl); err != nil {
			return err
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

func (s *LiveSession) appendTemplateFrameLocked(seq int, frame protocol.TemplateFrame) {
	frameCopy := cloneTemplateFrame(frame)
	s.templateFrames = append(s.templateFrames, queuedTemplateFrame{seq: seq, frame: frameCopy})
	if s.frameCap > 0 && len(s.templateFrames) > s.frameCap {
		drop := len(s.templateFrames) - s.frameCap
		newLen := copy(s.templateFrames, s.templateFrames[drop:])
		for i := newLen; i < len(s.templateFrames); i++ {
			s.templateFrames[i] = queuedTemplateFrame{}
		}
		s.templateFrames = s.templateFrames[:newLen]
	}
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
		s.pruneTemplateFramesLocked()
		return
	}
	idx := sort.Search(len(s.frames), func(i int) bool {
		return s.frames[i].Seq > s.lastAck
	})
	if idx <= 0 {
		s.pruneTemplateFramesLocked()
		return
	}
	if idx >= len(s.frames) {
		for i := range s.frames {
			s.frames[i] = protocol.Frame{}
		}
		s.frames = s.frames[:0]
		s.pruneTemplateFramesLocked()
		return
	}
	newLen := copy(s.frames, s.frames[idx:])
	for i := newLen; i < len(s.frames); i++ {
		s.frames[i] = protocol.Frame{}
	}
	s.frames = s.frames[:newLen]
	s.pruneTemplateFramesLocked()
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

func (s *LiveSession) pruneTemplateFramesLocked() {
	if len(s.templateFrames) == 0 {
		return
	}
	keep := s.templateFrames[:0]
	for _, tpl := range s.templateFrames {
		if tpl.seq <= s.lastAck {
			continue
		}
		keep = append(keep, tpl)
	}
	for i := len(keep); i < len(s.templateFrames); i++ {
		s.templateFrames[i] = queuedTemplateFrame{}
	}
	s.templateFrames = keep
}

func (s *LiveSession) templateFramesFromLocked(start int) []protocol.TemplateFrame {
	if len(s.templateFrames) == 0 {
		return nil
	}
	out := make([]protocol.TemplateFrame, 0, len(s.templateFrames))
	for _, tpl := range s.templateFrames {
		if tpl.seq < start {
			continue
		}
		out = append(out, cloneTemplateFrame(tpl.frame))
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (s *LiveSession) buildInitLocked(errors []protocol.ServerError) protocol.Init {
	payload := s.templatePayloadFromSnapshot(s.snapshot)
	init := protocol.Init{
		TemplatePayload: payload,
		T:               "init",
		SID:             string(s.id),
		Ver:             s.version,
		Location:        s.snapshot.Location,
		Seq:             s.nextSeq,
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

	payload := cloneTemplatePayload(init.TemplatePayload)
	payload.HTML = html
	boot := protocol.Boot{
		TemplatePayload: payload,
		T:               "boot",
		SID:             init.SID,
		Ver:             init.Ver,
		Seq:             init.Seq,
		Location:        init.Location,
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
	Statics        []string
	staticsOwned   bool
	Dynamics       []protocol.DynamicSlot
	Slots          []protocol.SlotMeta
	SlotPaths      []protocol.SlotPath
	ListPaths      []protocol.ListPath
	ComponentPaths []protocol.ComponentPath
	Handlers       map[string]protocol.HandlerMeta
	Bindings       protocol.TemplateBindings
	Refs           map[string]protocol.RefMeta
	Location       protocol.Location
	Metadata       *Meta
}

func (s *LiveSession) buildSnapshot(structured render.Structured, loc SessionLocation, meta *Meta) snapshot {
	statics := globalTemplateIntern.InternStatics(structured.S)
	dynamics := encodeDynamics(structured.D)
	slots := make([]protocol.SlotMeta, len(dynamics))
	for i := range slots {
		slots[i] = protocol.SlotMeta{AnchorID: i}
	}
	slotPaths := encodeSlotPaths(structured.SlotPaths)
	listPaths := encodeListPaths(structured.ListPaths)
	componentPaths := encodeComponentPaths(structured.ComponentPaths)
	handlers := extractHandlerMeta(structured)
	var refs map[string]protocol.RefMeta
	if ids := extractRefIDs(structured); len(ids) > 0 {
		if comp := s.ComponentSession(); comp != nil {
			refs = comp.snapshotRefs(ids)
		}
	}
	protoLoc := protocol.Location{Path: loc.Path, Query: loc.Query}
	return snapshot{
		Statics:        statics,
		Dynamics:       dynamics,
		Slots:          slots,
		SlotPaths:      slotPaths,
		ListPaths:      listPaths,
		ComponentPaths: componentPaths,
		Handlers:       handlers,
		Bindings:       encodeTemplateBindings(structured.Bindings, structured.UploadBindings, structured.RefBindings, structured.RouterBindings),
		Refs:           refs,
		Location:       protoLoc,
		Metadata:       CloneMeta(meta),
		staticsOwned:   false,
	}
}

func (s *LiveSession) templatePayloadFromSnapshot(snap snapshot) protocol.TemplatePayload {
	payload := protocol.TemplatePayload{
		S:              append([]string(nil), snap.Statics...),
		D:              cloneDynamics(snap.Dynamics),
		Slots:          cloneSlots(snap.Slots),
		SlotPaths:      cloneSlotPaths(snap.SlotPaths),
		ListPaths:      cloneListPaths(snap.ListPaths),
		ComponentPaths: cloneComponentPaths(snap.ComponentPaths),
		Handlers:       cloneHandlers(snap.Handlers),
		Bindings:       cloneTemplateBindings(snap.Bindings),
	}
	if len(snap.Refs) > 0 {
		payload.Refs = protocol.RefDelta{Add: cloneRefs(snap.Refs)}
	}
	payload.TemplateHash = computeTemplateHash(payload)
	return payload
}

func templatePayloadFromComponentUpdate(update componentTemplateUpdate) protocol.TemplatePayload {
	payload := protocol.TemplatePayload{
		HTML:           update.html,
		S:              append([]string(nil), update.statics...),
		D:              cloneDynamics(update.dynamics),
		SlotPaths:      cloneSlotPaths(update.slotPaths),
		ListPaths:      cloneListPaths(update.listPaths),
		ComponentPaths: cloneComponentPaths(update.componentPaths),
		Handlers:       cloneHandlers(update.handlersAdd),
		Bindings:       cloneTemplateBindings(update.bindings),
	}
	if len(update.slots) > 0 {
		slots := make([]protocol.SlotMeta, len(update.slots))
		for i, slot := range update.slots {
			slots[i] = protocol.SlotMeta{AnchorID: slot}
		}
		payload.Slots = slots
	}
	payload.TemplateHash = computeTemplateHash(payload)
	return payload
}

func templateScopeFromUpdate(update componentTemplateUpdate) *protocol.TemplateScope {
	scope := &protocol.TemplateScope{ComponentID: update.id}
	for _, path := range update.componentPaths {
		if path.ComponentID != update.id {
			continue
		}
		if path.ParentID != "" {
			scope.ParentID = path.ParentID
		}
		if len(path.ParentPath) > 0 {
			scope.ParentPath = append([]int(nil), path.ParentPath...)
		}
		break
	}
	return scope
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
					bindings := encodeTemplateBindings(row.Bindings, row.UploadBindings, row.RefBindings, row.RouterBindings)
					if hasTemplateBindings(bindings) {
						rows[j].Bindings = bindings
					}
					if encoded := encodeSlotPaths(row.SlotPaths); len(encoded) > 0 {
						rows[j].SlotPaths = encoded
					}
					if encodedLists := encodeListPaths(row.ListPaths); len(encodedLists) > 0 {
						rows[j].ListPaths = encodedLists
					}
					if encodedComponents := encodeComponentPaths(row.ComponentPaths); len(encodedComponents) > 0 {
						rows[j].ComponentPaths = encodedComponents
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

func encodeTemplateBindings(
	handlerBindings []render.HandlerBinding,
	uploadBindings []render.UploadBinding,
	refBindings []render.RefBinding,
	routerBindings []render.RouterBinding,
) protocol.TemplateBindings {
	table := encodeBindingTable(handlerBindings)
	uploads := encodeUploadBindings(uploadBindings)
	refs := encodeRefBindings(refBindings)
	routers := encodeRouterBindings(routerBindings)
	if len(table) == 0 && len(uploads) == 0 && len(refs) == 0 && len(routers) == 0 {
		return protocol.TemplateBindings{}
	}
	return protocol.TemplateBindings{Slots: table, Uploads: uploads, Refs: refs, Router: routers}
}

func hasTemplateBindings(bindings protocol.TemplateBindings) bool {
	return len(bindings.Slots) > 0 || len(bindings.Uploads) > 0 || len(bindings.Refs) > 0 || len(bindings.Router) > 0
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

func encodeUploadBindings(bindings []render.UploadBinding) []protocol.UploadBinding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]protocol.UploadBinding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.UploadID == "" || binding.ComponentID == "" {
			continue
		}
		encoded := protocol.UploadBinding{
			ComponentID: binding.ComponentID,
			UploadID:    binding.UploadID,
			Multiple:    binding.Multiple,
			MaxSize:     binding.MaxSize,
		}
		if len(binding.Path) > 0 {
			encoded.Path = append([]int(nil), binding.Path...)
		}
		if len(binding.Accept) > 0 {
			encoded.Accept = append([]string(nil), binding.Accept...)
		}
		out = append(out, encoded)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func encodeRefBindings(bindings []render.RefBinding) []protocol.RefBinding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]protocol.RefBinding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.RefID == "" {
			continue
		}
		encoded := protocol.RefBinding{
			ComponentID: binding.ComponentID,
			RefID:       binding.RefID,
		}
		if len(binding.Path) > 0 {
			encoded.Path = append([]int(nil), binding.Path...)
		}
		out = append(out, encoded)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func encodeRouterBindings(bindings []render.RouterBinding) []protocol.RouterBinding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]protocol.RouterBinding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.PathValue == "" && binding.Query == "" && binding.Hash == "" && binding.Replace == "" {
			continue
		}
		encoded := protocol.RouterBinding{
			ComponentID: binding.ComponentID,
			PathValue:   binding.PathValue,
			Query:       binding.Query,
			Hash:        binding.Hash,
			Replace:     binding.Replace,
		}
		if len(binding.Path) > 0 {
			encoded.Path = append([]int(nil), binding.Path...)
		}
		out = append(out, encoded)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func encodeSlotPaths(paths []render.SlotPath) []protocol.SlotPath {
	if len(paths) == 0 {
		return nil
	}
	out := make([]protocol.SlotPath, len(paths))
	for i, path := range paths {
		clone := protocol.SlotPath{
			Slot:           path.Slot,
			ComponentID:    path.ComponentID,
			TextChildIndex: path.TextChildIndex,
		}
		if len(path.ElementPath) > 0 {
			clone.ElementPath = append([]int(nil), path.ElementPath...)
		}
		out[i] = clone
	}
	return out
}

func encodeListPaths(paths []render.ListPath) []protocol.ListPath {
	if len(paths) == 0 {
		return nil
	}
	out := make([]protocol.ListPath, len(paths))
	for i, path := range paths {
		clone := protocol.ListPath{
			Slot:        path.Slot,
			ComponentID: path.ComponentID,
		}
		if len(path.ElementPath) > 0 {
			clone.ElementPath = append([]int(nil), path.ElementPath...)
		}
		out[i] = clone
	}
	return out
}

func encodeComponentPaths(paths []render.ComponentPath) []protocol.ComponentPath {
	if len(paths) == 0 {
		return nil
	}
	out := make([]protocol.ComponentPath, len(paths))
	for i, path := range paths {
		clone := protocol.ComponentPath{
			ComponentID: path.ComponentID,
		}
		if path.ParentID != "" {
			clone.ParentID = path.ParentID
		}
		if len(path.ParentPath) > 0 {
			clone.ParentPath = append([]int(nil), path.ParentPath...)
		}
		if len(path.FirstChild) > 0 {
			clone.FirstChild = append([]int(nil), path.FirstChild...)
		}
		if len(path.LastChild) > 0 {
			clone.LastChild = append([]int(nil), path.LastChild...)
		}
		out[i] = clone
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
	for _, binding := range structured.RefBindings {
		id := strings.TrimSpace(binding.RefID)
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
					rows[j].Slots = append([]int(nil), row.Slots...)
				}
				if hasTemplateBindings(row.Bindings) {
					rows[j].Bindings = cloneTemplateBindings(row.Bindings)
				}
				if len(row.SlotPaths) > 0 {
					rows[j].SlotPaths = cloneSlotPaths(row.SlotPaths)
				}
				if len(row.ListPaths) > 0 {
					rows[j].ListPaths = cloneListPaths(row.ListPaths)
				}
				if len(row.ComponentPaths) > 0 {
					rows[j].ComponentPaths = cloneComponentPaths(row.ComponentPaths)
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

func cloneSlotPaths(paths []protocol.SlotPath) []protocol.SlotPath {
	if len(paths) == 0 {
		return nil
	}
	out := make([]protocol.SlotPath, len(paths))
	for i, path := range paths {
		clone := path
		if len(path.ElementPath) > 0 {
			clone.ElementPath = append([]int(nil), path.ElementPath...)
		}
		out[i] = clone
	}
	return out
}

func cloneListPaths(paths []protocol.ListPath) []protocol.ListPath {
	if len(paths) == 0 {
		return nil
	}
	out := make([]protocol.ListPath, len(paths))
	for i, path := range paths {
		clone := path
		if len(path.ElementPath) > 0 {
			clone.ElementPath = append([]int(nil), path.ElementPath...)
		}
		out[i] = clone
	}
	return out
}

func cloneComponentPaths(paths []protocol.ComponentPath) []protocol.ComponentPath {
	if len(paths) == 0 {
		return nil
	}
	out := make([]protocol.ComponentPath, len(paths))
	for i, path := range paths {
		clone := path
		if len(path.ParentPath) > 0 {
			clone.ParentPath = append([]int(nil), path.ParentPath...)
		}
		if len(path.FirstChild) > 0 {
			clone.FirstChild = append([]int(nil), path.FirstChild...)
		}
		if len(path.LastChild) > 0 {
			clone.LastChild = append([]int(nil), path.LastChild...)
		}
		out[i] = clone
	}
	return out
}

func replaceSlotPaths(existing []protocol.SlotPath, updates []protocol.SlotPath, slots []int) []protocol.SlotPath {
	if len(updates) == 0 {
		if len(existing) == 0 {
			return nil
		}
		out := make([]protocol.SlotPath, len(existing))
		copy(out, existing)
		return out
	}
	slotSet := make(map[int]struct{}, len(slots))
	for _, slot := range slots {
		slotSet[slot] = struct{}{}
	}
	filtered := make([]protocol.SlotPath, 0, len(existing)+len(updates))
	for _, path := range existing {
		if len(slotSet) > 0 {
			if _, ok := slotSet[path.Slot]; ok {
				continue
			}
		}
		filtered = append(filtered, path)
	}
	filtered = append(filtered, cloneSlotPaths(updates)...)
	return filtered
}

func replaceListPaths(existing []protocol.ListPath, updates []protocol.ListPath, slots []int) []protocol.ListPath {
	if len(updates) == 0 {
		if len(existing) == 0 {
			return nil
		}
		out := make([]protocol.ListPath, len(existing))
		copy(out, existing)
		return out
	}
	slotSet := make(map[int]struct{}, len(slots))
	for _, slot := range slots {
		slotSet[slot] = struct{}{}
	}
	filtered := make([]protocol.ListPath, 0, len(existing)+len(updates))
	for _, path := range existing {
		if len(slotSet) > 0 {
			if _, ok := slotSet[path.Slot]; ok {
				continue
			}
		}
		filtered = append(filtered, path)
	}
	filtered = append(filtered, cloneListPaths(updates)...)
	return filtered
}

func replaceComponentPaths(existing []protocol.ComponentPath, updates []protocol.ComponentPath) []protocol.ComponentPath {
	if len(updates) == 0 {
		if len(existing) == 0 {
			return nil
		}
		out := make([]protocol.ComponentPath, len(existing))
		copy(out, existing)
		return out
	}
	idSet := make(map[string]struct{}, len(updates))
	for _, path := range updates {
		if path.ComponentID == "" {
			continue
		}
		idSet[path.ComponentID] = struct{}{}
	}
	filtered := make([]protocol.ComponentPath, 0, len(existing)+len(updates))
	for _, path := range existing {
		if _, ok := idSet[path.ComponentID]; ok {
			continue
		}
		filtered = append(filtered, path)
	}
	filtered = append(filtered, cloneComponentPaths(updates)...)
	return filtered
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

func cloneTemplatePayload(payload protocol.TemplatePayload) protocol.TemplatePayload {
	cloned := protocol.TemplatePayload{
		HTML:         payload.HTML,
		TemplateHash: payload.TemplateHash,
	}
	if len(payload.S) > 0 {
		cloned.S = append([]string(nil), payload.S...)
	}
	if len(payload.D) > 0 {
		cloned.D = cloneDynamics(payload.D)
	}
	if len(payload.Slots) > 0 {
		cloned.Slots = cloneSlots(payload.Slots)
	}
	if len(payload.SlotPaths) > 0 {
		cloned.SlotPaths = cloneSlotPaths(payload.SlotPaths)
	}
	if len(payload.ListPaths) > 0 {
		cloned.ListPaths = cloneListPaths(payload.ListPaths)
	}
	if len(payload.ComponentPaths) > 0 {
		cloned.ComponentPaths = cloneComponentPaths(payload.ComponentPaths)
	}
	if len(payload.Handlers) > 0 {
		cloned.Handlers = cloneHandlers(payload.Handlers)
	}
	cloned.Bindings = cloneTemplateBindings(payload.Bindings)
	if len(payload.Refs.Add) > 0 {
		cloned.Refs.Add = cloneRefs(payload.Refs.Add)
	}
	if len(payload.Refs.Del) > 0 {
		cloned.Refs.Del = append([]string(nil), payload.Refs.Del...)
	}
	return cloned
}

func cloneTemplateFrame(frame protocol.TemplateFrame) protocol.TemplateFrame {
	cloned := protocol.TemplateFrame{
		TemplatePayload: cloneTemplatePayload(frame.TemplatePayload),
		T:               frame.T,
		SID:             frame.SID,
		Ver:             frame.Ver,
	}
	if frame.Scope != nil {
		cloned.Scope = cloneTemplateScope(frame.Scope)
	}
	return cloned
}

func cloneTemplateScope(scope *protocol.TemplateScope) *protocol.TemplateScope {
	if scope == nil {
		return nil
	}
	cloned := &protocol.TemplateScope{
		ComponentID: scope.ComponentID,
		ParentID:    scope.ParentID,
	}
	if len(scope.ParentPath) > 0 {
		cloned.ParentPath = append([]int(nil), scope.ParentPath...)
	}
	return cloned
}

func cloneTemplateBindings(bindings protocol.TemplateBindings) protocol.TemplateBindings {
	if len(bindings.Slots) == 0 && len(bindings.Uploads) == 0 && len(bindings.Refs) == 0 && len(bindings.Router) == 0 {
		return protocol.TemplateBindings{}
	}
	return protocol.TemplateBindings{
		Slots:   cloneBindingTable(bindings.Slots),
		Uploads: cloneUploadBindings(bindings.Uploads),
		Refs:    cloneRefBindings(bindings.Refs),
		Router:  cloneRouterBindings(bindings.Router),
	}
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

func cloneRefBindings(bindings []protocol.RefBinding) []protocol.RefBinding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]protocol.RefBinding, len(bindings))
	for i, binding := range bindings {
		clone := protocol.RefBinding{
			ComponentID: binding.ComponentID,
			RefID:       binding.RefID,
		}
		if len(binding.Path) > 0 {
			clone.Path = append([]int(nil), binding.Path...)
		}
		out[i] = clone
	}
	return out
}

func cloneRouterBindings(bindings []protocol.RouterBinding) []protocol.RouterBinding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]protocol.RouterBinding, len(bindings))
	for i, binding := range bindings {
		clone := protocol.RouterBinding{
			ComponentID: binding.ComponentID,
			PathValue:   binding.PathValue,
			Query:       binding.Query,
			Hash:        binding.Hash,
			Replace:     binding.Replace,
		}
		if len(binding.Path) > 0 {
			clone.Path = append([]int(nil), binding.Path...)
		}
		out[i] = clone
	}
	return out
}

type hashTemplateData struct {
	S              []string                 `json:"s,omitempty"`
	D              []hashDynamicSlot        `json:"d,omitempty"`
	Slots          []protocol.SlotMeta      `json:"slots,omitempty"`
	SlotPaths      []protocol.SlotPath      `json:"slotPaths,omitempty"`
	ListPaths      []protocol.ListPath      `json:"listPaths,omitempty"`
	ComponentPaths []protocol.ComponentPath `json:"componentPaths,omitempty"`
	Handlers       []hashHandlerEntry       `json:"handlers,omitempty"`
	Bindings       hashTemplateBindings     `json:"bindings,omitempty"`
	Refs           hashRefDelta             `json:"refs,omitempty"`
}

type hashDynamicSlot struct {
	Kind  string        `json:"k"`
	Text  string        `json:"t,omitempty"`
	Attrs []hashAttr    `json:"a,omitempty"`
	List  []hashListRow `json:"l,omitempty"`
}

type hashAttr struct {
	Key   string `json:"k"`
	Value string `json:"v"`
}

type hashListRow struct {
	Key            string                   `json:"k"`
	Slots          []int                    `json:"s,omitempty"`
	SlotPaths      []protocol.SlotPath      `json:"sp,omitempty"`
	ListPaths      []protocol.ListPath      `json:"lp,omitempty"`
	ComponentPaths []protocol.ComponentPath `json:"cp,omitempty"`
	Bindings       hashTemplateBindings     `json:"b,omitempty"`
}

type hashHandlerEntry struct {
	ID   string               `json:"id"`
	Meta protocol.HandlerMeta `json:"meta"`
}

type hashTemplateBindings struct {
	Slots   []hashSlotBindingEntry `json:"slots,omitempty"`
	Uploads []hashUploadBinding    `json:"uploads,omitempty"`
	Refs    []hashRefBinding       `json:"refs,omitempty"`
	Router  []hashRouterBinding    `json:"router,omitempty"`
}

type hashSlotBindingEntry struct {
	Slot     int                    `json:"slot"`
	Bindings []protocol.SlotBinding `json:"bindings,omitempty"`
}

type hashUploadBinding struct {
	ComponentID string   `json:"componentId"`
	Path        []int    `json:"path,omitempty"`
	UploadID    string   `json:"uploadId"`
	Accept      []string `json:"accept,omitempty"`
	Multiple    bool     `json:"multiple,omitempty"`
	MaxSize     int64    `json:"maxSize,omitempty"`
}

type hashRefBinding struct {
	ComponentID string `json:"componentId"`
	Path        []int  `json:"path,omitempty"`
	RefID       string `json:"refId"`
}

type hashRouterBinding struct {
	ComponentID string `json:"componentId"`
	Path        []int  `json:"path,omitempty"`
	PathValue   string `json:"pathValue,omitempty"`
	Query       string `json:"query,omitempty"`
	Hash        string `json:"hash,omitempty"`
	Replace     string `json:"replace,omitempty"`
}

type hashRefDelta struct {
	Add []hashRefAddEntry `json:"add,omitempty"`
	Del []string          `json:"del,omitempty"`
}

type hashRefAddEntry struct {
	ID   string      `json:"id"`
	Meta hashRefMeta `json:"meta"`
}

type hashRefMeta struct {
	Tag    string         `json:"tag"`
	Events []hashRefEvent `json:"events,omitempty"`
}

type hashRefEvent struct {
	Name   string   `json:"name"`
	Listen []string `json:"listen,omitempty"`
	Props  []string `json:"props,omitempty"`
}

func computeTemplateHash(payload protocol.TemplatePayload) string {
	data := hashTemplateData{
		S:              append([]string(nil), payload.S...),
		D:              buildHashDynamicSlots(payload.D),
		Slots:          cloneSlots(payload.Slots),
		SlotPaths:      cloneSlotPaths(payload.SlotPaths),
		ListPaths:      cloneListPaths(payload.ListPaths),
		ComponentPaths: cloneComponentPaths(payload.ComponentPaths),
		Handlers:       buildHashHandlers(payload.Handlers),
		Bindings:       buildHashTemplateBindings(payload.Bindings),
		Refs:           buildHashRefDelta(payload.Refs),
	}
	encoded, err := json.Marshal(data)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(encoded)
	return fmt.Sprintf("sha256:%x", sum[:])
}

func buildHashDynamicSlots(slots []protocol.DynamicSlot) []hashDynamicSlot {
	if len(slots) == 0 {
		return nil
	}
	out := make([]hashDynamicSlot, len(slots))
	for i, slot := range slots {
		hashSlot := hashDynamicSlot{Kind: slot.Kind}
		if slot.Text != "" {
			hashSlot.Text = slot.Text
		}
		if len(slot.Attrs) > 0 {
			keys := make([]string, 0, len(slot.Attrs))
			for key := range slot.Attrs {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			attrs := make([]hashAttr, len(keys))
			for idx, key := range keys {
				attrs[idx] = hashAttr{Key: key, Value: slot.Attrs[key]}
			}
			hashSlot.Attrs = attrs
		}
		if len(slot.List) > 0 {
			hashSlot.List = buildHashListRows(slot.List)
		}
		out[i] = hashSlot
	}
	return out
}

func buildHashListRows(rows []protocol.ListRow) []hashListRow {
	if len(rows) == 0 {
		return nil
	}
	out := make([]hashListRow, len(rows))
	for i, row := range rows {
		hashRow := hashListRow{Key: row.Key}
		if len(row.Slots) > 0 {
			hashRow.Slots = append([]int(nil), row.Slots...)
		}
		if len(row.SlotPaths) > 0 {
			hashRow.SlotPaths = cloneSlotPaths(row.SlotPaths)
		}
		if len(row.ListPaths) > 0 {
			hashRow.ListPaths = cloneListPaths(row.ListPaths)
		}
		if len(row.ComponentPaths) > 0 {
			hashRow.ComponentPaths = cloneComponentPaths(row.ComponentPaths)
		}
		hashRow.Bindings = buildHashTemplateBindings(row.Bindings)
		out[i] = hashRow
	}
	return out
}

func buildHashHandlers(handlers map[string]protocol.HandlerMeta) []hashHandlerEntry {
	if len(handlers) == 0 {
		return nil
	}
	ids := make([]string, 0, len(handlers))
	for id := range handlers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]hashHandlerEntry, len(ids))
	for i, id := range ids {
		out[i] = hashHandlerEntry{ID: id, Meta: handlers[id]}
	}
	return out
}

func buildHashTemplateBindings(bindings protocol.TemplateBindings) hashTemplateBindings {
	return hashTemplateBindings{
		Slots:   buildHashSlotBindingEntries(bindings.Slots),
		Uploads: buildHashUploadBindings(bindings.Uploads),
		Refs:    buildHashRefBindings(bindings.Refs),
		Router:  buildHashRouterBindings(bindings.Router),
	}
}

func buildHashSlotBindingEntries(table protocol.BindingTable) []hashSlotBindingEntry {
	if len(table) == 0 {
		return nil
	}
	slots := make([]int, 0, len(table))
	for slot := range table {
		slots = append(slots, slot)
	}
	sort.Ints(slots)
	out := make([]hashSlotBindingEntry, 0, len(slots))
	for _, slot := range slots {
		bindings := table[slot]
		entry := hashSlotBindingEntry{Slot: slot}
		if len(bindings) > 0 {
			cloned := copySlotBindings(bindings)
			sort.Slice(cloned, func(i, j int) bool {
				if cloned[i].Event != cloned[j].Event {
					return cloned[i].Event < cloned[j].Event
				}
				return cloned[i].Handler < cloned[j].Handler
			})
			entry.Bindings = cloned
		}
		out = append(out, entry)
	}
	return out
}

func buildHashUploadBindings(bindings []protocol.UploadBinding) []hashUploadBinding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]hashUploadBinding, len(bindings))
	for i, binding := range bindings {
		out[i] = hashUploadBinding{
			ComponentID: binding.ComponentID,
			Path:        append([]int(nil), binding.Path...),
			UploadID:    binding.UploadID,
			Accept:      append([]string(nil), binding.Accept...),
			Multiple:    binding.Multiple,
			MaxSize:     binding.MaxSize,
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ComponentID != out[j].ComponentID {
			return out[i].ComponentID < out[j].ComponentID
		}
		if cmp := compareIntSlices(out[i].Path, out[j].Path); cmp != 0 {
			return cmp < 0
		}
		return out[i].UploadID < out[j].UploadID
	})
	return out
}

func buildHashRefBindings(bindings []protocol.RefBinding) []hashRefBinding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]hashRefBinding, len(bindings))
	for i, binding := range bindings {
		out[i] = hashRefBinding{
			ComponentID: binding.ComponentID,
			Path:        append([]int(nil), binding.Path...),
			RefID:       binding.RefID,
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ComponentID != out[j].ComponentID {
			return out[i].ComponentID < out[j].ComponentID
		}
		if cmp := compareIntSlices(out[i].Path, out[j].Path); cmp != 0 {
			return cmp < 0
		}
		return out[i].RefID < out[j].RefID
	})
	return out
}

func buildHashRouterBindings(bindings []protocol.RouterBinding) []hashRouterBinding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]hashRouterBinding, len(bindings))
	for i, binding := range bindings {
		out[i] = hashRouterBinding{
			ComponentID: binding.ComponentID,
			Path:        append([]int(nil), binding.Path...),
			PathValue:   binding.PathValue,
			Query:       binding.Query,
			Hash:        binding.Hash,
			Replace:     binding.Replace,
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ComponentID != out[j].ComponentID {
			return out[i].ComponentID < out[j].ComponentID
		}
		if cmp := compareIntSlices(out[i].Path, out[j].Path); cmp != 0 {
			return cmp < 0
		}
		if out[i].PathValue != out[j].PathValue {
			return out[i].PathValue < out[j].PathValue
		}
		if out[i].Query != out[j].Query {
			return out[i].Query < out[j].Query
		}
		if out[i].Hash != out[j].Hash {
			return out[i].Hash < out[j].Hash
		}
		return out[i].Replace < out[j].Replace
	})
	return out
}

func buildHashRefDelta(refs protocol.RefDelta) hashRefDelta {
	var result hashRefDelta
	if len(refs.Add) > 0 {
		entries := make([]hashRefAddEntry, 0, len(refs.Add))
		ids := make([]string, 0, len(refs.Add))
		for id := range refs.Add {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			entries = append(entries, hashRefAddEntry{ID: id, Meta: buildHashRefMeta(refs.Add[id])})
		}
		result.Add = entries
	}
	if len(refs.Del) > 0 {
		del := append([]string(nil), refs.Del...)
		sort.Strings(del)
		result.Del = del
	}
	return result
}

func buildHashRefMeta(meta protocol.RefMeta) hashRefMeta {
	result := hashRefMeta{Tag: meta.Tag}
	if len(meta.Events) > 0 {
		names := make([]string, 0, len(meta.Events))
		for name := range meta.Events {
			names = append(names, name)
		}
		sort.Strings(names)
		events := make([]hashRefEvent, len(names))
		for i, name := range names {
			entry := hashRefEvent{Name: name}
			if listeners := meta.Events[name].Listen; len(listeners) > 0 {
				entry.Listen = append([]string(nil), listeners...)
			}
			if props := meta.Events[name].Props; len(props) > 0 {
				entry.Props = append([]string(nil), props...)
			}
			events[i] = entry
		}
		result.Events = events
	}
	return result
}

func compareIntSlices(a, b []int) int {
	lenA := len(a)
	lenB := len(b)
	min := lenA
	if lenB < min {
		min = lenB
	}
	for i := 0; i < min; i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	if lenA < lenB {
		return -1
	}
	if lenA > lenB {
		return 1
	}
	return 0
}

func cloneUploadBindings(bindings []protocol.UploadBinding) []protocol.UploadBinding {
	if len(bindings) == 0 {
		return nil
	}
	out := make([]protocol.UploadBinding, len(bindings))
	for i, binding := range bindings {
		clone := protocol.UploadBinding{
			ComponentID: binding.ComponentID,
			UploadID:    binding.UploadID,
			Multiple:    binding.Multiple,
			MaxSize:     binding.MaxSize,
		}
		if len(binding.Path) > 0 {
			clone.Path = append([]int(nil), binding.Path...)
		}
		if len(binding.Accept) > 0 {
			clone.Accept = append([]string(nil), binding.Accept...)
		}
		out[i] = clone
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
