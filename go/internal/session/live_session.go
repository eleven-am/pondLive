package session

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/dom/diff"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/router"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// SessionID uniquely identifies a client session.
type SessionID string

const (
	defaultDOMRequestTimeout = 15 * time.Second
)

var (
	errDOMRequestNoTransport = errors.New("session: dom request requires active transport")
	errDOMRequestTimeout     = errors.New("session: dom request timed out")
)

// LiveSession integrates runtime2.ComponentSession with HTTP/WebSocket transport.
type LiveSession struct {
	id      SessionID
	version int

	component *runtime.ComponentSession
	header    *headerState

	// Router support - initial location from HTTP request
	initialPath  string
	initialQuery url.Values
	initialHash  string

	mu        sync.Mutex
	lifecycle *Lifecycle

	transport Transport
	devMode   bool

	nextSeq   int
	lastAck   int
	clientSeq int

	cookieBatches map[string]cookieBatch
	cookieCounter atomic.Uint64
	cookieQueue   []string

	routerState struct {
		mu  sync.RWMutex
		set func(Location)
	}

	domGetTimeout  time.Duration
	domCallTimeout time.Duration

	domGetCounter  atomic.Uint64
	domCallCounter atomic.Uint64

	domGetPending  map[string]chan domGetResult
	domCallPending map[string]chan domCallResult
	domGetMu       sync.Mutex
	domCallMu      sync.Mutex

	touchObservers map[int]func(time.Time)
	nextObserverID int

	flushing    bool
	clientAsset string
}

// Config captures the optional configuration for a LiveSession.
type Config struct {
	Transport      Transport
	PubsubProvider runtime.PubsubProvider
	TTL            time.Duration
	Clock          func() time.Time
	DevMode        bool
	DOMGetTimeout  time.Duration
	DOMCallTimeout time.Duration
	ClientAsset    string
}

// NewLiveSession constructs a session runtime for the given component tree.
// The root component is wrapped with DocumentRoot to provide context (headers, router, etc).
//
// Example:
//
//	sess := NewLiveSession(
//	    SessionID("user-123"),
//	    1,
//	    MyApp,
//	    cfg,
//	)
func NewLiveSession(id SessionID, version int, root Component, cfg *Config) *LiveSession {
	effectiveConfig := defaultConfig()
	if cfg != nil {
		if cfg.Transport != nil {
			effectiveConfig.Transport = cfg.Transport
		}
		if cfg.TTL > 0 {
			effectiveConfig.TTL = cfg.TTL
		}
		if cfg.Clock != nil {
			effectiveConfig.Clock = cfg.Clock
		}
		effectiveConfig.DevMode = cfg.DevMode
		effectiveConfig.ClientAsset = cfg.ClientAsset
	}

	sess := &LiveSession{
		id:          id,
		version:     version,
		header:      newHeaderState(),
		lifecycle:   NewLifecycle(effectiveConfig.Clock, effectiveConfig.TTL),
		transport:   effectiveConfig.Transport,
		devMode:     effectiveConfig.DevMode,
		clientAsset: effectiveConfig.ClientAsset,
		nextSeq:     1,
	}

	wrapped := documentRoot(sess, root)
	sess.component = runtime.NewSession(wrapped, struct{}{})

	sess.configureRuntime(effectiveConfig)

	sess.component.SetInitialLocationProvider(func() (string, map[string]string, string) {
		sess.mu.Lock()
		defer sess.mu.Unlock()
		query := make(map[string]string)
		for k, v := range sess.initialQuery {
			if len(v) > 0 {
				query[k] = v[0]
			}
		}
		return sess.initialPath, query, sess.initialHash
	})

	sess.component.SetAutoFlush(func() {
		sess.mu.Lock()
		transport := sess.transport
		alreadyFlushing := sess.flushing
		sess.mu.Unlock()

		if transport != nil && transport.IsLive() && !alreadyFlushing {
			_ = sess.Flush()
		}
	})

	return sess
}

