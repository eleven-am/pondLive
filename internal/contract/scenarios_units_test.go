package contract

import (
	"time"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

var fragmentRow = runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[row]) node {
	v := runtime.UseComputed(c, func() string { return it.Get().V })
	return work.NewFragment(
		el("dt", attr("data-k", itoa(it.Peek().K)), txt("key "+itoa(it.Peek().K))),
		el("dd", bound(v)),
	)
})

func fragmentUnits() scenario {
	listView := func(opts ...runtime.ForOpt) func(c *ctx, b *bag) node {
		return func(c *ctx, b *bag) node {
			list := use[[]row](c, b, "rows")
			return doc(el("dl", runtime.For(runtime.Items(list, rowKey), fragmentRow, opts...)))
		}
	}
	armView := func(c *ctx, b *bag) node {
		on := use[bool](c, b, "on")
		sel := use[int](c, b, "sel")
		label := use[string](c, b, "label")
		return doc(el("div",
			runtime.Show(on,
				func() node { return work.NewFragment(el("p", txt("one ")), el("p", bound(label))) },
				runtime.Else(func() node { return work.NewFragment(txt("plain "), txt("text")) })),
			runtime.Switch(sel,
				runtime.Case(1, func() node { return work.NewFragment(el("i", txt("a")), el("i", bound(label))) }),
				runtime.Case(2, func() node { return txt("just text") }),
				runtime.Default(func() node { return work.NewFragment() })),
			el("footer", txt("end"))))
	}
	return scenario{file: "fragment_units", description: "a row or arm may be a fragment or any multi-node content and is wrapped in its own region", cases: []scenarioCase{
		bagCase("fragment_rows", rowsInit(1, 2, 3), listView(),
			setRows(3, 1, 2), setRows(3, 4, 1, 2), setRows(4, 2), setRows(), setRows(5, 6), set("rows", []row{{5, "five"}, {6, "six"}})),
		bagCase("fragment_rows_with_exit", rowsInit(1, 2, 3), listView(runtime.Exit(100*time.Millisecond)),
			setRows(1, 3), setRows(1, 2, 3), setRows(3)),
		bagCase("fragment_arms", map[string]any{"on": true, "sel": 1, "label": "x"}, armView,
			set("on", false), set("sel", 2), set("label", "y"), set("sel", 0), set("on", true), set("sel", 1)),
	}}
}
