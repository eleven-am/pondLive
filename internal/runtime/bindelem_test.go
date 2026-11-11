package runtime_test

import (
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func bindItem(v any, opts ...work.BindOpt) Item { return work.Bind(v, opts...) }

func firstID(app *runtimetest.App, tag string) uint32 {
	for _, id := range app.Model.IDs() {
		if n := app.Model.Node(id); n != nil && n.Data == tag {
			return id
		}
	}
	return 0
}

func attrEdits(edits []protocol.Edit, name string) []protocol.Edit {
	var out []protocol.Edit
	for _, e := range edits {
		if (e.Op == protocol.OpAttr || e.Op == protocol.OpRemoveAttr) && e.Name == name {
			out = append(out, e)
		}
	}
	return out
}

func TestBindConstantsAreStaticAttributes(t *testing.T) {
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		return el("div",
			el("input", bindItem("hello")),
			el("input", work.Attribute("type", "checkbox"), bindItem(true)),
			el("input", work.Attribute("type", "checkbox"), bindItem(false)))
	})
	if !strings.Contains(app.SSR, `<input value="hello">`) || strings.Count(app.SSR, "checked") != 1 || strings.Contains(app.SSR, "data-pon") {
		t.Fatalf("ssr = %s", app.SSR)
	}
}

func TestBindWritableSignalIsTwoWayWithoutAnEchoToTheSameElement(t *testing.T) {
	var sig *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sig = runtime.UseSignal(ctx, "a")
		return el("div", el("input", bindItem(sig)), el("p", bound(sig)))
	})
	if !strings.Contains(app.SSR, `value="a"`) || !strings.Contains(app.SSR, `"e":"~change"`) || !strings.Contains(app.SSR, `"p":"value"`) {
		t.Fatalf("ssr = %s", app.SSR)
	}
	id := firstID(app, "input")

	edits := app.Type(id, "typed")
	if got := attrEdits(edits, "value"); len(got) != 0 {
		t.Fatalf("the element's own input must not be echoed back to it: %v", edits)
	}
	if !strings.Contains(app.Normalized(), "<p>typed</p>") {
		t.Fatalf("other readers must see the new value: %s", app.Normalized())
	}
	var now string
	app.On(func() { now = sig.Peek() })
	if now != "typed" {
		t.Fatalf("signal = %q", now)
	}

	edits = app.Do(func() { sig.Set("") })
	if got := attrEdits(edits, "value"); len(got) != 1 || got[0].Value != "" {
		t.Fatalf("a server-side Set must reach the element: %v", edits)
	}
	edits = app.Do(func() { sig.Set("typed") })
	if got := attrEdits(edits, "value"); len(got) != 1 {
		t.Fatalf("an old echo must not suppress a later server change: %v", edits)
	}
}

func TestBindSendsBackAValueTheAppChangedOnInput(t *testing.T) {
	var sig *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sig = runtime.UseSignal(ctx, "")
		runtime.UseEffect(ctx, func() func() {
			if v := sig.Get(); v != strings.ToUpper(v) {
				sig.Set(strings.ToUpper(v))
			}
			return nil
		})
		return el("input", bindItem(sig))
	})
	id := firstID(app, "input")
	edits := app.Type(id, "abc")
	got := attrEdits(edits, "value")
	if len(got) != 1 || got[0].Value != "ABC" {
		t.Fatalf("a value the app rewrote must be sent back: %v", edits)
	}
}

func TestBindLazyUsesChangeAndDebounceIsInTheHandlerSpec(t *testing.T) {
	var sig *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sig = runtime.UseSignal(ctx, "")
		return el("div",
			el("input", bindItem(sig, work.Lazy{})),
			el("textarea", bindItem(sig, work.Debounce(250*time.Millisecond))),
			el("input", bindItem(sig, work.Lazy{}, work.Debounce(40*time.Millisecond))))
	})
	html := app.SSR
	if !strings.Contains(html, `{"e":"change","p":"value"}`) {
		t.Fatalf("lazy must listen to change: %s", html)
	}
	if !strings.Contains(html, `{"e":"~change","db":250,"p":"value"}`) {
		t.Fatalf("debounce must be in the spec: %s", html)
	}
	if !strings.Contains(html, `{"e":"change","db":40,"p":"value"}`) {
		t.Fatalf("lazy and debounce must compose: %s", html)
	}
}

