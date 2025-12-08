package runtime

import (
	"context"
	"testing"

	"github.com/eleven-am/pondlive/internal/work"
)

func TestUseErrorBoundary_NoError(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	root := &Instance{
		ID:        "root",
		Fn:        func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame: []HookSlot{},
	}
	sess.Root = root

	ctx := &Ctx{
		instance:  root,
		session:   sess,
		hookIndex: 0,
	}

	err := UseErrorBoundary(ctx)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestUseErrorBoundary_RenderError(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	panicComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		panic("intentional panic")
	}

	child := &Instance{
		ID:        "child",
		Fn:        panicComponent,
		HookFrame: []HookSlot{},
	}

	parent := &Instance{
		ID:        "parent",
		Fn:        func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame: []HookSlot{},
		Children:  []*Instance{child},
	}
	child.Parent = parent

	sess.Root = parent

	child.Render(sess, context.Background())

	ctx := &Ctx{
		instance:  parent,
		session:   sess,
		hookIndex: 0,
	}

	err := UseErrorBoundary(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Message != "intentional panic" {
		t.Errorf("expected 'intentional panic', got '%s'", err.Message)
	}

	if err.ComponentID != "child" {
		t.Errorf("expected component_id 'child', got '%s'", err.ComponentID)
	}

	if err.Phase != "render" {
		t.Errorf("expected phase 'render', got '%s'", err.Phase)
	}
}

func TestUseErrorBoundary_MemoError(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	panicMemoComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {

		UseMemo(ctx, func() int {
			panic("memo panic")
		})
		return nil
	}

	child := &Instance{
		ID:        "child",
		Fn:        panicMemoComponent,
		HookFrame: []HookSlot{},
	}

	parent := &Instance{
		ID:        "parent",
		Fn:        func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame: []HookSlot{},
		Children:  []*Instance{child},
	}
	child.Parent = parent

	sess.Root = parent

	child.Render(sess, context.Background())

	ctx := &Ctx{
		instance:  parent,
		session:   sess,
		hookIndex: 0,
	}

	err := UseErrorBoundary(ctx)
	if err == nil {
		t.Fatal("expected error from memo, got nil")
	}

	if err.Message != "memo panic" {
		t.Errorf("expected 'memo panic', got '%s'", err.Message)
	}

	if err.Phase != "memo" {
		t.Errorf("expected phase 'memo', got '%s'", err.Phase)
	}

	if err.HookIndex != 0 {
		t.Errorf("expected hook_index 0, got %d", err.HookIndex)
	}
}

func TestErrorBoundary_CatchChildError(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	childComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		panic("child error")
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		err := UseErrorBoundary(ctx)
		if err != nil {

			return &work.Text{Value: "Error: " + err.Message}
		}

		return nil
	}

	child := &Instance{
		ID:        "child",
		Fn:        childComponent,
		HookFrame: []HookSlot{},
	}

	parent := &Instance{
		ID:        "parent",
		Fn:        parentComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{child},
	}
	child.Parent = parent

	sess.Root = parent

	child.Render(sess, context.Background())

	result := parent.Render(sess, context.Background())

	if result == nil {
		t.Fatal("expected parent to render fallback UI")
	}

	textNode, ok := result.(*work.Text)
	if !ok {
		t.Fatalf("expected *work.Text, got %T", result)
	}

	if textNode.Value != "Error: child error" {
		t.Errorf("expected 'Error: child error', got '%s'", textNode.Value)
	}
}

func TestErrorBoundary_MultipleErrors(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	child1 := &Instance{
		ID: "child1",
		Fn: func(ctx *Ctx, _ any, _ []work.Item) work.Node {
			panic("error 1")
		},
		HookFrame: []HookSlot{},
	}

	child2 := &Instance{
		ID: "child2",
		Fn: func(ctx *Ctx, _ any, _ []work.Item) work.Node {
			panic("error 2")
		},
		HookFrame: []HookSlot{},
	}

	parent := &Instance{
		ID:        "parent",
		Fn:        func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame: []HookSlot{},
		Children:  []*Instance{child1, child2},
	}
	child1.Parent = parent
	child2.Parent = parent

	sess.Root = parent

	child1.Render(sess, context.Background())
	child2.Render(sess, context.Background())

	ctx := &Ctx{
		instance:  parent,
		session:   sess,
		hookIndex: 0,
	}

	err := UseErrorBoundary(ctx)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if err.Message != "error 1" && err.Message != "error 2" {
		t.Errorf("expected 'error 1' or 'error 2', got '%s'", err.Message)
	}
}

