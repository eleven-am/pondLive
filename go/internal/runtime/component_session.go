package runtime

import (
	"errors"
	"fmt"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom2/diff"
)

const maxEffectsPerFlush = 64

// ComponentSession drives component rendering, diffing, and event handling.
type ComponentSession struct {
	root       *component
	dirty      map[*component]struct{}
	components map[string]*component

	prevTree *dom2.StructuredNode

	handlers   map[string]dom2.EventHandler
	handlersMu sync.RWMutex

	uploads  map[string]*uploadSlot
	uploadMu sync.Mutex

	pendingEffects  []effectTask
	pendingCleanups []cleanupTask
	pendingPubsub   []pubsubTask

	pubsubSubs     map[string]pubsubSubscription
	pubsubProvider PubsubProvider
	pubsubMu       sync.RWMutex

	mountedComponents map[*component]struct{}

	domActions      []dom2.DOMActionEffect
	domActionSender func([]dom2.DOMActionEffect) error
	domGetHandler   func(ref string, selectors ...string) (map[string]any, error)
	domCallHandler  func(ref string, method string, args ...any) (any, error)

	nextRefID int

	sendPatch func([]dom2diff.Patch) error
	reporter  DiagnosticReporter

	// Router support - callback to get initial location (set by session package)
	getInitialLocation func() (path string, query map[string]string, hash string)

	mu sync.Mutex
}

// DiagnosticReporter receives structured diagnostics captured during panic recovery.
type DiagnosticReporter interface {
	ReportDiagnostic(Diagnostic)
}

// Diagnostic captures error context for debugging.
type Diagnostic struct {
	Phase      string
	Message    string
	StackTrace string
	Metadata   map[string]any
}

// NewSession constructs a session rooted at the provided component function.
func NewSession[P any](root Component[P], props P) *ComponentSession {
	sess := &ComponentSession{
		dirty:      make(map[*component]struct{}),
		components: make(map[string]*component),
		handlers:   make(map[string]dom2.EventHandler),
		uploads:    make(map[string]*uploadSlot),
		pubsubSubs: make(map[string]pubsubSubscription),
	}
	sess.root = newComponent(sess, nil, "root", root, props)
	return sess
}

// SetPatchSender installs the transport used to deliver diff operations.
func (s *ComponentSession) SetPatchSender(fn func([]dom2diff.Patch) error) {
	s.sendPatch = fn
}

// SetDiagnosticReporter installs the error reporter.
func (s *ComponentSession) SetDiagnosticReporter(r DiagnosticReporter) {
	s.reporter = r
}

// SetPubsubProvider installs the pubsub provider for UsePubsub hooks.
func (s *ComponentSession) SetPubsubProvider(p PubsubProvider) {
	if s == nil {
		return
	}
	s.pubsubProvider = p
}

// SetInitialLocationProvider installs a callback to get the initial location for router support.
func (s *ComponentSession) SetInitialLocationProvider(fn func() (path string, query map[string]string, hash string)) {
	s.getInitialLocation = fn
}

// GetInitialLocation returns the initial location if available.
func (s *ComponentSession) GetInitialLocation() (path string, query map[string]string, hash string, ok bool) {
	if s == nil || s.getInitialLocation == nil {
		return "", nil, "", false
	}
	p, q, h := s.getInitialLocation()
	return p, q, h, true
}

// Tree returns the last rendered StructuredNode tree.
// Returns nil if no render has occurred yet.
func (s *ComponentSession) Tree() *dom2.StructuredNode {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.prevTree
}

