package session

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/upload"
	"github.com/eleven-am/pondlive/internal/work"
)

type Component = func(*runtime.Ctx) work.Node

type LiveSession struct {
	id      SessionID
	version int

	session   *runtime.Session
	transport Transport

	clientAsset string

	mu          sync.Mutex
	transportMu sync.RWMutex
	outboundSub *protocol.Subscription
	closed      bool
}

func NewLiveSession(id SessionID, version int, root Component, cfg *Config) *LiveSession {
	effectiveCfg := DefaultConfig()
	if cfg != nil {
		effectiveCfg.DevMode = cfg.DevMode
		effectiveCfg.ClientAsset = cfg.ClientAsset
		effectiveCfg.DOMTimeout = cfg.DOMTimeout
	}

	sess := &LiveSession{
		id:          id,
		version:     version,
		clientAsset: effectiveCfg.ClientAsset,
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
		UploadRegistry:    upload.NewRegistry(),
	}

	rtSession.SetDevMode(effectiveCfg.DevMode)
	if effectiveCfg.DOMTimeout > 0 {
		rtSession.SetDOMTimeout(effectiveCfg.DOMTimeout)
	}

	sess.session = rtSession

	sess.outboundSub = rtSession.Bus.SubscribeAll(func(topic protocol.Topic, event string, data interface{}) {
		if !isClientTopic(topic, event) {
			return
		}
		sess.transportMu.RLock()
		t := sess.transport
		sess.transportMu.RUnlock()

		if t != nil {
			if err := t.Send(string(topic), event, data); err != nil {
				rtSession.Bus.Publish("session:error", "send_error", map[string]any{
					"topic": string(topic),
					"event": event,
					"error": err.Error(),
				})
			}
		}
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
		if ws, ok := t.(*WebSocketTransport); ok {
			if state := old.RequestState(); state != nil {
				ws.UpdateRequestState(state)
			}
		}
		_ = old.Close()
	}
}

func (s *LiveSession) Receive(topic, event string, data any) {
	if s == nil || s.session == nil || s.session.Bus == nil {
		return
	}
	s.session.Bus.Publish(protocol.Topic(topic), event, data)
}

func (s *LiveSession) Flush() error {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.Flush()
}

func (s *LiveSession) Close() error {
	if s == nil {
		return nil
	}

	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true

	outboundSub := s.outboundSub
	s.outboundSub = nil
	session := s.session
	s.session = nil
	s.mu.Unlock()

	if outboundSub != nil {
		outboundSub.Unsubscribe()
	}

	if session != nil {
		session.Close()
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

func (s *LiveSession) ChannelManager() *runtime.ChannelManager {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.ChannelManager()
}

func (s *LiveSession) UploadRegistry() *upload.Registry {
	if s == nil || s.session == nil {
		return nil
	}
	return s.session.UploadRegistry
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
	s.session.ServeHTTP(w, r)
}

func isClientTopic(topic protocol.Topic, event string) bool {
	switch topic {
	case protocol.TopicFrame, protocol.RouteHandler, protocol.DOMHandler, protocol.AckTopic:
		return true
	default:
		if strings.HasPrefix(string(topic), "script:") {
			return event == string(protocol.ScriptSendAction)
		}
		return false
	}
}