func TestErrorBoundary_ClearsOnSuccessfulRender(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	shouldPanic := true
	testComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		if shouldPanic {
			panic("first render fails")
		}
		return nil
	}

	inst := &Instance{
		ID:        "test",
		Fn:        testComponent,
		HookFrame: []HookSlot{},
	}

	sess.Root = inst

	inst.Render(sess, context.Background())

	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	err := UseErrorBoundary(ctx)
	if err == nil {
		t.Fatal("expected error from first render")
	}

	shouldPanic = false
	inst.Render(sess, context.Background())

	ctx.hookIndex = 0

	err = UseErrorBoundary(ctx)
	if err != nil {
		t.Errorf("expected error to be cleared after successful render, got %v", err)
	}
}

func TestErrorBoundary_LayeredInnerCatches(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	widgetComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		panic("widget error")
	}

	dashboardComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		err := UseErrorBoundary(ctx)
		if err != nil {
			return &work.Text{Value: "Dashboard Error: " + err.Message}
		}
		return nil
	}

	appComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		err := UseErrorBoundary(ctx)
		if err != nil {
			return &work.Text{Value: "App Error: " + err.Message}
		}
		return nil
	}

	widget := &Instance{
		ID:        "widget",
		Fn:        widgetComponent,
		HookFrame: []HookSlot{},
	}

	dashboard := &Instance{
		ID:        "dashboard",
		Fn:        dashboardComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{widget},
	}
	widget.Parent = dashboard

	app := &Instance{
		ID:        "app",
		Fn:        appComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{dashboard},
	}
	dashboard.Parent = app

	sess.Root = app

	widget.Render(sess, context.Background())
	dashboardResult := dashboard.Render(sess, context.Background())
	appResult := app.Render(sess, context.Background())

	dashText, ok := dashboardResult.(*work.Text)
	if !ok {
		t.Fatalf("expected dashboard to render error UI, got %T", dashboardResult)
	}
	if dashText.Value != "Dashboard Error: widget error" {
		t.Errorf("expected 'Dashboard Error: widget error', got '%s'", dashText.Value)
	}

	if appResult != nil {
		t.Errorf("expected app to render normally (nil), got %v", appResult)
	}
}

func TestErrorBoundary_LayeredOuterCatchesWhenInnerMissing(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	widgetComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		panic("widget error")
	}

	dashboardComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return nil
	}

	appComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		err := UseErrorBoundary(ctx)
		if err != nil {
			return &work.Text{Value: "App Error: " + err.Message}
		}
		return nil
	}

	widget := &Instance{
		ID:        "widget",
		Fn:        widgetComponent,
		HookFrame: []HookSlot{},
	}

	dashboard := &Instance{
		ID:        "dashboard",
		Fn:        dashboardComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{widget},
	}
	widget.Parent = dashboard

	app := &Instance{
		ID:        "app",
		Fn:        appComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{dashboard},
	}
	dashboard.Parent = app

	sess.Root = app

	widget.Render(sess, context.Background())
	dashboard.Render(sess, context.Background())
	appResult := app.Render(sess, context.Background())

	appText, ok := appResult.(*work.Text)
	if !ok {
		t.Fatalf("expected app to render error UI, got %T", appResult)
	}
	if appText.Value != "App Error: widget error" {
		t.Errorf("expected 'App Error: widget error', got '%s'", appText.Value)
	}
}

