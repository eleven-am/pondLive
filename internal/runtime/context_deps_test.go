package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/work"
)

// Helper to build a session with a root instance.
func newTestSession(rootFn any) *Session {
	root := &Instance{
		ID:        "root",
		Fn:        rootFn,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}

	sess := &Session{
		Root:       root,
		Components: map[string]*Instance{"root": root},
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}

	return sess
}

func TestContextChangeRerendersConsumersButNotNonConsumers(t *testing.T) {
	theme := CreateContext("light")

	var setTheme func(string)
	renderRoot := 0
	renderChild := 0
	renderGrandchild := 0

	grandchildFn := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
		renderGrandchild++
		_ = theme.UseContextValue(ctx)
		return &work.Text{Value: "gc"}
	}

	childFn := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
		renderChild++
		return work.Component(grandchildFn)
	}

	rootFn := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
		renderRoot++
		_, setter := theme.UseProvider(ctx, "light")
		if setTheme == nil {
			setTheme = func(v string) { setter(v) }
		}
		return work.Component(childFn)
	}

	sess := newTestSession(rootFn)

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush: %v", err)
	}

	if setTheme == nil {
		t.Fatalf("setter not captured")
	}

	setTheme("dark")

	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush: %v", err)
	}

	if renderRoot != 2 {
		t.Fatalf("expected root render 2x, got %d", renderRoot)
	}
	if renderChild != 1 {
		t.Fatalf("expected child render 1x (reused), got %d", renderChild)
	}
	if renderGrandchild != 2 {
		t.Fatalf("expected grandchild render 2x (consumes context), got %d", renderGrandchild)
	}
}

func TestContextChangeSkipsUnrelatedContext(t *testing.T) {
	ctxA := CreateContext("a")
	ctxB := CreateContext("b")

	var setA func(string)
	var setB func(string)

	renderChild := 0

	childFn := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
		renderChild++
		_ = ctxA.UseContextValue(ctx)
		return &work.Text{Value: "child"}
	}

	rootFn := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
		_, sa := ctxA.UseProvider(ctx, "a")
		if setA == nil {
			setA = func(v string) { sa(v) }
		}
		_, sb := ctxB.UseProvider(ctx, "b")
		if setB == nil {
			setB = func(v string) { sb(v) }
		}
		return work.Component(childFn)
	}

	sess := newTestSession(rootFn)

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush: %v", err)
	}

	setB("bb")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after ctxB change: %v", err)
	}
	if renderChild != 1 {
		t.Fatalf("ctxB change should not re-render child consuming ctxA; got %d renders", renderChild)
	}

	setA("aa")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after ctxA change: %v", err)
	}
	if renderChild != 2 {
		t.Fatalf("ctxA change should re-render child; got %d renders", renderChild)
	}
}

func TestContextOverrideHonorsNearestProvider(t *testing.T) {
	valueCtx := CreateContext("x")

	var setParent func(string)
	var setChild func(string)

	renderConsumer := 0

	consumer := func(c *Ctx, _ any, _ []work.Node) work.Node {
		renderConsumer++
		_ = valueCtx.UseContextValue(c)
		return &work.Text{Value: "c"}
	}

	childWithProvider := func(c *Ctx, _ any, _ []work.Node) work.Node {
		_, setter := valueCtx.UseProvider(c, "child")
		if setChild == nil {
			setChild = func(v string) { setter(v) }
		}
		return work.Component(consumer)
	}

	rootFn := func(c *Ctx, _ any, _ []work.Node) work.Node {
		_, setter := valueCtx.UseProvider(c, "root")
		if setParent == nil {
			setParent = func(v string) { setter(v) }
		}
		return work.Component(childWithProvider)
	}

	sess := newTestSession(rootFn)

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush: %v", err)
	}

	setParent("root-updated")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after parent provider change: %v", err)
	}

	if renderConsumer != 1 {
		t.Fatalf("consumer should not re-render on ancestor provider change; got %d", renderConsumer)
	}

	setChild("child-updated")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after child provider change: %v", err)
	}

	if renderConsumer != 2 {
		t.Fatalf("consumer should re-render on nearest provider change; got %d", renderConsumer)
	}
}