func (s *LiveSession) configureRuntime(cfg Config) {
	if s == nil {
		return
	}
	if cfg.Transport != nil {
		s.transport = cfg.Transport
	}
	if cfg.PubsubProvider != nil && s.component != nil {
		s.component.SetPubsubProvider(cfg.PubsubProvider)
	}
	if cfg.Clock != nil && s.lifecycle != nil {
		s.lifecycle.SetClock(cfg.Clock)
	}
	if cfg.TTL > 0 && s.lifecycle != nil {
		s.lifecycle.SetTTL(cfg.TTL)
	}
	if cfg.DevMode {
		s.devMode = true
	}

	s.domGetTimeout = cfg.DOMGetTimeout
	if s.domGetTimeout <= 0 {
		s.domGetTimeout = defaultDOMRequestTimeout
	}
	s.domCallTimeout = cfg.DOMCallTimeout
	if s.domCallTimeout <= 0 {
		s.domCallTimeout = defaultDOMRequestTimeout
	}

	if s.component != nil {
		s.component.SetPatchSender(s.onPatch)
		s.component.SetDOMActionSender(s.sendDOMActions)
		s.component.SetDOMRequestHandlers(s.performDOMGet, s.performDOMCall)
	}

	s.clientAsset = cfg.ClientAsset
}

// ID returns the session identifier.
func (s *LiveSession) ID() SessionID {
	if s == nil {
		return ""
	}
	return s.id
}

// Version returns the session version number.
func (s *LiveSession) Version() int {
	if s == nil {
		return 0
	}
	return s.version
}

// Header returns the header state for this session.
func (s *LiveSession) Header() HeaderState {
	if s == nil {
		return noopHeaderState{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.header
}

// SetTransport updates the transport for this session.
func (s *LiveSession) SetTransport(t Transport) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.transport = t
	s.mu.Unlock()
}

// Touch updates the last activity timestamp.
func (s *LiveSession) Touch() {
	if s == nil {
		return
	}
	var ts time.Time
	if s.lifecycle != nil {
		s.lifecycle.Touch()
		ts = s.lifecycle.LastTouch()
	} else {
		ts = time.Now()
	}
	s.notifyTouchObservers(ts)
}

// Ack records that the client received a frame with the given sequence number.
// This tracks delivery confirmation and updates the last activity timestamp.
func (s *LiveSession) Ack(seq int) {
	if s == nil {
		return
	}
	s.mu.Lock()
	if seq > s.lastAck {
		s.lastAck = seq
	}
	s.mu.Unlock()
	s.Touch()
}

// IsExpired returns true if the session has exceeded its TTL.
func (s *LiveSession) IsExpired() bool {
	if s == nil {
		return true
	}
	if s.lifecycle == nil {
		return false
	}
	return s.lifecycle.IsExpired()
}

// TTL returns the configured session TTL.
func (s *LiveSession) TTL() time.Duration {
	if s == nil || s.lifecycle == nil {
		return 0
	}
	return s.lifecycle.TTL()
}

// AddTouchObserver registers a callback invoked whenever the session is touched.
func (s *LiveSession) AddTouchObserver(cb func(time.Time)) func() {
	if s == nil || cb == nil {
		return func() {}
	}
	s.mu.Lock()
	if s.touchObservers == nil {
		s.touchObservers = make(map[int]func(time.Time))
	}
	id := s.nextObserverID
	s.nextObserverID++
	s.touchObservers[id] = cb
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		delete(s.touchObservers, id)
		s.mu.Unlock()
	}
}

func (s *LiveSession) notifyTouchObservers(ts time.Time) {
	s.mu.Lock()
	obs := make([]func(time.Time), 0, len(s.touchObservers))
	for _, cb := range s.touchObservers {
		if cb != nil {
			obs = append(obs, cb)
		}
	}
	s.mu.Unlock()
	for _, cb := range obs {
		cb(ts)
	}
}

// MergeRequest populates headers, cookies, and initial location from an HTTP request.
func (s *LiveSession) MergeRequest(r *http.Request) {
	if s == nil || r == nil {
		return
	}

	if s.header != nil {
		s.header.mergeRequest(r)
	}

	loc := Location{
		Path:  r.URL.Path,
		Query: cloneQuery(r.URL.Query()),
		Hash:  strings.TrimSpace(r.URL.Fragment),
	}
	s.mu.Lock()
	s.initialPath = loc.Path
	s.initialQuery = cloneQuery(r.URL.Query())
	s.initialHash = loc.Hash
	s.mu.Unlock()

	s.seedRouterState(loc)
}