func TestBindCheckboxIsTwoWayOnChecked(t *testing.T) {
	var sig *runtime.Signal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sig = runtime.UseSignal(ctx, true)
		return el("div", el("input", work.Attribute("type", "checkbox"), bindItem(sig)), el("p", bound(runtime.UseComputed(ctx, func() string {
			if sig.Get() {
				return "on"
			}
			return "off"
		}))))
	})
	if !strings.Contains(app.SSR, "checked") || !strings.Contains(app.SSR, `{"e":"~change","p":"checked"}`) {
		t.Fatalf("ssr = %s", app.SSR)
	}
	id := firstID(app, "input")
	edits := app.Check(id, false)
	if got := attrEdits(edits, "checked"); len(got) != 0 {
		t.Fatalf("checkbox echo: %v", edits)
	}
	if !strings.Contains(app.Normalized(), "<p>off</p>") {
		t.Fatalf("html = %s", app.Normalized())
	}
	edits = app.Do(func() { sig.Set(true) })
	if got := attrEdits(edits, "checked"); len(got) != 1 || got[0].Op != protocol.OpAttr {
		t.Fatalf("a server-side change must check the box: %v", edits)
	}
}

func TestBindSelectSingleAndMultiple(t *testing.T) {
	var one *runtime.Signal[string]
	var many *runtime.Signal[[]string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		one = runtime.UseSignal(ctx, "b")
		many = runtime.UseSignal(ctx, []string{"x", "z"})
		opts := func() []Item {
			return []Item{
				el("option", work.Attribute("value", "a"), txt("A")),
				el("option", work.Attribute("value", "b"), txt("B")),
				el("option", work.Attribute("value", "x"), txt("X")),
				el("option", work.Attribute("value", "z"), txt("Z")),
			}
		}
		return el("div",
			el("select", append([]Item{work.Attribute("id", "one"), bindItem(one)}, opts()...)...),
			el("select", append([]Item{work.Attribute("id", "many"), work.Attribute("multiple", true), bindItem(many)}, opts()...)...))
	})
	html := app.SSR
	if strings.Count(html, `selected=""`) != 3 || !strings.Contains(html, `data-psel='["x","z"]'`) {
		t.Fatalf("ssr = %s", html)
	}
	if !strings.Contains(html, `{"e":"~change","p":"value"}`) || !strings.Contains(html, `{"e":"~change","p":"selected"}`) {
		t.Fatalf("handlers: %s", html)
	}
	var ids []uint32
	for _, id := range app.Model.IDs() {
		if n := app.Model.Node(id); n != nil && n.Data == "select" {
			ids = append(ids, id)
		}
	}
	edits := app.Choose(ids[0], "a")
	if len(attrEdits(edits, "value")) != 0 {
		t.Fatalf("select echo: %v", edits)
	}
	edits = app.Choose(ids[1], "a", "x")
	if len(attrEdits(edits, "data-psel")) != 0 {
		t.Fatalf("multiple echo: %v", edits)
	}
	var got []string
	var cur string
	app.On(func() { got = many.Peek(); cur = one.Peek() })
	if strings.Join(got, ",") != "a,x" || cur != "a" {
		t.Fatalf("signals: %v %q", got, cur)
	}
	edits = app.Do(func() { many.Set([]string{"z"}) })
	if e := attrEdits(edits, "data-psel"); len(e) != 1 || e[0].Value != `["z"]` {
		t.Fatalf("server change of a multiple select: %v", edits)
	}
}

func TestBindTextareaCarriesItsValueInTheBody(t *testing.T) {
	var sig *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sig = runtime.UseSignal(ctx, "line 1")
		return el("textarea", bindItem(sig))
	})
	if !strings.Contains(app.SSR, ">line 1</textarea>") {
		t.Fatalf("ssr = %s", app.SSR)
	}
	edits := app.Do(func() { sig.Set("line 2") })
	if got := attrEdits(edits, "value"); len(got) != 1 {
		t.Fatalf("textarea update: %v", edits)
	}
}

