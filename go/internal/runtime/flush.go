package runtime

import (
	"errors"
	"fmt"

	"github.com/eleven-am/pondlive/go/internal/view/diff"
)

func (s *Session) SetAutoFlush(fn func()) {
	if s == nil {
		return
	}
	s.flushMu.Lock()
	s.autoFlush = fn
	s.flushMu.Unlock()
}

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

func (s *Session) IsFlushing() bool {
	if s == nil {
		return false
	}
	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	return s.flushing
}

func (s *Session) IsFlushPending() bool {
	if s == nil {
		return false
	}
	s.flushMu.Lock()
	defer s.flushMu.Unlock()
	return s.pendingFlush
}

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
		s.Root.NextHandlerIndex = 0
		s.View = s.convertWorkToView(s.Root.WorkTree, s.Root)
	}

	if s.Bus != nil {
		patches := diff.Diff(s.PrevView, s.View)
		if len(patches) > 0 {
			s.Bus.PublishFramePatch(patches)
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

func (s *Session) clearRenderedFlags(inst *Instance) {
	if inst == nil {
		return
	}

	inst.RenderedThisFlush = false
	inst.ChildRenderIndex = 0
	inst.ReferencedChildren = make(map[string]bool)

	inst.mu.Lock()
	children := make([]*Instance, len(inst.Children))
	copy(children, inst.Children)
	inst.mu.Unlock()

	for _, child := range children {
		s.clearRenderedFlags(child)
	}
}

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

func (s *Session) collectRenderedComponents(inst *Instance, result map[*Instance]struct{}) {
	if inst == nil {
		return
	}

	result[inst] = struct{}{}

	inst.mu.Lock()
	children := make([]*Instance, len(inst.Children))
	copy(children, inst.Children)
	inst.mu.Unlock()

	for _, child := range children {
		s.collectRenderedComponents(child, result)
	}
}

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

func (s *Session) pruneUnreferencedChildren(inst *Instance) {
	if inst == nil {
		return
	}

	inst.mu.Lock()
	referencedChildren := inst.ReferencedChildren
	currentChildren := make([]*Instance, len(inst.Children))
	copy(currentChildren, inst.Children)
	inst.mu.Unlock()

	var kept []*Instance
	for _, child := range currentChildren {
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

	inst.mu.Lock()
	inst.Children = kept
	inst.mu.Unlock()
}

func (s *Session) clearCurrentHandlers() {
	s.handlerIDsMu.Lock()
	s.currentHandlerIDs = make(map[string]bool)
	s.handlerIDsMu.Unlock()
}

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
