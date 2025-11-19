package runtime

// Cleanup is a function returned by effect setup to clean up resources.
type Cleanup func()

type effectCell struct {
	deps    []any
	cleanup Cleanup
	hasRun  bool
}

// UseEffect schedules side effects after the next flush and handles cleanup when deps change or unmount.
func UseEffect(ctx Ctx, setup func() Cleanup, deps ...any) {
	if ctx.frame == nil {
		panic("runtime2: UseEffect called outside render")
	}
	idx := ctx.frame.idx
	ctx.frame.idx++
	if idx >= len(ctx.frame.cells) {
		cell := &effectCell{deps: cloneDeps(deps)}
		ctx.frame.cells = append(ctx.frame.cells, cell)
		if ctx.sess != nil {
			ctx.sess.enqueueEffect(ctx.comp, idx, setup)
		}
		return
	}
	cell, ok := ctx.frame.cells[idx].(*effectCell)
	if !ok {
		panicHookMismatch(ctx.comp, idx, "UseEffect", ctx.frame.cells[idx])
	}

	if len(deps) == 0 {
		if ctx.sess != nil {
			ctx.sess.enqueueEffect(ctx.comp, idx, setup)
		}
		return
	}

	if !depsEqual(cell.deps, deps) {
		if ctx.sess != nil {
			ctx.sess.enqueueCleanup(ctx.comp, idx)
			ctx.sess.enqueueEffect(ctx.comp, idx, setup)
		}
		cell.deps = cloneDeps(deps)
	} else if !cell.hasRun {

		if ctx.sess != nil {
			ctx.sess.enqueueEffect(ctx.comp, idx, setup)
		}
	}
}

func (s *ComponentSession) enqueueEffect(comp *component, hookIndex int, setup func() Cleanup) {
	if s == nil || comp == nil || setup == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	task := effectTask{
		run: func() {
			cleanup := setup()

			if comp.frame != nil && hookIndex < len(comp.frame.cells) {
				if cell, ok := comp.frame.cells[hookIndex].(*effectCell); ok {
					cell.cleanup = cleanup
					cell.hasRun = true
				}
			}
		},
	}
	s.pendingEffects = append(s.pendingEffects, task)
}

func (s *ComponentSession) enqueueCleanup(comp *component, hookIndex int) {
	if s == nil || comp == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	task := cleanupTask{
		run: func() {
			if comp.frame != nil && hookIndex < len(comp.frame.cells) {
				if cell, ok := comp.frame.cells[hookIndex].(*effectCell); ok {
					if cell.cleanup != nil {
						cell.cleanup()
						cell.cleanup = nil
					}
					cell.hasRun = false
				}
			}
		},
	}
	s.pendingCleanups = append(s.pendingCleanups, task)
}
