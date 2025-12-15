package errors

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestCrashPage(t *testing.T) {
	node := crashPage()
	if node == nil {
		t.Fatal("expected non-nil node")
	}

	elem, ok := node.(*work.Element)
	if !ok {
		t.Fatal("expected *work.Element")
	}
	if elem.Tag != "div" {
		t.Errorf("expected div tag, got %s", elem.Tag)
	}

	if len(elem.Children) < 2 {
		t.Error("expected at least 2 children (h1 and p)")
	}
}

func TestBuildComponentPath_Nil(t *testing.T) {
	err := runtime.NewError(runtime.ErrCodeApp, "test error")
	result := buildComponentPath(err)
	if result != "" {
		t.Errorf("expected empty string for error without metadata, got %q", result)
	}
}

func TestBuildComponentPath_NoPath(t *testing.T) {
	err := runtime.NewError(runtime.ErrCodeApp, "test error")
	err.Meta = map[string]any{"other_key": "value"}
	result := buildComponentPath(err)
	if result != "" {
		t.Errorf("expected empty string for error without component_name_path, got %q", result)
	}
}

func TestBuildComponentPath_EmptyPath(t *testing.T) {
	err := runtime.NewError(runtime.ErrCodeApp, "test error")
	err.Meta = map[string]any{"component_name_path": []string{}}
	result := buildComponentPath(err)
	if result != "" {
		t.Errorf("expected empty string for empty path, got %q", result)
	}
}

func TestBuildComponentPath_SingleComponent(t *testing.T) {
	err := runtime.NewError(runtime.ErrCodeApp, "test error")
	err.Meta = map[string]any{"component_name_path": []string{"App"}}
	result := buildComponentPath(err)
	if result != "App" {
		t.Errorf("expected 'App', got %q", result)
	}
}

func TestBuildComponentPath_MultipleComponents(t *testing.T) {
	err := runtime.NewError(runtime.ErrCodeApp, "test error")
	err.Meta = map[string]any{"component_name_path": []string{"App", "Header", "Button"}}
	result := buildComponentPath(err)
	if result != "App → Header → Button" {
		t.Errorf("expected 'App → Header → Button', got %q", result)
	}
}

func TestBuildComponentPath_WrongType(t *testing.T) {
	err := runtime.NewError(runtime.ErrCodeApp, "test error")
	err.Meta = map[string]any{"component_name_path": "not a slice"}
	result := buildComponentPath(err)
	if result != "" {
		t.Errorf("expected empty string for wrong type, got %q", result)
	}
}

func TestErrorItem(t *testing.T) {
	err := runtime.NewError(runtime.ErrCodeApp, "test error message")
	err.Phase = "render"

	node := errorItem(err)
	if node == nil {
		t.Fatal("expected non-nil node")
	}

	elem, ok := node.(*work.Element)
	if !ok {
		t.Fatal("expected *work.Element")
	}
	if elem.Tag != "div" {
		t.Errorf("expected div tag, got %s", elem.Tag)
	}
}

func TestErrorItem_WithComponentPath(t *testing.T) {
	err := runtime.NewError(runtime.ErrCodeApp, "test error")
	err.Phase = "mount"
	err.Meta = map[string]any{"component_name_path": []string{"App", "Child"}}

	node := errorItem(err)
	if node == nil {
		t.Fatal("expected non-nil node")
	}
}

func TestErrorItem_WithStackFrames(t *testing.T) {
	err := runtime.NewError(runtime.ErrCodeApp, "test error")
	err.Phase = "render"
	err.StackTrace = `github.com/test/pkg.TestFunc
	/path/to/file.go:123`

	node := errorItem(err)
	if node == nil {
		t.Fatal("expected non-nil node")
	}
}

func TestDevOverlay(t *testing.T) {
	err1 := runtime.NewError(runtime.ErrCodeApp, "error one")
	err2 := runtime.NewError(runtime.ErrCodeHandler, "error two")
	batch := runtime.NewErrorBatch(err1, err2)

	node := devOverlay(batch)
	if node == nil {
		t.Fatal("expected non-nil node")
	}

	elem, ok := node.(*work.Element)
	if !ok {
		t.Fatal("expected *work.Element")
	}
	if elem.Tag != "div" {
		t.Errorf("expected div tag, got %s", elem.Tag)
	}
}

func TestDevOverlay_EmptyBatch(t *testing.T) {
	batch := runtime.NewErrorBatch()

	node := devOverlay(batch)
	if node == nil {
		t.Fatal("expected non-nil node")
	}
}

func TestOverlayStyles(t *testing.T) {
	item := overlayStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestContainerStyles(t *testing.T) {
	item := containerStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestHeaderStyles(t *testing.T) {
	item := headerStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestCountBadgeStyles(t *testing.T) {
	item := countBadgeStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestErrorListStyles(t *testing.T) {
	item := errorListStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestErrorItemStyles(t *testing.T) {
	item := errorItemStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestErrorHeaderStyles(t *testing.T) {
	item := errorHeaderStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestCodeStyles(t *testing.T) {
	item := codeStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestPhaseStyles(t *testing.T) {
	item := phaseStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestMessageStyles(t *testing.T) {
	item := messageStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestComponentPathStyles(t *testing.T) {
	item := componentPathStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestStackStyles(t *testing.T) {
	item := stackStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestFrameStyles(t *testing.T) {
	item := frameStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestFuncNameStyles(t *testing.T) {
	item := funcNameStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestFileStyles(t *testing.T) {
	item := fileStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestCrashContainerStyles(t *testing.T) {
	item := crashContainerStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestCrashTitleStyles(t *testing.T) {
	item := crashTitleStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestCrashMessageStyles(t *testing.T) {
	item := crashMessageStyles()
	if item == nil {
		t.Fatal("expected non-nil item")
	}
}

func TestProps(t *testing.T) {
	p := Props{DevMode: true}
	if !p.DevMode {
		t.Error("expected DevMode to be true")
	}

	p2 := Props{DevMode: false}
	if p2.DevMode {
		t.Error("expected DevMode to be false")
	}
}

func TestProviderExists(t *testing.T) {
	if Provider == nil {
		t.Fatal("expected Provider to be non-nil")
	}
}

func TestProvider_NoErrors_ReturnsChildren(t *testing.T) {
	childCalled := false

	childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		childCalled = true
		return &work.Text{Value: "child content"}
	}

	root := &runtime.Instance{
		ID: "root",
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			return Provider(ctx, Props{DevMode: false}, work.Component(childFn))
		},
		HookFrame: []runtime.HookSlot{},
		Children:  []*runtime.Instance{},
	}

	sess := &runtime.Session{
		Root:              root,
		Components:        map[string]*runtime.Instance{"root": root},
		Handlers:          make(map[string]work.Handler),
		MountedComponents: make(map[*runtime.Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if !childCalled {
		t.Error("expected child to be called")
	}
}

func TestProvider_NoErrors_DevMode_ReturnsChildren(t *testing.T) {
	childCalled := false

	childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		childCalled = true
		return &work.Text{Value: "child content"}
	}

	root := &runtime.Instance{
		ID: "root",
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			return Provider(ctx, Props{DevMode: true}, work.Component(childFn))
		},
		HookFrame: []runtime.HookSlot{},
		Children:  []*runtime.Instance{},
	}

	sess := &runtime.Session{
		Root:              root,
		Components:        map[string]*runtime.Instance{"root": root},
		Handlers:          make(map[string]work.Handler),
		MountedComponents: make(map[*runtime.Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if !childCalled {
		t.Error("expected child to be called in dev mode without errors")
	}
}
