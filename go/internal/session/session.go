package session

import (
	"net/http"
	"sync"
	"time"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type Component = func(*runtime.Ctx) work.Node

type TouchObserver func(time.Time)

type LiveSession struct {
	id      SessionID
	version int

	session   *runtime.Session
	transport Transport
	lifecycle *Lifecycle

	touchObservers   map[int]TouchObserver
	nextObserverID   int
	touchObserversMu sync.Mutex

	clientAsset string

	mu          sync.Mutex
	transportMu sync.RWMutex
	outboundSub *protocol.Subscription
}

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

	sess := &LiveSession{
		id:             id,
		version:        version,
		lifecycle:      NewLifecycle(effectiveCfg.Clock, effectiveCfg.TTL),
		touchObservers: make(map[int]TouchObserver),
		clientAsset:    effectiveCfg.ClientAsset,
	}

	rootInst := &runtime.Instance{
		ID:        "root",
		Fn:        loadBootComponent(sess, root, effectiveCfg.ClientAsset),
		HookFrame: []runtime.HookSlot{},
		Children:  []*runtime.Instance{},
	}

	rtSession := &runtime.Session{
		Root:              rootInst,
		Components:        map[string]*runtime.Instance{"root": rootInst},
		Handlers:          make(map[string]work.Handler),
		MountedComponents: make(map[*runtime.Instance]struct{}),
		Bus:               protocol.NewBus(),
		SessionID:         string(id),
	}

	rtSession.SetDevMode(effectiveCfg.DevMode)
	if effectiveCfg.DOMTimeout > 0 {
		rtSession.SetDOMTimeout(effectiveCfg.DOMTimeout)
	}

	sess.session = rtSession

	sess.outboundSub = rtSession.Bus.SubscribeAll(func(topic protocol.Topic, event string, data interface{}) {
		if !isClientTopic(topic) {
			return
		}
		sess.transportMu.RLock()
		t := sess.transport
		if t != nil {
			_ = t.Send(string(topic), event, data)
		}
		sess.transportMu.RUnlock()
	})

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

func (s *LiveSession) Session() *runtime.Session {
	if s == nil {
		return nil
	}
	return s.session
}

func (s *LiveSession) SetTransport(t Transport) {
	if s == nil {
		return
	}
	s.transportMu.Lock()
	old := s.transport
	s.transport = t
	s.transportMu.Unlock()

	if old != nil && old != t {
		_ = old.Close()
	}
}

func (s *LiveSession) Receive(topic, event string, data any) {
	if s == nil || s.session == nil || s.session.Bus == nil {
		return
	}
	s.Touch()
	s.session.Bus.Publish(protocol.Topic(topic), event, data)
}

func (s *LiveSession) Flush() error {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Flush()
}

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
			func() {
				defer func() { recover() }()
				obs(now)
			}()
		}
	}
}

func (s *LiveSession) IsExpired() bool {
	if s == nil || s.lifecycle == nil {
		return true
	}
	return s.lifecycle.IsExpired()
}

func (s *LiveSession) TTL() time.Duration {
	if s == nil || s.lifecycle == nil {
		return 0
	}
	return s.lifecycle.TTL()
}

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

	s.transportMu.Lock()
	t := s.transport
	s.transport = nil
	s.transportMu.Unlock()

	if t != nil {
		_ = t.Close()
	}

	return nil
}

func (s *LiveSession) SetDevMode(enabled bool) {
	if s == nil || s.session == nil {
		return
	}
	s.session.SetDevMode(enabled)
}

func (s *LiveSession) SetDiagnosticReporter(reporter runtime.DiagnosticReporter) {
	if s == nil || s.session == nil {
		return
	}
	s.session.SetDiagnosticReporter(reporter)
}

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

func (s *LiveSession) ClientAsset() string {
	if s == nil {
		return ""
	}
	return s.clientAsset
}

func (s *LiveSession) SetClientAsset(path string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.clientAsset = path
	s.mu.Unlock()
}

func (s *LiveSession) Bus() *protocol.Bus {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Bus
}

func (s *LiveSession) SetAutoFlush(fn func()) {
	if s == nil || s.session == nil {
		return
	}
	s.session.SetAutoFlush(fn)
}

func (s *LiveSession) SetDOMTimeout(timeout time.Duration) {
	if s == nil || s.session == nil {
		return
	}
	s.session.SetDOMTimeout(timeout)
}

func (s *LiveSession) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.session == nil {
		http.NotFound(w, r)
		return
	}
	s.Touch()
	s.session.ServeHTTP(w, r)
}

func isClientTopic(topic protocol.Topic) bool {
	switch topic {
	case protocol.TopicFrame, protocol.RouteHandler, protocol.DOMHandler, protocol.AckTopic:
		return true
	default:
		return false
	}
}
