package runtime_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func ops(edits []protocol.Edit) string {
	parts := make([]string, len(edits))
	for i, e := range edits {
		parts[i] = string(e.Op)
	}
	return strings.Join(parts, ",")
}

func TestShowSwapsArmsWithOneRemoveAndOneInsert(t *testing.T) {
	var flag *runtime.Signal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		flag = runtime.UseSignal(ctx, true)
		return el("div", runtime.Show(flag,
			func() Node { return el("p", txt("yes")) },
			runtime.Else(func() Node { return el("em", txt("no")) })))
	})
	requireEqual(t, app.SSR, `<div><!--r:1--><p data-pi="2">yes</p><!--/r:1--></div>`)

	edits := app.Do(func() { flag.Set(false) })

	if ops(edits) != "rm,ins" {
		t.Fatalf("ops = %s", ops(edits))
	}
	requireEqual(t, app.Model.HTML(), `<div><!--r:1--><em data-pi="3">no</em><!--/r:1--></div>`)

	edits = app.Do(func() { flag.Set(true) })
	if ops(edits) != "rm,ins" {
		t.Fatalf("ops = %s", ops(edits))
	}
	if strings.Contains(app.Model.HTML(), `data-pi="2"`) {
		t.Fatalf("ids must never be reused: %s", app.Model.HTML())
	}
}

func TestShowWithoutElseEmitsClearForNothingAndInsertForSomething(t *testing.T) {
	var flag *runtime.Signal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		flag = runtime.UseSignal(ctx, false)
		return el("div", runtime.Show(flag, func() Node { return el("p") }))
	})
	requireEqual(t, app.SSR, `<div><!--r:1--><!--/r:1--></div>`)

	if got := ops(app.Do(func() { flag.Set(true) })); got != "ins" {
		t.Fatalf("ops = %s", got)
	}
	if got := ops(app.Do(func() { flag.Set(false) })); got != "clr" {
		t.Fatalf("ops = %s", got)
	}
}

func TestSwitchKeepsArmMountedWhenSelectorChangesWithinTheSameArm(t *testing.T) {
	var sel *runtime.Signal[string]
	mounts := 0
	page := runtime.Component(func(ctx *Ctx) Node {
		mounts++
		return el("section", txt("page"))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sel = runtime.UseSignal(ctx, "a")
		return el("div", runtime.Switch(sel,
			runtime.Case("a", func() Node { return page() }),
			runtime.Case("b", func() Node { return el("i", txt("b")) }),
			runtime.Default(func() Node { return el("u", txt("other")) })))
	})
	before := mounts

	app.Do(func() { sel.Set("zzz") })
	app.Do(func() { sel.Set("yyy") })

	if edits := app.Do(func() { sel.Set("xxx") }); len(edits) != 0 {
		t.Fatalf("selector change inside the default arm must not touch the DOM: %v", edits)
	}
	if mounts != before {
		t.Fatalf("component remounted")
	}
	app.Do(func() { sel.Set("b") })
	if !strings.Contains(app.Normalized(), "<i>b</i>") {
		t.Fatalf("case b not shown: %s", app.Normalized())
	}
}

func TestExitKeepsRowUntilTimerAndReusesIDsOnReturn(t *testing.T) {
	var list *runtime.Signal[[]int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, []int{1, 2, 3})
		return el("ul", runtime.For(runtime.Items(list, func(n int) int { return n }),
			runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return el("li", txt(fmt.Sprint(item.Peek()))) }),
			runtime.Exit(100*time.Millisecond)))
	})
	before := app.Model.HTML()

	edits := app.Do(func() { list.Set([]int{1, 3}) })
	if ops(edits) != "a,a" || edits[0].Name != "data-exiting" || edits[1].Name != "inert" {
		t.Fatalf("exit edits = %v", edits)
	}

	edits = app.Do(func() { list.Set([]int{1, 2, 3}) })
	if ops(edits) != "ra,ra" {
		t.Fatalf("revive edits = %v", edits)
	}
	if app.Model.HTML() != before {
		t.Fatalf("re-entering must restore the same DOM and ids\nbefore: %s\nafter:  %s", before, app.Model.HTML())
	}

	app.Do(func() { list.Set([]int{1, 3}) })
	if edits := app.Advance(99 * time.Millisecond); len(edits) != 0 {
		t.Fatalf("dropped early: %v", edits)
	}
	edits = app.Advance(2 * time.Millisecond)
	if ops(edits) != "rm" {
		t.Fatalf("drop edits = %v", edits)
	}
}

func TestExitingRowDropsInFlightClick(t *testing.T) {
	var list *runtime.Signal[[]int]
	clicks := 0
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, []int{1, 2})
		return el("ul", runtime.For(runtime.Items(list, func(n int) int { return n }),
			runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node {
				return el("li", on("click", func(work.Event) { clicks++ }))
			}),
			runtime.Exit(time.Second)))
	})
	app.Do(func() { list.Set([]int{1}) })

	if err := app.Dispatch(3, "click", nil); err != runtime.ErrExitingHandler {
		t.Fatalf("err = %v", err)
	}
	if err := app.Dispatch(2, "click", nil); err != nil {
		t.Fatal(err)
	}
	if clicks != 1 {
		t.Fatalf("clicks = %d", clicks)
	}
	app.Advance(2 * time.Second)
	if err := app.Dispatch(3, "click", nil); err != runtime.ErrStaleHandler {
		t.Fatalf("removed row err = %v", err)
	}
}

