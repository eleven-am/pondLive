package runtime_test

import (
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/shared"
)

type effectPage struct {
	app      *runtimetest.App
	boundary *runtime.Boundary
	show     *runtime.Signal[bool]
}

func newEffectPage(t *testing.T, dev bool, effect func(ctx *Ctx) Node) *effectPage {
	t.Helper()
	p := &effectPage{}
	child := runtime.Component(func(ctx *Ctx) Node { return effect(ctx) })
	p.app = runtimetest.New(t, func(ctx *Ctx) Node {
		p.boundary = runtime.UseErrorBoundary(ctx)
		p.show = runtime.UseSignal(ctx, true)
		return el("main", runtime.Show(p.show, func() Node { return child() }))
	}, runtimetest.Dev(dev))
	return p
}

func (p *effectPage) errors() []*runtime.Error {
	var out []*runtime.Error
	p.app.On(func() { out = p.boundary.Errors().Peek().All() })
	return out
}

func (p *effectPage) warnings() []*runtime.Warning {
	var out []*runtime.Warning
	p.app.On(func() { out = p.boundary.Warnings().Peek() })
	return out
}

func TestRenderEffectRunsOnceBeforeItsComponentsHTMLAndNotInSerializePasses(t *testing.T) {
	runs := 0
	value := 0
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		runtime.UseRenderEffect(ctx, func() func() { runs++; value = 42; return nil })
		return el("p", txt(fmt.Sprint(value)))
	})
	requireEqual(t, app.SSR, `<p>42</p>`)

	app.CheckReset()
	app.CheckReset()

	if runs != 1 {
		t.Fatalf("runs = %d, want 1: a serialize pass must skip render effects", runs)
	}
}

func TestRenderEffectRerunsWhenASignalItReadChangesAndCleansUpBetween(t *testing.T) {
	var sig *runtime.Signal[int]
	var log []string
	p := newEffectPage(t, true, func(ctx *Ctx) Node {
		sig = runtime.UseSignal(ctx, 1)
		runtime.UseRenderEffect(ctx, func() func() {
			v := sig.Get()
			log = append(log, fmt.Sprint("run", v))
			return func() { log = append(log, fmt.Sprint("clean", v)) }
		})
		return el("p", txt("x"))
	})
	p.app.Do(func() { sig.Set(2) })
	p.app.Do(func() { p.show.Set(false) })

	if got := strings.Join(log, ","); got != "run1,clean1,run2,clean2" {
		t.Fatalf("log = %s", got)
	}
}

func TestRenderEffectPanicReachesTheBoundary(t *testing.T) {
	p := newEffectPage(t, true, func(ctx *Ctx) Node {
		runtime.UseRenderEffect(ctx, func() func() { panic("effect boom") })
		return el("p", txt("x"))
	})
	errs := p.errors()
	if len(errs) != 1 || !strings.Contains(errs[0].Message, "effect boom") || !strings.HasPrefix(errs[0].Phase, "render-effect:run") {
		t.Fatalf("errors = %v", errs)
	}
}

type forbiddenCase struct {
	op   string
	call func(ctx *Ctx, script runtime.ScriptHandle, hub *hub)
}

type hub struct {
	shared    *shared.Shared[int]
	broadcast *shared.Broadcast[int]
	ran       atomic.Bool
}

