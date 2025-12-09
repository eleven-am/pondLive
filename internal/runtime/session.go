package runtime

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/view"
	"github.com/eleven-am/pondlive/internal/work"
)

type Session struct {
	Root       *Instance
	View       view.Node
	PrevView   view.Node
	Components map[string]*Instance

	Handlers   map[string]work.Handler
	handlersMu sync.RWMutex
	DirtyQueue []*Instance
	DirtySet   map[*Instance]struct{}
	dirtyMu    sync.Mutex

	Scripts   map[string]*scriptSlot
	scriptsMu sync.RWMutex

	Bus *protocol.Bus

	channelManager *ChannelManager

	domReqMgr   *domRequestManager
	domReqMgrMu sync.Mutex
	domTimeout  time.Duration

	currentHandlerIDs map[string]bool
	allHandlerSubs    map[string]*protocol.Subscription
	handlerIDsMu      sync.Mutex

	httpHandlers  map[string]*handlerEntry
	httpHandlerMu sync.RWMutex

	PendingEffects  []effectTask
	PendingCleanups []cleanupTask

	MountedComponents map[*Instance]struct{}

	SessionID string

	devMode bool

	pendingFlush bool
	flushing     bool
	autoFlush    func()
	flushMu      sync.Mutex

	flushCtx    context.Context
	flushCancel context.CancelFunc

	mu sync.Mutex
}

type effectTask struct {
	instance  *Instance
	hookIndex int
	fn        func() func()
}

type cleanupTask struct {
	instance *Instance
	fn       func()
}

func (s *Session) getDOMRequestManager() *domRequestManager {
	if s == nil || s.Bus == nil {
		return nil
	}

	s.domReqMgrMu.Lock()
	defer s.domReqMgrMu.Unlock()

	if s.domReqMgr == nil {
		timeout := s.domTimeout
		if timeout == 0 {
			timeout = defaultDOMTimeout
		}
		s.domReqMgr = newDOMRequestManager(s.Bus, timeout)
	}

	return s.domReqMgr
}

func (s *Session) SetDOMTimeout(timeout time.Duration) {
	if s == nil {
		return
	}
	if timeout == 0 {
		timeout = defaultDOMTimeout
	}

	s.domReqMgrMu.Lock()
	s.domTimeout = timeout
	if s.domReqMgr != nil {
		s.domReqMgr.setTimeout(timeout)
	}
	s.domReqMgrMu.Unlock()
}

func (s *Session) Close() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.flushCancel != nil {
		s.flushCancel()
		s.flushCancel = nil
		s.flushCtx = nil
	}

	s.cleanupInstanceTree(s.Root)

	for _, task := range s.PendingCleanups {
		if task.fn != nil {
			func() {
				defer func() { recover() }()
				task.fn()
			}()
		}
	}
	s.PendingCleanups = nil

	s.handlerIDsMu.Lock()
	for _, sub := range s.allHandlerSubs {
		if sub != nil {
			sub.Unsubscribe()
		}
	}
	s.allHandlerSubs = nil
	s.currentHandlerIDs = nil
	s.handlerIDsMu.Unlock()

	s.scriptsMu.Lock()
	s.Scripts = nil
	s.scriptsMu.Unlock()

	s.domReqMgrMu.Lock()
	if s.domReqMgr != nil {
		s.domReqMgr.close()
		s.domReqMgr = nil
	}
	s.domReqMgrMu.Unlock()

	if s.channelManager != nil {
		s.channelManager.Close()
		s.channelManager = nil
	}

	s.Root = nil
	s.View = nil
	s.PrevView = nil
	s.Components = nil
	s.Handlers = nil
	s.DirtyQueue = nil
	s.DirtySet = nil
	s.PendingEffects = nil
	s.MountedComponents = nil
}

func (s *Session) SetDevMode(enabled bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.devMode = enabled
	s.mu.Unlock()
}

func (s *Session) ChannelManager() *ChannelManager {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.channelManager == nil && s.Bus != nil && s.SessionID != "" {
		s.channelManager = NewChannelManager(s.SessionID, s.Bus)
	}
	return s.channelManager
}

func (s *Session) withRecovery(phase string, fn func() error) error {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			if s != nil && s.Bus != nil {
				s.Bus.ReportDiagnostic(protocol.Diagnostic{
					Phase:      phase,
					Message:    fmt.Sprintf("panic: %v", r),
					StackTrace: stack,
					Metadata: map[string]any{
						"panic_value": r,
						"session_id":  s.SessionID,
					},
				})
			}
		}
	}()

	if s == nil {
		return fmt.Errorf("runtime: session is nil")
	}

	return fn()
}

func (s *Session) cleanupInstanceTree(inst *Instance) {
	if inst == nil {
		return
	}

	inst.mu.Lock()
	children := make([]*Instance, len(inst.Children))
	copy(children, inst.Children)
	inst.mu.Unlock()

	for _, child := range children {
		s.cleanupInstanceTree(child)
	}

	s.cleanupInstance(inst)
}
