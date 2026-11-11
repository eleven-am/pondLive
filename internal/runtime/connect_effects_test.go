package runtime_test

import (
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/shared"
)

type clockedApp struct {
	*deferredApp
	clock *runtimetest.FakeClock
}

func startClocked(t testing.TB, dev bool, root func(*Ctx) Node) *clockedApp {
	t.Helper()
	clock := runtimetest.NewFakeClock()
	s := runtime.NewSession(runtime.Config{SessionID: "clocked", Root: root, DevMode: dev, Clock: clock})
	s.DeferEffects()
	runtimetest.Start(t, s)
	d := &deferredApp{t: t, s: s}
	d.ask(func() {
		if _, err := s.RenderSnapshot(); err != nil {
			t.Fatalf("render: %v", err)
		}
	})
	return &clockedApp{deferredApp: d, clock: clock}
}

func TestEffectsAreQueuedDuringSSRAndRunExactlyOnceAtConnectParentsFirstInMountOrder(t *testing.T) {
	var log []string
	rec := func(name string) func() func() {
		return func() func() { log = append(log, name); return nil }
	}
	grandchild := runtime.Component(func(ctx *Ctx) Node {
		runtime.UseEffect(ctx, rec("a1"))
		return el("i", txt("a1"))
	})
	a := runtime.Component(func(ctx *Ctx) Node {
		runtime.UseEffect(ctx, rec("a"))
		return el("b", grandchild())
	})
	b := runtime.Component(func(ctx *Ctx) Node {
		runtime.UseEffect(ctx, rec("b"))
		return el("u", txt("b"))
	})
	c := runtime.Component(func(ctx *Ctx) Node {
		runtime.UseEffect(ctx, rec("c"))
		return el("s", txt("c"))
	})
	d := rawStartDeferred(t, true, func(*Ctx) Node { return el("main", a(), b(), c()) })

	var before []string
	d.ask(func() { before = append(before, log...) })
	if len(before) != 0 {
		t.Fatalf("effects ran before connect: %v", before)
	}

	d.goLive()
	d.goLive()
	d.ask(func() {})

	if got := strings.Join(log, ","); got != "a,b,c,a1" {
		t.Fatalf("effect order = %s", got)
	}
}

func TestEffectsOfComponentsMountedAfterConnectRunAsToday(t *testing.T) {
	var runs atomic.Int32
	var show *runtime.Signal[bool]
	late := runtime.Component(func(ctx *Ctx) Node {
		runtime.UseEffect(ctx, func() func() { runs.Add(1); return nil })
		return el("p", txt("late"))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		show = runtime.UseSignal(ctx, false)
		return el("main", runtime.Show(show, func() Node { return late() }))
	})
	if runs.Load() != 0 {
		t.Fatal("effect ran for an unmounted component")
	}
	app.Do(func() { show.Set(true) })
	if runs.Load() != 1 {
		t.Fatalf("runs = %d", runs.Load())
	}
}

func TestATickingEffectDoesNotTickBeforeConnect(t *testing.T) {
	var ticks atomic.Int32
	root := func(ctx *Ctx) Node {
		runtime.UseEffect(ctx, func() func() {
			var tick func()
			tick = func() {
				ticks.Add(1)
				ctx.After(time.Second, tick)
			}
			timer := ctx.After(time.Second, tick)
			return func() { timer.Stop() }
		})
		return el("p", txt("clock"))
	}
	a := startClocked(t, true, root)

	a.ask(func() { _ = a.s.SettleIfNeeded() })
	a.clock.Advance(time.Hour)
	a.ask(func() {})
	if ticks.Load() != 0 {
		t.Fatalf("%d ticks before connect", ticks.Load())
	}

	a.goLive()
	for i := 0; i < 3; i++ {
		a.clock.Advance(time.Second)
		a.ask(func() {})
	}
	if ticks.Load() != 3 {
		t.Fatalf("ticks after connect = %d", ticks.Load())
	}
}

func TestASessionThatNeverConnectsDiscardsItsEffectQueueOnClose(t *testing.T) {
	var runs atomic.Int32
	var cleaned atomic.Int32
	d := rawStartDeferred(t, true, func(ctx *Ctx) Node {
		runtime.UseEffect(ctx, func() func() { runs.Add(1); return func() { cleaned.Add(1) } })
		return el("p", txt("x"))
	})
	if err := d.s.RequestClose(); err != nil {
		t.Fatal(err)
	}
	<-d.s.Done()
	if runs.Load() != 0 || cleaned.Load() != 0 {
		t.Fatalf("runs=%d cleaned=%d for a session that never connected", runs.Load(), cleaned.Load())
	}
}

func TestSharedMirrorIsCurrentAtConnectAndBroadcastWaitsForConnect(t *testing.T) {
	hub := shared.NewShared(1)
	bc := shared.NewBroadcast[int]()
	var got atomic.Int32
	root := func(ctx *Ctx) Node {
		v := runtime.UseShared(ctx, hub)
		runtime.UseBroadcast(ctx, bc, func(n int) { got.Add(int32(n)) })
		return el("p", bound(v))
	}
	a := startClocked(t, true, root)

	hub.Set(5)
	bc.Send(10)
	a.ask(func() {})
	if got.Load() != 0 {
		t.Fatalf("broadcast handler ran before connect: %d", got.Load())
	}

	a.goLive()
	a.ask(func() {})
	if got.Load() != 10 {
		t.Fatalf("broadcast handler total after connect = %d, want 10", got.Load())
	}
	var html string
	a.ask(func() {
		html, _ = a.s.DocumentHTML()
	})
	if !strings.Contains(html, ">5<") {
		t.Fatalf("shared mirror stale after connect: %s", html)
	}
	bc.Send(1)
	a.ask(func() {})
	a.ask(func() {})
	if got.Load() != 11 {
		t.Fatalf("live broadcast after connect = %d", got.Load())
	}
}
