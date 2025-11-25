package runtime

import (
	"errors"
	"fmt"

	"github.com/eleven-am/pondlive/go/internal/view/diff"
)

// Bus topic constants for frame publishing.
const (
	TopicFrame = "frame"
	EventPatch = "patch"
)

// SetAutoFlush sets the callback that will be invoked when a flush is requested.
// This allows external code (e.g., event loop) to schedule flushes appropriately.
// The callback should eventually call Flush() to execute the render cycle.
func (s *Session) SetAutoFlush(fn func()) {
	if s == nil {
		return
	}
	s.flushMu.Lock()
	s.autoFlush = fn
	s.flushMu.Unlock()
}

// RequestFlush requests a flush to be scheduled.
// Multiple requests are batched - only one flush will execute.
// If an autoFlush callback is set, it will be invoked.
// If no callback is set, Flush() is called directly.
func (s *Session) RequestFlush() {
	if s == nil {
		return
	}

	s.flushMu.Lock()

	if s.pendingFlush {
		s.flushMu.Unlock()
		return
	}

	s.pendingFlush = true

	if s.flushing {
		s.flushMu.Unlock()
		return
	}

	autoFlush := s.autoFlush
	s.flushMu.Unlock()

	if autoFlush != nil {
		autoFlush()
	} else {

		_ = s.Flush()
	}
}

// IsFlushing returns true if a flush is currently in progress.
func (s *Session) IsFlushing() bool {
	if s == nil {
		return false
	}
	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	return s.flushing
}

// IsFlushPending returns true if a flush has been requested but not yet started.
func (s *Session) IsFlushPending() bool {
	if s == nil {
		return false
	}
	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	return s.pendingFlush
}

// Flush executes the render/flush cycle.
// Renders dirty components, converts work tree to view tree, and runs effects.
// IMPORTANT: User code (effects/cleanups) runs OUTSIDE the session lock to prevent
// deadlocks when effects call back into session methods.
func (s *Session) Flush() error {
	if s == nil || s.Root == nil {
		return errors.New("runtime: session not initialized")
	}

	s.flushMu.Lock()
	if s.flushing {

		s.pendingFlush = true
		s.flushMu.Unlock()
		return nil
	}
	s.flushing = true
	s.pendingFlush = false
	s.flushMu.Unlock()

	defer func() {
		s.flushMu.Lock()
		s.flushing = false
		needsReflush := s.pendingFlush
		s.flushMu.Unlock()

		if needsReflush {
			_ = s.Flush()
		}
	}()

	s.mu.Lock()

	dirtyComponents := s.collectDirtyComponentsLocked()
	isFirstRender := s.PrevView == nil

	s.clearRenderedFlags(s.Root)

	if isFirstRender {
		s.resetRefsForComponent(s.Root)
		s.Root.Render(s)
	} else {
		for _, inst := range dirtyComponents {
			s.resetRefsForComponent(inst)
			inst.Render(s)
		}
	}

	s.clearCurrentHandlers()

	s.PrevView = s.View
	if s.Root.WorkTree != nil {
		s.View = s.convertWorkToView(s.Root.WorkTree, s.Root)
	}

	if s.Bus != nil {
		patches := diff.Diff(s.PrevView, s.View)
		if len(patches) > 0 {
			s.Bus.Publish(TopicFrame, EventPatch, patches)
		}
	}

	s.cleanupStaleHandlers()

	s.detectAndCleanupUnmounted()
	s.pruneUnreferencedChildren(s.Root)

	pendingEffects := append([]effectTask(nil), s.PendingEffects...)
	pendingCleanups := append([]cleanupTask(nil), s.PendingCleanups...)
	s.PendingEffects = s.PendingEffects[:0]
	s.PendingCleanups = s.PendingCleanups[:0]

	s.mu.Unlock()

	s.runEffectsOutsideLock(pendingEffects, pendingCleanups)

	return nil
}

