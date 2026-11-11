package runtime_test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestHandlerIdentityIsNodeIDAndEvent(t *testing.T) {
	var log []string
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		mk := func(name string) []Item {
			return []Item{
				on("click", func(work.Event) { log = append(log, name+":click") }),
				work.OnWith("input", metadata.EventOptions{Debounce: 100, Fields: []string{"value"}}, func(work.Event) { log = append(log, name+":input") }),
			}
		}
		return el("div", el("a", mk("a")...), el("b", mk("b")...))
	})
	requireEqual(t, app.SSR, `<div><a data-pi="1" data-pon='[{"e":"click"},{"e":"input","db":100,"p":"value"}]'></a><b data-pi="2" data-pon='[{"e":"click"},{"e":"input","db":100,"p":"value"}]'></b></div>`)

	for _, c := range []struct {
		id    uint32
		event string
	}{{2, "input"}, {1, "click"}, {2, "click"}, {1, "input"}} {
		if err := app.Dispatch(c.id, c.event, nil); err != nil {
			t.Fatal(err)
		}
	}

	if got := strings.Join(log, ","); got != "b:input,a:click,b:click,a:input" {
		t.Fatalf("log = %s", got)
	}
}

func TestHandlerSpecsCoverEveryOptionAndAreDeterministic(t *testing.T) {
	opts := metadata.EventOptions{Prevent: true, Stop: true, Passive: true, Once: true, Capture: true, Debounce: 5, Throttle: 7, CapturePointer: true, Keys: []string{"a", "b\"c"}, Fields: []string{"clientX", "clientY"}}
	build := func(ctx *Ctx) Node {
		return el("div", work.OnWith("scroll", opts, noopHandler), on("click", noopHandler))
	}
	app := runtimetest.New(t, build)
	want := `[{"e":"click"},{"e":"scroll","pv":1,"sp":1,"ps":1,"o":1,"c":1,"cp":1,"db":5,"k":["a","b\"c"],"p":"clientX clientY"}]`
	if !strings.Contains(app.SSR, strings.ReplaceAll(want, `&`, "&amp;")) {
		t.Fatalf("spec mismatch:\n%s\nwant %s", app.SSR, want)
	}
}

func TestIDsAreNeverReusedAcrossForShowAndSwitch(t *testing.T) {
	var list *runtime.Signal[[]int]
	var flag *runtime.Signal[bool]
	var sel *runtime.Signal[int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, []int{1, 2, 3})
		flag = runtime.UseSignal(ctx, true)
		sel = runtime.UseSignal(ctx, 0)
		return el("div",
			runtime.For(runtime.Items(list, func(n int) int { return n }),
				runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return el("p", bound(item)) }),
				runtime.Exit(time.Second)),
			runtime.Show(flag, func() Node { return el("em", bound(sel)) }, runtime.Exit(time.Second)),
			runtime.Switch(sel,
				runtime.Case(0, func() Node { return el("i", bound(sel)) }),
				runtime.Case(1, func() Node { return el("u", bound(sel)) }),
				runtime.Exit(time.Second)))
	})
	seen := map[uint32]bool{}
	record := func() {
		t.Helper()
		for _, id := range app.Model.IDs() {
			seen[id] = true
		}
	}
	record()
	removedEver := map[uint32]bool{}
	liveBefore := map[uint32]bool{}
	for _, id := range app.Model.IDs() {
		liveBefore[id] = true
	}
	steps := []func(){
		func() { list.Set([]int{3, 4}) },
		func() { flag.Set(false) },
		func() { sel.Set(1) },
		func() { flag.Set(true) },
		func() { list.Set([]int{1, 2, 3, 4, 5}) },
		func() { sel.Set(0) },
		func() { list.Set(nil) },
	}
	for i, step := range steps {
		app.Do(step)
		app.Advance(2 * time.Second)
		live := map[uint32]bool{}
		for _, id := range app.Model.IDs() {
			live[id] = true
		}
		for id := range liveBefore {
			if !live[id] {
				removedEver[id] = true
			}
		}
		for id := range live {
			if removedEver[id] {
				t.Fatalf("step %d: id %d came back after its node was removed", i, id)
			}
		}
		liveBefore = live
	}
	if len(removedEver) < 5 {
		t.Fatalf("scenario removed only %v", removedEver)
	}
}

