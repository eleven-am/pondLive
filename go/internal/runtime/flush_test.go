package runtime

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/work"
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
		HookFrame:          []HookSlot{},
		Children:           []*Instance{},
		ReferencedChildren: make(map[string]bool),
		Providers:          make(map[any]any),
	}

	child := &Instance{
		ID:                 "child",
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
		HookFrame:          []HookSlot{},
		Children:           []*Instance{},
		ReferencedChildren: make(map[string]bool),
	}

	child := &Instance{
		ID:        "child",
		Fn:        func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:        func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn: func(*Ctx, any, []work.Node) work.Node {
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
		HookFrame:          []HookSlot{},
		Children:           []*Instance{},
		ReferencedChildren: make(map[string]bool),
	}
	sess.Root = root
	sess.Components["root"] = root

	for i := 0; i < 10; i++ {
		child := &Instance{
			ID:                 "child-" + string(rune('0'+i)),
			Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
		Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
			Fn:                 func(*Ctx, any, []work.Node) work.Node { return nil },
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
