package session

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"golang.org/x/time/rate"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/upload"
	"github.com/eleven-am/pondlive/internal/work"
)

const documentType = "<!DOCTYPE html>"

const (
	closeBound         = 10 * time.Second
	defaultRenderBound = 10 * time.Second
)

var (
	ErrTransportClosed = errors.New("session: transport closed before attach")
	ErrNotLive         = errors.New("session: session is closing")
	ErrNoView          = errors.New("session: no view to reset to")
	ErrStaleConnection = errors.New("session: connection was replaced")
)

type Component = func(*runtime.Ctx) work.Node

type Option func(*LiveSession)

func WithRequest(r *http.Request) Option {
	return func(s *LiveSession) {
		s.requestState = headers.NewRequestState(headers.NewRequestInfo(r, s.retained...))
	}
}

type LiveSession struct {
	id      SessionID
	version int

	clientAsset string
	devMode     bool
	retained    []string

	rt           *runtime.Session
	rootBoundary *runtime.Boundary
	requestState *headers.RequestState
	clientTopic  protocol.Topic
	clientAction string
	out          *outbound
	clock        Clock
	logger       *slog.Logger

	pendingTTL    time.Duration
	renderBound   time.Duration
	disconnectTTL time.Duration
	onClose       func(*LiveSession, CloseReason)
	onLeavePend   func(*LiveSession)

	state  atomic.Uint32
	reason atomic.Uint32

	gen    atomic.Uint64
	inMu   sync.Mutex
	lastCS uint64
	tr     Transport

	onReset     func(*LiveSession, string)
	resetLimit  *rate.Limiter
	resetQueued bool
	resyncBS    uint64
	epoch       uint64
	stateTimer  Timer
	lifeTimer   Timer
}

func NewLiveSession(id SessionID, version int, root Component, cfg *Config, opts ...Option) *LiveSession {
	effectiveCfg := DefaultConfig()
	if cfg != nil {
		effectiveCfg = *cfg
	}

	logger := effectiveCfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	clock := effectiveCfg.Clock
	if clock == nil {
		clock = realClock{}
	}
	replayFrames := effectiveCfg.ReplayFrames
	if replayFrames <= 0 {
		replayFrames = DefaultReplayFrames
	}
	replayBytes := effectiveCfg.ReplayBytes
	if replayBytes <= 0 {
		replayBytes = DefaultReplayBytes
	}
	resetLimit := effectiveCfg.ResetLimit
	if resetLimit <= 0 {
		resetLimit = DefaultOutboundBudget / 2
	}
	resyncInterval := effectiveCfg.ResyncInterval
	if resyncInterval <= 0 {
		resyncInterval = DefaultResyncInterval
	}
	lifetime := effectiveCfg.MaxLifetime
	if lifetime <= 0 {
		lifetime = DefaultMaxLifetime
	}

	renderBound := effectiveCfg.RenderTimeout
	if renderBound <= 0 {
		renderBound = defaultRenderBound
	}

	sess := &LiveSession{
		renderBound:   renderBound,
		id:            id,
		version:       version,
		clientAsset:   effectiveCfg.ClientAsset,
		devMode:       effectiveCfg.DevMode,
		retained:      append([]string(nil), effectiveCfg.RetainedHeaders...),
		clock:         clock,
		logger:        logger,
		pendingTTL:    effectiveCfg.PendingTTL,
		disconnectTTL: effectiveCfg.DisconnectTTL,
		onClose:       effectiveCfg.OnClose,
		onLeavePend:   effectiveCfg.OnLeavePending,
		onReset:       effectiveCfg.OnReset,
		resetLimit:    rate.NewLimiter(rate.Every(resyncInterval), resyncBurst),
		out:           &outbound{maxFrames: replayFrames, maxBytes: replayBytes, maxReset: resetLimit},
	}
	for _, opt := range opts {
		opt(sess)
	}
	if sess.requestState == nil {
		sess.requestState = headers.NewRequestState(headers.NewRequestInfo(nil, sess.retained...))
	}

	rtSession := runtime.NewSession(runtime.Config{
		SessionID:       string(id),
		Root:            bootRoot(sess, root, effectiveCfg.ClientAsset, effectiveCfg.DevMode),
		DevMode:         effectiveCfg.DevMode,
		Bus:             protocol.NewBus(),
		UploadRegistry:  upload.NewRegistry(),
		UploadEndpoint:  effectiveCfg.UploadEndpoint,
		UploadScriptURL: effectiveCfg.UploadScriptURL,
		HandlerQueueCap: effectiveCfg.HandlerQueueCap,
		DOMTimeout:      effectiveCfg.DOMTimeout,
	})
	sess.rt = rtSession

	rtSession.Bus.SubscribeAll(func(topic protocol.Topic, event string, data interface{}) {
		if !isClientTopic(topic, event) {
			return
		}
		if topic == sess.clientTopic && event == sess.clientAction {
			return
		}
		if !sess.out.send(string(topic), event, data) {
			sess.Report(protocol.Diagnostic{
				Phase:    "protocol:encode",
				Message:  "outbound message could not be encoded",
				Metadata: map[string]any{"topic": string(topic), "event": event},
			})
		}
	})
	rtSession.Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) {
		logger.Log(context.Background(), diagnosticLevel(d.Phase), "pondlive diagnostic", "session", string(id), "phase", d.Phase, "message", d.Message, "stack", d.StackTrace, "meta", d.Metadata)
	})

	rtSession.DeferEffects()
	rtSession.OnClosed(sess.closed)
	rtSession.Start()

	sess.lifeTimer = clock.AfterFunc(lifetime, func() { sess.RequestClose(ReasonLifetimeExpired) })
	if sess.pendingTTL > 0 {
		_ = rtSession.Post(func() { sess.armState(sess.pendingTTL) })
	}

	return sess
}

