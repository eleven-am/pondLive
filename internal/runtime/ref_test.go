package runtime_test

import (
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func useRef(ctx *Ctx, name string, accepts func(string) bool) *runtime.Ref {
	return runtime.UseCell(ctx, func() *runtime.Ref { return runtime.NewRef(ctx, name, accepts) })
}

func onlyTag(tag string) func(string) bool { return func(t string) bool { return t == tag } }

type domLog struct {
	mu     sync.Mutex
	events []string
	data   []any
}

func watchDOM(app *runtimetest.App) *domLog {
	l := &domLog{}
	app.S.Bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		l.mu.Lock()
		l.events = append(l.events, event)
		l.data = append(l.data, data)
		l.mu.Unlock()
	})
	return l
}

func (l *domLog) names() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return strings.Join(l.events, ",")
}

func (l *domLog) last() any {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.data) == 0 {
		return nil
	}
	return l.data[len(l.data)-1]
}

func connect(app *runtimetest.App) { app.On(func() { app.S.SetConnected(true) }) }

func TestRefCallPublishesTheAttachedID(t *testing.T) {
	var ref *runtime.Ref
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		ref = useRef(ctx, "ElementRef", nil)
		return el("button", runtime.AttachItem(ref), txt("go"))
	})
	log := watchDOM(app)
	connect(app)
	app.On(func() { ref.Call("scrollIntoView", 0, 10) })

	got, ok := log.last().(protocol.DOMCallPayload)
	if !ok || got.ID != ref.ID() || got.ID == 0 || got.Method != "scrollIntoView" || len(got.Args) != 2 {
		t.Fatalf("published %#v (events %s)", log.last(), log.names())
	}
}

func TestRefCallBeforeConnectIsReportedNotSilentlyDropped(t *testing.T) {
	for _, dev := range []bool{false, true} {
		var ref *runtime.Ref
		d := rawStartDeferred(t, dev, func(ctx *Ctx) Node {
			ref = useRef(ctx, "ElementRef", nil)
			return el("button", runtime.AttachItem(ref))
		})
		var calls []string
		d.s.Bus.Subscribe(protocol.DOMHandler, func(event string, _ interface{}) { calls = append(calls, event) })
		var diags []string
		d.s.Bus.SubscribeToDiagnostics(func(x protocol.Diagnostic) { diags = append(diags, x.Phase) })
		d.ask(func() { ref.Call("focus") })
		if len(calls) != 0 {
			t.Fatalf("dev=%v: a pre-connect call reached the wire: %v", dev, calls)
		}
		if strings.Join(diags, ",") != "warning:REF_NOT_CONNECTED" {
			t.Fatalf("dev=%v: diagnostics = %v", dev, diags)
		}
		d.goLive()
		d.ask(func() { ref.Call("focus") })
		if len(calls) != 1 {
			t.Fatalf("dev=%v: a call after the session went live was dropped: %v", dev, calls)
		}
	}
}

func TestRefCallOnADetachedRefIsReported(t *testing.T) {
	var ref *runtime.Ref
	var flag *runtime.Signal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		flag = runtime.UseSignal(ctx, true)
		ref = useRef(ctx, "ElementRef", nil)
		return el("div", runtime.Show(flag, func() Node { return el("button", runtime.AttachItem(ref)) }))
	})
	log := watchDOM(app)
	connect(app)
	app.Do(func() { flag.Set(false) })
	app.On(func() { ref.Call("focus") })

	if log.names() != "" {
		t.Fatalf("a detached call reached the wire: %s", log.names())
	}
	if d := app.Diagnostics("warning:REF_DETACHED"); len(d) != 1 {
		t.Fatalf("diagnostics = %v", app.Diags)
	}
}

func TestRefIncompatibleTagIsADevErrorAndAProdDiagnostic(t *testing.T) {
	var ref *runtime.Ref
	body := func(ctx *Ctx) Node {
		ref = useRef(ctx, "MediaRef", onlyTag("video"))
		return el("div", runtime.AttachItem(ref))
	}
	prod, err := rawStart(t, false, body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prod.phases(), "warning:REF_INCOMPATIBLE_TAG") || ref.IsAttached() {
		t.Fatalf("prod: attached=%v diagnostics=%s", ref.IsAttached(), prod.phases())
	}
	if _, err := rawStart(t, true, body); err == nil || !strings.Contains(err.Error(), "MediaRef cannot be attached to <div>") {
		t.Fatalf("dev: want an incompatible-tag error, got %v", err)
	}
}

