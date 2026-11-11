package pkg

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/router"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

var buttonIDPattern = regexp.MustCompile(`<button[^>]*data-pi="(\d+)"`)

func buttonID(t *testing.T, html string) uint32 {
	t.Helper()
	m := buttonIDPattern.FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("no button with an id in %s", html)
	}
	n, err := strconv.ParseUint(m[1], 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	return uint32(n)
}

func requireHTML(t *testing.T, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("html mismatch\n got: %s\nwant: %s", got, want)
	}
}

func TestTextAndAttributesFollowSignals(t *testing.T) {
	var label *Signal[string]
	var hidden *Signal[bool]
	var n *Signal[int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		label = UseSignal(ctx, "a")
		hidden = UseSignal(ctx, false)
		n = UseSignal(ctx, 1)
		on := UseComputed(ctx, func() string {
			if hidden.Get() {
				return "on"
			}
			return ""
		})
		return Div(Attr("title", label), Attr("hidden", hidden), Class("x", on), Text(label), Span(Text(n)))
	})
	requireHTML(t, app.Model.HTML(), `<div class="x" data-pi="1" title="a"><!--t:2-->a<span data-pi="3">1</span></div>`)

	app.Do(func() { label.Set("b"); hidden.Set(true); n.Set(2) })

	requireHTML(t, app.Model.HTML(), `<div class="x on" data-pi="1" hidden="" title="b"><!--t:2-->b<span data-pi="3">2</span></div>`)
}

func TestTextRejectsUnsupportedValues(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Text(struct) must panic")
		}
	}()
	Text(struct{}{})
}

func TestShowElseAndSwitchSwapArms(t *testing.T) {
	var on *Signal[bool]
	var mode *Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		on = UseSignal(ctx, true)
		mode = UseSignal(ctx, "a")
		return Div(
			Show(on, func() Node { return P(Text("yes")) }, Else(func() Node { return P(Text("no")) })),
			Switch(mode, Case("a", func() Node { return B(Text("A")) }), Case("b", func() Node { return I(Text("B")) }), Default(func() Node { return U(Text("?")) })),
		)
	})
	if !strings.Contains(app.Model.HTML(), "yes") || !strings.Contains(app.Model.HTML(), ">A</b>") {
		t.Fatalf("initial html: %s", app.Model.HTML())
	}

	app.Do(func() { on.Set(false); mode.Set("b") })
	if h := app.Model.HTML(); !strings.Contains(h, "no") || strings.Contains(h, "yes") || !strings.Contains(h, ">B</i>") {
		t.Fatalf("after swap: %s", h)
	}
	app.Do(func() { mode.Set("zzz") })
	if !strings.Contains(app.Model.HTML(), ">?</u>") {
		t.Fatalf("default arm missing: %s", app.Model.HTML())
	}
}

func TestForItemsInsertsOneRow(t *testing.T) {
	var list *Signal[[]string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = UseSignal(ctx, []string{"a", "b"})
		return Ul(For(Items(list, func(s string) string { return s }), PropsComponent(func(_ *Ctx, item ReadSignal[string]) Node {
			return Li(Text(item.Peek()))
		})))
	})

	edits := app.Do(func() { list.Set([]string{"a", "x", "b"}) })

	if len(edits) != 1 {
		t.Fatalf("edits = %v, want one insert", edits)
	}
	if h := app.Model.HTML(); strings.Index(h, ">x<") < strings.Index(h, ">a<") || strings.Index(h, ">x<") > strings.Index(h, ">b<") {
		t.Fatalf("row not between a and b: %s", h)
	}
}

func TestForFeedAppendsAndDeletesByKey(t *testing.T) {
	var feed *Feed[string, string]
	store := []string{"a"}
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		f := NewFeed(func(s string) string { return s }, func() []string { return append([]string(nil), store...) })
		if feed == nil {
			feed = f
		}
		return Ul(For(f, PropsComponent(func(_ *Ctx, item ReadSignal[string]) Node { return Li(Text(item.Peek())) })))
	})

	app.Do(func() { store = append(store, "b"); feed.Append("b") })
	if !strings.Contains(app.Model.HTML(), "b") {
		t.Fatalf("append missing: %s", app.Model.HTML())
	}
	app.Do(func() { store = store[1:]; feed.Delete("a") })
	if strings.Contains(app.Model.HTML(), ">a<") {
		t.Fatalf("delete failed: %s", app.Model.HTML())
	}
}