func (s *LiveSession) ID() SessionID {
	if s == nil {
		return ""
	}
	return s.id
}

func (s *LiveSession) Version() int {
	if s == nil {
		return 0
	}
	return s.version
}

func (s *LiveSession) ClientAsset() string {
	if s == nil {
		return ""
	}
	return s.clientAsset
}

func (s *LiveSession) Session() *runtime.Session {
	if s == nil {
		return nil
	}
	return s.rt
}

func (s *LiveSession) Generation() uint64 {
	if s == nil {
		return 0
	}
	return s.gen.Load()
}

func (s *LiveSession) RequestState() *headers.RequestState {
	if s == nil {
		return nil
	}
	return s.requestState
}

func (s *LiveSession) State() State {
	if s == nil {
		return StateClosed
	}
	return State(s.state.Load())
}

func (s *LiveSession) CloseReason() CloseReason {
	if s == nil {
		return ReasonNone
	}
	return CloseReason(s.reason.Load())
}

func (s *LiveSession) Retained() (frames, bytes int) {
	if s == nil {
		return 0, 0
	}
	return s.out.retained()
}

func (s *LiveSession) Done() <-chan struct{} {
	if s == nil {
		return nil
	}
	return s.rt.Done()
}

func (s *LiveSession) Report(d protocol.Diagnostic) {
	if s == nil || s.rt.Bus == nil {
		return
	}
	s.rt.Report(nil, runtime.Report{Severity: runtime.SeverityDiagnostic, Phase: d.Phase, Message: d.Message, Stack: d.StackTrace, Meta: d.Metadata})
}

func (s *LiveSession) armState(ttl time.Duration) {
	s.epoch++
	epoch := s.epoch
	if s.stateTimer != nil {
		s.stateTimer.Stop()
	}
	if ttl <= 0 {
		s.stateTimer = nil
		return
	}
	s.stateTimer = s.clock.AfterFunc(ttl, func() {
		_ = s.rt.Post(func() { s.expire(epoch) })
	})
}

func (s *LiveSession) expire(epoch uint64) {
	if epoch != s.epoch {
		return
	}
	switch s.State() {
	case StatePending:
		s.RequestClose(ReasonPendingExpired)
	case StateDisconnected:
		s.RequestClose(ReasonDisconnectExpired)
	}
}

func (s *LiveSession) transition(from, to State) bool {
	return s.state.CompareAndSwap(uint32(from), uint32(to))
}

func (s *LiveSession) RequestClose(reason CloseReason) {
	if s == nil {
		return
	}
	for {
		cur := State(s.state.Load())
		if cur == StateClosing || cur == StateClosed {
			break
		}
		if s.transition(cur, StateClosing) {
			s.reason.CompareAndSwap(uint32(ReasonNone), uint32(reason))
			break
		}
	}
	s.rt.CloseDOM()
	_ = s.rt.RequestClose()
}