func forbiddenCases() []forbiddenCase {
	return []forbiddenCase{
		{"Shared.Set", func(_ *Ctx, _ runtime.ScriptHandle, h *hub) { h.shared.Set(7) }},
		{"Shared.Update", func(_ *Ctx, _ runtime.ScriptHandle, h *hub) { h.shared.Update(func(int) int { return 7 }) }},
		{"Broadcast.Send", func(_ *Ctx, _ runtime.ScriptHandle, h *hub) { h.broadcast.Send(7) }},
		{runtime.PostOp, func(ctx *Ctx, _ runtime.ScriptHandle, h *hub) { runtime.RunOnActor(ctx, func() { h.ran.Store(true) }) }},
		{runtime.PostOp, func(ctx *Ctx, _ runtime.ScriptHandle, h *hub) {
			_ = runtime.SessionOf(ctx).Post(func() { h.ran.Store(true) })
		}},
		{"Script.Send", func(_ *Ctx, s runtime.ScriptHandle, _ *hub) { s.Send("evt", 1) }},
		{"ElementRef.focus", func(ctx *Ctx, _ runtime.ScriptHandle, _ *hub) { runtime.NewRef(ctx, "ElementRef", nil).Call("focus") }},
		{"ElementRef.Set value", func(ctx *Ctx, _ runtime.ScriptHandle, _ *hub) {
			runtime.NewRef(ctx, "ElementRef", nil).Set("value", "x")
		}},
		{"ElementRef.Query", func(ctx *Ctx, _ runtime.ScriptHandle, _ *hub) {
			runtime.NewRef(ctx, "ElementRef", nil).Query(func(map[string]any, error) {}, "value")
		}},
		{"ElementRef.toDataURL", func(ctx *Ctx, _ runtime.ScriptHandle, _ *hub) {
			runtime.NewRef(ctx, "ElementRef", nil).CallThen(func(any, error) {}, "toDataURL")
		}},
	}
}

func TestRenderEffectForbiddenCallsPanicInDevWithAClearMessage(t *testing.T) {
	for _, tc := range forbiddenCases() {
		t.Run(tc.op, func(t *testing.T) {
			h := &hub{shared: shared.NewShared(0), broadcast: shared.NewBroadcast[int]()}
			p := newEffectPage(t, true, func(ctx *Ctx) Node {
				script := runtime.UseScript(ctx, "1")
				runtime.UseRenderEffect(ctx, func() func() { tc.call(ctx, script, h); return nil })
				return el("p", txt("x"))
			})
			errs := p.errors()
			if len(errs) != 1 {
				t.Fatalf("errors = %v", errs)
			}
			msg := errs[0].Message
			if !strings.Contains(msg, tc.op) || !strings.Contains(msg, "UseRenderEffect") || !strings.Contains(msg, "newEffectPage") {
				t.Fatalf("message = %q", msg)
			}
			if h.shared.Get() != 0 || h.ran.Load() {
				t.Fatal("the forbidden call took effect")
			}
		})
	}
}

func TestRenderEffectForbiddenCallsAreDroppedInProdWithADiagnostic(t *testing.T) {
	for _, tc := range forbiddenCases() {
		t.Run(tc.op, func(t *testing.T) {
			h := &hub{shared: shared.NewShared(0), broadcast: shared.NewBroadcast[int]()}
			p := newEffectPage(t, false, func(ctx *Ctx) Node {
				script := runtime.UseScript(ctx, "1")
				runtime.UseRenderEffect(ctx, func() func() { tc.call(ctx, script, h); return nil })
				return el("p", txt("x"))
			})
			p.app.Settle()
			if errs := p.errors(); len(errs) != 0 {
				t.Fatalf("prod must not raise an error: %v", errs)
			}
			diags := p.app.Diagnostics("warning:RENDER_EFFECT_FORBIDDEN_CALL")
			if len(diags) != 1 || diags[0].Metadata["call"] != tc.op {
				t.Fatalf("diagnostics = %v", diags)
			}
			if h.shared.Get() != 0 || h.ran.Load() {
				t.Fatal("the dropped call took effect")
			}
		})
	}
}

func TestForbiddenCallsOutsideARenderEffectStillWork(t *testing.T) {
	h := &hub{shared: shared.NewShared(0), broadcast: shared.NewBroadcast[int]()}
	var ran atomic.Bool
	p := newEffectPage(t, true, func(ctx *Ctx) Node {
		runtime.UseEffect(ctx, func() func() {
			h.shared.Set(7)
			runtime.RunOnActor(ctx, func() { ran.Store(true) })
			return nil
		})
		return el("p", txt("x"))
	})
	p.app.Settle()
	if errs := p.errors(); len(errs) != 0 {
		t.Fatalf("errors = %v", errs)
	}
	if h.shared.Get() != 7 || !ran.Load() {
		t.Fatal("UseEffect must keep its full power")
	}
}