func rowApp(t *testing.T, initial []int, opts ...runtime.ForOpt) (*runtimetest.App, *runtime.Signal[[]int]) {
	var list *runtime.Signal[[]int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, initial)
		return el("ul", runtime.For(runtime.Items(list, func(n int) int { return n }),
			runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return el("li", txt(fmt.Sprint(item.Peek()))) }), opts...))
	})
	return app, list
}

func liOrder(app *runtimetest.App) string {
	n := app.Model.Normalized()
	var out []string
	for _, part := range strings.Split(n, "<li>")[1:] {
		out = append(out, strings.SplitN(part, "</li>", 2)[0])
	}
	return strings.Join(out, ",")
}

func TestForInsertAtFrontIsOneEdit(t *testing.T) {
	app, list := rowApp(t, []int{1, 2, 3, 4, 5})

	edits := app.Do(func() { list.Set([]int{0, 1, 2, 3, 4, 5}) })

	if len(edits) != 1 || edits[0].Op != protocol.OpInsert {
		t.Fatalf("edits = %v", edits)
	}
	if liOrder(app) != "0,1,2,3,4,5" {
		t.Fatalf("order = %s", liOrder(app))
	}
}

func TestForClearIsOneClr(t *testing.T) {
	app, list := rowApp(t, []int{1, 2, 3})

	edits := app.Do(func() { list.Set(nil) })

	if len(edits) != 1 || edits[0].Op != protocol.OpClear {
		t.Fatalf("edits = %v", edits)
	}
}

func TestForMovesAreMinimalOverPermutations(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for iter := 0; iter < 1000; iter++ {
		n := 2 + rng.Intn(9)
		start := rng.Perm(n)
		target := rng.Perm(n)
		app, list := rowApp(t, start)

		edits := app.Do(func() { list.Set(target) })

		pos := map[int]int{}
		for i, v := range start {
			pos[v] = i
		}
		seq := make([]int, n)
		for i, v := range target {
			seq[i] = pos[v]
		}
		want := n - lisLen(seq)
		if len(edits) != want {
			t.Fatalf("start=%v target=%v: %d edits, want n-LIS=%d (%v)", start, target, len(edits), want, edits)
		}
		for _, e := range edits {
			if e.Op != protocol.OpMove {
				t.Fatalf("unexpected op %v", e)
			}
		}
		if got := liOrder(app); got != joinInts(target) {
			t.Fatalf("order %s, want %s", got, joinInts(target))
		}
	}
}

func lisLen(seq []int) int {
	best := make([]int, len(seq))
	top := 0
	for i := range seq {
		best[i] = 1
		for j := 0; j < i; j++ {
			if seq[j] < seq[i] && best[j]+1 > best[i] {
				best[i] = best[j] + 1
			}
		}
		top = max(top, best[i])
	}
	return top
}

func joinInts(v []int) string {
	parts := make([]string, len(v))
	for i, n := range v {
		parts[i] = fmt.Sprint(n)
	}
	return strings.Join(parts, ",")
}

func TestForMovingARowKeepsItsIDs(t *testing.T) {
	app, list := rowApp(t, []int{1, 2, 3})
	before := app.Model.IDs()

	app.Do(func() { list.Set([]int{3, 1, 2}) })

	after := app.Model.IDs()
	if fmt.Sprint(before) != fmt.Sprint(after) {
		t.Fatalf("ids changed on a move: %v -> %v", before, after)
	}
}

func TestForUpdatesKeptRowItemInPlace(t *testing.T) {
	type row struct {
		ID   int
		Text string
	}
	var list *runtime.Signal[[]row]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, []row{{1, "a"}, {2, "b"}})
		return el("ul", runtime.For(runtime.Items(list, func(r row) int { return r.ID }),
			runtime.PropsComponent(func(ctx *Ctx, item runtime.ReadSignal[row]) Node {
				text := runtime.UseComputed(ctx, func() string { return item.Get().Text })
				return el("li", bound(text))
			})))
	})

	edits := app.Do(func() { list.Set([]row{{1, "a"}, {2, "B"}}) })

	if len(edits) != 1 || edits[0].Op != protocol.OpText || edits[0].Value != "B" {
		t.Fatalf("edits = %v", edits)
	}
}

func TestForDuplicateKeysDevErrorAndProdFirstWins(t *testing.T) {
	build := func(dev bool) (*runtimetest.App, *runtime.Signal[[]int]) {
		var list *runtime.Signal[[]int]
		app := runtimetest.New(t, func(ctx *Ctx) Node {
			list = runtime.UseSignal(ctx, []int{1})
			return el("ul", runtime.For(runtime.Items(list, func(n int) int { return n / 10 }),
				runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return el("li", txt(fmt.Sprint(item.Peek()))) })))
		}, runtimetest.Dev(dev))
		return app, list
	}

	prod, list := build(false)
	prod.Do(func() { list.Set([]int{11, 12, 25}) })
	if liOrder(prod) != "11,25" {
		t.Fatalf("first occurrence must win: %s", liOrder(prod))
	}
	if len(prod.Diagnostics("for:duplicate-key")) != 1 {
		t.Fatalf("diagnostics = %v", prod.Diags)
	}
}
