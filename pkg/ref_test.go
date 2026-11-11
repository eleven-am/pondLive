package pkg

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
)

type wire struct {
	mu   sync.Mutex
	data []any
	evts []string
}

func tap(app *runtimetest.App) *wire {
	w := &wire{}
	app.S.Bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		w.mu.Lock()
		w.evts = append(w.evts, event)
		w.data = append(w.data, data)
		w.mu.Unlock()
	})
	return w
}

func (w *wire) last() any {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.data) == 0 {
		return nil
	}
	return w.data[len(w.data)-1]
}

func (w *wire) count(event string) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, e := range w.evts {
		if e == event {
			n++
		}
	}
	return n
}

func pushValue(app *runtimetest.App, id uint32, prop string, v any) {
	app.On(func() {
		app.S.Bus.Publish(protocol.DOMHandler, string(protocol.DOMValueAction), protocol.DOMValuePayload{ID: id, Prop: prop, Value: v})
	})
	app.Sync()
}

func renderError(t *testing.T, root func(*Ctx) Node) error {
	t.Helper()
	s := runtime.NewSession(runtime.Config{SessionID: "dev", Root: root, DevMode: true})
	runtimetest.Start(t, s)
	var err error
	if aerr := s.Ask(time.Second, func() error {
		_, err = s.RenderSnapshot()
		return nil
	}); aerr != nil {
		t.Fatal(aerr)
	}
	return err
}

func TestUseRefAcceptsEachCapabilityOnItsTagsAndRejectsOthers(t *testing.T) {
	type tc struct {
		name string
		tag  string
		use  func(ctx *Ctx) Item
		ok   bool
	}
	cases := []tc{
		{"ElementRef/div", "div", func(ctx *Ctx) Item { return Attach(UseRef[ElementRef](ctx)) }, true},
		{"InputRef/input", "input", func(ctx *Ctx) Item { return Attach(UseRef[InputRef](ctx)) }, true},
		{"InputRef/select", "select", func(ctx *Ctx) Item { return Attach(UseRef[InputRef](ctx)) }, true},
		{"InputRef/button", "button", func(ctx *Ctx) Item { return Attach(UseRef[InputRef](ctx)) }, true},
		{"InputRef/div", "div", func(ctx *Ctx) Item { return Attach(UseRef[InputRef](ctx)) }, false},
		{"FormRef/form", "form", func(ctx *Ctx) Item { return Attach(UseRef[FormRef](ctx)) }, true},
		{"FormRef/div", "div", func(ctx *Ctx) Item { return Attach(UseRef[FormRef](ctx)) }, false},
		{"MediaRef/video", "video", func(ctx *Ctx) Item { return Attach(UseRef[MediaRef](ctx)) }, true},
		{"MediaRef/audio", "audio", func(ctx *Ctx) Item { return Attach(UseRef[MediaRef](ctx)) }, true},
		{"MediaRef/input", "input", func(ctx *Ctx) Item { return Attach(UseRef[MediaRef](ctx)) }, false},
		{"DialogRef/dialog", "dialog", func(ctx *Ctx) Item { return Attach(UseRef[DialogRef](ctx)) }, true},
		{"DialogRef/div", "div", func(ctx *Ctx) Item { return Attach(UseRef[DialogRef](ctx)) }, false},
		{"CanvasRef/canvas", "canvas", func(ctx *Ctx) Item { return Attach(UseRef[CanvasRef](ctx)) }, true},
		{"CanvasRef/img", "img", func(ctx *Ctx) Item { return Attach(UseRef[CanvasRef](ctx)) }, false},
		{"DetailsRef/details", "details", func(ctx *Ctx) Item { return Attach(UseRef[DetailsRef](ctx)) }, true},
		{"DetailsRef/section", "section", func(ctx *Ctx) Item { return Attach(UseRef[DetailsRef](ctx)) }, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := func(ctx *Ctx) Node { return El(c.tag, c.use(ctx)) }
			prod := runtimetest.New(t, root, runtimetest.Dev(false))
			rejected := len(prod.Diagnostics("warning:REF_INCOMPATIBLE_TAG")) == 1
			if c.ok == rejected {
				t.Fatalf("prod: ok=%v rejected=%v diagnostics=%v", c.ok, rejected, prod.Diags)
			}
			if c.ok {
				return
			}
			if err := renderError(t, root); err == nil || !strings.Contains(err.Error(), "cannot be attached") {
				t.Fatalf("dev: attaching to an incompatible tag must be an error, got %v", err)
			}
		})
	}
}

