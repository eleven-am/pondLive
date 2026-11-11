package runtime_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
)

type rawApp struct {
	t     testing.TB
	s     *runtime.Session
	html  string
	diags []protocol.Diagnostic
}

func rawStart(t testing.TB, dev bool, root func(*Ctx) Node) (*rawApp, error) {
	t.Helper()
	r := &rawApp{t: t}
	r.s = runtime.NewSession(runtime.Config{SessionID: "raw", Root: root, DevMode: dev})
	r.s.Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) { r.diags = append(r.diags, d) })
	runtimetest.Start(t, r.s)
	var renderErr error
	if err := r.s.Ask(time.Second, func() error {
		snap, err := r.s.RenderSnapshot()
		r.html, renderErr = snap.HTML, err
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return r, renderErr
}

func (r *rawApp) reset() (string, error) {
	r.t.Helper()
	var html string
	var err error
	if aerr := r.s.Ask(time.Second, func() error { html, err = r.s.ResetStep(); return nil }); aerr != nil {
		r.t.Fatal(aerr)
	}
	return html, err
}

func (r *rawApp) phases() string {
	var out []string
	for _, d := range r.diags {
		out = append(out, d.Phase)
	}
	return strings.Join(out, ",")
}

func TestNondeterministicBodyAbortsResetInProd(t *testing.T) {
	calls := 0
	armed := false
	app, err := rawStart(t, false, func(ctx *Ctx) Node {
		calls++
		if armed && calls%2 == 0 {
			return el("div", el("p", bound(runtime.Const("a"))), el("p", bound(runtime.Const("b"))))
		}
		return el("div", el("p", bound(runtime.Const("a"))))
	})
	if err != nil {
		t.Fatal(err)
	}
	armed = true

	_, err = app.reset()

	if !errors.Is(err, runtime.ErrNondeterministic) {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(app.phases(), "reset:nondeterministic") {
		t.Fatalf("diagnostics = %s", app.phases())
	}
}

func TestConditionalHookIsCaught(t *testing.T) {
	armed := false
	app, err := rawStart(t, false, func(ctx *Ctx) Node {
		if !armed {
			runtime.UseSignal(ctx, 1)
		}
		return el("div")
	})
	if err != nil {
		t.Fatal(err)
	}
	armed = true

	_, err = app.reset()

	var nd *runtime.NondeterministicError
	if !errors.As(err, &nd) || !strings.Contains(nd.Detail, "cell") {
		t.Fatalf("err = %v", err)
	}
}

func TestDifferentChildFunctionIsCaught(t *testing.T) {
	a := runtime.Component(func(ctx *Ctx) Node { return el("i", bound(runtime.UseSignal(ctx, 1))) })
	b := runtime.Component(func(ctx *Ctx) Node { return el("b", bound(runtime.UseSignal(ctx, 1))) })
	armed := false
	app, err := rawStart(t, false, func(ctx *Ctx) Node {
		if armed {
			return el("div", b())
		}
		return el("div", a())
	})
	if err != nil {
		t.Fatal(err)
	}
	armed = true

	_, err = app.reset()

	if !errors.Is(err, runtime.ErrNondeterministic) {
		t.Fatalf("err = %v", err)
	}
}

func TestNondeterministicRegionStructureIsCaught(t *testing.T) {
	armed := false
	app, err := rawStart(t, false, func(ctx *Ctx) Node {
		if armed {
			return el("div")
		}
		return el("div", runtime.Show(runtime.Const(true), func() Node { return el("p") }))
	})
	if err != nil {
		t.Fatal(err)
	}
	armed = true

	if _, err = app.reset(); !errors.Is(err, runtime.ErrNondeterministic) {
		t.Fatalf("err = %v", err)
	}
}

func TestMountSelfCheckFlagsTimeNow(t *testing.T) {
	n := 0
	_, err := rawStart(t, true, func(ctx *Ctx) Node {
		n++
		return el("p", txt(fmt.Sprint("run ", n)))
	})

	var sc *runtime.SelfCheckError
	if !errors.As(err, &sc) {
		t.Fatalf("err = %v, want a SelfCheckError", err)
	}
	if !strings.Contains(sc.Error(), "byte") {
		t.Fatalf("message lacks the first differing offset: %v", sc)
	}
}

func TestMountSelfCheckFlagsMapOrder(t *testing.T) {
	flagged := 0
	for i := 0; i < 40; i++ {
		m := map[string]int{"a": 1, "b": 2, "c": 3, "d": 4, "e": 5, "f": 6, "g": 7, "h": 8}
		_, err := rawStart(t, true, func(ctx *Ctx) Node {
			var parts []Item
			for k := range m {
				parts = append(parts, el("i", txt(k)))
			}
			return el("div", parts...)
		})
		var sc *runtime.SelfCheckError
		if errors.As(err, &sc) {
			flagged++
		}
	}
	if flagged == 0 {
		t.Fatal("map iteration order was never flagged in 40 attempts")
	}
}

func TestMountSelfCheckAfterInsertedUnit(t *testing.T) {
	var sig *runtime.Signal[bool]
	runs := 0
	app, err := rawStart(t, true, func(ctx *Ctx) Node {
		sig = runtime.UseSignal(ctx, false)
		return el("div", runtime.Show(sig, func() Node {
			runs++
			return el("p", txt(fmt.Sprint("run ", runs)))
		}))
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := app.s.Ask(time.Second, func() error { sig.Set(true); return app.s.SettleIfNeeded() }); err != nil {
		t.Fatal(err)
	}

	found := false
	for _, d := range app.diags {
		found = found || strings.Contains(d.Message, "serialize output differs")
	}
	if !found {
		t.Fatalf("an inserted unit that differs from its serialization must be flagged; diagnostics: %v", app.phases())
	}
}

func TestDoubleSerializeFlagsNonDeterminismInsideThePass(t *testing.T) {
	armed := false
	toggle := false
	app, err := rawStart(t, true, func(ctx *Ctx) Node {
		if armed {
			toggle = !toggle
		}
		return el("p", txt(fmt.Sprint(toggle)))
	})
	if err != nil {
		t.Fatal(err)
	}
	armed = true

	_, err = app.reset()

	if !errors.Is(err, runtime.ErrNondeterministic) || !strings.Contains(err.Error(), "two serialize passes differ") {
		t.Fatalf("err = %v", err)
	}
}

func TestWriteDuringSerializeIsRejected(t *testing.T) {
	var sig *runtime.Signal[int]
	armed := false
	build := func(dev bool) *rawApp {
		armed = false
		app, err := rawStart(t, dev, func(ctx *Ctx) Node {
			sig = runtime.UseSignal(ctx, 1)
			if armed {
				sig.Set(99)
			}
			return el("p", bound(sig))
		})
		if err != nil {
			t.Fatal(err)
		}
		armed = true
		return app
	}

	dev := build(true)
	if _, err := dev.reset(); err == nil {
		t.Fatal("dev: a write in a serialize pass must be an error")
	}

	prod := build(false)
	html, err := prod.reset()
	if err != nil {
		t.Fatalf("prod: %v", err)
	}
	if !strings.Contains(html, ">1<") || sig.Peek() != 1 {
		t.Fatalf("prod: write was not ignored: %s / %d", html, sig.Peek())
	}
	if !strings.Contains(prod.phases(), "reactive:violation") {
		t.Fatalf("prod: diagnostics = %s", prod.phases())
	}
}

func TestCreateDuringSerializeIsRejected(t *testing.T) {
	armed := false
	app, err := rawStart(t, false, func(ctx *Ctx) Node {
		if armed {
			ctx.After(time.Hour, func() {})
		}
		return el("p")
	})
	if err != nil {
		t.Fatal(err)
	}
	armed = true

	if _, err = app.reset(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(app.phases(), "serialize:create") {
		t.Fatalf("diagnostics = %s", app.phases())
	}
}

func TestGetInBodyIsADevErrorAndProdDiagnostic(t *testing.T) {
	body := func(ctx *Ctx) Node {
		sig := runtime.UseSignal(ctx, 1)
		return el("p", txt(fmt.Sprint(sig.Get())))
	}
	devApp, err := rawStart(t, true, body)
	if err != nil {
		t.Fatal(err)
	}
	_ = devApp
	var crashed bool
	if derr := devApp.s.Ask(time.Second, func() error {
		snap, _ := devApp.s.RenderSnapshot()
		crashed = snap.Crashed
		return nil
	}); derr != nil {
		t.Fatal(derr)
	}
	if !crashed {
		t.Fatal("dev: Get() in a body must be an error")
	}

	prod, err := rawStart(t, false, body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(prod.phases(), "reactive:violation") {
		t.Fatalf("prod: diagnostics = %s", prod.phases())
	}
}

func TestDroppedStaticComponentMustStillRenderNoDynamicNodes(t *testing.T) {
	armed := false
	static := runtime.Component(func(ctx *Ctx) Node {
		if armed {
			return el("b", on("click", noopHandler))
		}
		return el("b")
	})
	app, err := rawStart(t, false, func(ctx *Ctx) Node { return el("div", static()) })
	if err != nil {
		t.Fatal(err)
	}
	if runtime.RootKidCount(app.s) != 0 {
		t.Fatal("the static component must have been dropped")
	}
	armed = true

	if _, err = app.reset(); !errors.Is(err, runtime.ErrNondeterministic) {
		t.Fatalf("a re-run of a dropped kid that now needs ids must be caught: %v", err)
	}
}
