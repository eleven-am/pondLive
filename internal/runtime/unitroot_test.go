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

func fragRow(n int) Node {
	return frag(el("dt", txt(fmt.Sprint("k", n))), el("dd", txt(fmt.Sprint("v", n))))
}

func TestForRowMayBeAFragmentAndKeepsMovesInsertsAndRemoves(t *testing.T) {
	var list *runtime.Signal[[]int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, []int{1, 2, 3})
		return el("dl", runtime.For(runtime.Items(list, func(n int) int { return n }),
			runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return fragRow(item.Peek()) })))
	})
	if strings.Count(app.SSR, "<!--r:") != 4 {
		t.Fatalf("each fragment row needs its own region: %s", app.SSR)
	}

	if got := ops(app.Do(func() { list.Set([]int{3, 1, 2}) })); got != "mv" {
		t.Fatalf("move ops = %s", got)
	}
	if got := ops(app.Do(func() { list.Set([]int{3, 4, 1, 2}) })); got != "ins" {
		t.Fatalf("insert ops = %s", got)
	}
	if got := ops(app.Do(func() { list.Set([]int{3, 1}) })); got != "rm,rm" {
		t.Fatalf("remove ops = %s", got)
	}
	want := `<dl><!--r:1--><!--r:`
	if !strings.HasPrefix(app.Model.HTML(), want) {
		t.Fatalf("html = %s", app.Model.HTML())
	}
}

func TestForFragmentRowsKeepTheirIDsOnMove(t *testing.T) {
	var list *runtime.Signal[[]int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, []int{1, 2, 3})
		return el("dl", runtime.For(runtime.Items(list, func(n int) int { return n }),
			runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node {
				return frag(el("dt", on("click", noopHandler), txt("k")), el("dd", txt(fmt.Sprint(item.Peek()))))
			})))
	})
	before := fmt.Sprint(app.Model.IDs())
	app.Do(func() { list.Set([]int{2, 3, 1}) })
	if after := fmt.Sprint(app.Model.IDs()); after != before {
		t.Fatalf("ids changed on a move: %s -> %s", before, after)
	}
}

func TestSingleElementRowsStayRegionFree(t *testing.T) {
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list := runtime.UseSignal(ctx, []int{1, 2})
		return el("ul", runtime.For(runtime.Items(list, func(n int) int { return n }),
			runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return frag(el("li", txt(fmt.Sprint(item.Peek())))) })))
	})
	if strings.Count(app.SSR, "<!--r:") != 1 {
		t.Fatalf("a single-element row must not get a region: %s", app.SSR)
	}
}

func TestShowAndSwitchArmsMayBeFragments(t *testing.T) {
	var flag *runtime.Signal[bool]
	var sel *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		flag = runtime.UseSignal(ctx, true)
		sel = runtime.UseSignal(ctx, "a")
		return el("div",
			runtime.Show(flag,
				func() Node { return frag(el("p", txt("one")), el("p", txt("two"))) },
				runtime.Else(func() Node { return frag(txt("plain "), txt("text")) })),
			runtime.Switch(sel,
				runtime.Case("a", func() Node { return frag(el("i", txt("a1")), el("i", txt("a2"))) }),
				runtime.Default(func() Node { return frag() })))
	})
	if got := ops(app.Do(func() { flag.Set(false) })); got != "rm,ins" {
		t.Fatalf("show ops = %s", got)
	}
	if got := ops(app.Do(func() { sel.Set("z") })); got != "rm,ins" {
		t.Fatalf("switch ops = %s", got)
	}
	if got := ops(app.Do(func() { flag.Set(true); sel.Set("a") })); got != "rm,ins,rm,ins" {
		t.Fatalf("both ops = %s", got)
	}
}

