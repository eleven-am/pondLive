package runtime

import (
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

	batch := UseErrorBoundary(ctx)
	if batch.HasErrors() {
		t.Errorf("expected no errors, got %v", batch.First())
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

	child.Render(sess)

	ctx := &Ctx{
		instance:  parent,
		session:   sess,
		hookIndex: 0,
	}

	batch := UseErrorBoundary(ctx)
	if !batch.HasErrors() {
		t.Fatal("expected error, got nil")
	}

	err := batch.First()
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

	child.Render(sess)

	ctx := &Ctx{
		instance:  parent,
		session:   sess,
		hookIndex: 0,
	}

	batch := UseErrorBoundary(ctx)
	if !batch.HasErrors() {
		t.Fatal("expected error from memo, got nil")
	}

	err := batch.First()
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
		batch := UseErrorBoundary(ctx)
		if batch.HasErrors() {

			return &work.Text{Value: "Error: " + batch.First().Message}
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

	child.Render(sess)

	result := parent.Render(sess)

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

	child1.Render(sess)
	child2.Render(sess)

	ctx := &Ctx{
		instance:  parent,
		session:   sess,
		hookIndex: 0,
	}

	batch := UseErrorBoundary(ctx)
	if !batch.HasErrors() {
		t.Fatal("expected errors, got nil")
	}

	if batch.Count() != 2 {
		t.Errorf("expected 2 errors, got %d", batch.Count())
	}

	allErrors := batch.All()
	messages := make(map[string]bool)
	for _, err := range allErrors {
		messages[err.Message] = true
	}

	if !messages["error 1"] {
		t.Error("expected 'error 1' in errors")
	}
	if !messages["error 2"] {
		t.Error("expected 'error 2' in errors")
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

	inst.Render(sess)

	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	batch := UseErrorBoundary(ctx)
	if !batch.HasErrors() {
		t.Fatal("expected error from first render")
	}

	shouldPanic = false
	inst.Render(sess)

	ctx.hookIndex = 0

	batch = UseErrorBoundary(ctx)
	if batch.HasErrors() {
		t.Errorf("expected error to be cleared after successful render, got %v", batch.First())
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
		batch := UseErrorBoundary(ctx)
		if batch.HasErrors() {
			return &work.Text{Value: "Dashboard Error: " + batch.First().Message}
		}
		return nil
	}

	appComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		batch := UseErrorBoundary(ctx)
		if batch.HasErrors() {
			return &work.Text{Value: "App Error: " + batch.First().Message}
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

	widget.Render(sess)
	dashboardResult := dashboard.Render(sess)
	appResult := app.Render(sess)

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
		batch := UseErrorBoundary(ctx)
		if batch.HasErrors() {
			return &work.Text{Value: "App Error: " + batch.First().Message}
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

	widget.Render(sess)
	dashboard.Render(sess)
	appResult := app.Render(sess)

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
		batch := UseErrorBoundary(ctx)
		if batch.HasErrors() {
			return &work.Text{Value: "Dashboard Error: " + batch.First().Message}
		}
		return nil
	}

	appComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		batch := UseErrorBoundary(ctx)
		if batch.HasErrors() {
			return &work.Text{Value: "App Error: " + batch.First().Message}
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

	chart.Render(sess)
	widget.Render(sess)
	dashboardResult := dashboard.Render(sess)
	appResult := app.Render(sess)

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

func TestErrorBoundary_HasDescendantErrorBubblesUp(t *testing.T) {
	grandchild := &Instance{
		ID:        "grandchild",
		HookFrame: []HookSlot{},
	}

	child := &Instance{
		ID:        "child",
		HookFrame: []HookSlot{},
		Children:  []*Instance{grandchild},
	}
	grandchild.Parent = child

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
		Children:  []*Instance{child},
	}
	child.Parent = parent

	root := &Instance{
		ID:        "root",
		HookFrame: []HookSlot{},
		Children:  []*Instance{parent},
	}
	parent.Parent = root

	if root.hasDescendantError || parent.hasDescendantError || child.hasDescendantError {
		t.Fatal("expected all hasDescendantError flags to be false initially")
	}

	testErr := &Error{
		ErrorCode: ErrCodeRender,
		Message:   "test error",
		Phase:     "render",
	}
	grandchild.setRenderError(testErr)

	if !child.hasDescendantError {
		t.Error("expected child.hasDescendantError to be true after grandchild error")
	}
	if !parent.hasDescendantError {
		t.Error("expected parent.hasDescendantError to be true after grandchild error")
	}
	if !root.hasDescendantError {
		t.Error("expected root.hasDescendantError to be true after grandchild error")
	}

	if grandchild.hasDescendantError {
		t.Error("grandchild should not have hasDescendantError set (error is on self, not descendant)")
	}
}

func TestErrorBoundary_HasDescendantErrorStopsAtMarked(t *testing.T) {
	child := &Instance{
		ID:        "child",
		HookFrame: []HookSlot{},
	}

	parent := &Instance{
		ID:                 "parent",
		HookFrame:          []HookSlot{},
		Children:           []*Instance{child},
		hasDescendantError: true,
	}
	child.Parent = parent

	root := &Instance{
		ID:                 "root",
		HookFrame:          []HookSlot{},
		Children:           []*Instance{parent},
		hasDescendantError: true,
	}
	parent.Parent = root

	testErr := &Error{
		ErrorCode: ErrCodeRender,
		Message:   "another error",
		Phase:     "render",
	}
	child.setRenderError(testErr)

	if !parent.hasDescendantError {
		t.Error("parent.hasDescendantError should still be true")
	}
	if !root.hasDescendantError {
		t.Error("root.hasDescendantError should still be true")
	}
}

func TestErrorBoundary_PruningSkipsCleanSubtrees(t *testing.T) {
	cleanGrandchild := &Instance{
		ID:        "cleanGrandchild",
		HookFrame: []HookSlot{},
	}

	cleanChild := &Instance{
		ID:        "cleanChild",
		HookFrame: []HookSlot{},
		Children:  []*Instance{cleanGrandchild},
	}
	cleanGrandchild.Parent = cleanChild

	dirtyGrandchild := &Instance{
		ID:        "dirtyGrandchild",
		HookFrame: []HookSlot{},
	}

	dirtyChild := &Instance{
		ID:        "dirtyChild",
		HookFrame: []HookSlot{},
		Children:  []*Instance{dirtyGrandchild},
	}
	dirtyGrandchild.Parent = dirtyChild

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
		Children:  []*Instance{cleanChild, dirtyChild},
	}
	cleanChild.Parent = parent
	dirtyChild.Parent = parent

	testErr := &Error{
		ErrorCode: ErrCodeRender,
		Message:   "dirty error",
		Phase:     "render",
	}
	dirtyGrandchild.setRenderError(testErr)

	if cleanChild.hasDescendantError {
		t.Error("cleanChild should not have hasDescendantError")
	}
	if !dirtyChild.hasDescendantError {
		t.Error("dirtyChild should have hasDescendantError")
	}
	if !parent.hasDescendantError {
		t.Error("parent should have hasDescendantError")
	}

	errors := parent.collectChildErrors()
	if len(errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errors))
	}
	if errors[0].Message != "dirty error" {
		t.Errorf("expected 'dirty error', got '%s'", errors[0].Message)
	}
}

func TestErrorBoundary_ClearResetsFlags(t *testing.T) {
	grandchild := &Instance{
		ID:        "grandchild",
		HookFrame: []HookSlot{},
	}

	child := &Instance{
		ID:        "child",
		HookFrame: []HookSlot{},
		Children:  []*Instance{grandchild},
	}
	grandchild.Parent = child

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
		Children:  []*Instance{child},
	}
	child.Parent = parent

	testErr := &Error{
		ErrorCode: ErrCodeRender,
		Message:   "test error",
		Phase:     "render",
	}
	grandchild.setRenderError(testErr)

	if !parent.hasDescendantError || !child.hasDescendantError {
		t.Fatal("flags should be set before clear")
	}

	parent.clearChildErrors()

	if parent.hasDescendantError {
		t.Error("parent.hasDescendantError should be false after clear")
	}
	if child.hasDescendantError {
		t.Error("child.hasDescendantError should be false after clear")
	}
	if grandchild.hasDescendantError {
		t.Error("grandchild.hasDescendantError should be false after clear")
	}
	if grandchild.RenderError != nil {
		t.Error("grandchild.RenderError should be nil after clear (cleared by boundary)")
	}
	if !grandchild.errorHandledByBoundary {
		t.Error("grandchild.errorHandledByBoundary should be true after clear (prevents infinite loop)")
	}
}

func TestErrorBoundary_EffectErrorBubblesUp(t *testing.T) {
	child := &Instance{
		ID:        "child",
		HookFrame: []HookSlot{},
	}

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
		Children:  []*Instance{child},
	}
	child.Parent = parent

	testErr := &Error{
		ErrorCode: ErrCodeEffect,
		Message:   "effect error",
		Phase:     "effect",
	}
	child.setEffectError(testErr)

	if !parent.hasDescendantError {
		t.Error("parent.hasDescendantError should be true after child effect error")
	}

	errors := parent.collectChildEffectErrors()
	if len(errors) != 1 {
		t.Fatalf("expected 1 effect error, got %d", len(errors))
	}
	if errors[0].Message != "effect error" {
		t.Errorf("expected 'effect error', got '%s'", errors[0].Message)
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
		batch := UseErrorBoundary(ctx)
		if batch.HasErrors() {
			return &work.Text{Value: "Critical Fallback: " + batch.First().Message}
		}
		return nil
	}

	optionalWrapperComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		batch := UseErrorBoundary(ctx)
		if batch.HasErrors() {
			return &work.Text{Value: "Optional Fallback: " + batch.First().Message}
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

	critical.Render(sess)
	optional.Render(sess)
	criticalWrapperResult := criticalWrapper.Render(sess)
	optionalWrapperResult := optionalWrapper.Render(sess)

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
