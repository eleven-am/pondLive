package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/work"
)

func TestCtxContext_ReturnsSessionContext(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}
	sess.InitContext()

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

	if goCtx != sess.SessionContext() {
		t.Fatal("expected context to be the session context")
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

func TestCtxContext_NilSessionReturnsBackground(t *testing.T) {
	ctx := &Ctx{
		instance: &Instance{},
		session:  nil,
	}

	goCtx := ctx.Context()
	if goCtx != context.Background() {
		t.Error("expected context.Background() for nil session")
	}
}

func TestContextSurvivesRerender(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}
	sess.InitContext()

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

	inst.Render(sess)
	firstCtx := capturedCtx

	select {
	case <-firstCtx.Done():
		t.Fatal("first context should not be cancelled")
	default:
	}

	inst.Render(sess)
	secondCtx := capturedCtx

	if firstCtx != secondCtx {
		t.Fatal("context should be the same across re-renders (session-scoped)")
	}

	select {
	case <-firstCtx.Done():
		t.Fatal("context should NOT be cancelled after re-render")
	default:
	}
}

func TestContextSurvivesFlush(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}
	sess.InitContext()

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
		t.Fatal("first flush context should not be cancelled")
	default:
	}

	err = sess.Flush()
	if err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	select {
	case <-firstCtx.Done():
		t.Fatal("context should NOT be cancelled after second flush (session-scoped)")
	default:
	}
}

func TestSessionCloseCancelsContext(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}
	sess.InitContext()

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

func TestAllComponentsShareSessionContext(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}
	sess.InitContext()

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

	if parentCapturedCtx != childCapturedCtx {
		t.Fatal("parent and child should share the same session context")
	}

	if parentCapturedCtx != sess.SessionContext() {
		t.Fatal("captured context should be the session context")
	}
}

func TestSessionContextInitialization(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	ctx1 := sess.SessionContext()
	if ctx1 != context.Background() {
		t.Fatal("uninitialized session should return Background context")
	}

	sess.InitContext()

	ctx2 := sess.SessionContext()
	if ctx2 == context.Background() {
		t.Fatal("initialized session should not return Background context")
	}

	sess.InitContext()
	ctx3 := sess.SessionContext()
	if ctx2 != ctx3 {
		t.Fatal("calling InitContext twice should not create new context")
	}
}

func TestRenderContextCancelledOnRerender(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}
	sess.InitContext()

	var capturedRenderCtx context.Context
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		capturedRenderCtx = ctx.RenderContext()
		return nil
	}

	inst := &Instance{
		ID:        "test",
		Fn:        component,
		HookFrame: []HookSlot{},
	}
	sess.Root = inst

	sess.flushCtx, sess.flushCancel = context.WithCancel(context.Background())

	inst.Render(sess)
	firstRenderCtx := capturedRenderCtx

	if firstRenderCtx != sess.flushCtx {
		t.Fatal("render context should be the flush context")
	}

	select {
	case <-firstRenderCtx.Done():
		t.Fatal("render context should not be cancelled yet")
	default:
	}

	sess.flushCancel()

	select {
	case <-firstRenderCtx.Done():
	default:
		t.Fatal("render context should be cancelled when flush is cancelled")
	}
}

func TestMarkDirtyCancelsRenderContext(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}
	sess.InitContext()

	renderCount := 0
	var firstRenderCtx context.Context
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		renderCount++
		if renderCount == 1 {
			firstRenderCtx = ctx.RenderContext()
		}
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

	if renderCount != 1 {
		t.Fatalf("expected 1 render, got %d", renderCount)
	}

	select {
	case <-firstRenderCtx.Done():
		t.Fatal("render context should not be cancelled yet")
	default:
	}

	sess.MarkDirty(inst)

	select {
	case <-firstRenderCtx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("render context should be cancelled after MarkDirty. renderCount=%d", renderCount)
	}
}

func TestChildRenderContextCancelledWhenParentMarkedDirty(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}
	sess.InitContext()

	var firstParentRenderCtx, firstChildRenderCtx context.Context
	parentRenderCount := 0
	childRenderCount := 0

	childComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		childRenderCount++
		if childRenderCount == 1 {
			firstChildRenderCtx = ctx.RenderContext()
		}
		return nil
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		parentRenderCount++
		if parentRenderCount == 1 {
			firstParentRenderCtx = ctx.RenderContext()
		}
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

	if firstParentRenderCtx == nil || firstChildRenderCtx == nil {
		t.Fatal("render contexts were not captured")
	}

	select {
	case <-firstChildRenderCtx.Done():
		t.Fatal("child render context should not be cancelled yet")
	default:
	}

	sess.MarkDirty(parent)

	select {
	case <-firstParentRenderCtx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("parent render context should be cancelled after MarkDirty")
	}

	select {
	case <-firstChildRenderCtx.Done():
	case <-time.After(100 * time.Millisecond):
		t.Fatal("child render context should be cancelled when parent is cancelled")
	}
}

func TestRenderContextDifferentFromSessionContext(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}
	sess.InitContext()

	var capturedSessionCtx, capturedRenderCtx context.Context
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		capturedSessionCtx = ctx.Context()
		capturedRenderCtx = ctx.RenderContext()
		return nil
	}

	inst := &Instance{
		ID:        "test",
		Fn:        component,
		HookFrame: []HookSlot{},
	}
	sess.Root = inst

	err := sess.Flush()
	if err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if capturedSessionCtx == capturedRenderCtx {
		t.Fatal("session context and render context should be different")
	}

	if capturedSessionCtx != sess.SessionContext() {
		t.Fatal("captured session context should match session.SessionContext()")
	}
}

func TestRenderContextNilCtx(t *testing.T) {
	var ctx *Ctx
	goCtx := ctx.RenderContext()

	if goCtx == nil {
		t.Fatal("expected non-nil context")
	}

	if goCtx != context.Background() {
		t.Error("expected context.Background() for nil Ctx")
	}
}

func TestRenderContextNilSession(t *testing.T) {
	ctx := &Ctx{
		instance: &Instance{},
		session:  nil,
	}

	goCtx := ctx.RenderContext()
	if goCtx != context.Background() {
		t.Error("expected context.Background() for nil session")
	}
}

func TestFlushContextNilSession(t *testing.T) {
	var sess *Session
	ctx := sess.FlushContext()

	if ctx != context.Background() {
		t.Error("expected context.Background() for nil session")
	}
}

func TestFlushContextNilFlushCtx(t *testing.T) {
	sess := &Session{
		flushCtx: nil,
	}
	ctx := sess.FlushContext()

	if ctx != context.Background() {
		t.Error("expected context.Background() for nil flushCtx")
	}
}