func TestFragmentRowExitsAfterItsDuration(t *testing.T) {
	var list *runtime.Signal[[]int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, []int{1, 2})
		return el("dl", runtime.For(runtime.Items(list, func(n int) int { return n }),
			runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return fragRow(item.Peek()) }),
			runtime.Exit(100*time.Millisecond)))
	})
	app.Do(func() { list.Set([]int{2}) })
	if edits := app.Advance(99 * time.Millisecond); len(edits) != 0 {
		t.Fatalf("dropped early: %v", edits)
	}
	if got := ops(app.Advance(2 * time.Millisecond)); got != "rm" {
		t.Fatalf("drop ops = %s", got)
	}
	app.Do(func() { list.Set([]int{1, 2}) })
}

func TestShowExitKeepsTheArmMarkedUntilTheDurationEnds(t *testing.T) {
	var on *runtime.Signal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		on = runtime.UseSignal(ctx, true)
		return el("main", runtime.Show(on, func() Node { return el("p", txt("arm")) },
			runtime.Else(func() Node { return el("i", txt("else")) }), runtime.Exit(100*time.Millisecond)))
	})
	app.Do(func() { on.Set(false) })
	html := app.Model.HTML()
	if !strings.Contains(html, `<p data-exiting="" data-pi="`) || !strings.Contains(html, "else") {
		t.Fatalf("the leaving arm must stay marked while the else arm mounts: %s", html)
	}
	app.Advance(101 * time.Millisecond)
	if html := app.Model.HTML(); strings.Contains(html, "arm") || strings.Contains(html, "data-exiting") {
		t.Fatalf("the arm outlived its exit: %s", html)
	}
}

func TestExitingFragmentRowMarksEveryTopLevelElement(t *testing.T) {
	var list *runtime.Signal[[]int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, []int{1, 2})
		return el("dl", runtime.For(runtime.Items(list, func(n int) int { return n }),
			runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return fragRow(item.Peek()) }),
			runtime.Exit(100*time.Millisecond)))
	})
	app.Do(func() { list.Set([]int{2}) })
	html := app.Model.HTML()
	if strings.Count(html, ` data-exiting=""`) != 2 || strings.Count(html, ` inert=""`) != 2 {
		t.Fatalf("both dt and dd of the exiting row need data-exiting and inert: %s", html)
	}
	if !strings.Contains(html, `<dt data-exiting="" data-pi="3" inert="">k1</dt>`) || strings.Contains(html, `<dt data-pi="6" data-exiting`) {
		t.Fatalf("dt of row 1 is not marked: %s", html)
	}
	app.Do(func() { list.Set([]int{1, 2}) })
	if html := app.Model.HTML(); strings.Contains(html, "data-exiting") || strings.Contains(html, "inert") {
		t.Fatalf("a revived row must lose its exit markers: %s", html)
	}
	app.Do(func() { list.Set([]int{2}) })
	app.Advance(101 * time.Millisecond)
	if html := app.Model.HTML(); strings.Contains(html, "k1") || strings.Contains(html, "data-exiting") {
		t.Fatalf("row not dropped: %s", html)
	}
}

func TestFragmentFeedRowsInsertAndDelete(t *testing.T) {
	items := []int{1, 2}
	var feed *runtime.Feed[int, int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		feed = runtime.NewFeed(func(n int) int { return n }, func() []int { return append([]int(nil), items...) })
		return el("dl", runtime.For(feed, runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return fragRow(item.Peek()) })))
	})
	app.Do(func() { items = append(items, 3); feed.Append(3) })
	app.Do(func() { items = items[1:]; feed.Delete(1) })
	if strings.Count(app.Model.HTML(), "<dt") != 2 {
		t.Fatalf("html = %s", app.Model.HTML())
	}
}

func TestUnitRootAcceptsTextAndEmptyContent(t *testing.T) {
	var list *runtime.Signal[[]int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, []int{1, 2})
		return el("p", runtime.For(runtime.Items(list, func(n int) int { return n }),
			runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node {
				if item.Peek() == 2 {
					return frag()
				}
				return work.TextOf(item)
			})))
	})
	app.Do(func() { list.Set([]int{2, 1, 3}) })
}