func (s *LiveSession) closed() {
	s.state.Store(uint32(StateClosed))
	s.reason.CompareAndSwap(uint32(ReasonNone), uint32(ReasonRuntime))

	if s.lifeTimer != nil {
		s.lifeTimer.Stop()
	}
	if s.stateTimer != nil {
		s.stateTimer.Stop()
		s.stateTimer = nil
	}
	tr := s.tr
	s.tr = nil
	s.out.shut()
	if tr != nil {
		tr.Terminate(websocket.StatusGoingAway, "session closed")
	}
	if s.onClose != nil {
		s.onClose(s, s.CloseReason())
	}
}

func (s *LiveSession) Close() error {
	if s == nil {
		return nil
	}
	s.RequestClose(ReasonRemoved)
	return s.rt.CloseWithin(closeBound)
}

func (s *LiveSession) Wedge(reason string) {
	if s == nil {
		return
	}
	for {
		cur := State(s.state.Load())
		if cur == StateClosed {
			return
		}
		if cur == StateClosing || s.transition(cur, StateClosing) {
			break
		}
	}
	s.reason.Store(uint32(ReasonWedged))
	if tr := s.out.current(); tr != nil {
		tr.Terminate(websocket.StatusInternalError, "session wedged")
	}
	s.rt.Wedge(reason)
}

type Redirect struct {
	URL  string
	Code int
}

type RenderResult struct {
	Status   int
	Header   http.Header
	Redirect *Redirect
	HTML     string
	Boot     protocol.Boot
	Crash    *runtime.Error
}

func (s *LiveSession) Render(ctx context.Context) (RenderResult, error) {
	var snap runtime.Snapshot
	var renderErr error
	var location route.Location
	var head uint64
	err := s.rt.AskCtx(ctx, s.renderBound, func() error {
		s.out.suppressFrames(true)
		defer s.out.suppressFrames(false)
		snap, renderErr = s.rt.RenderSnapshot()
		if renderErr == nil && !snap.Crashed && s.rootBoundary != nil {
			if all := s.rootBoundary.Errors().Peek().All(); len(all) > 0 {
				snap.Crashed, snap.Crash = true, all[0]
			}
		}
		location = s.requestState.URL()
		head = s.out.headSeq()
		return nil
	})
	if err == nil {
		err = renderErr
	}
	if err != nil {
		s.RequestClose(ReasonRenderFailed)
		return RenderResult{}, err
	}

	res := RenderResult{Status: http.StatusOK, Header: s.requestState.ResponseHeaders()}

	if url, code, ok := s.requestState.Redirect(); ok {
		res.Status = code
		res.Redirect = &Redirect{URL: url, Code: code}
		s.RequestClose(ReasonRedirected)
		return res, nil
	}

	if snap.Crashed {
		res.Status = http.StatusInternalServerError
		res.HTML = documentType + snap.HTML
		res.Crash = snap.Crash
		s.RequestClose(ReasonRenderFailed)
		return res, nil
	}

	var clientCfg *protocol.ClientConfig
	if s.devMode {
		debug := true
		clientCfg = &protocol.ClientConfig{Debug: &debug}
	}
	res.HTML = documentType + snap.HTML
	res.Boot = protocol.Boot{
		T:        protocol.FrameBoot,
		SID:      string(s.id),
		Ver:      s.version,
		Seq:      head,
		Location: location,
		Client:   clientCfg,
	}
	return res, nil
}

type AttachResult struct {
	Gen  uint64
	CS   uint64
	Mode string
}

func (s *LiveSession) Attach(ctx context.Context, tr Transport, upgrade http.Header, bs uint64, wantReset bool) (AttachResult, error) {
	if s == nil || tr == nil {
		return AttachResult{}, ErrNotLive
	}
	var res AttachResult
	err := s.rt.AskCtx(ctx, 0, func() error {
		r, err := s.attach(tr, upgrade, bs, wantReset)
		res = r
		return err
	})
	if err != nil {
		return AttachResult{}, err
	}
	return res, nil
}

