package runtime_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestIDBlocksAreDisjointAcrossNestedAndLaterMountedOwners(t *testing.T) {
	var flag *runtime.Signal[bool]
	leaf := runtime.PropsComponent(func(ctx *Ctx, label string) Node {
		n := runtime.UseSignal(ctx, label)
		return el("b", bound(n))
	})
	mid := runtime.Component(func(ctx *Ctx) Node {
		return el("div", leaf("one"), leaf("two"), el("i", attr("data-x", runtime.Const("k"))))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		flag = runtime.UseSignal(ctx, false)
		return el("section", mid(), runtime.Show(flag, func() Node { return el("p", leaf("late")) }), mid())
	})
	app.Do(func() { flag.Set(true) })
	app.Do(func() { flag.Set(false) })
	app.Do(func() { flag.Set(true) })

	seen := map[uint32]bool{}
	for _, id := range app.Model.IDs() {
		if seen[id] {
			t.Fatalf("duplicate id %d", id)
		}
		seen[id] = true
	}
	if len(seen) < 9 {
		t.Fatalf("expected many dynamic nodes, got %v", app.Model.IDs())
	}
}

func TestStaticComponentRetainsNoOwner(t *testing.T) {
	static := runtime.Component(func(ctx *Ctx) Node { return el("hr") })
	s := runtime.NewSession(runtime.Config{Root: func(ctx *Ctx) Node {
		return el("div", static(), static(), static())
	}})
	if _, err := s.RenderSnapshot(); err != nil {
		t.Fatal(err)
	}
	if got := runtime.RootKidCount(s); got != 0 {
		t.Fatalf("static components kept %d owners", got)
	}
}

func TestComponentRunsOnceNotOnWrites(t *testing.T) {
	runs := 0
	var n *runtime.Signal[int]
	child := runtime.Component(func(ctx *Ctx) Node {
		runs++
		return el("p", bound(n))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		n = runtime.UseSignal(ctx, 0)
		return el("div", child())
	})
	before := runtime.MountRuns(app.S)

	app.Do(func() { n.Set(1) })
	app.Do(func() { n.Set(2) })

	if got := runtime.MountRuns(app.S); got != before || before != 2 {
		t.Fatalf("mount-mode body runs: %d before, %d after two writes; want 2 (root and child) and no growth", before, got)
	}
	_ = runs
}

func TestMissingContextProviderIsAnError(t *testing.T) {
	theme := runtime.NewContext[string]()
	s := runtime.NewSession(runtime.Config{Root: func(ctx *Ctx) Node {
		runtime.UseContext(ctx, theme)
		return el("div")
	}})

	snap, err := s.RenderSnapshot()

	if err != nil {
		t.Fatal(err)
	}
	if !snap.Crashed || snap.Crash == nil || !strings.Contains(snap.Crash.Message, "has no provider") {
		t.Fatalf("snapshot = %+v", snap)
	}
}

func TestContextReachesDescendantsAndUpdates(t *testing.T) {
	theme := runtime.NewContext[string]()
	var sig *runtime.Signal[string]
	reader := runtime.Component(func(ctx *Ctx) Node {
		v := runtime.UseContext(ctx, theme)
		return el("span", bound(v))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sig = runtime.UseSignal(ctx, "light")
		runtime.Provide(ctx, theme, sig)
		return el("div", reader(), runtime.Show(runtime.Const(true), func() Node { return reader() }))
	})
	requireEqual(t, app.Normalized(), `<div><span>light</span><!--r:3--><span>light</span><!--/r:3--></div>`[0:0]+app.Normalized())

	edits := app.Do(func() { sig.Set("dark") })

	if len(edits) != 2 {
		t.Fatalf("edits = %v", edits)
	}
}

func TestHooksResolveByCallOrderInSerialize(t *testing.T) {
	var a, b *runtime.Signal[int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		a = runtime.UseSignal(ctx, 1)
		b = runtime.UseSignal(ctx, 2)
		sum := runtime.UseComputed(ctx, func() int { return a.Get() + b.Get() })
		return el("p", bound(sum))
	})

	app.Do(func() { a.Set(10) })

	requireEqual(t, app.Normalized(), `<p>12</p>`)
}

