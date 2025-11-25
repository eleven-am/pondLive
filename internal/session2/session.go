package session2

import (
	"net/http"
	"sync"
	"time"

	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// Component is a function that renders a component tree.
type Component = func(*runtime2.Ctx) work.Node

// TouchObserver is called when the session is touched (activity detected).
type TouchObserver func(time.Time)

// LiveSession is a thin naive bridge between WebSocket/transport and the Bus.
// It owns transport/HTTP/TTL but has no knowledge of frames, events, navigation, or DOM.
// It just forwards messages between the transport and the Bus.
type LiveSession struct {
	id      SessionID
	version int

	session   *runtime2.Session
	transport Transport
	lifecycle *Lifecycle

	// Touch observers for activity notifications
	touchObservers   map[int]TouchObserver
	nextObserverID   int
	touchObserversMu sync.Mutex

	// Client asset path for cache busting
	clientAsset string

	mu          sync.Mutex
	outboundSub *runtime2.Subscription
}

// NewLiveSession creates a new session with the given root component.
// The session wires the Bus to the transport for bidirectional messaging.
func NewLiveSession(id SessionID, version int, root Component, cfg *Config) *LiveSession {
	effectiveCfg := DefaultConfig()
	if cfg != nil {
		if cfg.TTL > 0 {
			effectiveCfg.TTL = cfg.TTL
		}
		if cfg.Clock != nil {
			effectiveCfg.Clock = cfg.Clock
		}
		effectiveCfg.DevMode = cfg.DevMode
		effectiveCfg.ClientAsset = cfg.ClientAsset
		effectiveCfg.DOMTimeout = cfg.DOMTimeout
	}

	rootInst := &runtime2.Instance{
		ID:        "root",
		Fn:        root,
		HookFrame: []runtime2.HookSlot{},
		Children:  []*runtime2.Instance{},
	}

	rtSession := &runtime2.Session{
		Root:              rootInst,
		Components:        map[string]*runtime2.Instance{"root": rootInst},
		Handlers:          make(map[string]work.Handler),
		MountedComponents: make(map[*runtime2.Instance]struct{}),
		Bus:               runtime2.NewBus(),
		SessionID:         string(id),
	}

	rtSession.SetDevMode(effectiveCfg.DevMode)
	if effectiveCfg.DOMTimeout > 0 {
		rtSession.SetDOMTimeout(effectiveCfg.DOMTimeout)
	}

	sess := &LiveSession{
		id:             id,
		version:        version,
		session:        rtSession,
		lifecycle:      NewLifecycle(effectiveCfg.Clock, effectiveCfg.TTL),
		touchObservers: make(map[int]TouchObserver),
		clientAsset:    effectiveCfg.ClientAsset,
	}

	sess.outboundSub = rtSession.Bus.SubscribeAll(func(topic string, event string, data interface{}) {
		sess.mu.Lock()
		t := sess.transport
		sess.mu.Unlock()

		if t != nil {
			_ = t.Send(topic, event, data)
		}
	})

	return sess
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

// Session returns the underlying runtime2.Session.
func (s *LiveSession) Session() *runtime2.Session {
	if s == nil {
		return nil
	}
	return s.session
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

// Receive handles inbound messages from the transport.
// It publishes the message to the Bus for subscribers (handlers, router, etc.) to process.
func (s *LiveSession) Receive(topic, event string, data any) {
	if s == nil || s.session == nil || s.session.Bus == nil {
		return
	}
	s.Touch()
	s.session.Bus.Publish(topic, event, data)
}

// Flush triggers a render/flush cycle on the runtime session.
func (s *LiveSession) Flush() error {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Flush()
}

// Touch updates the last activity timestamp and notifies observers.
func (s *LiveSession) Touch() {
	if s == nil || s.lifecycle == nil {
		return
	}
	s.lifecycle.Touch()

	s.touchObserversMu.Lock()
	observers := make([]TouchObserver, 0, len(s.touchObservers))
	for _, obs := range s.touchObservers {
		observers = append(observers, obs)
	}
	s.touchObserversMu.Unlock()

	now := s.lifecycle.LastTouch()
	for _, obs := range observers {
		if obs != nil {
			obs(now)
		}
	}
}

// IsExpired returns true if the session has exceeded its TTL.
func (s *LiveSession) IsExpired() bool {
	if s == nil || s.lifecycle == nil {
		return true
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

// Close releases session resources.
func (s *LiveSession) Close() error {
	if s == nil {
		return nil
	}

	if s.outboundSub != nil {
		s.outboundSub.Unsubscribe()
	}

	if s.session != nil {
		s.session.Close()
	}

	s.mu.Lock()
	t := s.transport
	s.transport = nil
	s.mu.Unlock()

	if t != nil {
		_ = t.Close()
	}

	return nil
}

// SetDevMode enables or disables development mode.
// Delegates to the underlying runtime2.Session.
func (s *LiveSession) SetDevMode(enabled bool) {
	if s == nil || s.session == nil {
		return
	}
	s.session.SetDevMode(enabled)
}

// SetDiagnosticReporter installs a diagnostic reporter.
// Delegates to the underlying runtime2.Session.
func (s *LiveSession) SetDiagnosticReporter(reporter runtime2.DiagnosticReporter) {
	if s == nil || s.session == nil {
		return
	}
	s.session.SetDiagnosticReporter(reporter)
}

// OnTouch registers an observer to be called when the session is touched.
// Returns an unsubscribe function.
func (s *LiveSession) OnTouch(observer TouchObserver) func() {
	if s == nil || observer == nil {
		return func() {}
	}

	s.touchObserversMu.Lock()
	id := s.nextObserverID
	s.nextObserverID++
	s.touchObservers[id] = observer
	s.touchObserversMu.Unlock()

	return func() {
		s.touchObserversMu.Lock()
		delete(s.touchObservers, id)
		s.touchObserversMu.Unlock()
	}
}

// ClientAsset returns the versioned client JS bundle path.
func (s *LiveSession) ClientAsset() string {
	if s == nil {
		return ""
	}
	return s.clientAsset
}

// SetClientAsset updates the client asset path.
func (s *LiveSession) SetClientAsset(path string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.clientAsset = path
	s.mu.Unlock()
}

// Bus returns the session's message bus.
func (s *LiveSession) Bus() *runtime2.Bus {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Bus
}

// SetAutoFlush sets the callback for automatic flush scheduling.
// Delegates to the underlying runtime2.Session.
func (s *LiveSession) SetAutoFlush(fn func()) {
	if s == nil || s.session == nil {
		return
	}
	s.session.SetAutoFlush(fn)
}

// SetDOMTimeout sets the timeout for blocking DOM operations.
// Delegates to the underlying runtime2.Session.
func (s *LiveSession) SetDOMTimeout(timeout time.Duration) {
	if s == nil || s.session == nil {
		return
	}
	s.session.SetDOMTimeout(timeout)
}

// ServeHTTP dispatches HTTP requests to registered handlers.
// Routes /_handlers/{sessionID}/{handlerID} to the appropriate handler.
// Delegates to the underlying runtime.Session.ServeHTTP.
func (s *LiveSession) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.session == nil {
		http.NotFound(w, r)
		return
	}
	s.Touch()
	s.session.ServeHTTP(w, r)
}