func TestTypedHandlerUpdatesASignalAndSendsOneTextEdit(t *testing.T) {
	var count *Signal[int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		count = UseSignal(ctx, 0)
		inc := func(ClickEvent) { count.Update(func(n int) int { return n + 1 }) }
		return Div(Button(OnClick(inc), Text("+")), P(Text(count)))
	})

	for range 3 {
		if err := app.Dispatch(buttonID(t, app.SSR), "click", nil); err != nil {
			t.Fatal(err)
		}
	}

	if !strings.Contains(app.Model.HTML(), ">3</p>") {
		t.Fatalf("html: %s", app.Model.HTML())
	}
}

func TestOnHandlerReceivesTheEvent(t *testing.T) {
	var got Event
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		return Button(On("click", func(e Event) { got = e }), Text("x"))
	})

	if err := app.Dispatch(buttonID(t, app.SSR), "click", map[string]any{"k": "v"}); err != nil {
		t.Fatal(err)
	}

	if got.Name != "click" || got.Payload["k"] != "v" {
		t.Fatalf("event = %+v", got)
	}
}

func TestComputedAndEffectTrackSignals(t *testing.T) {
	var n *Signal[int]
	var seen []int
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		n = UseSignal(ctx, 1)
		double := UseComputed(ctx, func() int { return n.Get() * 2 })
		UseEffect(ctx, func() func() { seen = append(seen, double.Get()); return nil })
		return P(Text(double))
	})

	app.Do(func() { n.Set(5) })

	if len(seen) < 2 || seen[len(seen)-1] != 10 {
		t.Fatalf("watch saw %v", seen)
	}
	requireHTML(t, app.Model.HTML(), `<p data-pi="1">10</p>`)
}

var themeCtx = NewContext[string]()

func TestContextProvideAndUse(t *testing.T) {
	var theme *Signal[string]
	child := Component(func(ctx *Ctx) Node {
		return Span(Text(UseContext(ctx, themeCtx)))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		theme = UseSignal(ctx, "dark")
		Provide(ctx, themeCtx, theme)
		return Div(child())
	})
	if !strings.Contains(app.Model.HTML(), "dark") {
		t.Fatalf("html: %s", app.Model.HTML())
	}

	app.Do(func() { theme.Set("light") })

	if !strings.Contains(app.Model.HTML(), "light") {
		t.Fatalf("html: %s", app.Model.HTML())
	}
}

func TestMissingProviderIsAnError(t *testing.T) {
	app, err := NewApp(Component(func(ctx *Ctx) Node {
		return Div(Text(UseContext(ctx, themeCtx)))
	}))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()

	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}

func TestSharedReachesEverySessionIncludingTheWriter(t *testing.T) {
	counter := NewShared(0)
	var writer *runtimetest.App
	build := func(ctx *Ctx) Node {
		v := UseShared(ctx, counter)
		return P(Text(v))
	}
	writer = runtimetest.New(t, build)
	reader := runtimetest.New(t, build)

	writer.Do(func() { counter.Set(7) })
	reader.Sync()

	for name, app := range map[string]*runtimetest.App{"writer": writer, "reader": reader} {
		if !strings.Contains(app.Model.HTML(), ">7</p>") {
			t.Fatalf("%s html: %s", name, app.Model.HTML())
		}
	}
}

func TestSharedFamilyKeysAreIndependent(t *testing.T) {
	fam := NewSharedFamily(func(k string) int { return len(k) })
	a, b := fam.Key("a"), fam.Key("bb")
	if a == b {
		t.Fatal("keys must map to distinct values")
	}
	build := func(k *Shared[int]) func(*Ctx) Node {
		return func(ctx *Ctx) Node { return P(Text(UseShared(ctx, k))) }
	}
	appA := runtimetest.New(t, build(a))
	appB := runtimetest.New(t, build(b))

	appA.Do(func() { a.Set(9) })
	appB.Sync()

	if !strings.Contains(appA.Model.HTML(), ">9<") || !strings.Contains(appB.Model.HTML(), ">2<") {
		t.Fatalf("a=%s b=%s", appA.Model.HTML(), appB.Model.HTML())
	}
}