func TestRefAttachedTwiceIsAnError(t *testing.T) {
	app, err := rawStart(t, true, func(ctx *Ctx) Node {
		ref := useRef(ctx, "ElementRef", nil)
		return el("div", el("a", runtime.AttachItem(ref)), el("b", runtime.AttachItem(ref)))
	})
	if err == nil {
		t.Fatalf("attaching one ref to two live elements must be an error, got html %q", app.html)
	}
}

func TestRefQueryAnswersLaterOnTheActor(t *testing.T) {
	var ref *runtime.Ref
	var got map[string]any
	var gotErr error
	calls := 0
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		ref = useRef(ctx, "InputRef", nil)
		return el("button", runtime.AttachItem(ref))
	})
	connect(app)
	log := watchDOM(app)
	app.On(func() {
		ref.Query(func(m map[string]any, err error) { got, gotErr, calls = m, err, calls+1 }, "selectionStart")
	})
	if calls != 0 {
		t.Fatal("Query must not call back before the browser answers")
	}
	req := log.last().(protocol.DOMQueryPayload)
	app.S.ResolveDOMResponse(protocol.DOMResponsePayload{RequestID: req.RequestID, Values: map[string]any{"selectionStart": "hi"}})
	app.Settle()
	app.On(func() {})
	if calls != 1 || gotErr != nil || got["selectionStart"] != "hi" {
		t.Fatalf("calls=%d err=%v got=%v", calls, gotErr, got)
	}
}

func TestRefCallThenReportsBrowserErrorsAndTimeouts(t *testing.T) {
	var ref *runtime.Ref
	var errs []error
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		ref = useRef(ctx, "CanvasRef", nil)
		return el("canvas", runtime.AttachItem(ref))
	})
	connect(app)
	log := watchDOM(app)
	ask := func() {
		app.On(func() { ref.CallThen(func(_ any, err error) { errs = append(errs, err) }, "toDataURL") })
	}
	ask()
	req := log.last().(protocol.DOMAsyncPayload)
	app.S.ResolveDOMResponse(protocol.DOMResponsePayload{RequestID: req.RequestID, Error: "tainted"})
	app.On(func() {})
	if len(errs) != 1 || errs[0] == nil || errs[0].Error() != "tainted" {
		t.Fatalf("errs = %v", errs)
	}

	ask()
	app.Advance(6 * time.Second)
	if len(errs) != 2 || errs[1] != runtime.ErrQueryTimeout {
		t.Fatalf("errs = %v", errs)
	}
}

func TestRefQueryOnAnUnattachedRefCallsBackWithAnError(t *testing.T) {
	var gotErr error
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		ref := useRef(ctx, "InputRef", nil)
		runtime.UseEffect(ctx, func() func() {
			ref.Query(func(_ map[string]any, err error) { gotErr = err }, "selectionStart")
			return nil
		})
		return el("div")
	})
	connect(app)
	app.On(func() {})
	if gotErr != runtime.ErrRefNotAttached {
		t.Fatalf("err = %v", gotErr)
	}
}

func TestASlowBrowserReplyDoesNotDelayAnotherEvent(t *testing.T) {
	var ref *runtime.Ref
	var n *runtime.Signal[int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		ref = useRef(ctx, "InputRef", nil)
		n = runtime.UseSignal(ctx, 0)
		return el("div",
			el("button", runtime.AttachItem(ref), work.On("click", func(work.Event) {
				ref.Query(func(map[string]any, error) {}, "selectionStart")
			})),
			el("button", work.On("click", func(work.Event) { n.Set(n.Peek() + 1) })),
			el("p", bound(n)))
	})
	connect(app)
	slowID := ref.ID()
	var fastID uint32
	for _, id := range app.Model.IDs() {
		if nd := app.Model.Node(id); nd != nil && nd.Data == "button" && id != slowID {
			fastID = id
		}
	}
	start := time.Now()
	if err := app.Dispatch(slowID, "click", nil); err != nil {
		t.Fatal(err)
	}
	if err := app.Dispatch(fastID, "click", nil); err != nil {
		t.Fatal(err)
	}
	app.Sync()
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("the second event waited %v for a browser that never answered", elapsed)
	}
	if got := app.Normalized(); !strings.Contains(got, "<p>1</p>") {
		t.Fatalf("second event not applied while the first query is pending: %s", got)
	}
}

func TestRefHandlerAttachesOnlyToElements(t *testing.T) {
	var hits int
	var ref *runtime.Ref
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		ref = useRef(ctx, "ElementRef", nil)
		return el("button", runtime.AttachItem(ref), work.On("click", func(work.Event) { hits++ }), txt("go"))
	})
	requireEqual(t, app.SSR, `<button data-pi="1" data-pon='[{"e":"click"}]'>go</button>`)
	if err := app.Dispatch(ref.ID(), "click", nil); err != nil || hits != 1 {
		t.Fatalf("err=%v hits=%d", err, hits)
	}
}