func TestRenderEffectStartedGoroutineIsReportedAtConnectWithComponentAndCreationStack(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	p := newEffectPage(t, true, func(ctx *Ctx) Node {
		runtime.UseRenderEffect(ctx, func() func() {
			go func() { <-block }()
			return nil
		})
		return el("p", txt("x"))
	})
	warns := p.warnings()
	if len(warns) != 1 || warns[0].Code != runtime.WarnRenderEffectLeak {
		t.Fatalf("warnings = %+v", warns)
	}
	w := warns[0]
	if !strings.Contains(w.Component, "newEffectPage") || !strings.Contains(w.Stack, "created by") || !strings.Contains(w.Stack, "rendereffect_test.go") {
		t.Fatalf("warning = %+v", w)
	}
}

func TestRenderEffectGoroutineThatFinishedIsNotReported(t *testing.T) {
	done := make(chan struct{})
	p := newEffectPage(t, true, func(ctx *Ctx) Node {
		runtime.UseRenderEffect(ctx, func() func() {
			go func() { close(done) }()
			<-done
			time.Sleep(time.Millisecond)
			return nil
		})
		return el("p", txt("x"))
	})
	if warns := p.warnings(); len(warns) != 0 {
		t.Fatalf("warnings = %+v", warns)
	}
}

func TestRenderEffectWarningsAreHeldUntilConnectAndNeverInProd(t *testing.T) {
	block := make(chan struct{})
	defer close(block)
	body := func(ctx *Ctx) Node {
		runtime.UseRenderEffect(ctx, func() func() {
			go func() { <-block }()
			time.Sleep(2 * runtime.SlowRenderEffect)
			return nil
		})
		return el("p", txt("x"))
	}
	var boundary *runtime.Boundary
	root := func(ctx *Ctx) Node {
		boundary = runtime.UseErrorBoundary(ctx)
		return el("main", runtime.Component(body)())
	}

	dev := rawStartDeferred(t, true, root)
	if n := dev.warnCount(boundary); n != 0 {
		t.Fatalf("%d warnings visible before connect", n)
	}
	dev.goLive()
	if n := dev.warnCount(boundary); n != 2 {
		t.Fatalf("%d warnings after connect, want slow + leak", n)
	}

	boundary = nil
	prod := rawStartDeferred(t, false, root)
	prod.goLive()
	if n := prod.warnCount(boundary); n != 0 {
		t.Fatalf("prod raised %d warnings", n)
	}
}

func TestRenderEffectSlowWarningHasThresholdAndNamesTheComponent(t *testing.T) {
	var slow bool
	p := newEffectPage(t, true, func(ctx *Ctx) Node {
		runtime.UseRenderEffect(ctx, func() func() {
			if slow {
				time.Sleep(3 * runtime.SlowRenderEffect)
			}
			return nil
		})
		return el("p", txt("x"))
	})
	if warns := p.warnings(); len(warns) != 0 {
		t.Fatalf("a fast effect warned: %+v", warns)
	}
	slow = true
	p.app.Do(func() { p.show.Set(false) })
	p.app.Do(func() { p.show.Set(true) })
	warns := p.warnings()
	if len(warns) != 1 || warns[0].Code != runtime.WarnRenderEffectSlow || !strings.Contains(warns[0].Component, "newEffectPage") || !strings.Contains(warns[0].Message, "5ms") {
		t.Fatalf("warnings = %+v", warns)
	}
}