// Flush renders dirty components, diffs the tree, and sends patches.
func (s *ComponentSession) Flush() error {
	if s == nil || s.root == nil {
		return errors.New("runtime2: session not initialized")
	}

	return s.withRecovery("flush", func() error {

		s.mu.Lock()
		dirtyComponents := s.collectDirtyComponentsLocked()
		isFirstRender := s.prevTree == nil
		s.mu.Unlock()

		s.clearRenderedFlags()

		if isFirstRender {
			s.root.render()
		} else {
			for _, comp := range dirtyComponents {
				comp.render()
			}
		}

		s.detectAndCleanupUnmounted()

		nextTree := s.root.node

		var patches []dom2diff.Patch
		if s.prevTree != nil {
			patches = dom2diff.Diff(s.prevTree, nextTree)
		}

		handlers := s.collectHandlersFromTree(nextTree)

		s.mu.Lock()
		sendPatchFn := s.sendPatch
		effects := s.takeEffectsBatchLocked()
		cleanups := append([]cleanupTask(nil), s.pendingCleanups...)
		pubsubs := append([]pubsubTask(nil), s.pendingPubsub...)
		s.pendingCleanups = nil
		s.pendingPubsub = nil
		s.mu.Unlock()

		s.handlersMu.Lock()
		s.handlers = handlers
		s.handlersMu.Unlock()

		if sendPatchFn != nil && (len(patches) > 0 || s.prevTree == nil) {
			if err := sendPatchFn(patches); err != nil {
				return err
			}
		}

		snapshot := cloneTree(nextTree)
		s.mu.Lock()
		s.prevTree = snapshot
		s.mu.Unlock()

		runCleanups(cleanups)
		s.runPubsubTasks(pubsubs)
		runEffects(effects)

		s.mu.Lock()
		actions := append([]dom2.DOMActionEffect(nil), s.domActions...)
		s.domActions = nil
		sender := s.domActionSender
		s.mu.Unlock()

		if sender != nil && len(actions) > 0 {
			if err := sender(actions); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *ComponentSession) collectDirtyComponentsLocked() []*component {
	if len(s.dirty) == 0 {
		return nil
	}
	result := make([]*component, 0, len(s.dirty))
	for comp := range s.dirty {
		result = append(result, comp)
		delete(s.dirty, comp)
	}
	return result
}

func (s *ComponentSession) collectHandlersFromTree(node *dom2.StructuredNode) map[string]dom2.EventHandler {
	handlers := make(map[string]dom2.EventHandler)
	if node == nil {
		return handlers
	}
	s.collectHandlersFromNode(node, handlers)
	return handlers
}

func (s *ComponentSession) collectHandlersFromNode(node *dom2.StructuredNode, handlers map[string]dom2.EventHandler) {
	if node == nil {
		return
	}

	for _, binding := range node.Events {
		if binding.Key != "" && binding.Handler != nil {
			handlers[binding.Key] = binding.Handler
		}
	}

	for i := range node.Children {
		s.collectHandlersFromNode(node.Children[i], handlers)
	}
}

func runCleanups(tasks []cleanupTask) {
	for _, task := range tasks {
		task.run()
	}
}

func runEffects(tasks []effectTask) {
	for _, task := range tasks {
		task.run()
	}
}

// clearRenderedFlags clears the renderedThisFlush flag for all components that were rendered last flush.
func (s *ComponentSession) clearRenderedFlags() {
	for comp := range s.mountedComponents {
		comp.renderedThisFlush = false
	}
}

// detectAndCleanupUnmounted finds components that were rendered last flush but not this flush,
// runs their effect cleanups, and updates the mounted set.
func (s *ComponentSession) detectAndCleanupUnmounted() {
	newMounted := make(map[*component]struct{})

	s.collectRenderedComponents(s.root, newMounted)

	for comp := range s.mountedComponents {
		if _, stillRendered := newMounted[comp]; !stillRendered {
			s.runComponentCleanups(comp)
		}
	}

	s.mountedComponents = newMounted
}

// collectRenderedComponents recursively collects all components that were rendered this flush.
func (s *ComponentSession) collectRenderedComponents(comp *component, rendered map[*component]struct{}) {
	if comp == nil {
		return
	}

	if comp.renderedThisFlush {
		rendered[comp] = struct{}{}

		for _, child := range comp.children {
			if child.renderedThisFlush {
				s.collectRenderedComponents(child, rendered)
			}
		}
	}
}

// runComponentCleanups runs all effect cleanups for a component.
func (s *ComponentSession) runComponentCleanups(comp *component) {
	if comp == nil || comp.frame == nil {
		return
	}

	for _, cell := range comp.frame.cells {
		if effectCell, ok := cell.(*effectCell); ok {
			if effectCell.cleanup != nil {
				effectCell.cleanup()
				effectCell.cleanup = nil
			}
		}
	}
}

func (s *ComponentSession) runPubsubTasks(tasks []pubsubTask) {

}

func (s *ComponentSession) takeEffectsBatchLocked() []effectTask {
	if len(s.pendingEffects) == 0 {
		return nil
	}
	limit := len(s.pendingEffects)
	if limit > maxEffectsPerFlush {
		limit = maxEffectsPerFlush
	}
	batch := make([]effectTask, limit)
	copy(batch, s.pendingEffects[:limit])
	if len(s.pendingEffects) > limit {
		s.pendingEffects = append([]effectTask(nil), s.pendingEffects[limit:]...)
	} else {
		s.pendingEffects = nil
	}
	return batch
}

func (s *ComponentSession) markDirty(comp *component) {
	if s == nil || comp == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.dirty == nil {
		s.dirty = make(map[*component]struct{})
	}
	s.dirty[comp] = struct{}{}
}

func (s *ComponentSession) enqueueDOMAction(effect dom2.DOMActionEffect) {
	if s == nil {
		return
	}
	if effect.Type == "" {
		effect.Type = "dom"
	}
	s.mu.Lock()
	s.domActions = append(s.domActions, effect)
	s.mu.Unlock()
}

func (s *ComponentSession) SetDOMActionSender(fn func([]dom2.DOMActionEffect) error) {
	s.mu.Lock()
	s.domActionSender = fn
	s.mu.Unlock()
}

func (s *ComponentSession) SetDOMRequestHandlers(get func(ref string, selectors ...string) (map[string]any, error), call func(ref string, method string, args ...any) (any, error)) {
	s.mu.Lock()
	s.domGetHandler = get
	s.domCallHandler = call
	s.mu.Unlock()
}

func (s *ComponentSession) domGet(ref string, selectors ...string) (map[string]any, error) {
	s.mu.Lock()
	handler := s.domGetHandler
	s.mu.Unlock()
	if handler == nil {
		return nil, fmt.Errorf("runtime2: DOMGet handler not configured")
	}
	return handler(ref, selectors...)
}

func (s *ComponentSession) domAsyncCall(ref string, method string, args ...any) (any, error) {
	s.mu.Lock()
	handler := s.domCallHandler
	s.mu.Unlock()
	if handler == nil {
		return nil, fmt.Errorf("runtime2: DOMAsyncCall handler not configured")
	}
	return handler(ref, method, args...)
}

func (s *ComponentSession) allocateElementRefID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("ref:%d", s.nextRefID)
	s.nextRefID++
	return id
}

func (s *ComponentSession) withRecovery(phase string, fn func() error) error {
	defer func() {
		if r := recover(); r != nil {
			if s.reporter != nil {
				s.reporter.ReportDiagnostic(Diagnostic{
					Phase:   phase,
					Message: fmt.Sprintf("%v", r),
				})
			}
		}
	}()
	return fn()
}

// HandleEvent dispatches an event to the registered handler.
func (s *ComponentSession) HandleEvent(id string, ev dom2.Event) error {
	if s == nil {
		return errors.New("runtime2: session is nil")
	}

	return s.withRecovery(fmt.Sprintf("event:%s", id), func() error {
		s.handlersMu.RLock()
		handler := s.handlers[id]
		s.handlersMu.RUnlock()

		if handler == nil {
			return fmt.Errorf("runtime2: handler not found: %s", id)
		}

		if updates := handler(ev); updates != nil {
			s.markDirty(s.root)
		}
		return nil
	})
}

// ComponentByID looks up a component by its ID.
// Returns nil if the component is not found.
func (s *ComponentSession) ComponentByID(id string) *component {
	if s == nil || id == "" {
		return nil
	}
	s.mu.Lock()
	comp := s.components[id]
	s.mu.Unlock()
	return comp
}

// Reset clears all component state and schedules a complete re-render.
// This is used for development mode error recovery.
// Returns true if reset was performed, false if session is not in a valid state.
func (s *ComponentSession) Reset() bool {
	if s == nil || s.root == nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Collect all cleanups before resetting
	var allCleanups []cleanupTask
	allCleanups = append(allCleanups, s.pendingCleanups...)

	// Walk component tree and collect effect cleanups
	s.walkComponentTree(s.root, func(comp *component) {
		if comp.frame != nil {
			for _, cell := range comp.frame.cells {
				if ec, ok := cell.(*effectCell); ok && ec.cleanup != nil {
					allCleanups = append(allCleanups, cleanupTask{run: ec.cleanup})
				}
			}
		}
	})

	// Clear all pending tasks
	s.pendingEffects = nil
	s.pendingCleanups = nil
	s.pendingPubsub = nil

	// Mark root dirty to trigger full re-render
	s.markDirtyLocked(s.root)

	// Run all cleanups after releasing lock
	go func() {
		for _, task := range allCleanups {
			if task.run != nil {
				task.run()
			}
		}
	}()

	return true
}

// ResetComponent resets a specific component and marks it dirty for re-render.
// This is used for router resets where only a specific component needs to be reinitialized.
// Returns true if the component was found and reset.
func (s *ComponentSession) ResetComponent(id string) bool {
	if s == nil || id == "" {
		return false
	}

	comp := s.ComponentByID(id)
	if comp == nil {
		return false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Collect effect cleanups for this component
	var cleanups []cleanupTask
	if comp.frame != nil {
		for _, cell := range comp.frame.cells {
			if ec, ok := cell.(*effectCell); ok && ec.cleanup != nil {
				cleanups = append(cleanups, cleanupTask{run: ec.cleanup})
			}
		}
	}

	// Mark component dirty to trigger re-render
	s.markDirtyLocked(comp)

	// Run cleanups after releasing lock
	go func() {
		for _, task := range cleanups {
			if task.run != nil {
				task.run()
			}
		}
	}()

	return true
}

func (s *ComponentSession) walkComponentTree(root *component, fn func(*component)) {
	if root == nil || fn == nil {
		return
	}
	fn(root)
	for _, child := range root.children {
		s.walkComponentTree(child, fn)
	}
}

func (s *ComponentSession) markDirtyLocked(comp *component) {
	if comp == nil {
		return
	}
	if s.dirty == nil {
		s.dirty = make(map[*component]struct{})
	}
	s.dirty[comp] = struct{}{}
}

type effectTask struct {
	run func()
}

type cleanupTask struct {
	run func()
}

type pubsubTask struct {
	run func()
	// ... existing fields ...
}

type pubsubSubscription struct {
	token    string
	topic    string
	handler  func([]byte, map[string]string)
	provider PubsubProvider
}