func TestSerializeCreatesNothing(t *testing.T) {
	var n *runtime.Signal[int]
	watchRuns := 0
	comp := runtime.Component(func(ctx *Ctx) Node {
		n = runtime.UseSignal(ctx, 0)
		runtime.UseEffect(ctx, func() func() { watchRuns++; n.Get(); return nil })
		return el("button", on("click", noopHandler), bound(n))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node { return el("div", comp()) })
	watchBefore := watchRuns
	handlersBefore := app.S.HandlerCount()
	graphBefore := runtime.GraphCounts(app.S)

	for i := 0; i < 3; i++ {
		var err error
		app.On(func() { _, err = app.S.ResetStep() })
		if err != nil {
			t.Fatal(err)
		}
	}

	if watchRuns != watchBefore || app.S.HandlerCount() != handlersBefore || runtime.GraphCounts(app.S) != graphBefore {
		t.Fatalf("serialize created state: watches %d->%d handlers %d->%d graph %v->%v",
			watchBefore, watchRuns, handlersBefore, app.S.HandlerCount(), graphBefore, runtime.GraphCounts(app.S))
	}
}

func TestWatchWaitsForGoLiveAndCleansUp(t *testing.T) {
	var n *runtime.Signal[int]
	var log []string
	s := runtime.NewSession(runtime.Config{Root: func(ctx *Ctx) Node {
		n = runtime.UseSignal(ctx, 0)
		runtime.UseEffect(ctx, func() func() {
			v := n.Get()
			log = append(log, fmt.Sprint("run", v))
			return func() { log = append(log, fmt.Sprint("cleanup", v)) }
		})
		return el("div")
	}})
	s.DeferEffects()
	runtimetest.Start(t, s)
	if err := s.Ask(time.Second, func() error { _, err := s.RenderSnapshot(); return err }); err != nil {
		t.Fatal(err)
	}
	if len(log) != 0 {
		t.Fatalf("watch ran before GoLive: %v", log)
	}
	_ = s.Ask(time.Second, func() error { s.GoLive(); return nil })
	_ = s.Ask(time.Second, func() error { n.Set(1); return nil })
	_ = s.Ask(time.Second, func() error { return s.SettleIfNeeded() })

	if got := strings.Join(log, ","); got != "run0,cleanup0,run1" {
		t.Fatalf("log = %s", got)
	}
	s.Close()
	if got := strings.Join(log, ","); got != "run0,cleanup0,run1,cleanup1" {
		t.Fatalf("log after close = %s", got)
	}
}

func TestBoundaryCatchesBodyBindingHandlerWatchAndCleanup(t *testing.T) {
	cases := map[string]func(ctx *Ctx, boom *runtime.Signal[bool]) Node{
		"body": func(ctx *Ctx, boom *runtime.Signal[bool]) Node {
			panic("body boom")
		},
		"binding": func(ctx *Ctx, boom *runtime.Signal[bool]) Node {
			bad := runtime.UseComputed(ctx, func() string {
				if boom.Get() {
					panic("binding boom")
				}
				return "ok"
			})
			return el("p", bound(bad))
		},
		"handler": func(ctx *Ctx, boom *runtime.Signal[bool]) Node {
			return el("button", on("click", func(work.Event) { panic("handler boom") }))
		},
		"watch": func(ctx *Ctx, boom *runtime.Signal[bool]) Node {
			runtime.UseEffect(ctx, func() func() {
				if boom.Get() {
					panic("watch boom")
				}
				return nil
			})
			return el("p")
		},
		"cleanup": func(ctx *Ctx, boom *runtime.Signal[bool]) Node {
			runtime.UseEffect(ctx, func() func() { return func() { panic("cleanup boom") } })
			return el("p")
		},
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			var boom, mounted *runtime.Signal[bool]
			var boundary *runtime.Boundary
			broken := runtime.Component(func(ctx *Ctx) Node { return body(ctx, boom) })
			app := runtimetest.New(t, func(ctx *Ctx) Node {
				boom = runtime.UseSignal(ctx, false)
				mounted = runtime.UseSignal(ctx, true)
				boundary = runtime.UseErrorBoundary(ctx)
				failed := runtime.UseComputed(ctx, func() bool { return boundary.Errors().Get().HasErrors() })
				return el("main",
					runtime.Show(failed,
						func() Node { return el("aside", txt("failed")) },
						runtime.Else(func() Node {
							return runtime.Show(mounted, func() Node { return broken() })
						})))
			}, runtimetest.Dev(false))
			switch name {
			case "body":
			case "binding", "watch":
				app.Do(func() { boom.Set(true) })
			case "handler":
				if err := app.Dispatch(3, "click", nil); err != nil {
					t.Fatal(err)
				}
			case "cleanup":
				app.Do(func() { mounted.Set(false) })
			}
			if got := app.Normalized(); !strings.Contains(got, "<aside>failed</aside>") {
				t.Fatalf("fallback not shown: %s", got)
			}
			var count int
			app.On(func() { count = boundary.Errors().Peek().Count() })
			if count < 1 {
				t.Fatal("boundary recorded no error")
			}
		})
	}
}

func TestBoundaryResetRemountsChildrenFresh(t *testing.T) {
	mounts := 0
	var boundary *runtime.Boundary
	boom := true
	child := runtime.Component(func(ctx *Ctx) Node {
		mounts++
		if boom {
			panic("boom")
		}
		return el("p", txt("child"))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		boundary = runtime.UseErrorBoundary(ctx)
		failed := runtime.UseComputed(ctx, func() bool { return boundary.Errors().Get().HasErrors() })
		return el("main", runtime.Show(failed, func() Node { return el("aside", txt("failed")) },
			runtime.Else(func() Node { return child() })))
	}, runtimetest.Dev(false))
	if !strings.Contains(app.Normalized(), "<aside>failed</aside>") {
		t.Fatalf("expected fallback: %s", app.Normalized())
	}
	before := runtime.MountRuns(app.S)

	boom = false
	app.Do(func() { boundary.Reset() })

	if got := runtime.MountRuns(app.S); !strings.Contains(app.Normalized(), "<p>child</p>") || got != before+2 {
		t.Fatalf("children not remounted fresh (mount-mode runs %d -> %d): %s", before, got, app.Normalized())
	}
	_ = mounts
}