func TestWarningsCanBeDismissedOneAtATime(t *testing.T) {
	var slow bool
	p := newEffectPage(t, true, func(ctx *Ctx) Node {
		runtime.UseRenderEffect(ctx, func() func() {
			if slow {
				time.Sleep(2 * runtime.SlowRenderEffect)
			}
			return nil
		})
		return el("p", txt("x"))
	})
	slow = true
	p.app.Do(func() { p.show.Set(false) })
	p.app.Do(func() { p.show.Set(true) })
	p.app.Do(func() { p.show.Set(false) })
	p.app.Do(func() { p.show.Set(true) })
	warns := p.warnings()
	if len(warns) != 2 {
		t.Fatalf("warnings = %d", len(warns))
	}
	p.app.Do(func() { p.boundary.Dismiss(warns[0].ID) })
	if left := p.warnings(); len(left) != 1 || left[0].ID != warns[1].ID {
		t.Fatalf("left = %+v", left)
	}
}

func TestRenderEffectWritingEarlierEmittedUntrackedStateIsCaughtByTheDevSelfCheck(t *testing.T) {
	stale := 0
	reader := runtime.Component(func(ctx *Ctx) Node { return el("i", txt(fmt.Sprint(stale))) })
	writer := runtime.Component(func(ctx *Ctx) Node {
		runtime.UseRenderEffect(ctx, func() func() { stale = 9; return nil })
		return el("b", txt("w"))
	})
	root := func(ctx *Ctx) Node { return el("div", reader(), writer()) }

	_, err := rawStart(t, true, root)
	var sc *runtime.SelfCheckError
	if !asSelfCheck(err, &sc) {
		t.Fatalf("dev render error = %v, want the mount self-check", err)
	}

	stale = 0
	if _, err := rawStart(t, false, root); err != nil {
		t.Fatalf("prod has no self-check: %v", err)
	}
}

func TestRenderEffectWritingATrackedSignalAlreadyEmittedStillRendersTheFinalValue(t *testing.T) {
	var title *runtime.Signal[string]
	reader := runtime.Component(func(ctx *Ctx) Node {
		title = runtime.UseSignal(ctx, "before")
		return el("i", bound(title))
	})
	writer := runtime.Component(func(ctx *Ctx) Node {
		runtime.UseRenderEffect(ctx, func() func() { title.Set("after"); return nil })
		return el("b", txt("w"))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node { return el("div", reader(), writer()) })

	if !strings.Contains(app.SSR, ">after<") {
		t.Fatalf("ssr = %s", app.SSR)
	}
}

func TestRenderEffectCleanupRunsOnUnmountOnly(t *testing.T) {
	cleaned := 0
	p := newEffectPage(t, true, func(ctx *Ctx) Node {
		runtime.UseRenderEffect(ctx, func() func() { return func() { cleaned++ } })
		return el("p", txt("x"))
	})
	p.app.CheckReset()
	if cleaned != 0 {
		t.Fatalf("cleanup ran during a reset")
	}
	p.app.Do(func() { p.show.Set(false) })
	if cleaned != 1 {
		t.Fatalf("cleaned = %d", cleaned)
	}
}

func asSelfCheck(err error, target **runtime.SelfCheckError) bool { return errors.As(err, target) }

type deferredApp struct {
	t testing.TB
	s *runtime.Session
}

func rawStartDeferred(t testing.TB, dev bool, root func(*Ctx) Node) *deferredApp {
	t.Helper()
	d := &deferredApp{t: t, s: runtime.NewSession(runtime.Config{SessionID: "deferred", Root: root, DevMode: dev})}
	d.s.DeferEffects()
	runtimetest.Start(t, d.s)
	d.ask(func() {
		if _, err := d.s.RenderSnapshot(); err != nil {
			t.Fatalf("render: %v", err)
		}
	})
	return d
}

func (d *deferredApp) ask(fn func()) {
	d.t.Helper()
	if err := d.s.Ask(time.Second, func() error { fn(); return nil }); err != nil {
		d.t.Fatal(err)
	}
}

func (d *deferredApp) goLive() {
	d.t.Helper()
	d.ask(d.s.GoLive)
}

func (d *deferredApp) warnCount(b *runtime.Boundary) (n int) {
	d.t.Helper()
	d.ask(func() { n = len(b.Warnings().Peek()) })
	return n
}
