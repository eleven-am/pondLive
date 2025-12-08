package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/work"
)

func TestCtxContext_ReturnsContext(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	inst := &Instance{
		ID:        "test",
		Fn:        func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame: []HookSlot{},
	}
	sess.Root = inst

	inst.Render(sess, context.Background())

	ctx := &Ctx{
		instance: inst,
		session:  sess,
		goCtx:    inst.renderCtx,
	}

	goCtx := ctx.Context()
	if goCtx == nil {
		t.Fatal("expected non-nil context")
	}

	select {
	case <-goCtx.Done():
		t.Fatal("context should not be done")
	default:
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

func TestCtxContext_NilGoCtxReturnsBackground(t *testing.T) {
	ctx := &Ctx{
		instance: &Instance{},
		session:  &Session{},
		goCtx:    nil,
	}

	goCtx := ctx.Context()
	if goCtx != context.Background() {
		t.Error("expected context.Background() for nil goCtx")
	}
}

func TestRenderCancelsOnNewRender(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	var capturedCtx context.Context
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		capturedCtx = ctx.Context()
		return nil
	}

	inst := &Instance{
		ID:        "test",
		Fn:        component,
		HookFrame: []HookSlot{},
	}
	sess.Root = inst

	inst.Render(sess, context.Background())
	firstCtx := capturedCtx

	select {
	case <-firstCtx.Done():
		t.Fatal("first context should not be cancelled yet")
	default:
	}

	inst.Render(sess, context.Background())

	select {
	case <-firstCtx.Done():
	default:
		t.Fatal("first context should be cancelled after re-render")
	}
}

func TestChildContextDerivedFromParent(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	parentCtx, parentCancel := context.WithCancel(context.Background())
	defer parentCancel()

	var capturedCtx context.Context
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		capturedCtx = ctx.Context()
		return nil
	}

	inst := &Instance{
		ID:        "child",
		Fn:        component,
		HookFrame: []HookSlot{},
	}
	sess.Root = inst

	inst.Render(sess, parentCtx)

	select {
	case <-capturedCtx.Done():
		t.Fatal("child context should not be cancelled yet")
	default:
	}

	parentCancel()

	select {
	case <-capturedCtx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("child context should be cancelled when parent is cancelled")
	}
}

func TestFlushCancelsOnNewFlush(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	var capturedCtx context.Context
	var mu sync.Mutex
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		mu.Lock()
		capturedCtx = ctx.Context()
		mu.Unlock()
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
		t.Fatalf("first flush failed: %v", err)
	}

	mu.Lock()
	firstCtx := capturedCtx
	mu.Unlock()

	select {
	case <-firstCtx.Done():
		t.Fatal("first flush context should not be cancelled yet")
	default:
	}

	err = sess.Flush()
	if err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	select {
	case <-firstCtx.Done():
	default:
		t.Fatal("first flush context should be cancelled after second flush")
	}
}

func TestSessionCloseCancelsFlushContext(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	var capturedCtx context.Context
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		capturedCtx = ctx.Context()
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

	select {
	case <-capturedCtx.Done():
		t.Fatal("context should not be cancelled before Close")
	default:
	}

	sess.Close()

	select {
	case <-capturedCtx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("context should be cancelled after Close")
	}
}

func TestContextPropagatesThroughComponentHierarchy(t *testing.T) {
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

	sess.flushCancel()

	select {
	case <-parentCapturedCtx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("parent context should be cancelled")
	}

	select {
	case <-childCapturedCtx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("child context should be cancelled when parent is cancelled")
	}
}

func TestContextWithValuePassesThrough(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	type ctxKey string
	const testKey ctxKey = "test-key"
	const testValue = "test-value"

	parentCtx := context.WithValue(context.Background(), testKey, testValue)

	var capturedCtx context.Context
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		capturedCtx = ctx.Context()
		return nil
	}

	inst := &Instance{
		ID:        "test",
		Fn:        component,
		HookFrame: []HookSlot{},
	}
	sess.Root = inst

	inst.Render(sess, parentCtx)

	if capturedCtx.Value(testKey) != testValue {
		t.Errorf("expected context value %q, got %v", testValue, capturedCtx.Value(testKey))
	}
}
