package runtime_test

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func counterApp(t *testing.T) (*runtimetest.App, *runtime.Signal[int]) {
	var count *runtime.Signal[int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		count = runtime.UseSignal(ctx, 0)
		return el("div",
			el("p", txt("Count: "), bound(count)),
			el("span", bound(count)),
		)
	})
	return app, count
}

func TestSSRGoldenAndIDsOnlyOnDynamicNodes(t *testing.T) {
	app, _ := counterApp(t)

	requireEqual(t, app.SSR, `<div><p>Count: <!--t:1-->0</p><span data-pi="2">0</span></div>`)
}

func TestBoundUpdateSendsOnlyExactEdits(t *testing.T) {
	app, count := counterApp(t)

	edits := app.Do(func() { count.Set(7) })

	want := []protocol.Edit{protocol.TextEdit(1, "7"), protocol.TextEdit(2, "7")}
	if len(edits) != len(want) || edits[0] != want[0] || edits[1] != want[1] {
		t.Fatalf("edits = %v, want %v", edits, want)
	}
	requireEqual(t, app.Model.HTML(), `<div><p>Count: <!--t:1-->7</p><span data-pi="2">7</span></div>`)
}

func TestEqualWriteSendsNothing(t *testing.T) {
	app, count := counterApp(t)

	if edits := app.Do(func() { count.Set(0) }); len(edits) != 0 {
		t.Fatalf("edits = %v", edits)
	}
}

func TestTwoRelativeWritesInOneHandler(t *testing.T) {
	app, count := counterApp(t)

	edits := app.Do(func() {
		count.Update(func(n int) int { return n + 1 })
		count.Update(func(n int) int { return n + 1 })
	})

	if count.Peek() != 2 {
		t.Fatalf("count = %d, relative writes were lost", count.Peek())
	}
	if len(edits) != 2 || edits[0].Value != "2" {
		t.Fatalf("edits = %v, want one batched pair for 2", edits)
	}
}

func TestHandlerIdentityAndDispatch(t *testing.T) {
	var count *runtime.Signal[int]
	var evt work.Event
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		count = runtime.UseSignal(ctx, 0)
		return el("button", on("click", func(e work.Event) { evt = e; count.Update(func(n int) int { return n + 1 }) }), bound(count))
	})
	requireEqual(t, app.SSR, `<button data-pi="1" data-pon='[{"e":"click"}]'>0</button>`)

	if err := app.Dispatch(1, "click", map[string]any{"k": 1}); err != nil {
		t.Fatal(err)
	}

	if count.Peek() != 1 || evt.Name != "click" || evt.Payload["k"] != 1 {
		t.Fatalf("count=%d evt=%+v", count.Peek(), evt)
	}
	if err := app.Dispatch(1, "dblclick", nil); err != runtime.ErrStaleHandler {
		t.Fatalf("unknown event err = %v", err)
	}
	if err := app.Dispatch(99, "click", nil); err != runtime.ErrStaleHandler {
		t.Fatalf("unknown id err = %v", err)
	}
}

func TestAttributeClassAndStyleBindings(t *testing.T) {
	var on, label, color *runtime.Signal[string]
	var flag *runtime.Signal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		label = runtime.UseSignal(ctx, "a")
		flag = runtime.UseSignal(ctx, false)
		color = runtime.UseSignal(ctx, "red")
		on = runtime.UseSignal(ctx, "x")
		return el("div",
			attr("title", label),
			attr("hidden", flag),
			cls("base", classWhen{"active", flag}, on),
			work.Style("color", color),
			work.Style("width", "10px"),
		)
	})
	requireEqual(t, app.SSR, `<div data-pi="1" title="a" class="base x" style="color: red; width: 10px;"></div>`)

	app.Do(func() { label.Set("b"); flag.Set(true); color.Set(""); on.Set("y") })

	requireEqual(t, app.Model.HTML(), `<div class="base active y" data-pi="1" hidden="" style="width: 10px;" title="b"></div>`)
	if !strings.Contains(app.Model.HTML(), `hidden=""`) {
		t.Fatal("presence attribute missing")
	}
}