// InitialLocation returns the initial location from the HTTP request.
// This is used by the Router component to seed its initial state.
func (s *LiveSession) InitialLocation() Location {
	if s == nil {
		return Location{Path: "/"}
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	return Location{
		Path:  s.initialPath,
		Query: cloneQuery(s.initialQuery),
		Hash:  s.initialHash,
	}
}

func (s *LiveSession) registerRouterState(set func(Location)) {
	if s == nil {
		return
	}
	s.routerState.mu.Lock()
	s.routerState.set = set
	s.routerState.mu.Unlock()
}

func (s *LiveSession) seedRouterState(loc Location) {
	if s == nil {
		return
	}
	s.routerState.mu.RLock()
	set := s.routerState.set
	s.routerState.mu.RUnlock()
	if set != nil {
		set(loc)
	}
}

// Flush renders dirty components, diffs the tree, and sends patches.
// For SSR (non-live transport), flushes up to 3 times to handle synchronous
// state updates like router matching and layout effects.
// For WebSocket (live transport), flushes once - async updates trigger auto-flush.
func (s *LiveSession) Flush() error {
	if s == nil || s.component == nil {
		return errors.New("session: not initialized")
	}

	s.mu.Lock()
	transport := s.transport
	s.flushing = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.flushing = false
		s.mu.Unlock()
	}()

	isLive := transport != nil && transport.IsLive()

	if isLive {

		if err := s.component.Flush(); err != nil {
			return err
		}
		return s.dispatchPendingCookies()
	}

	const maxFlushes = 3
	for i := 0; i < maxFlushes; i++ {
		if err := s.component.Flush(); err != nil {
			return err
		}

		if !s.component.HasDirtyComponents() {
			break
		}
	}

	return s.dispatchPendingCookies()
}

// Tree returns the last rendered StructuredNode tree. Must be called after Flush().
// Returns nil if no render has occurred yet.
func (s *LiveSession) Tree() *dom.StructuredNode {
	if s == nil || s.component == nil {
		return nil
	}
	return s.component.Tree()
}

// ComponentSession returns the underlying component session for upload and other operations.
func (s *LiveSession) ComponentSession() *runtime.ComponentSession {
	if s == nil {
		return nil
	}
	return s.component
}

// DeliverPubsub delivers a pubsub message to the component session.
func (s *LiveSession) DeliverPubsub(topic string, payload []byte, meta map[string]string) {
	if s == nil || s.component == nil {
		return
	}
	s.Touch()
	s.component.DeliverPubsub(topic, payload, meta)
}

// HandleEvent dispatches an event to the registered handler.
func (s *LiveSession) HandleEvent(id string, ev dom.Event) error {
	if s == nil || s.component == nil {
		return errors.New("session: not initialized")
	}
	s.Touch()
	return s.component.HandleEvent(id, ev)
}

// HandleNavigation processes a client-side navigation event.
func (s *LiveSession) HandleNavigation(path, rawQuery, hash string) error {
	return s.handleLocationMessage(router.NavMsg{
		T:    "nav",
		Path: path,
		Q:    rawQuery,
		Hash: hash,
	})
}

// HandlePopState processes a client-side popstate event.
func (s *LiveSession) HandlePopState(path, rawQuery, hash string) error {
	return s.handleLocationMessage(router.PopMsg{
		T:    "pop",
		Path: path,
		Q:    rawQuery,
		Hash: hash,
	})
}

func (s *LiveSession) handleLocationMessage(msg router.NavMsg) error {
	if s == nil {
		return errors.New("session: nil session")
	}
	loc := Location{
		Path:  normalizePathValue(msg.Path),
		Query: parseRawQuery(msg.Q),
		Hash:  strings.TrimSpace(msg.Hash),
	}

	s.seedRouterState(loc)
	s.Touch()
	return s.Flush()
}

// onPatch is called by ComponentSession when patches are ready to send.
func (s *LiveSession) onPatch(patches []diff.Patch) error {
	if s == nil {
		return errors.New("session: nil session")
	}

	s.mu.Lock()
	transport := s.transport
	seq := s.nextSeq
	s.nextSeq++
	s.mu.Unlock()

	if transport == nil {
		return errors.New("session: no transport")
	}

	frame := protocol.Frame{
		T:     "frame",
		SID:   string(s.id),
		Seq:   seq,
		Ver:   s.version,
		Patch: patches,
	}

	if s.component != nil {
		if navDelta := s.component.TakeNavDelta(); navDelta != nil {
			frame.Nav = &protocol.NavDelta{
				Push:    navDelta.Push,
				Replace: navDelta.Replace,
			}
		}
	}

	return transport.SendFrame(frame)
}

func (s *LiveSession) sendDOMActions(effects []dom.DOMActionEffect) error {
	if s == nil || len(effects) == 0 {
		return nil
	}

	s.mu.Lock()
	transport := s.transport
	seq := s.nextSeq
	s.nextSeq++
	version := s.version
	sid := s.id
	s.mu.Unlock()

	if transport == nil {
		return errDOMRequestNoTransport
	}

	payload := make([]any, 0, len(effects))
	for _, eff := range effects {
		payload = append(payload, convertDOMActionEffect(eff))
	}

	frame := protocol.Frame{
		T:       "frame",
		SID:     string(sid),
		Seq:     seq,
		Ver:     version,
		Effects: payload,
	}
	return transport.SendFrame(frame)
}