func (s *LiveSession) attach(tr Transport, upgrade http.Header, bs uint64, wantReset bool) (AttachResult, error) {
	switch s.State() {
	case StateClosing, StateClosed:
		return AttachResult{}, runtime.ErrSessionClosed
	}
	if tr.Closed() {
		return AttachResult{}, ErrTransportClosed
	}

	if !s.rt.Mounted() {
		s.out.suppressFrames(true)
		_, err := s.rt.RenderSnapshot()
		s.out.suppressFrames(false)
		if err != nil {
			return AttachResult{}, err
		}
	}

	old := s.tr
	if old == tr {
		old = nil
	}
	if old != nil {
		s.out.detach()
	}

	s.requestState.SetIsLive(true)
	s.requestState.MergeHeaders(upgrade)

	if err := s.rt.SettleIfNeeded(); err != nil && !errors.Is(err, runtime.ErrNotSettled) {
		return AttachResult{}, s.abortAttach(old, err, false)
	}
	s.rt.GoLive()
	if err := s.rt.SettleIfNeeded(); err != nil && !errors.Is(err, runtime.ErrNotSettled) {
		return AttachResult{}, s.abortAttach(old, err, true)
	}
	if tr.Closed() {
		return AttachResult{}, s.abortAttach(old, ErrTransportClosed, true)
	}

	s.rt.SetConnected(true)
	if err := s.rt.SettleIfNeeded(); err != nil && !errors.Is(err, runtime.ErrNotSettled) {
		return AttachResult{}, s.abortAttach(old, err, true)
	}

	mode, cause, err := s.out.attachPlanned(tr, bs, wantReset, s.buildReset)
	if err != nil {
		return AttachResult{}, s.abortAttach(old, err, true)
	}
	if cause == protocol.CauseClientAhead {
		s.Report(protocol.Diagnostic{
			Phase:    "protocol:client-ahead",
			Message:  "client claims a sequence beyond the server head",
			Metadata: map[string]any{"bs": bs, "head": s.out.headSeq()},
		})
	}

	if mode == protocol.ModeReset {
		s.noteReset(cause)
	}

	s.inMu.Lock()
	if old != nil {
		old.Terminate(CloseReplaced, "replaced")
	}
	gen := s.gen.Add(1)
	cs := s.lastCS
	s.inMu.Unlock()
	s.tr = tr
	if s.transition(StatePending, StateLive) {
		s.leftPending()
		s.armState(0)
	} else if s.transition(StateDisconnected, StateLive) {
		s.armState(0)
	}
	return AttachResult{Gen: gen, CS: cs, Mode: mode}, nil
}

func (s *LiveSession) buildReset(cause string) (any, error) {
	html, err := s.rt.ResetStep()
	if err != nil {
		if errors.Is(err, runtime.ErrNondeterministic) {
			s.RequestClose(ReasonResetNondeterministic)
		}
		return nil, err
	}
	return protocol.ResetPayload{
		HTML:     html,
		Location: s.requestState.URL(),
		Cause:    cause,
	}, nil
}

func (s *LiveSession) Ack(gen, seq uint64) error {
	if s == nil {
		return runtime.ErrSessionClosed
	}
	if gen != s.gen.Load() {
		return nil
	}
	return s.out.ack(seq)
}

func (s *LiveSession) abortAttach(old Transport, cause error, wentLive bool) error {
	if old != nil && !old.Closed() {
		if _, _, err := s.out.attachPlanned(old, s.out.detachedFrom(), false, s.buildReset); err == nil {
			return cause
		}
		old.Terminate(websocket.StatusInternalError, "session unavailable")
	}
	s.tr = nil
	s.out.detach()
	s.disconnected()
	if s.transition(StateLive, StateDisconnected) {
		s.armState(s.disconnectTTL)
	} else if wentLive && s.transition(StatePending, StateDisconnected) {
		s.leftPending()
		s.armState(s.disconnectTTL)
	}
	return cause
}

func (s *LiveSession) leftPending() {
	if s.onLeavePend != nil {
		s.onLeavePend(s)
	}
}

func (s *LiveSession) Detach(tr Transport) {
	if s == nil || tr == nil {
		return
	}
	_ = s.rt.Post(func() {
		if s.tr != tr {
			return
		}
		s.tr = nil
		s.out.detach()
		s.disconnected()
		if s.transition(StateLive, StateDisconnected) {
			s.armState(s.disconnectTTL)
		}
	})
}

func (s *LiveSession) disconnected() {
	s.rt.SetConnected(false)
	_ = s.rt.SettleIfNeeded()
}

func (s *LiveSession) Deliver(gen uint64, evt protocol.ClientEvt) (bool, error) {
	if s == nil {
		return false, runtime.ErrSessionClosed
	}
	s.inMu.Lock()
	defer s.inMu.Unlock()
	if gen != s.gen.Load() {
		return false, ErrStaleConnection
	}
	if evt.CS <= s.lastCS {
		return false, nil
	}
	if evt.CS > s.lastCS+1 {
		s.Report(protocol.Diagnostic{
			Phase:    "protocol:client-seq-gap",
			Message:  "client events lost before delivery",
			Metadata: map[string]any{"from": s.lastCS + 1, "to": evt.CS - 1},
		})
	}
	if err := s.post(evt); err != nil {
		return false, err
	}
	s.lastCS = evt.CS
	return true, nil
}

