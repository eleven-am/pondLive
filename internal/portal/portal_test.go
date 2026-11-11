package portal

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func el(tag string, items ...work.Item) work.Node { return work.BuildElement(tag, items...) }

func TestPortalRendersAtTarget(t *testing.T) {
	app := runtimetest.New(t, func(ctx *runtime.Ctx) work.Node {
		return el("div",
			el("section", Portal(el("dialog", work.NewText("hello")))),
			el("footer", Target()),
		)
	})

	if got := app.Normalized(); got != `<div><section></section><footer><dialog>hello</dialog></footer></div>` {
		t.Fatalf("got %s", got)
	}
}

func TestPortalPreservesContext(t *testing.T) {
	theme := runtime.NewContext[string]()
	reader := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		return el("b", work.TextOf(runtime.UseContext(ctx, theme)))
	})
	app := runtimetest.New(t, func(ctx *runtime.Ctx) work.Node {
		sig := runtime.UseSignal(ctx, "dark")
		runtime.Provide(ctx, theme, sig)
		return el("div", el("section", Portal(el("p", reader()))), el("footer", Target()))
	})

	if got := app.Normalized(); !strings.Contains(got, "<footer><p><b>dark</b></p></footer>") {
		t.Fatalf("the portal did not see its creator's context: %s", got)
	}
}

func TestMultiplePortalsRenderInOrder(t *testing.T) {
	app := runtimetest.New(t, func(ctx *runtime.Ctx) work.Node {
		return el("div",
			Portal(el("i", work.NewText("one"))),
			Portal(el("i", work.NewText("two"))),
			el("footer", Target()),
		)
	})

	if got := app.Normalized(); !strings.Contains(got, "<footer><i>one</i><i>two</i></footer>") {
		t.Fatalf("got %s", got)
	}
}

func TestEmptyPortalRendersNothing(t *testing.T) {
	app := runtimetest.New(t, func(ctx *runtime.Ctx) work.Node {
		return el("div", Portal(), el("footer", Target()))
	}, runtimetest.Dev(false))

	if got := app.Normalized(); !strings.Contains(got, "<footer></footer>") {
		t.Fatalf("got %s", got)
	}
}

func TestPortalLeavesWhenItsCreatorIsDisposed(t *testing.T) {
	var on *runtime.Signal[bool]
	app := runtimetest.New(t, func(ctx *runtime.Ctx) work.Node {
		on = runtime.UseSignal(ctx, true)
		return el("div",
			runtime.Show(on, func() work.Node { return el("section", Portal(el("dialog"))) }),
			el("footer", Target()),
		)
	})
	if !strings.Contains(app.Normalized(), "<dialog></dialog>") {
		t.Fatalf("got %s", app.Normalized())
	}

	app.Do(func() { on.Set(false) })

	if strings.Contains(app.Normalized(), "dialog") {
		t.Fatalf("portal entry outlived its creator: %s", app.Normalized())
	}
}

func TestPortalEntryMayBeSeveralNodes(t *testing.T) {
	var show *runtime.Signal[bool]
	app := runtimetest.New(t, func(ctx *runtime.Ctx) work.Node {
		show = runtime.UseSignal(ctx, true)
		return el("div",
			runtime.Show(show, func() work.Node {
				return el("section", Portal(el("i", work.NewText("one")), work.NewText("loose text"), el("b", work.NewText("two"))))
			}),
			el("footer", Target()),
		)
	})

	if got := app.Normalized(); !strings.Contains(got, "<footer>") || !strings.Contains(got, "<i>one</i>loose text<b>two</b>") {
		t.Fatalf("got %s", got)
	}

	app.Do(func() { show.Set(false) })

	if got := app.Normalized(); strings.Contains(got, "loose text") {
		t.Fatalf("the multi-node entry outlived its creator: %s", got)
	}
}