func (s *LiveSession) performDOMGet(ref string, selectors ...string) (map[string]any, error) {
	if s == nil {
		return nil, errors.New("session: nil session")
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, errors.New("session: domget requires element ref")
	}
	props := sanitizeSelectors(selectors)
	if len(props) == 0 {
		return map[string]any{}, nil
	}

	s.mu.Lock()
	transport := s.transport
	timeout := s.domGetTimeout
	if timeout <= 0 {
		timeout = defaultDOMRequestTimeout
	}
	s.mu.Unlock()

	if transport == nil {
		return nil, errDOMRequestNoTransport
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
		T:     "dom_req",
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
		return nil, errDOMRequestTimeout
	}
}

func (s *LiveSession) performDOMCall(ref string, method string, args ...any) (any, error) {
	if s == nil {
		return nil, errors.New("session: nil session")
	}
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return nil, errors.New("session: domcall requires element ref")
	}
	method = strings.TrimSpace(method)
	if method == "" {
		return nil, errors.New("session: domcall requires method name")
	}

	s.mu.Lock()
	transport := s.transport
	timeout := s.domCallTimeout
	if timeout <= 0 {
		timeout = defaultDOMRequestTimeout
	}
	s.mu.Unlock()

	if transport == nil {
		return nil, errDOMRequestNoTransport
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
		T:      "dom_req",
		ID:     id,
		Ref:    ref,
		Method: method,
		Args:   append([]any(nil), args...),
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
		return nil, errDOMRequestTimeout
	}
}

// HandleDOMResponse resolves pending DOM requests using client responses.
func (s *LiveSession) HandleDOMResponse(resp protocol.DOMResponse) {
	if s == nil || resp.ID == "" {
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
		} else if len(resp.Values) > 0 {
			result.values = cloneAnyMap(resp.Values)
		} else {
			result.values = map[string]any{}
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
	}
}

// HandleRouterReset processes a client-side router reset request.
// This is called when a router component needs to be reset to a clean state.
func (s *LiveSession) HandleRouterReset(componentID string) error {
	if s == nil {
		return errors.New("session: nil session")
	}
	trimmed := strings.TrimSpace(componentID)
	if trimmed == "" {
		return errors.New("session: router reset requires component identifier")
	}
	if s.component == nil {
		return errors.New("session: session has no component")
	}

	s.Touch()

	if !s.component.ResetComponent(trimmed) {
		return fmt.Errorf("session: component %s not found", trimmed)
	}

	return nil
}

// Recover attempts to recover the session from an error state.
// Only available in development mode.
func (s *LiveSession) Recover() error {
	if s == nil {
		return errors.New("session: nil session")
	}
	if !s.devMode {
		return errors.New("session: recovery only available in dev mode")
	}
	if s.component == nil {
		return errors.New("session: session has no component")
	}

	s.Touch()

	if !s.component.Reset() {
		return errors.New("session: reset failed")
	}

	return s.Flush()
}

// HandleUploadMessage processes upload control messages from the client.
func (s *LiveSession) HandleUploadMessage(msg protocol.UploadClient) error {
	if s == nil {
		return errors.New("session: nil session")
	}
	if s.component == nil {
		return errors.New("session: session has no component")
	}

	s.Touch()

	switch msg.Op {
	case "change":
		if msg.Meta != nil {
			meta := runtime.FileMeta{
				Name: msg.Meta.Name,
				Size: msg.Meta.Size,
				Type: msg.Meta.Type,
			}
			s.component.HandleUploadChange(msg.ID, meta)
		}
	case "progress":
		s.component.HandleUploadProgress(msg.ID, msg.Loaded, msg.Total)
	case "error":
		errMsg := msg.Error
		if strings.TrimSpace(errMsg) == "" {
			errMsg = "upload failed"
		}
		s.component.HandleUploadError(msg.ID, errors.New(errMsg))
	case "complete":
		if msg.Meta != nil {
			meta := runtime.FileMeta{
				Name: msg.Meta.Name,
				Size: msg.Meta.Size,
				Type: msg.Meta.Type,
			}
			s.component.HandleUploadComplete(msg.ID, meta)
		}
	default:
		return fmt.Errorf("session: unknown upload op: %s", msg.Op)
	}

	return nil
}