// MarkDirty marks an instance as dirty and adds it to the dirty queue.
// Automatically requests a flush to schedule re-rendering.
func (s *Session) MarkDirty(inst *Instance) {
	if s == nil || inst == nil {
		return
	}

	s.dirtyMu.Lock()

	if s.DirtySet == nil {
		s.DirtySet = make(map[*Instance]struct{})
	}

	if _, exists := s.DirtySet[inst]; exists {
		s.dirtyMu.Unlock()
		return
	}

	inst.SetDirty(true)
	s.DirtySet[inst] = struct{}{}
	s.DirtyQueue = append(s.DirtyQueue, inst)
	s.dirtyMu.Unlock()

	s.RequestFlush()
}

// collectDirtyComponentsLocked collects all dirty components with ancestor pruning.
// If both a parent and child are dirty, only the parent is returned.
// Must be called with s.mu held.
func (s *Session) collectDirtyComponentsLocked() []*Instance {
	if s == nil {
		return nil
	}

	s.dirtyMu.Lock()
	defer s.dirtyMu.Unlock()

	var pruned []*Instance
	for _, inst := range s.DirtyQueue {
		hasAncestorDirty := false
		for ancestor := inst.Parent; ancestor != nil; ancestor = ancestor.Parent {
			if _, dirty := s.DirtySet[ancestor]; dirty {
				hasAncestorDirty = true
				break
			}
		}
		if !hasAncestorDirty {
			pruned = append(pruned, inst)
		}
	}

	s.DirtyQueue = s.DirtyQueue[:0]
	s.DirtySet = make(map[*Instance]struct{})

	return pruned
}

// clearRenderedFlags clears the RenderedThisFlush flag and resets child tracking for all components.
// This must be called at the start of each flush to ensure ChildRenderIndex is consistent
// when convertWorkToView walks cached work trees.
func (s *Session) clearRenderedFlags(inst *Instance) {
	if inst == nil {
		return
	}

	inst.RenderedThisFlush = false
	inst.ChildRenderIndex = 0
	inst.ReferencedChildren = make(map[string]bool)

	for _, child := range inst.Children {
		s.clearRenderedFlags(child)
	}
}

// resetRefsForComponent calls ResetAttachment on all element refs for a component.
func (s *Session) resetRefsForComponent(inst *Instance) {
	if inst == nil {
		return
	}

	for _, slot := range inst.HookFrame {
		if slot.Type == HookTypeElement {
			if ref, ok := slot.Value.(*ElementRef); ok {
				ref.ResetAttachment()
			}
		}
	}
}

// detectAndCleanupUnmounted detects unmounted components and runs cleanup.
func (s *Session) detectAndCleanupUnmounted() {
	if s == nil {
		return
	}

	rendered := make(map[*Instance]struct{})
	s.collectRenderedComponents(s.Root, rendered)

	if s.MountedComponents == nil {
		s.MountedComponents = make(map[*Instance]struct{})
	}

	var unmounted []*Instance
	for inst := range s.MountedComponents {
		if _, stillMounted := rendered[inst]; !stillMounted {
			unmounted = append(unmounted, inst)
		}
	}

	for _, inst := range unmounted {
		s.cleanupInstance(inst)
		delete(s.MountedComponents, inst)
	}

	s.MountedComponents = rendered
}

// collectRenderedComponents collects all components in the component tree.
// This includes both components that rendered this flush AND memoized components
// that skipped rendering but are still mounted. This is critical for correct
// unmount detection - a memoized parent that didn't render is still mounted.
func (s *Session) collectRenderedComponents(inst *Instance, result map[*Instance]struct{}) {
	if inst == nil {
		return
	}

	result[inst] = struct{}{}

	for _, child := range inst.Children {
		s.collectRenderedComponents(child, result)
	}
}