func TestBroadcastRunsTheHandlerInEverySession(t *testing.T) {
	bc := NewBroadcast[string]()
	var total *Signal[int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		total = UseSignal(ctx, 0)
		UseBroadcast(ctx, bc, func(string) { total.Update(func(n int) int { return n + 1 }) })
		return P(Text(total))
	})

	bc.Send("hi")
	app.Sync()

	if !strings.Contains(app.Model.HTML(), ">1</p>") {
		t.Fatalf("html: %s", app.Model.HTML())
	}
}

func attrNames(el *work.Element) []string {
	names := make([]string, len(el.Attrs))
	for i, a := range el.Attrs {
		names[i] = a.Name
	}
	return names
}

func TestRequestURLIsReadOnlyAndFollowsTheRouter(t *testing.T) {
	var r *Router
	var req *Request
	state := headers.NewRequestState(&headers.RequestInfo{Path: "/home", Headers: http.Header{}})
	app := runtimetest.New(t, func(*Ctx) Node {
		return headers.Provider(headers.ProviderProps{State: state, Children: router.Provider(router.ProviderProps{
			Children: Div(Component(func(ctx *Ctx) Node {
				req = UseRequest(ctx)
				r = UseRouter(ctx)
				path := UseComputed(ctx, func() string { return req.URL().Get().Path })
				return Span(Text(path))
			})()),
		})})
	})
	app.On(func() { app.S.SetConnected(true) })
	app.Sync()
	if !strings.Contains(app.Model.HTML(), ">/home<") {
		t.Fatalf("html = %s", app.Model.HTML())
	}
	if edits := app.Do(func() { r.Navigate("/users/7") }); len(edits) == 0 {
		t.Fatal("no edit frame")
	}
	if !strings.Contains(app.Model.HTML(), ">/users/7<") {
		t.Fatalf("html = %s", app.Model.HTML())
	}
	if edits := app.Do(func() { req.Push("/users/8") }); len(edits) == 0 || !strings.Contains(app.Model.HTML(), ">/users/8<") {
		t.Fatalf("req.Push: html = %s", app.Model.HTML())
	}
	before := req.URL().Peek()
	app.Do(func() { req.Push("javascript:alert(1)") })
	if got := req.URL().Peek(); got.Path != before.Path {
		t.Fatalf("an unsafe Push moved the URL: %+v", got)
	}
	_ = Status(301)
	if _, ok := reflect.TypeOf(req).MethodByName("SetURL"); ok {
		t.Fatal("Request exposes a URL setter")
	}
}

func TestUseRenderEffectRunsBeforeItsComponentHTMLOnceAndNotInResets(t *testing.T) {
	runs := 0
	label := ""
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		UseRenderEffect(ctx, func() func() { runs++; label = "ready"; return nil })
		return P(Text(label))
	})
	requireHTML(t, app.Model.HTML(), `<p>ready</p>`)

	app.CheckReset()
	app.CheckReset()

	if runs != 1 {
		t.Fatalf("render effect ran %d times", runs)
	}
}

func TestUseRenderEffectRefusesAGlobalWriteThroughTheSharedHandle(t *testing.T) {
	hub := NewShared(0)
	body := func(ctx *Ctx) Node {
		UseRenderEffect(ctx, func() func() { hub.Set(1); return nil })
		return P(Text("x"))
	}
	prod, err := NewApp(Component(body))
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	prod.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || hub.Get() != 0 {
		t.Fatalf("prod: status=%d shared=%d", rec.Code, hub.Get())
	}

	dev, err := NewApp(func() Node { return Div(Component(body)()) }, WithDevMode())
	if err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	dev.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if got := rec.Body.String(); !strings.Contains(got, "Shared.Set") || !strings.Contains(got, "UseRenderEffect") || hub.Get() != 0 {
		t.Fatalf("dev: shared=%d body=%s", hub.Get(), got)
	}
}