func (s *LiveSession) dispatchPendingCookies() error {
	if s == nil {
		return nil
	}
	if s.header != nil {
		if batch := s.header.drainCookieMutations(); !batch.Empty() {
			if token := s.registerCookieBatch(batch); token != "" {
				s.enqueueCookieToken(token)
			}
		}
	}
	return s.flushCookieQueue()
}

func (s *LiveSession) enqueueCookieToken(token string) {
	if token == "" {
		return
	}
	s.mu.Lock()
	s.cookieQueue = append(s.cookieQueue, token)
	s.mu.Unlock()
}

func (s *LiveSession) flushCookieQueue() error {
	s.mu.Lock()
	if len(s.cookieQueue) == 0 {
		s.mu.Unlock()
		return nil
	}
	tokens := append([]string(nil), s.cookieQueue...)
	s.cookieQueue = nil
	transport := s.transport
	seq := s.nextSeq
	s.nextSeq++
	s.mu.Unlock()

	if transport == nil {
		s.mu.Lock()
		s.cookieQueue = append(tokens, s.cookieQueue...)
		s.mu.Unlock()
		return errors.New("session: no transport for cookie effects")
	}

	effects := make([]any, 0, len(tokens))
	for _, token := range tokens {
		effects = append(effects, protocol.CookieEffect{
			Type:     "cookies",
			Endpoint: CookieEndpointPath,
			SID:      string(s.id),
			Token:    token,
		})
	}

	frame := protocol.Frame{
		T:       "frame",
		SID:     string(s.id),
		Seq:     seq,
		Ver:     s.version,
		Effects: effects,
	}

	if err := transport.SendFrame(frame); err != nil {
		s.mu.Lock()
		s.cookieQueue = append(tokens, s.cookieQueue...)
		s.mu.Unlock()
		return err
	}
	return nil
}

// ConsumeCookieBatch retrieves and clears a pending cookie batch identified by token.
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
	entry, ok := s.cookieBatches[trimmed]
	if !ok {
		return CookieBatch{}, false
	}
	delete(s.cookieBatches, trimmed)
	return cloneCookieBatch(entry.Mutations), true
}

func (s *LiveSession) registerCookieBatch(batch CookieBatch) string {
	if s == nil || batch.Empty() {
		return ""
	}
	token := s.nextCookieToken()
	if token == "" {
		return ""
	}
	clone := cloneCookieBatch(batch)
	s.mu.Lock()
	if s.cookieBatches == nil {
		s.cookieBatches = make(map[string]cookieBatch)
	}
	s.cookieBatches[token] = cookieBatch{Mutations: clone}
	s.mu.Unlock()
	return token
}

func (s *LiveSession) nextCookieToken() string {
	buf := make([]byte, 18)
	if _, err := rand.Read(buf); err == nil {
		return base64.RawURLEncoding.EncodeToString(buf)
	}
	fallback := s.cookieCounter.Add(1)
	ts := time.Now().UnixNano()
	return strconv.FormatInt(ts, 36) + strconv.FormatUint(fallback, 36)
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

func convertDOMActionEffect(effect dom.DOMActionEffect) protocol.DOMActionEffect {
	out := protocol.DOMActionEffect{
		Type:     effect.Type,
		Kind:     effect.Kind,
		Ref:      effect.Ref,
		Method:   effect.Method,
		Prop:     effect.Prop,
		Value:    effect.Value,
		Class:    effect.Class,
		Behavior: effect.Behavior,
		Block:    effect.Block,
		Inline:   effect.Inline,
	}
	if len(effect.Args) > 0 {
		args := make([]any, len(effect.Args))
		copy(args, effect.Args)
		out.Args = args
	}
	if effect.On != nil {
		out.On = *effect.On
	}
	return out
}

func cloneAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func sanitizeSelectors(selectors []string) []string {
	if len(selectors) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(selectors))
	out := make([]string, 0, len(selectors))
	for _, selector := range selectors {
		token := strings.TrimSpace(selector)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	return out
}

func parseRawQuery(raw string) url.Values {
	if strings.TrimSpace(raw) == "" {
		return url.Values{}
	}
	values, err := url.ParseQuery(raw)
	if err != nil {
		return url.Values{}
	}
	return values
}

func normalizePathValue(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		return "/" + path
	}
	return path
}

type domGetResult struct {
	values map[string]any
	err    error
}

type domCallResult struct {
	result any
	err    error
}

type cookieBatch struct {
	Mutations CookieBatch
}