func TestExitingRowsAreIncludedInSerializeAtTheirMergedPosition(t *testing.T) {
	var list *runtime.Signal[[]int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, []int{1, 2, 3, 4})
		return el("ul", runtime.For(runtime.Items(list, func(n int) int { return n }),
			runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return el("li", bound(item)) }),
			runtime.Exit(time.Second)))
	})

	app.Do(func() { list.Set([]int{4, 1, 3}) })

	got := app.Normalized()
	order := liOrderOf(got)
	if order != "4,1,2,3" && order != "2,4,1,3" && order != "4,2,1,3" {
		t.Fatalf("exiting row 2 must keep a merged position, got %s", order)
	}
	if !strings.Contains(got, `data-exiting=""`) {
		t.Fatalf("missing exit marker: %s", got)
	}
	app.Do(func() { list.Set([]int{2, 4}) })
	if strings.Count(app.Model.HTML(), "data-exiting") != 2 {
		t.Fatalf("1 and 3 should be exiting while 2 revives: %s", app.Model.HTML())
	}
}

func liOrderOf(markup string) string {
	var out []string
	for _, part := range strings.Split(markup, "<li")[1:] {
		body := part[strings.Index(part, ">")+1:]
		out = append(out, strings.SplitN(body, "<", 2)[0])
	}
	return strings.Join(out, ",")
}

func TestSwitchWaitExitHoldsTheNewArmBack(t *testing.T) {
	var sel *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sel = runtime.UseSignal(ctx, "a")
		return el("div", runtime.Switch(sel,
			runtime.Case("a", func() Node { return el("p", txt("A")) }),
			runtime.Case("b", func() Node { return el("p", txt("B")) }),
			runtime.Exit(100*time.Millisecond), runtime.WaitExit()))
	})

	app.Do(func() { sel.Set("b") })
	if got := app.Normalized(); !strings.Contains(got, "A") || strings.Contains(got, "B") {
		t.Fatalf("the new arm must wait for the old one to leave: %s", got)
	}
	app.Advance(150 * time.Millisecond)

	if got := app.Normalized(); strings.Contains(got, "A") || !strings.Contains(got, "B") {
		t.Fatalf("after the exit the new arm must show: %s", got)
	}
}

func TestSwitchSyncModeShowsBothDuringTheExit(t *testing.T) {
	var sel *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sel = runtime.UseSignal(ctx, "a")
		return el("div", runtime.Switch(sel,
			runtime.Case("a", func() Node { return el("p", txt("A")) }),
			runtime.Case("b", func() Node { return el("p", txt("B")) }),
			runtime.Exit(100*time.Millisecond)))
	})

	app.Do(func() { sel.Set("b") })

	if got := app.Normalized(); !strings.Contains(got, "A") || !strings.Contains(got, "B") {
		t.Fatalf("sync mode shows both arms while the old one exits: %s", got)
	}
}

func TestNoEditTargetsANodeRemovedInTheSameFlush(t *testing.T) {
	var list *runtime.Signal[[]int]
	var label *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		list = runtime.UseSignal(ctx, []int{1, 2})
		label = runtime.UseSignal(ctx, "x")
		return el("ul", runtime.For(runtime.Items(list, func(n int) int { return n }),
			runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return el("li", attr("title", label), bound(label)) })))
	})

	edits := app.Do(func() {
		label.Set("y")
		list.Set([]int{1})
	})

	removed := map[uint32]bool{}
	for _, e := range edits {
		if e.Op == protocol.OpRemove {
			removed[e.ID] = true
		}
		if removed[e.ID] && e.Op != protocol.OpRemove {
			t.Fatalf("edit %v targets a node removed earlier in the frame: %v", e, edits)
		}
	}
	if len(removed) != 1 {
		t.Fatalf("edits = %v", edits)
	}
}

func TestSSRIsDeterministicAcrossSessions(t *testing.T) {
	build := func() string {
		s := runtime.NewSession(runtime.Config{Root: func(ctx *Ctx) Node {
			list := runtime.UseSignal(ctx, []int{3, 1, 2})
			return el("main", runtime.For(runtime.Items(list, func(n int) int { return n }),
				runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node {
					return el("p", on("click", noopHandler), cls("a", "b"), bound(item), txt(fmt.Sprint("-", item.Peek())))
				})), runtime.Show(runtime.Const(true), func() Node { return el("i") }))
		}})
		snap, err := s.RenderSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		return snap.HTML
	}
	first := build()
	for i := 0; i < 5; i++ {
		if got := build(); got != first {
			t.Fatalf("SSR differs between sessions:\n%s\n%s", first, got)
		}
	}
}

func TestDuplicateKeysAreADevError(t *testing.T) {
	snap, rerr := func() (runtime.Snapshot, error) {
		s := runtime.NewSession(runtime.Config{DevMode: true, Root: func(ctx *Ctx) Node {
			list := runtime.UseSignal(ctx, []int{1, 1})
			return el("ul", runtime.For(runtime.Items(list, func(n int) int { return n }),
				runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[int]) Node { return el("li") })))
		}})
		return s.RenderSnapshot()
	}()
	if rerr != nil || !snap.Crashed || !strings.Contains(snap.Crash.Message, "duplicate key") {
		t.Fatalf("snapshot=%+v err=%v", snap, rerr)
	}
}
