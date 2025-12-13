package runtime

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/view"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestMemoizedParentNotUnmounted(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	parent := &Instance{
		ID:                 "parent",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		Children:           []*Instance{},
		ReferencedChildren: make(map[string]bool),
		Providers:          make(map[any]any),
	}

	child := &Instance{
		ID:                 "child",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		Parent:             parent,
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}

	parent.Children = append(parent.Children, child)
	parent.ReferencedChildren["child"] = true

	sess.Root = parent
	sess.Components["parent"] = parent
	sess.Components["child"] = child

	parent.RenderedThisFlush = true
	child.RenderedThisFlush = true

	parentProviderCleared := false
	parent.Providers["test-context"] = "test-value"
	parent.RegisterCleanup(func() {
		parentProviderCleared = true
	})

	sess.MountedComponents[parent] = struct{}{}
	sess.MountedComponents[child] = struct{}{}

	parent.RenderedThisFlush = false
	child.RenderedThisFlush = true

	sess.detectAndCleanupUnmounted()

	if _, mounted := sess.MountedComponents[parent]; !mounted {
		t.Error("memoized parent should still be in MountedComponents")
	}

	if parent.Providers == nil || parent.Providers["test-context"] == nil {
		t.Error("memoized parent's providers should NOT be cleared")
	}

	if parentProviderCleared {
		t.Error("memoized parent's cleanup should NOT have run")
	}

	if _, mounted := sess.MountedComponents[child]; !mounted {
		t.Error("child should still be in MountedComponents")
	}
}

func TestActualUnmountStillWorks(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	parent := &Instance{
		ID:                 "parent",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		Children:           []*Instance{},
		ReferencedChildren: make(map[string]bool),
	}

	child := &Instance{
		ID:        "child",
		Fn:        func(*Ctx, any, []work.Item) work.Node { return nil },
		Parent:    parent,
		HookFrame: []HookSlot{},
	}

	parent.Children = append(parent.Children, child)
	sess.Root = parent
	sess.Components["parent"] = parent
	sess.Components["child"] = child

	sess.MountedComponents[parent] = struct{}{}
	sess.MountedComponents[child] = struct{}{}

	childCleanupRan := false
	child.RegisterCleanup(func() {
		childCleanupRan = true
	})

	parent.Children = []*Instance{}
	parent.RenderedThisFlush = true
	child.RenderedThisFlush = false

	sess.detectAndCleanupUnmounted()

	if _, mounted := sess.MountedComponents[child]; mounted {
		t.Error("removed child should NOT be in MountedComponents")
	}

	if !childCleanupRan {
		t.Error("removed child's cleanup should have run")
	}
}

func TestEffectRunsOutsideLock(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		Bus:               protocol.NewBus(),
	}

	inst := &Instance{
		ID:                 "test",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = inst
	sess.Components["test"] = inst
	sess.MountedComponents[inst] = struct{}{}

	markDirtyCalled := atomic.Bool{}
	effectCompleted := atomic.Bool{}

	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	UseEffect(ctx, func() func() {

		sess.MarkDirty(inst)
		markDirtyCalled.Store(true)
		effectCompleted.Store(true)
		return nil
	})

	done := make(chan struct{})
	go func() {
		sess.runPendingEffects()
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(2 * time.Second):
		t.Fatal("effect execution deadlocked - effects are running under session lock")
	}

	if !effectCompleted.Load() {
		t.Error("effect should have completed")
	}

	if !markDirtyCalled.Load() {
		t.Error("effect should have been able to call MarkDirty")
	}
}

func TestEffectCleanupRunsOutsideLock(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	inst := &Instance{
		ID:        "test",
		Fn:        func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame: []HookSlot{},
	}
	sess.Root = inst
	sess.Components["test"] = inst

	cleanupCompleted := atomic.Bool{}
	markDirtyInCleanup := atomic.Bool{}

	sess.PendingCleanups = append(sess.PendingCleanups, cleanupTask{
		instance: inst,
		fn: func() {

			sess.MarkDirty(inst)
			markDirtyInCleanup.Store(true)
			cleanupCompleted.Store(true)
		},
	})

	done := make(chan struct{})
	go func() {
		sess.runPendingEffects()
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(2 * time.Second):
		t.Fatal("cleanup execution deadlocked")
	}

	if !cleanupCompleted.Load() {
		t.Error("cleanup should have completed")
	}

	if !markDirtyInCleanup.Load() {
		t.Error("cleanup should have been able to call MarkDirty")
	}
}