func TestRefAttachedSignalFollowsTheElement(t *testing.T) {
	var ref *runtime.Ref
	var flag *runtime.Signal[bool]
	var shown runtime.ReadSignal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		flag = runtime.UseSignal(ctx, true)
		ref = useRef(ctx, "ElementRef", nil)
		shown = ref.AttachedSignal()
		return el("div", runtime.Show(flag, func() Node { return el("button", runtime.AttachItem(ref)) }),
			el("p", bound(runtime.UseComputed(ctx, func() string {
				if shown.Get() {
					return "in"
				}
				return "out"
			}))))
	})
	if !strings.Contains(app.Normalized(), "<p>in</p>") {
		t.Fatalf("attached ref must read true: %s", app.Normalized())
	}
	app.Do(func() { flag.Set(false) })
	if !strings.Contains(app.Normalized(), "<p>out</p>") {
		t.Fatalf("a detached ref must read false: %s", app.Normalized())
	}
	app.Do(func() { flag.Set(true) })
	if !strings.Contains(app.Normalized(), "<p>in</p>") {
		t.Fatalf("a re-attached ref must read true: %s", app.Normalized())
	}
}

func TestScriptAndRefShareOneAttachMechanism(t *testing.T) {
	var script runtime.ScriptHandle
	body := func(ctx *Ctx) Node {
		script = runtime.UseScript(ctx, "(el)=>{}")
		return el("div", el("a", runtime.AttachItem(script)), el("b", runtime.AttachItem(script)))
	}
	if _, err := rawStart(t, true, body); err == nil || !strings.Contains(err.Error(), "is attached to two live elements") {
		t.Fatalf("dev: a script attached twice must be an error, got %v", err)
	}
	prod, err := rawStart(t, false, body)
	if err != nil || !strings.Contains(prod.phases(), "warning:REF_ATTACHED_TWICE") {
		t.Fatalf("prod: err=%v diagnostics=%s", err, prod.phases())
	}
}

func TestScriptCanBeAttachedAgainAfterItsElementUnmounts(t *testing.T) {
	var flag *runtime.Signal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		flag = runtime.UseSignal(ctx, true)
		script := runtime.UseScript(ctx, "(el)=>{}")
		return el("div", runtime.Show(flag, func() Node { return el("p", runtime.AttachItem(script)) }))
	})
	app.Do(func() { flag.Set(false) })
	app.Do(func() { flag.Set(true) })
	if strings.Count(app.Normalized(), "data-pscript") != 1 {
		t.Fatalf("html = %s", app.Normalized())
	}
}

func TestRefDropsAnythingNotOnItsTypeList(t *testing.T) {
	var ref *runtime.Ref
	var queryErr error
	var sig runtime.ReadSignal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		ref = useRef(ctx, "ElementRef", nil)
		sig = runtime.RefSignal(ref, runtime.RefSpec[string]{Prop: "value", Zero: "zero", Decode: func(v any) (string, bool) { s, ok := v.(string); return s, ok }})
		return el("div", runtime.AttachItem(ref))
	})
	log := watchDOM(app)
	app.On(func() {
		ref.Call("submit")
		ref.Call("remove")
		ref.Set("innerHTML", "<b>x</b>")
		ref.Set("value", "x")
		ref.Query(func(_ map[string]any, err error) { queryErr = err }, "value")
		ref.CallThen(func(any, error) {}, "eval", "1")
		if got := sig.Peek(); got != "zero" {
			t.Errorf("a prop outside the list must not be observable, got %q", got)
		}
	})
	app.On(func() {})
	if n := log.names(); n != "" {
		t.Fatalf("forbidden requests reached the wire: %s", n)
	}
	if queryErr != runtime.ErrForbiddenCall {
		t.Fatalf("query error = %v", queryErr)
	}
	if got := len(app.Diagnostics("warning:REF_FORBIDDEN_CALL")); got != 7 {
		t.Fatalf("each refused request must be reported, got %d: %v", got, app.Diags)
	}
}

func TestRefPayloadsNameTheirKind(t *testing.T) {
	var ref *runtime.Ref
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		ref = useRef(ctx, "MediaRef", nil)
		return el("video", runtime.AttachItem(ref))
	})
	log := watchDOM(app)
	app.On(func() { ref.Call("play"); ref.Set("volume", 0.5) })
	if p, ok := log.last().(protocol.DOMSetPayload); !ok || p.K != "MediaRef" {
		t.Fatalf("set payload = %#v", log.last())
	}
}
