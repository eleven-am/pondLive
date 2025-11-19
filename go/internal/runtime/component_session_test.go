package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom2/diff"
)

func TestNewSession(t *testing.T) {
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})

	if sess == nil {
		t.Fatal("expected session to be created")
	}
	if sess.root == nil {
		t.Fatal("expected root component to be initialized")
	}
	if sess.dirty == nil {
		t.Fatal("expected dirty map to be initialized")
	}
	if sess.handlers == nil {
		t.Fatal("expected handlers map to be initialized")
	}
}

func TestFlushBasic(t *testing.T) {
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})

	sess.SetPatchSender(func(patches []dom2diff.Patch) error {
		return nil
	})

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if sess.prevTree == nil {
		t.Fatal("expected prevTree to be set after flush")
	}

	if sess.prevTree.Tag != "div" {
		t.Errorf("expected prevTree to be <div>, got <%s>", sess.prevTree.Tag)
	}
}

func TestFlushDirtyTracking(t *testing.T) {
	count := 0
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		count++
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected component to render once, got %d", count)
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected component to not re-render, but count is %d", count)
	}

	sess.markDirty(sess.root)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if count != 2 {
		t.Errorf("expected component to render twice, got %d", count)
	}
}

func TestHandlerCollection(t *testing.T) {
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		node := &dom2.StructuredNode{Tag: "button"}
		if node.Events == nil {
			node.Events = make(map[string]dom2.EventBinding)
		}
		node.Events["click"] = dom2.EventBinding{
			Key: "test:h0",
			Handler: func(ev dom2.Event) dom2.Updates {
				return nil
			},
		}
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	sess.handlersMu.RLock()
	handler, exists := sess.handlers["test:h0"]
	sess.handlersMu.RUnlock()

	if !exists {
		t.Fatal("expected handler to be collected")
	}
	if handler == nil {
		t.Fatal("expected handler to not be nil")
	}
}

func TestHandleEvent(t *testing.T) {
	called := false
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		node := &dom2.StructuredNode{Tag: "button"}
		if node.Events == nil {
			node.Events = make(map[string]dom2.EventBinding)
		}
		node.Events["click"] = dom2.EventBinding{
			Key: "test:h0",
			Handler: func(ev dom2.Event) dom2.Updates {
				called = true
				return dom2.Rerender()
			},
		}
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	ev := dom2.Event{Name: "click"}
	if err := sess.HandleEvent("test:h0", ev); err != nil {
		t.Fatalf("HandleEvent failed: %v", err)
	}

	if !called {
		t.Error("expected handler to be called")
	}

	sess.mu.Lock()
	_, isDirty := sess.dirty[sess.root]
	sess.mu.Unlock()

	if !isDirty {
		t.Error("expected root to be marked dirty after handler returned Updates")
	}
}

func TestHandleEventNotFound(t *testing.T) {
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	ev := dom2.Event{Name: "click"}
	err := sess.HandleEvent("unknown:h0", ev)

	if err == nil {
		t.Fatal("expected error for unknown handler")
	}
}

func TestAllocateRefID(t *testing.T) {
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})

	id1 := sess.allocateElementRefID()
	id2 := sess.allocateElementRefID()
	id3 := sess.allocateElementRefID()

	if id1 != "ref:0" {
		t.Errorf("expected ref:0, got %s", id1)
	}
	if id2 != "ref:1" {
		t.Errorf("expected ref:1, got %s", id2)
	}
	if id3 != "ref:2" {
		t.Errorf("expected ref:2, got %s", id3)
	}
}

func TestComponentTree(t *testing.T) {
	childComp := func(ctx Ctx, props string) *dom2.StructuredNode {
		return &dom2.StructuredNode{Tag: "span", Text: props}
	}

	parentComp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		child := Render(ctx, childComp, "hello")
		return &dom2.StructuredNode{
			Tag:      "div",
			Children: []*dom2.StructuredNode{child},
		}
	}

	sess := NewSession(parentComp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if sess.prevTree == nil {
		t.Fatal("expected tree to exist")
	}
	if sess.prevTree.Tag != "div" {
		t.Errorf("expected root to be <div>, got <%s>", sess.prevTree.Tag)
	}
	if len(sess.prevTree.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(sess.prevTree.Children))
	}
	wrapper := sess.prevTree.Children[0]
	if wrapper.ComponentID == "" {
		t.Fatalf("expected component wrapper, got %+v", wrapper)
	}
	if len(wrapper.Children) != 1 {
		t.Fatalf("expected wrapper to have 1 child, got %d", len(wrapper.Children))
	}
	if wrapper.Children[0].Tag != "span" {
		t.Errorf("expected wrapped child to be <span>, got <%s>", wrapper.Children[0].Tag)
	}
}