func TestErrorBoundary_LayeredMultiLevel(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	chartComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		panic("chart error")
	}

	widgetComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return nil
	}

	dashboardComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		err := UseErrorBoundary(ctx)
		if err != nil {
			return &work.Text{Value: "Dashboard Error: " + err.Message}
		}
		return nil
	}

	appComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		err := UseErrorBoundary(ctx)
		if err != nil {
			return &work.Text{Value: "App Error: " + err.Message}
		}
		return nil
	}

	chart := &Instance{
		ID:        "chart",
		Fn:        chartComponent,
		HookFrame: []HookSlot{},
	}

	widget := &Instance{
		ID:        "widget",
		Fn:        widgetComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{chart},
	}
	chart.Parent = widget

	dashboard := &Instance{
		ID:        "dashboard",
		Fn:        dashboardComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{widget},
	}
	widget.Parent = dashboard

	app := &Instance{
		ID:        "app",
		Fn:        appComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{dashboard},
	}
	dashboard.Parent = app

	sess.Root = app

	chart.Render(sess, context.Background())
	widget.Render(sess, context.Background())
	dashboardResult := dashboard.Render(sess, context.Background())
	appResult := app.Render(sess, context.Background())

	dashText, ok := dashboardResult.(*work.Text)
	if !ok {
		t.Fatalf("expected dashboard to render error UI, got %T", dashboardResult)
	}
	if dashText.Value != "Dashboard Error: chart error" {
		t.Errorf("expected 'Dashboard Error: chart error', got '%s'", dashText.Value)
	}

	if appResult != nil {
		t.Errorf("expected app to render normally (nil), got %v", appResult)
	}
}

func TestErrorBoundary_LayeredSiblingIsolation(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	criticalComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		panic("critical error")
	}

	optionalComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return &work.Text{Value: "optional works"}
	}

	criticalWrapperComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		err := UseErrorBoundary(ctx)
		if err != nil {
			return &work.Text{Value: "Critical Fallback: " + err.Message}
		}
		return nil
	}

	optionalWrapperComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		err := UseErrorBoundary(ctx)
		if err != nil {
			return &work.Text{Value: "Optional Fallback: " + err.Message}
		}
		return nil
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return nil
	}

	critical := &Instance{
		ID:        "critical",
		Fn:        criticalComponent,
		HookFrame: []HookSlot{},
	}

	optional := &Instance{
		ID:        "optional",
		Fn:        optionalComponent,
		HookFrame: []HookSlot{},
	}

	criticalWrapper := &Instance{
		ID:        "criticalWrapper",
		Fn:        criticalWrapperComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{critical},
	}
	critical.Parent = criticalWrapper

	optionalWrapper := &Instance{
		ID:        "optionalWrapper",
		Fn:        optionalWrapperComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{optional},
	}
	optional.Parent = optionalWrapper

	parent := &Instance{
		ID:        "parent",
		Fn:        parentComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{criticalWrapper, optionalWrapper},
	}
	criticalWrapper.Parent = parent
	optionalWrapper.Parent = parent

	sess.Root = parent

	critical.Render(sess, context.Background())
	optional.Render(sess, context.Background())
	criticalWrapperResult := criticalWrapper.Render(sess, context.Background())
	optionalWrapperResult := optionalWrapper.Render(sess, context.Background())

	criticalText, ok := criticalWrapperResult.(*work.Text)
	if !ok {
		t.Fatalf("expected critical wrapper to render fallback, got %T", criticalWrapperResult)
	}
	if criticalText.Value != "Critical Fallback: critical error" {
		t.Errorf("expected 'Critical Fallback: critical error', got '%s'", criticalText.Value)
	}

	if optionalWrapperResult != nil {
		t.Errorf("expected optional wrapper to render normally (nil), got %v", optionalWrapperResult)
	}

	optionalText, ok := optional.WorkTree.(*work.Text)
	if !ok {
		t.Fatalf("expected optional to render text, got %T", optional.WorkTree)
	}
	if optionalText.Value != "optional works" {
		t.Errorf("expected 'optional works', got '%s'", optionalText.Value)
	}
}