func TestUseRefBuildsTheTypedWrapperAndRepeatedLiveCallsShareOneSignal(t *testing.T) {
	var in *InputRef
	var a, b ReadSignal[bool]
	runtimetest.New(t, func(ctx *Ctx) Node {
		in = UseRef[InputRef](ctx)
		a, b = in.Focused(), in.Focused()
		return Input(Attach(in))
	})
	if in == nil || in.ElementRef == nil || in.ref == nil {
		t.Fatalf("ref not built: %+v", in)
	}
	if a != b {
		t.Fatal("two Focused() calls must return the same signal")
	}
}

func TestMediaRefSignalsStartFromTheHTMLAndFollowTheBrowser(t *testing.T) {
	var m *MediaRef
	var cur, dur, vol ReadSignal[float64]
	var paused, ended, muted ReadSignal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		m = UseRef[MediaRef](ctx)
		cur, dur, vol = m.CurrentTime(), m.Duration(), m.Volume()
		paused, ended, muted = m.Paused(), m.Ended(), m.Muted()
		return Video(Attach(m), Muted(true), Autoplay(true))
	})
	app.On(func() {
		if cur.Peek() != 0 || dur.Peek() != 0 || vol.Peek() != 1 || paused.Peek() || ended.Peek() || !muted.Peek() {
			t.Errorf("start values: time=%v dur=%v vol=%v paused=%v ended=%v muted=%v", cur.Peek(), dur.Peek(), vol.Peek(), paused.Peek(), ended.Peek(), muted.Peek())
		}
	})
	pushValue(app, m.ref.ID(), "currentTime", 12.5)
	pushValue(app, m.ref.ID(), "duration", nil)
	pushValue(app, m.ref.ID(), "paused", true)
	pushValue(app, m.ref.ID(), "ended", true)
	pushValue(app, m.ref.ID(), "volume", 0.25)
	pushValue(app, m.ref.ID(), "muted", false)
	app.On(func() {
		if cur.Peek() != 12.5 || dur.Peek() != 0 || vol.Peek() != 0.25 || !paused.Peek() || !ended.Peek() || muted.Peek() {
			t.Errorf("after events: time=%v dur=%v vol=%v paused=%v ended=%v muted=%v", cur.Peek(), dur.Peek(), vol.Peek(), paused.Peek(), ended.Peek(), muted.Peek())
		}
	})
}

func TestElementRefFocusedRectAndScroll(t *testing.T) {
	var r *ElementRef
	var focused ReadSignal[bool]
	var rect ReadSignal[DOMRect]
	var scroll ReadSignal[ScrollMetrics]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		r = UseRef[ElementRef](ctx)
		focused, rect, scroll = r.Focused(), r.Rect(Throttle(40*time.Millisecond)), r.Scroll()
		return Div(Attach(r))
	})
	w := tap(app)
	app.On(func() { _, _, _ = focused.Peek(), rect.Peek(), scroll.Peek() })
	if w.count("observe") != 3 {
		t.Fatalf("expected three lazy subscriptions, got %d", w.count("observe"))
	}
	pushValue(app, r.ref.ID(), "focused", true)
	pushValue(app, r.ref.ID(), "rect", map[string]any{"x": 1.0, "y": 2.0, "width": 30.0, "height": 40.0, "top": 2.0, "right": 31.0, "bottom": 42.0, "left": 1.0})
	pushValue(app, r.ref.ID(), "scroll", map[string]any{"scrollTop": 5.0, "scrollHeight": 900.0, "clientHeight": 300.0})
	app.On(func() {
		if !focused.Peek() || rect.Peek().Width != 30 || rect.Peek().Bottom != 42 || scroll.Peek().ScrollTop != 5 || scroll.Peek().ScrollHeight != 900 {
			t.Errorf("focused=%v rect=%+v scroll=%+v", focused.Peek(), rect.Peek(), scroll.Peek())
		}
	})
	pushValue(app, r.ref.ID(), "rect", "garbage")
	app.On(func() {
		if rect.Peek().Width != 30 {
			t.Errorf("an undecodable value changed the signal: %+v", rect.Peek())
		}
	})
}

func TestRefSettersAndCallsReachTheWire(t *testing.T) {
	var in *InputRef
	var med *MediaRef
	var dlg *DialogRef
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		in, med, dlg = UseRef[InputRef](ctx), UseRef[MediaRef](ctx), UseRef[DialogRef](ctx)
		return Div(Input(Attach(in)), Video(Attach(med)), Dialog(Attach(dlg)))
	})
	w := tap(app)
	app.On(func() {
		in.Select()
		in.SetSelectionRange(1, 2)
		med.SetCurrentTime(3)
		dlg.ShowModal()
	})
	if w.count("set") != 1 || w.count("call") != 3 {
		t.Fatalf("set=%d call=%d events=%v", w.count("set"), w.count("call"), w.evts)
	}
}