// cleanupInstance runs cleanup for an instance (effects, refs, providers, etc).
func (s *Session) cleanupInstance(inst *Instance) {
	if inst == nil {
		return
	}

	for _, slot := range inst.HookFrame {
		if slot.Type == HookTypeEffect {
			if cell, ok := slot.Value.(*effectCell); ok {
				if cell.cleanup != nil {
					s.PendingCleanups = append(s.PendingCleanups, cleanupTask{
						instance: inst,
						fn:       cell.cleanup,
					})
				}
			}
		}
	}

	inst.mu.Lock()
	inst.Providers = nil
	inst.mu.Unlock()

	inst.cleanupsMu.Lock()
	cleanups := inst.cleanups
	inst.cleanups = nil
	inst.cleanupsMu.Unlock()

	for _, cleanup := range cleanups {
		if cleanup != nil {
			func() {
				defer func() { recover() }()
				cleanup()
			}()
		}
	}
}

// pruneUnreferencedChildren removes children that weren't referenced during render.
func (s *Session) pruneUnreferencedChildren(inst *Instance) {
	if inst == nil {
		return
	}

	inst.mu.Lock()
	referencedChildren := inst.ReferencedChildren
	inst.mu.Unlock()

	var kept []*Instance
	for _, child := range inst.Children {
		if referencedChildren != nil && referencedChildren[child.ID] {
			kept = append(kept, child)
			s.pruneUnreferencedChildren(child)
		} else {
			s.cleanupInstance(child)
			if s.Components != nil {
				delete(s.Components, child.ID)
			}
		}
	}

	inst.Children = kept
}

// clearCurrentHandlers clears the tracking of current handlers before convert.
func (s *Session) clearCurrentHandlers() {
	s.handlerIDsMu.Lock()
	s.currentHandlerIDs = make(map[string]bool)
	s.handlerIDsMu.Unlock()
}

// cleanupStaleHandlers unsubscribes handlers that are no longer in the view tree.
func (s *Session) cleanupStaleHandlers() {
	s.handlerIDsMu.Lock()
	defer s.handlerIDsMu.Unlock()

	for handlerID, sub := range s.allHandlerSubs {
		if !s.currentHandlerIDs[handlerID] {

			if sub != nil {
				sub.Unsubscribe()
			}
			delete(s.allHandlerSubs, handlerID)
		}
	}
}

// runPendingEffects executes pending effects from the session.
// This is a convenience wrapper for tests and external callers.
// It collects pending effects under the lock, then runs them outside the lock.
func (s *Session) runPendingEffects() {
	if s == nil {
		return
	}

	s.mu.Lock()
	effects := append([]effectTask(nil), s.PendingEffects...)
	cleanups := append([]cleanupTask(nil), s.PendingCleanups...)
	s.PendingEffects = s.PendingEffects[:0]
	s.PendingCleanups = s.PendingCleanups[:0]
	s.mu.Unlock()

	s.runEffectsOutsideLock(effects, cleanups)
}

// runEffectsOutsideLock executes all pending effects/cleanups outside the session lock.
// This prevents deadlocks when effects call back into session methods like MarkDirty or Flush.
// MUST be called without holding s.mu.
func (s *Session) runEffectsOutsideLock(effects []effectTask, cleanups []cleanupTask) {
	if s == nil {
		return
	}

	for _, task := range cleanups {
		if task.fn != nil {
			phase := fmt.Sprintf("effect:cleanup:%s", task.instance.ID)
			_ = s.withRecovery(phase, func() error {
				task.fn()
				return nil
			})
		}
	}

	for _, task := range effects {
		if task.fn != nil {
			phase := fmt.Sprintf("effect:run:%s:%d", task.instance.ID, task.hookIndex)
			var cleanup func()
			_ = s.withRecovery(phase, func() error {
				cleanup = task.fn()
				return nil
			})

			if cleanup != nil && task.instance != nil && task.hookIndex < len(task.instance.HookFrame) {
				task.instance.mu.Lock()
				if slot := task.instance.HookFrame[task.hookIndex]; slot.Type == HookTypeEffect {
					if cell, ok := slot.Value.(*effectCell); ok {
						cell.cleanup = cleanup
					}
				}
				task.instance.mu.Unlock()
			}
		}
	}
}
