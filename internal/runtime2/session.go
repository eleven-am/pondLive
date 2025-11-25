package runtime2

import (
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/eleven-am/pondlive/go/internal/view"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// Session manages the entire component tree and view.
type Session struct {
	Root       *Instance            // Root component instance
	View       view.Node            // Current view tree
	PrevView   view.Node            // Previous view tree for diffing
	Components map[string]*Instance // Component ID -> Instance lookup

	Handlers      map[string]work.Handler // Handler ID -> Handler
	handlersMu    sync.RWMutex            // Protects Handlers map
	DirtyQueue    []*Instance             // Components needing re-render (slice for ordering)
	DirtySet      map[*Instance]struct{}  // Fast lookup for dirty components
	dirtyMu       sync.Mutex              // Protects dirty tracking
	nextHandlerID int                     // Counter for generating handler IDs

	// Element refs
	nextElementRefID int // Counter for generating element ref IDs

	// Scripts
	Scripts           map[string]*scriptSlot // Script ID -> scriptSlot
	scriptsMu         sync.RWMutex
	scriptEventSender func(scriptID, event string, data interface{}) error
	scriptSenderMu    sync.RWMutex

	// Message bus
	Bus *Bus // Pubsub message router for bidirectional messaging

	// DOM request manager for blocking queries
	domReqMgr   *domRequestManager
	domReqMgrMu sync.Mutex
	domTimeout  time.Duration // Configurable timeout for DOM operations

	// Handler tracking for cleanup
	currentHandlerIDs map[string]bool          // Handler IDs active in current flush
	allHandlerSubs    map[string]*Subscription // All handler subscriptions (for cleanup)
	handlerIDsMu      sync.Mutex

	// HTTP handlers (UseHandler hook)
	httpHandlers  map[string]*handlerEntry
	httpHandlerMu sync.RWMutex

	// Effects
	PendingEffects  []effectTask  // Effects to run after flush
	PendingCleanups []cleanupTask // Cleanup functions to run

	// Component lifecycle tracking
	MountedComponents map[*Instance]struct{} // Track mounted components

	// Session metadata
	SessionID string // Unique session identifier

	// Development mode and diagnostics
	devMode  bool               // Enable panic recovery and diagnostic reporting
	reporter DiagnosticReporter // Optional error reporter

	// Flush batching and control
	pendingFlush bool       // True if a flush has been requested but not yet started
	flushing     bool       // True if a flush is currently in progress
	autoFlush    func()     // Callback to trigger auto-flush (e.g., schedule on event loop)
	flushMu      sync.Mutex // Protects flush state (separate from mu to avoid deadlocks)

	mu sync.Mutex // General session lock
}

// DiagnosticReporter receives structured diagnostics captured during panic recovery.
type DiagnosticReporter interface {
	ReportDiagnostic(Diagnostic)
}

// Diagnostic captures error context for debugging.
type Diagnostic struct {
	Phase      string         // Where the panic occurred (e.g., "script:message", "event:h-1")
	Message    string         // Error message
	StackTrace string         // Stack trace
	Metadata   map[string]any // Additional context
}

// effectTask represents an effect to run after render.
type effectTask struct {
	instance  *Instance
	hookIndex int // Hook index to locate the effectCell for storing cleanup
	fn        func() func()
}

// cleanupTask represents a cleanup function to run.
type cleanupTask struct {
	instance *Instance
	fn       func()
}

// allocateElementRefID generates a unique element ref ID.
func (s *Session) allocateElementRefID() string {
	s.nextElementRefID++
	return fmt.Sprintf("ref-%d", s.nextElementRefID)
}

// getDOMRequestManager returns the DOM request manager, creating it if needed.
// Uses lazy initialization to avoid creating manager for sessions that don't use DOM queries.
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

// SetDOMTimeout sets the timeout for blocking DOM operations (Query, AsyncCall).
// Must be called before any DOM operations are performed.
// If timeout is 0, the default (5 seconds) is used.
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

// Close tears down the session and cleans up all resources.
// Runs all effect cleanups, unsubscribes handlers, and clears state.
func (s *Session) Close() {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

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

// SetDevMode enables or disables development mode features.
func (s *Session) SetDevMode(enabled bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.devMode = enabled
	s.mu.Unlock()
}

// SetDiagnosticReporter installs the error reporter.
func (s *Session) SetDiagnosticReporter(r DiagnosticReporter) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.reporter = r
	s.mu.Unlock()
}

// withRecovery wraps a function with panic recovery.
// Always recovers, logs, and reports diagnostics; devMode controls verbosity.
// Note: devMode and reporter are read without locking as they are
// configuration fields set during initialization and rarely change.
func (s *Session) withRecovery(phase string, fn func() error) error {
	defer func() {
		if r := recover(); r != nil {
			stack := string(debug.Stack())
			if s != nil && s.reporter != nil {
				s.reporter.ReportDiagnostic(Diagnostic{
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

// cleanupInstanceTree recursively cleans up an instance and all its children.
func (s *Session) cleanupInstanceTree(inst *Instance) {
	if inst == nil {
		return
	}

	for _, child := range inst.Children {
		s.cleanupInstanceTree(child)
	}

	s.cleanupInstance(inst)
}