func TestBindReadOnlySignalOnAnEditableControlWarns(t *testing.T) {
	body := func(readonly bool) func(*Ctx) Node {
		return func(ctx *Ctx) Node {
			sig := runtime.UseSignal(ctx, "x")
			ro := runtime.UseComputed(ctx, func() string { return sig.Get() })
			items := []Item{bindItem(ro)}
			if readonly {
				items = append(items, work.Attribute("readonly", true))
			}
			return el("input", items...)
		}
	}
	for _, dev := range []bool{false, true} {
		app := runtimetest.New(t, body(false), runtimetest.Dev(dev))
		d := app.Diagnostics("warning:BIND_READ_ONLY_SIGNAL")
		if len(d) != 1 || !strings.Contains(d[0].Message, "typing here won't update this value") {
			t.Fatalf("dev=%v diagnostics = %v", dev, app.Diags)
		}
		if strings.Contains(app.SSR, "data-pon") {
			t.Fatalf("a read-only bind must not listen: %s", app.SSR)
		}
	}
	quiet := runtimetest.New(t, body(true))
	if len(quiet.Diagnostics("warning:BIND_READ_ONLY_SIGNAL")) != 0 {
		t.Fatalf("a readonly control must not warn: %v", quiet.Diags)
	}
}

func TestBindDevWarningReachesTheOverlayPanel(t *testing.T) {
	var boundary *runtime.Boundary
	child := runtime.Component(func(ctx *Ctx) Node {
		sig := runtime.UseSignal(ctx, "x")
		return el("input", bindItem(runtime.UseComputed(ctx, func() string { return sig.Get() })))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		boundary = runtime.UseErrorBoundary(ctx)
		return el("div", child())
	}, runtimetest.Dev(true))
	var n int
	app.On(func() { n = len(boundary.Warnings().Peek()) })
	if n != 1 {
		t.Fatalf("warnings = %d", n)
	}
}

func TestBindNonEditableElementsAreOneWay(t *testing.T) {
	var sig *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sig = runtime.UseSignal(ctx, "5")
		return el("div",
			el("progress", bindItem(sig)), el("meter", bindItem(sig)), el("li", bindItem(sig)),
			el("button", bindItem(sig)), el("data", bindItem(sig)), el("option", bindItem(sig)))
	})
	if strings.Contains(app.SSR, "data-pon") || strings.Count(app.SSR, `value="5"`) != 6 {
		t.Fatalf("ssr = %s", app.SSR)
	}
	if len(app.Diagnostics("warning:BIND_READ_ONLY_SIGNAL")) != 0 {
		t.Fatalf("no warning is due on a non-editable element: %v", app.Diags)
	}
	edits := app.Do(func() { sig.Set("6") })
	if len(attrEdits(edits, "value")) != 6 {
		t.Fatalf("one-way updates: %v", edits)
	}
}

func TestBindDisabledOrReadonlyControlsDoNotListen(t *testing.T) {
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sig := runtime.UseSignal(ctx, "x")
		return el("div",
			el("input", work.Attribute("disabled", true), bindItem(sig)),
			el("input", work.Attribute("readonly", true), bindItem(sig)))
	})
	if strings.Contains(app.SSR, "data-pon") {
		t.Fatalf("ssr = %s", app.SSR)
	}
}

func TestBindWrongValueTypeIsADevErrorAndAProdDiagnostic(t *testing.T) {
	body := func(ctx *Ctx) Node {
		flag := runtime.UseSignal(ctx, true)
		return el("input", work.Attribute("type", "text"), bindItem(flag))
	}
	prod := runtimetest.New(t, body, runtimetest.Dev(false))
	if len(prod.Diagnostics("warning:BIND_TYPE_MISMATCH")) != 1 || strings.Contains(prod.SSR, "data-pon") {
		t.Fatalf("prod: %v %s", prod.Diags, prod.SSR)
	}
	r, err := rawStart(t, true, body)
	if err == nil || !strings.Contains(err.Error(), "Bind on <input type=text> needs a string, got a bool") {
		t.Fatalf("dev error = %v (html %q)", err, r.html)
	}
}

func TestBindOfAnUnsupportedTypePanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Bind(42) must panic")
		}
	}()
	work.Bind(42)
}
