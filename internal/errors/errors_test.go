package errors

import (
	"strings"
	"sync"
	"testing"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/shared"
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
	errs := runtime.Const([]*runtime.Error{err1, err2})

	node := devOverlay(errs)
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
	node := devOverlay(runtime.Const([]*runtime.Error(nil)))
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

func broken(boom bool) func() work.Node {
	return runtime.Component(func(ctx *runtime.Ctx) work.Node {
		if boom {
			panic("kaboom")
		}
		return work.BuildElement("p", work.NewText("child content"))
	})
}

func TestProvider_NoErrors_ReturnsChildren(t *testing.T) {
	for _, dev := range []bool{false, true} {
		app := runtimetest.New(t, func(ctx *runtime.Ctx) work.Node {
			return Provider(Props{DevMode: dev, Children: broken(false)()})
		}, runtimetest.Dev(dev))

		if got := app.Normalized(); !strings.Contains(got, "<p>child content</p>") || strings.Contains(got, "Runtime Error") || strings.Contains(got, "Something went wrong") {
			t.Fatalf("dev=%v: %s", dev, got)
		}
	}
}

func TestProvider_Prod_ReplacesChildrenWithCrashPage(t *testing.T) {
	app := runtimetest.New(t, func(ctx *runtime.Ctx) work.Node {
		return Provider(Props{DevMode: false, Children: broken(true)()})
	}, runtimetest.Dev(false))

	got := app.Normalized()
	if !strings.Contains(got, "Something went wrong") || strings.Contains(got, "kaboom") {
		t.Fatalf("prod crash page expected without internals: %s", got)
	}
}

func TestProvider_Dev_ShowsTheOverlayWithTheError(t *testing.T) {
	app := runtimetest.New(t, func(ctx *runtime.Ctx) work.Node {
		return Provider(Props{DevMode: true, Children: broken(true)()})
	}, runtimetest.Dev(false))

	got := app.Normalized()
	if !strings.Contains(got, "Runtime Error") || !strings.Contains(got, "kaboom") {
		t.Fatalf("dev overlay expected: %s", got)
	}
}

func warnApp(t *testing.T, dev bool, body func(*Ctx) work.Node) *runtimetest.App {
	t.Helper()
	child := runtime.Component(body)
	return runtimetest.New(t, func(*Ctx) work.Node {
		return Provider(Props{DevMode: dev, Children: work.BuildElement("main", child())})
	}, runtimetest.Dev(dev))
}

type leaker struct {
	block chan struct{}
	wg    sync.WaitGroup
}

func newLeaker(t *testing.T) *leaker {
	l := &leaker{block: make(chan struct{})}
	t.Cleanup(func() {
		close(l.block)
		l.wg.Wait()
	})
	return l
}

func (l *leaker) body(ctx *Ctx) work.Node {
	runtime.UseRenderEffect(ctx, func() func() {
		l.wg.Add(1)
		go func() {
			defer l.wg.Done()
			<-l.block
		}()
		return nil
	})
	return work.BuildElement("p", work.NewText("page"))
}

func dismissButtons(app *runtimetest.App) []uint32 {
	var out []uint32
	for _, id := range app.Model.IDs() {
		n := app.Model.Node(id)
		if n == nil || n.Data != "button" || n.FirstChild == nil || n.FirstChild.Data != "Dismiss" {
			continue
		}
		out = append(out, id)
	}
	return out
}

func TestWarningsAreHeldOutOfTheServerRenderAndShownAfterConnect(t *testing.T) {
	app := warnApp(t, true, newLeaker(t).body)

	if strings.Contains(app.SSR, "data-pondlive-warnings") {
		t.Fatalf("a warning reached the server-rendered HTML: %s", app.SSR)
	}
	html := app.Normalized()
	for _, want := range []string{"data-pondlive-warnings", "RENDER_EFFECT_LEAKED_GOROUTINE", "leaker", "created by", "errors_test.go", "Dismiss"} {
		if !strings.Contains(html, want) {
			t.Fatalf("panel is missing %q after connect: %s", want, html)
		}
	}
	if strings.Contains(html, "Runtime Error") {
		t.Fatalf("a warning must not raise the blocking error overlay: %s", html)
	}
}

func TestWarningPanelIsDistinctFromTheErrorOverlayAndDismissible(t *testing.T) {
	app := warnApp(t, true, newLeaker(t).body)

	buttons := dismissButtons(app)
	if len(buttons) != 1 {
		t.Fatalf("want one per-warning dismiss button, got %d", len(buttons))
	}
	if err := app.Dispatch(buttons[0], "click", nil); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(app.Normalized(), "data-pondlive-warnings") {
		t.Fatalf("panel still open after dismissing the only warning: %s", app.Normalized())
	}
}

func TestProdShowsNoWarningPanel(t *testing.T) {
	app := warnApp(t, false, newLeaker(t).body)
	if strings.Contains(app.Normalized(), "data-pondlive-warnings") {
		t.Fatalf("prod rendered a warning panel: %s", app.Normalized())
	}
}

func TestRenderEffectForbiddenCallUsesTheExistingErrorOverlayInDev(t *testing.T) {
	hub := shared.NewShared(0)
	app := warnApp(t, true, func(ctx *Ctx) work.Node {
		runtime.UseRenderEffect(ctx, func() func() { hub.Set(1); return nil })
		return work.BuildElement("p", work.NewText("page"))
	})
	html := app.Normalized()
	if !strings.Contains(html, "Runtime Error") || !strings.Contains(html, "Shared.Set") || !strings.Contains(html, "UseRenderEffect") {
		t.Fatalf("overlay = %s", html)
	}
	if hub.Get() != 0 {
		t.Fatal("the forbidden write happened")
	}
}