func (s *LiveSession) post(evt protocol.ClientEvt) error {
	rt := s.rt

	if evt.Type == protocol.TopicFrame {
		return nil
	}

	if evt.Type == protocol.DOMHandler && evt.Action == string(protocol.DOMResponseAction) {
		if resp, ok := protocol.DecodePayload[protocol.DOMResponsePayload](evt.Payload); ok {
			rt.ResolveDOMResponse(resp)
		}
		return nil
	}

	if evt.Type == protocol.HandlerTopic {
		return rt.PostClientEvent(func() {
			if err := rt.DispatchHandler(evt.I, evt.Action, evt.Payload); err != nil {
				s.Report(protocol.Diagnostic{
					Phase:    err.Error(),
					Message:  "client event dropped",
					Metadata: map[string]any{"id": evt.I, "event": evt.Action},
				})
			}
		})
	}

	return rt.PostClientEvent(func() {
		s.clientTopic, s.clientAction = evt.Type, evt.Action
		defer func() { s.clientTopic, s.clientAction = "", "" }()
		if protocol.IsScriptTopic(evt.Type) && evt.Action == string(protocol.ScriptMessageAction) {
			if payload, ok := protocol.DecodePayload[protocol.ScriptPayload](evt.Payload); ok {
				rt.Bus.Publish(protocol.Topic(string(evt.Type)+":"+payload.Event), evt.Action, evt.Payload)
			}
			return
		}
		rt.Bus.Publish(evt.Type, evt.Action, evt.Payload)
	})
}

func (s *LiveSession) RequestResync(gen, bs uint64, reason string) {
	if s == nil || gen != s.gen.Load() {
		return
	}
	_ = s.rt.Post(func() { s.resync(gen, bs) })
}

func diagnosticLevel(phase string) slog.Level {
	switch phase {
	case "protocol:reset", "protocol:client-seq-gap", "handler:stale", "handler:exiting":
		return slog.LevelWarn
	}
	return slog.LevelError
}

func (s *LiveSession) resync(gen, bs uint64) {
	if gen != s.gen.Load() || s.tr == nil || s.resetQueued {
		return
	}
	switch s.State() {
	case StateClosing, StateClosed:
		return
	}
	if bs < s.out.lastResetSeq() {
		return
	}
	r := s.resetLimit.Reserve()
	if !r.OK() {
		return
	}
	delay := r.Delay()
	if delay <= 0 {
		s.emitReset(protocol.CauseClientRequest)
		return
	}
	s.resetQueued = true
	s.resyncBS = bs
	s.rt.After(delay, func() {
		s.resetQueued = false
		if s.resyncBS < s.out.lastResetSeq() {
			return
		}
		s.emitReset(protocol.CauseClientRequest)
	})
}

func (s *LiveSession) emitReset(cause string) {
	if s.tr == nil {
		return
	}
	switch s.State() {
	case StateClosing, StateClosed:
		return
	}
	s.out.suppressFrames(true)
	data, err := s.buildReset(cause)
	s.out.suppressFrames(false)
	if err != nil {
		return
	}
	if err := s.out.appendReset(data); err != nil {
		if errors.Is(err, ErrResetTooLarge) {
			s.RequestClose(ReasonResetTooLarge)
		}
		return
	}
	s.noteReset(cause)
}

func (s *LiveSession) noteReset(cause string) {
	s.Report(protocol.Diagnostic{
		Phase:    "protocol:reset",
		Message:  "client document reset",
		Metadata: map[string]any{"cause": cause},
	})
	s.rt.ResyncRefs()
	if s.onReset != nil {
		s.onReset(s, cause)
	}
}

func (s *LiveSession) Flush() error {
	if s == nil {
		return nil
	}
	return s.rt.Flush()
}

func (s *LiveSession) UploadRegistry() *upload.Registry {
	if s == nil {
		return nil
	}
	return s.rt.UploadRegistry
}

func (s *LiveSession) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s == nil {
		http.NotFound(w, r)
		return
	}
	s.rt.ServeHTTP(w, r)
}

func isClientTopic(topic protocol.Topic, event string) bool {
	switch topic {
	case protocol.TopicFrame, protocol.RouteHandler, protocol.DOMHandler:
		return true
	default:
		if protocol.IsScriptTopic(topic) {
			return event == string(protocol.ScriptSendAction)
		}
		return false
	}
}