func TestOneOffReadsAreCallbacksNotBlockingGetters(t *testing.T) {
	var cv *CanvasRef
	var url string
	var urlErr error
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		cv = UseRef[CanvasRef](ctx)
		return Canvas(Attach(cv))
	})
	w := tap(app)
	app.On(func() { cv.ToDataURL(func(s string, err error) { url, urlErr = s, err }, "image/png") })
	if url != "" || urlErr != nil {
		t.Fatal("ToDataURL must return before the browser answers")
	}
	req := w.last().(protocol.DOMAsyncPayload)
	if req.Method != "toDataURL" || len(req.Args) != 1 {
		t.Fatalf("request = %#v", req)
	}
	app.S.ResolveDOMResponse(protocol.DOMResponsePayload{RequestID: req.RequestID, Result: "data:image/png;base64,AAAA"})
	app.On(func() {})
	if url != "data:image/png;base64,AAAA" || urlErr != nil {
		t.Fatalf("url=%q err=%v", url, urlErr)
	}
}

func TestInputRefSelectionCallback(t *testing.T) {
	var in *InputRef
	var start, end int
	var dir string
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		in = UseRef[InputRef](ctx)
		return Input(Attach(in))
	})
	w := tap(app)
	app.On(func() { in.Selection(func(s, e int, d string, err error) { start, end, dir = s, e, d }) })
	req := w.last().(protocol.DOMQueryPayload)
	app.S.ResolveDOMResponse(protocol.DOMResponsePayload{RequestID: req.RequestID, Values: map[string]any{"selectionStart": 1.0, "selectionEnd": 4.0, "selectionDirection": "forward"}})
	app.On(func() {})
	if start != 1 || end != 4 || dir != "forward" {
		t.Fatalf("selection = %d,%d,%q", start, end, dir)
	}
}

func TestRefCallsOutsideARunningSessionAreReportedNotDropped(t *testing.T) {
	var r *ElementRef
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		r = UseRef[ElementRef](ctx)
		return Div()
	})
	app.On(func() { r.Focus() })
	if len(app.Diagnostics("warning:REF_DETACHED")) != 1 {
		t.Fatalf("an unattached ref call must be reported: %v", app.Diags)
	}
}

func TestRefCallsInsideRenderEffectAreRefused(t *testing.T) {
	var found bool
	runtimetest.New(t, func(ctx *Ctx) Node {
		r := UseRef[ElementRef](ctx)
		UseRenderEffect(ctx, func() func() {
			defer func() {
				found = recover() != nil
			}()
			r.Focus()
			return nil
		})
		return Div(Attach(r))
	})
	if !found {
		t.Fatal("a ref call inside UseRenderEffect must be refused")
	}
}

func TestReportUsesTheFrameworkDiagnosticPath(t *testing.T) {
	var boundary *Boundary
	prod := runtimetest.New(t, func(ctx *Ctx) Node {
		Report(ctx, ReportWarning, "APP_SLOW", "the thing is slow")
		Report(ctx, ReportError, "APP_BROKEN", "the thing broke")
		return Div()
	}, runtimetest.Dev(false))
	if len(prod.Diagnostics("warning:APP_SLOW")) != 1 || len(prod.Diagnostics("error:APP_BROKEN")) != 1 {
		t.Fatalf("prod diagnostics = %v", prod.Diags)
	}
	child := Component(func(ctx *Ctx) Node {
		Report(ctx, ReportWarning, "APP_SLOW", "the thing is slow")
		Report(ctx, ReportError, "APP_BROKEN", "the thing broke")
		return Div()
	})
	dev := runtimetest.New(t, func(ctx *Ctx) Node {
		boundary = UseErrorBoundary(ctx)
		return Div(child())
	}, runtimetest.Dev(true))
	var warns, errs int
	dev.On(func() {
		warns = len(boundary.Warnings().Peek())
		errs = boundary.Errors().Peek().Count()
	})
	if warns != 1 || errs != 1 {
		t.Fatalf("dev overlay: warnings=%d errors=%d", warns, errs)
	}
	if len(dev.Diagnostics("warning:APP_SLOW")) != 1 || len(dev.Diagnostics("error:APP_BROKEN")) != 1 {
		t.Fatalf("dev must also log: %v", dev.Diags)
	}
}