func TestConcurrentFlushAndMarkDirty(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	inst := &Instance{
		ID:                 "test",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = inst
	sess.Components["test"] = inst
	sess.MountedComponents[inst] = struct{}{}

	effectStarted := make(chan struct{})
	effectCanFinish := make(chan struct{})
	markDirtyDone := make(chan struct{})

	sess.PendingEffects = append(sess.PendingEffects, effectTask{
		instance:  inst,
		hookIndex: 0,
		fn: func() func() {
			close(effectStarted)
			<-effectCanFinish
			return nil
		},
	})

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		sess.runPendingEffects()
	}()

	go func() {
		defer wg.Done()
		<-effectStarted

		sess.MarkDirty(inst)
		close(markDirtyDone)
	}()

	select {
	case <-markDirtyDone:

		close(effectCanFinish)
	case <-time.After(2 * time.Second):
		t.Fatal("MarkDirty deadlocked while effect was running")
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(2 * time.Second):
		t.Fatal("test timed out")
	}
}

func TestSetAutoFlush(t *testing.T) {
	sess := &Session{}

	called := false
	sess.SetAutoFlush(func() {
		called = true
	})

	sess.Root = &Instance{
		ID:                 "root",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Components = map[string]*Instance{"root": sess.Root}

	sess.RequestFlush()

	if !called {
		t.Error("autoFlush callback should have been called")
	}
}

func TestSetAutoFlushNilSession(t *testing.T) {
	var sess *Session

	sess.SetAutoFlush(func() {})
}

func TestRequestFlushWithoutCallback(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	inst := &Instance{
		ID:                 "root",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = inst
	sess.Components["root"] = inst

	sess.RequestFlush()

	if sess.IsFlushPending() {
		t.Error("pendingFlush should be false after synchronous flush")
	}
}

func TestRequestFlushBatching(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	inst := &Instance{
		ID:                 "root",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = inst
	sess.Components["root"] = inst

	callCount := atomic.Int32{}
	sess.SetAutoFlush(func() {
		callCount.Add(1)

	})

	sess.RequestFlush()
	sess.RequestFlush()
	sess.RequestFlush()

	if callCount.Load() != 1 {
		t.Errorf("expected 1 callback invocation, got %d", callCount.Load())
	}

	if !sess.IsFlushPending() {
		t.Error("pendingFlush should be true until Flush is called")
	}
}

func TestFlushGuardPreventsReentrant(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	inst := &Instance{
		ID:                 "root",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = inst
	sess.Components["root"] = inst
	sess.MountedComponents[inst] = struct{}{}

	reentrantCompleted := atomic.Bool{}
	reentrantReturnedEarly := atomic.Bool{}

	sess.PendingEffects = append(sess.PendingEffects, effectTask{
		instance:  inst,
		hookIndex: 0,
		fn: func() func() {

			beforePending := sess.IsFlushPending()
			err := sess.Flush()

			if err == nil && sess.IsFlushPending() && !beforePending {
				reentrantReturnedEarly.Store(true)
			}
			reentrantCompleted.Store(true)
			return nil
		},
	})

	done := make(chan struct{})
	go func() {
		_ = sess.Flush()
		close(done)
	}()

	select {
	case <-done:

	case <-time.After(2 * time.Second):
		t.Fatal("re-entrant Flush caused deadlock")
	}

	if !reentrantCompleted.Load() {
		t.Error("re-entrant flush attempt should have completed")
	}

	if !reentrantReturnedEarly.Load() {
		t.Error("re-entrant Flush should return early and mark pending")
	}
}

func TestIsFlushing(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	inst := &Instance{
		ID:                 "root",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = inst
	sess.Components["root"] = inst
	sess.MountedComponents[inst] = struct{}{}

	if sess.IsFlushing() {
		t.Error("IsFlushing should be false before Flush")
	}

	flushingDuringEffect := atomic.Bool{}

	sess.PendingEffects = append(sess.PendingEffects, effectTask{
		instance:  inst,
		hookIndex: 0,
		fn: func() func() {
			flushingDuringEffect.Store(sess.IsFlushing())
			return nil
		},
	})

	_ = sess.Flush()

	if !flushingDuringEffect.Load() {
		t.Error("IsFlushing should be true during effect execution")
	}

	if sess.IsFlushing() {
		t.Error("IsFlushing should be false after Flush completes")
	}
}

func TestIsFlushPending(t *testing.T) {
	sess := &Session{}

	if sess.IsFlushPending() {
		t.Error("IsFlushPending should be false initially")
	}

	sess.Root = &Instance{
		ID:                 "root",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Components = map[string]*Instance{"root": sess.Root}
	sess.MountedComponents = make(map[*Instance]struct{})
	sess.PendingEffects = []effectTask{}
	sess.PendingCleanups = []cleanupTask{}

	flushCalled := make(chan struct{})
	sess.SetAutoFlush(func() {

		close(flushCalled)
	})

	sess.RequestFlush()

	select {
	case <-flushCalled:
	case <-time.After(time.Second):
		t.Fatal("autoFlush should have been called")
	}

	if !sess.IsFlushPending() {
		t.Error("IsFlushPending should be true after RequestFlush (before Flush)")
	}

	_ = sess.Flush()

	if sess.IsFlushPending() {
		t.Error("IsFlushPending should be false after Flush")
	}
}

func TestFlushNilSession(t *testing.T) {
	var sess *Session

	sess.RequestFlush()
	if sess.IsFlushing() {
		t.Error("IsFlushing on nil should return false")
	}
	if sess.IsFlushPending() {
		t.Error("IsFlushPending on nil should return false")
	}
}

func TestRequestFlushDuringFlushTriggersReflush(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	renderCount := atomic.Int32{}

	inst := &Instance{
		ID: "root",
		Fn: func(*Ctx, any, []work.Item) work.Node {
			renderCount.Add(1)
			return nil
		},
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = inst
	sess.Components["root"] = inst
	sess.MountedComponents[inst] = struct{}{}

	sess.PendingEffects = append(sess.PendingEffects, effectTask{
		instance:  inst,
		hookIndex: 0,
		fn: func() func() {
			if renderCount.Load() == 1 {

				sess.MarkDirty(inst)

				sess.RequestFlush()
			}
			return nil
		},
	})

	_ = sess.Flush()

	if renderCount.Load() < 2 {
		t.Errorf("expected at least 2 renders (original + reflush), got %d", renderCount.Load())
	}
}

func TestConcurrentRequestFlush(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	inst := &Instance{
		ID:                 "root",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = inst
	sess.Components["root"] = inst
	sess.MountedComponents[inst] = struct{}{}

	callbackCount := atomic.Int32{}
	sess.SetAutoFlush(func() {
		callbackCount.Add(1)

		_ = sess.Flush()
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			sess.RequestFlush()
		}()
	}

	wg.Wait()

	count := callbackCount.Load()
	if count > 50 {
		t.Errorf("expected batching to reduce callbacks, got %d (should be much less than 100)", count)
	}
}

func TestCollectRenderedComponentsIncludesAllTreeNodes(t *testing.T) {
	sess := &Session{}

	grandchild := &Instance{
		ID:        "grandchild",
		HookFrame: []HookSlot{},
	}

	child1 := &Instance{
		ID:        "child1",
		HookFrame: []HookSlot{},
		Children:  []*Instance{grandchild},
	}
	grandchild.Parent = child1

	child2 := &Instance{
		ID:        "child2",
		HookFrame: []HookSlot{},
	}

	root := &Instance{
		ID:        "root",
		HookFrame: []HookSlot{},
		Children:  []*Instance{child1, child2},
	}
	child1.Parent = root
	child2.Parent = root

	root.RenderedThisFlush = false
	child1.RenderedThisFlush = false
	child2.RenderedThisFlush = true
	grandchild.RenderedThisFlush = true

	result := make(map[*Instance]struct{})
	sess.collectRenderedComponents(root, result)

	if _, ok := result[root]; !ok {
		t.Error("root should be in result even though it didn't render")
	}
	if _, ok := result[child1]; !ok {
		t.Error("child1 should be in result even though it didn't render")
	}
	if _, ok := result[child2]; !ok {
		t.Error("child2 should be in result")
	}
	if _, ok := result[grandchild]; !ok {
		t.Error("grandchild should be in result")
	}

	if len(result) != 4 {
		t.Errorf("expected 4 components in result, got %d", len(result))
	}
}

func TestChildrenSliceConcurrentAccess(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	root := &Instance{
		ID:                 "root",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		Children:           []*Instance{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = root
	sess.Components["root"] = root

	for i := 0; i < 10; i++ {
		child := &Instance{
			ID:                 "child-" + string(rune('0'+i)),
			Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
			Parent:             root,
			HookFrame:          []HookSlot{},
			ReferencedChildren: make(map[string]bool),
		}
		root.Children = append(root.Children, child)
		root.ReferencedChildren[child.ID] = true
		sess.Components[child.ID] = child
		sess.MountedComponents[child] = struct{}{}
	}
	sess.MountedComponents[root] = struct{}{}

	done := make(chan struct{})

	go func() {
		for i := 0; i < 100; i++ {
			sess.clearRenderedFlags(root)
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			result := make(map[*Instance]struct{})
			sess.collectRenderedComponents(root, result)
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 100; i++ {
			root.mu.Lock()
			if len(root.Children) > 5 {
				root.Children = root.Children[:5]
			}
			for j := len(root.Children); j < 10; j++ {
				child := &Instance{
					ID:        "child-" + string(rune('0'+j)),
					Parent:    root,
					HookFrame: []HookSlot{},
				}
				root.Children = append(root.Children, child)
			}
			root.mu.Unlock()
		}
		done <- struct{}{}
	}()

	<-done
	<-done
	<-done
}

func TestPruneUnreferencedChildrenConcurrentSafety(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	root := &Instance{
		ID:                 "root",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		Children:           []*Instance{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = root
	sess.Components["root"] = root
	sess.MountedComponents[root] = struct{}{}

	for i := 0; i < 5; i++ {
		child := &Instance{
			ID:                 "child-" + string(rune('0'+i)),
			Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
			Parent:             root,
			HookFrame:          []HookSlot{},
			ReferencedChildren: make(map[string]bool),
		}
		root.Children = append(root.Children, child)
		root.ReferencedChildren[child.ID] = true
		sess.Components[child.ID] = child
		sess.MountedComponents[child] = struct{}{}
	}

	done := make(chan struct{})

	go func() {
		for i := 0; i < 50; i++ {
			sess.pruneUnreferencedChildren(root)
		}
		done <- struct{}{}
	}()

	go func() {
		for i := 0; i < 50; i++ {
			root.mu.Lock()
			root.ReferencedChildren = make(map[string]bool)
			for _, c := range root.Children {
				root.ReferencedChildren[c.ID] = true
			}
			root.mu.Unlock()
		}
		done <- struct{}{}
	}()

	<-done
	<-done
}

func TestHasErrorBoundary(t *testing.T) {
	sess := &Session{}

	t.Run("returns false for nil instance", func(t *testing.T) {
		if sess.hasErrorBoundary(nil) {
			t.Error("expected false for nil instance")
		}
	})

	t.Run("returns false for instance without error boundary", func(t *testing.T) {
		inst := &Instance{
			ID:        "test",
			HookFrame: []HookSlot{},
		}
		if sess.hasErrorBoundary(inst) {
			t.Error("expected false for empty hook frame")
		}
	})

	t.Run("returns false for instance with other hooks", func(t *testing.T) {
		inst := &Instance{
			ID: "test",
			HookFrame: []HookSlot{
				{Type: HookTypeState, Value: nil},
				{Type: HookTypeEffect, Value: nil},
				{Type: HookTypeMemo, Value: nil},
			},
		}
		if sess.hasErrorBoundary(inst) {
			t.Error("expected false when no error boundary hook")
		}
	})

	t.Run("returns true for instance with error boundary", func(t *testing.T) {
		inst := &Instance{
			ID: "test",
			HookFrame: []HookSlot{
				{Type: HookTypeState, Value: nil},
				{Type: HookTypeErrorBoundary, Value: nil},
				{Type: HookTypeEffect, Value: nil},
			},
		}
		if !sess.hasErrorBoundary(inst) {
			t.Error("expected true when error boundary hook is present")
		}
	})
}

func TestPropagateEffectErrors(t *testing.T) {
	t.Run("nil session", func(t *testing.T) {
		var sess *Session
		sess.propagateEffectErrors(nil)
	})

	t.Run("empty errors", func(t *testing.T) {
		sess := &Session{}
		sess.propagateEffectErrors([]*effectErrorRecord{})
	})

	t.Run("marks ancestor with error boundary dirty", func(t *testing.T) {
		sess := &Session{
			DirtyQueue: []*Instance{},
			DirtySet:   make(map[*Instance]struct{}),
		}

		parent := &Instance{
			ID: "parent",
			HookFrame: []HookSlot{
				{Type: HookTypeErrorBoundary, Value: nil},
			},
		}

		child := &Instance{
			ID:        "child",
			Parent:    parent,
			HookFrame: []HookSlot{},
		}

		errRec := &effectErrorRecord{
			instance:  child,
			hookIndex: 0,
			err:       &Error{Message: "test error"},
			phase:     "run",
		}

		sess.propagateEffectErrors([]*effectErrorRecord{errRec})

		if _, ok := sess.DirtySet[parent]; !ok {
			t.Error("expected parent with error boundary to be marked dirty")
		}
	})

	t.Run("sets effect error on instance", func(t *testing.T) {
		sess := &Session{
			DirtyQueue: []*Instance{},
			DirtySet:   make(map[*Instance]struct{}),
		}

		inst := &Instance{
			ID:        "test",
			HookFrame: []HookSlot{},
		}

		testErr := &Error{Message: "test error"}
		errRec := &effectErrorRecord{
			instance:  inst,
			hookIndex: 0,
			err:       testErr,
			phase:     "run",
		}

		sess.propagateEffectErrors([]*effectErrorRecord{errRec})

		if inst.EffectError != testErr {
			t.Error("expected effect error to be set on instance")
		}
	})
}

func TestRunEffectsOutsideLockNilSession(t *testing.T) {
	var sess *Session
	sess.runEffectsOutsideLock(nil, nil)
}

func TestCollectDirtyComponentsLockedNilSession(t *testing.T) {
	var sess *Session
	result := sess.collectDirtyComponentsLocked()
	if result != nil {
		t.Errorf("expected nil for nil session, got %v", result)
	}
}

func TestMarkDirtyNilSession(t *testing.T) {
	var sess *Session
	sess.MarkDirty(&Instance{ID: "test"})
}

func TestMarkDirtyNilInstance(t *testing.T) {
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	sess.MarkDirty(nil)
	if len(sess.DirtyQueue) != 0 {
		t.Error("expected empty dirty queue")
	}
}

func TestDetectAndCleanupUnmountedNilSession(t *testing.T) {
	var sess *Session
	sess.detectAndCleanupUnmounted()
}

func TestRunPendingEffectsNilSession(t *testing.T) {
	var sess *Session
	sess.runPendingEffects()
}

func TestRunWithEffectRecoveryNoPanic(t *testing.T) {
	sess := &Session{SessionID: "test-sess"}
	inst := &Instance{ID: "test-comp", HookFrame: []HookSlot{}}

	called := false
	err := sess.runWithEffectRecovery("run", inst, 0, func() {
		called = true
	})

	if !called {
		t.Error("expected function to be called")
	}
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRunWithEffectRecoveryRecoversPanic(t *testing.T) {
	sess := &Session{SessionID: "test-sess"}
	inst := &Instance{ID: "test-comp", HookFrame: []HookSlot{}}

	defer func() {
		if r := recover(); r != nil {
			t.Error("panic should have been recovered by runWithEffectRecovery")
		}
	}()

	sess.runWithEffectRecovery("run", inst, 0, func() {
		panic("test panic")
	})
}

func TestRunWithEffectRecoveryCleanupPhaseRecoversPanic(t *testing.T) {
	sess := &Session{SessionID: "test-sess"}
	inst := &Instance{ID: "test-comp", HookFrame: []HookSlot{}}

	defer func() {
		if r := recover(); r != nil {
			t.Error("cleanup panic should have been recovered")
		}
	}()

	sess.runWithEffectRecovery("cleanup", inst, 0, func() {
		panic("cleanup panic")
	})
}

func TestRunWithEffectRecoveryNegativeHookIndexRecoversPanic(t *testing.T) {
	sess := &Session{SessionID: "test-sess"}
	inst := &Instance{ID: "test-comp", HookFrame: []HookSlot{}}

	defer func() {
		if r := recover(); r != nil {
			t.Error("panic should have been recovered")
		}
	}()

	sess.runWithEffectRecovery("run", inst, -1, func() {
		panic("test panic")
	})
}

func TestRunWithEffectRecoveryWithParentRecoversPanic(t *testing.T) {
	sess := &Session{SessionID: "test-sess"}
	parent := &Instance{ID: "parent-comp"}
	inst := &Instance{ID: "test-comp", Parent: parent, HookFrame: []HookSlot{}}

	defer func() {
		if r := recover(); r != nil {
			t.Error("panic should have been recovered")
		}
	}()

	sess.runWithEffectRecovery("run", inst, 0, func() {
		panic("test panic")
	})
}

func TestRunWithEffectRecoveryWithBusRecoversPanic(t *testing.T) {
	bus := protocol.NewBus()
	sess := &Session{SessionID: "test-sess", Bus: bus}
	inst := &Instance{ID: "test-comp", HookFrame: []HookSlot{}}

	defer func() {
		if r := recover(); r != nil {
			t.Error("panic should have been recovered")
		}
	}()

	sess.runWithEffectRecovery("run", inst, 0, func() {
		panic("test panic with bus")
	})
}

func TestPropagateConvertRenderErrorsNilSession(t *testing.T) {
	var sess *Session
	sess.propagateConvertRenderErrors()
}

func TestPropagateConvertRenderErrorsEmptyErrors(t *testing.T) {
	sess := &Session{
		convertRenderErrors: []*Instance{},
	}
	sess.propagateConvertRenderErrors()
	if len(sess.convertRenderErrors) != 0 {
		t.Error("expected convertRenderErrors to be empty after propagation")
	}
}

func TestPropagateConvertRenderErrorsMarksAncestorDirty(t *testing.T) {
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}

	parent := &Instance{
		ID: "parent",
		HookFrame: []HookSlot{
			{Type: HookTypeErrorBoundary, Value: nil},
		},
	}

	child := &Instance{
		ID:          "child",
		Parent:      parent,
		HookFrame:   []HookSlot{},
		RenderError: &Error{Message: "test error"},
	}

	sess.convertRenderErrors = []*Instance{child}
	sess.propagateConvertRenderErrors()

	if _, ok := sess.DirtySet[parent]; !ok {
		t.Error("expected parent with error boundary to be marked dirty")
	}

	if sess.convertRenderErrors != nil {
		t.Error("expected convertRenderErrors to be cleared after propagation")
	}
}

func TestPropagateConvertRenderErrorsMultipleErrorsSameAncestor(t *testing.T) {
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}

	parent := &Instance{
		ID: "parent",
		HookFrame: []HookSlot{
			{Type: HookTypeErrorBoundary, Value: nil},
		},
	}

	child1 := &Instance{
		ID:          "child1",
		Parent:      parent,
		HookFrame:   []HookSlot{},
		RenderError: &Error{Message: "error 1"},
	}

	child2 := &Instance{
		ID:          "child2",
		Parent:      parent,
		HookFrame:   []HookSlot{},
		RenderError: &Error{Message: "error 2"},
	}

	sess.convertRenderErrors = []*Instance{child1, child2}
	sess.propagateConvertRenderErrors()

	if _, ok := sess.DirtySet[parent]; !ok {
		t.Error("expected parent with error boundary to be marked dirty")
	}

	if len(sess.DirtyQueue) != 1 {
		t.Errorf("expected parent to be marked dirty only once, got %d", len(sess.DirtyQueue))
	}
}

func TestPropagateConvertRenderErrorsNestedErrorBoundaries(t *testing.T) {
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}

	grandparent := &Instance{
		ID: "grandparent",
		HookFrame: []HookSlot{
			{Type: HookTypeErrorBoundary, Value: nil},
		},
	}

	parent := &Instance{
		ID:     "parent",
		Parent: grandparent,
		HookFrame: []HookSlot{
			{Type: HookTypeErrorBoundary, Value: nil},
		},
	}

	child := &Instance{
		ID:          "child",
		Parent:      parent,
		HookFrame:   []HookSlot{},
		RenderError: &Error{Message: "test error"},
	}

	sess.convertRenderErrors = []*Instance{child}
	sess.propagateConvertRenderErrors()

	if _, ok := sess.DirtySet[parent]; !ok {
		t.Error("expected inner parent with error boundary to be marked dirty")
	}

	if _, ok := sess.DirtySet[grandparent]; ok {
		t.Error("expected grandparent NOT to be marked dirty (inner boundary catches)")
	}
}

func TestPropagateConvertRenderErrorsNoErrorBoundary(t *testing.T) {
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
	}

	child := &Instance{
		ID:          "child",
		Parent:      parent,
		HookFrame:   []HookSlot{},
		RenderError: &Error{Message: "test error"},
	}

	sess.convertRenderErrors = []*Instance{child}
	sess.propagateConvertRenderErrors()

	if len(sess.DirtySet) != 0 {
		t.Error("expected no components marked dirty when no error boundary exists")
	}

	if sess.convertRenderErrors != nil {
		t.Error("expected convertRenderErrors to be cleared")
	}
}

func TestFlushClearsConvertRenderErrors(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		convertRenderErrors: []*Instance{
			{ID: "stale", RenderError: &Error{Message: "stale error"}},
		},
	}

	inst := &Instance{
		ID:                 "root",
		Fn:                 func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:          []HookSlot{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = inst
	sess.Components["root"] = inst
	sess.MountedComponents[inst] = struct{}{}

	_ = sess.Flush()

	if sess.convertRenderErrors != nil {
		t.Error("expected convertRenderErrors to be cleared at start of Flush")
	}
}

func TestConvertRenderErrorIntegrationWithErrorBoundary(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		MountedComponents: make(map[*Instance]struct{}),
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
	}

	panicChild := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		panic("child error")
	}

	var capturedError *Error
	parentComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		batch := UseErrorBoundary(ctx)
		if batch.HasErrors() {
			capturedError = batch.First()
			return &work.Text{Value: "Error: " + batch.First().Message}
		}
		return work.Component(panicChild)
	}

	root := &Instance{
		ID: "root",
		Fn: parentComponent,
	}
	sess.Root = root
	sess.Components["root"] = root

	_ = sess.Flush()

	if capturedError == nil {
		t.Fatal("expected error to be captured by UseErrorBoundary")
	}

	if capturedError.Message != "child error" {
		t.Errorf("expected error message 'child error', got '%s'", capturedError.Message)
	}

	text, ok := sess.View.(*view.Text)
	if !ok {
		t.Fatalf("expected view to be Text (fallback), got %T", sess.View)
	}
	if text.Text != "Error: child error" {
		t.Errorf("expected fallback text 'Error: child error', got '%s'", text.Text)
	}
}
