package runtime

import (
	"context"
	"testing"

	"github.com/eleven-am/pondlive/internal/work"
)

func TestCtxContext_ReturnsBackground(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	inst := &Instance{
		ID:        "test",
		Fn:        func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame: []HookSlot{},
	}
	sess.Root = inst

	inst.Render(sess)

	ctx := &Ctx{
		instance: inst,
		session:  sess,
	}

	goCtx := ctx.Context()
	if goCtx == nil {
		t.Fatal("expected non-nil context")
	}

	if goCtx != context.Background() {
		t.Error("expected context.Background()")
	}
}

func TestCtxContext_NilCtxReturnsBackground(t *testing.T) {
	var ctx *Ctx
	goCtx := ctx.Context()

	if goCtx == nil {
		t.Fatal("expected non-nil context")
	}

	if goCtx != context.Background() {
		t.Error("expected context.Background() for nil Ctx")
	}
}

func TestCtxContext_AlwaysReturnsBackground(t *testing.T) {
	ctx := &Ctx{
		instance: &Instance{},
		session:  &Session{},
	}

	goCtx := ctx.Context()
	if goCtx != context.Background() {
		t.Error("expected context.Background()")
	}
}

func TestFlushCompletesSuccessfully(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return nil
	}

	inst := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
	}
	sess.Root = inst

	err := sess.Flush()
	if err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

func TestSessionCloseHandlesNilFields(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return nil
	}

	inst := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
	}
	sess.Root = inst

	err := sess.Flush()
	if err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	sess.Close()
}

func TestContextThroughComponentHierarchy(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	var parentCapturedCtx, childCapturedCtx context.Context

	childComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		childCapturedCtx = ctx.Context()
		return nil
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		parentCapturedCtx = ctx.Context()
		return &work.ComponentNode{
			Fn:  childComponent,
			Key: "child",
		}
	}

	parent := &Instance{
		ID:        "parent",
		Fn:        parentComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess.Root = parent
	sess.Components = map[string]*Instance{
		"parent": parent,
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if parentCapturedCtx == nil {
		t.Fatal("parent context was not captured")
	}

	if childCapturedCtx == nil {
		t.Fatal("child context was not captured")
	}

	if parentCapturedCtx != context.Background() {
		t.Error("parent context should be context.Background()")
	}

	if childCapturedCtx != context.Background() {
		t.Error("child context should be context.Background()")
	}
}
