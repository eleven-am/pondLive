package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// TestGenericCleanupSystem verifies that RegisterCleanup works and cleanups run on unmount.
func TestGenericCleanupSystem(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{}

	cleanupRan := false
	inst.RegisterCleanup(func() {
		cleanupRan = true
	})

	if len(inst.cleanups) != 1 {
		t.Errorf("expected 1 cleanup registered, got %d", len(inst.cleanups))
	}

	sess.cleanupInstance(inst)

	if !cleanupRan {
		t.Error("expected cleanup to run")
	}

	if len(inst.cleanups) != 0 {
		t.Errorf("expected cleanups to be cleared after running, got %d", len(inst.cleanups))
	}
}

// TestScriptCleanupOnUnmount verifies that scripts are removed from registry when component unmounts.
func TestScriptCleanupOnUnmount(t *testing.T) {
	sess := &Session{
		Scripts:          make(map[string]*scriptSlot),
		nextElementRefID: 0,
	}

	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseScript(ctx, "console.log('test')")
	scriptID := handle.ID()

	if sess.Scripts[scriptID] == nil {
		t.Fatal("expected script to be registered")
	}

	sess.cleanupInstance(inst)

	if sess.Scripts[scriptID] != nil {
		t.Error("expected script to be removed from registry after cleanup")
	}
}

// TestMultipleCleanups verifies that multiple cleanups can be registered and all run.
func TestMultipleCleanups(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{}

	count := 0
	inst.RegisterCleanup(func() { count++ })
	inst.RegisterCleanup(func() { count++ })
	inst.RegisterCleanup(func() { count++ })

	if len(inst.cleanups) != 3 {
		t.Errorf("expected 3 cleanups registered, got %d", len(inst.cleanups))
	}

	sess.cleanupInstance(inst)

	if count != 3 {
		t.Errorf("expected all 3 cleanups to run, got %d", count)
	}
}

// TestCleanupWithEffects verifies that both effect cleanups and generic cleanups run.
func TestCleanupWithEffects(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		PendingEffects:  []effectTask{},
		PendingCleanups: []cleanupTask{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	effectCleanupRan := false
	UseEffect(ctx, func() func() {
		return func() {
			effectCleanupRan = true
		}
	}, "dep")

	sess.runPendingEffects()

	genericCleanupRan := false
	inst.RegisterCleanup(func() {
		genericCleanupRan = true
	})

	sess.cleanupInstance(inst)

	if !genericCleanupRan {
		t.Error("expected generic cleanup to run immediately")
	}

	if len(sess.PendingCleanups) != 1 {
		t.Errorf("expected 1 effect cleanup scheduled, got %d", len(sess.PendingCleanups))
	}

	sess.runPendingEffects()

	if !effectCleanupRan {
		t.Error("expected effect cleanup to run")
	}
}

// TestPruneUnreferencedChildrenRunsCleanup verifies cleanup runs when children are pruned.
func TestPruneUnreferencedChildrenRunsCleanup(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
		Scripts:    make(map[string]*scriptSlot),
	}

	parent := &Instance{
		ID:                 "parent",
		Fn:                 func(*Ctx) work.Node { return nil },
		HookFrame:          []HookSlot{},
		Children:           []*Instance{},
		ReferencedChildren: make(map[string]bool),
	}

	child := &Instance{
		ID:        "child",
		Fn:        func(*Ctx) work.Node { return nil },
		Parent:    parent,
		HookFrame: []HookSlot{},
	}

	parent.Children = append(parent.Children, child)
	sess.Components[child.ID] = child

	ctx := &Ctx{
		instance:  child,
		session:   sess,
		hookIndex: 0,
	}
	handle := UseScript(ctx, "console.log('child')")
	scriptID := handle.ID()

	if sess.Scripts[scriptID] == nil {
		t.Fatal("expected script to be registered")
	}

	sess.pruneUnreferencedChildren(parent)

	if len(parent.Children) != 0 {
		t.Errorf("expected child to be pruned, got %d children", len(parent.Children))
	}

	if sess.Scripts[scriptID] != nil {
		t.Error("expected script to be removed after child pruned")
	}
}

// TestSessionClose verifies that Close cleans up all resources.
func TestSessionClose(t *testing.T) {
	sess := &Session{
		Components:        make(map[string]*Instance),
		Scripts:           make(map[string]*scriptSlot),
		Bus:               protocol.NewBus(),
		allHandlerSubs:    make(map[string]*protocol.Subscription),
		currentHandlerIDs: make(map[string]bool),
		MountedComponents: make(map[*Instance]struct{}),
	}

	root := &Instance{
		ID:        "root",
		Fn:        func(*Ctx) work.Node { return nil },
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess.Root = root
	sess.Components["root"] = root

	child := &Instance{
		ID:        "child",
		Fn:        func(*Ctx) work.Node { return nil },
		Parent:    root,
		HookFrame: []HookSlot{},
	}
	root.Children = append(root.Children, child)
	sess.Components["child"] = child

	effectCleanupRan := false
	child.HookFrame = append(child.HookFrame, HookSlot{
		Type: HookTypeEffect,
		Value: &effectCell{
			cleanup: func() { effectCleanupRan = true },
		},
	})

	genericCleanupRan := false
	root.RegisterCleanup(func() { genericCleanupRan = true })

	sub := sess.Bus.Subscribe("test-handler", func(event string, data interface{}) {})
	sess.allHandlerSubs["test-handler"] = sub

	sess.Scripts["test-script"] = &scriptSlot{}

	sess.Close()

	if !genericCleanupRan {
		t.Error("expected generic cleanup to run")
	}
	if !effectCleanupRan {
		t.Error("expected effect cleanup to run")
	}

	if sess.Root != nil {
		t.Error("expected Root to be nil")
	}
	if sess.Components != nil {
		t.Error("expected Components to be nil")
	}
	if sess.Scripts != nil {
		t.Error("expected Scripts to be nil")
	}
	if sess.allHandlerSubs != nil {
		t.Error("expected allHandlerSubs to be nil")
	}
	if sess.MountedComponents != nil {
		t.Error("expected MountedComponents to be nil")
	}

	if sess.Bus.SubscriberCount("test-handler") != 0 {
		t.Error("expected handler to be unsubscribed from bus")
	}
}

// TestSessionCloseWithPanicingCleanup verifies that panics in cleanups don't prevent other cleanups.
func TestSessionCloseWithPanicingCleanup(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
		Bus:        protocol.NewBus(),
	}

	root := &Instance{
		ID:        "root",
		Fn:        func(*Ctx) work.Node { return nil },
		HookFrame: []HookSlot{},
	}
	sess.Root = root

	firstRan := false
	secondRan := false

	root.RegisterCleanup(func() { firstRan = true })
	root.RegisterCleanup(func() { panic("intentional panic") })

	root.HookFrame = append(root.HookFrame, HookSlot{
		Type: HookTypeEffect,
		Value: &effectCell{
			cleanup: func() {
				secondRan = true
				panic("effect panic")
			},
		},
	})

	sess.Close()

	if !firstRan {
		t.Error("expected first cleanup to run")
	}

	if !secondRan {
		t.Error("expected effect cleanup to run despite panic")
	}
}

// TestSessionCloseNil verifies that Close handles nil session.
func TestSessionCloseNil(t *testing.T) {
	var sess *Session

	sess.Close()
}
