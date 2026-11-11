package contract

import (
	"slices"
	"time"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
)

type row struct {
	K int
	V string
}

func rows(keys ...int) []row {
	out := make([]row, len(keys))
	for i, k := range keys {
		out[i] = row{K: k, V: "v" + itoa(k)}
	}
	return out
}

func rowKey(r row) int { return r.K }

var liRow = runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[row]) node {
	v := runtime.UseComputed(c, func() string { return it.Get().V })
	return el("li", attr("data-k", itoa(it.Peek().K)), bound(v))
})

func listView(opts ...runtime.ForOpt) func(c *ctx, b *bag) node {
	return func(c *ctx, b *bag) node {
		list := use[[]row](c, b, "rows")
		return doc(el("ul", runtime.For(runtime.Items(list, rowKey), liRow, opts...)))
	}
}

func rowsInit(keys ...int) map[string]any { return map[string]any{"rows": rows(keys...)} }

func setRows(keys ...int) stepFn { return set("rows", rows(keys...)) }

func forKeyed() scenario {
	return scenario{file: "for_keyed", description: "keyed rows are id-addressed units: insert, delete, replace, clear and refill", cases: []scenarioCase{
		bagCase("insert_positions", rowsInit(1, 2, 3), listView(),
			setRows(9, 1, 2, 3), setRows(9, 1, 8, 2, 3), setRows(9, 1, 8, 2, 3, 7), setRows(9, 1, 8, 2, 3, 7, 6, 5)),
		bagCase("delete_positions", rowsInit(1, 2, 3, 4, 5, 6), listView(),
			setRows(2, 3, 4, 5, 6), setRows(2, 3, 5, 6), setRows(2, 3, 5), setRows(3), setRows()),
		bagCase("replace_values", rowsInit(1, 2, 3), listView(),
			set("rows", []row{{1, "one"}, {2, "v2"}, {3, "v3"}}),
			set("rows", []row{{1, "one"}, {2, "two"}, {3, "three"}}),
			set("rows", []row{{1, ""}, {2, "<&>"}, {3, "a b"}})),
		bagCase("clear_and_refill", rowsInit(1, 2, 3, 4), listView(),
			setRows(), setRows(1, 2, 3, 4), setRows(4, 3), setRows(), setRows(7)),
		bagCase("start_empty", rowsInit(), listView(),
			setRows(1), setRows(1, 2), setRows(2), setRows(2, 3, 4), setRows()),
		bagCase("mixed_ops_in_one_step", rowsInit(1, 2, 3, 4, 5), listView(),
			setRows(5, 9, 2, 8, 1), setRows(1, 2, 3, 4, 5), setRows(3, 4, 6, 1)),
		bagCase("nested_lists", map[string]any{"groups": []group{{1, []int{1, 2}}, {2, []int{3}}, {3, nil}}}, nestedView,
			set("groups", []group{{1, []int{1, 2}}, {2, []int{3, 4}}, {3, nil}}),
			set("groups", []group{{3, []int{9}}, {1, []int{2, 1}}}),
			set("groups", []group{{1, []int{2, 1, 5}}, {4, []int{6}}, {3, []int{9}}}),
			set("groups", []group{})),
	}}
}

type group struct {
	K    int
	Kids []int
}

func nestedView(c *ctx, b *bag) node {
	groups := use[[]group](c, b, "groups")
	return doc(el("div", runtime.For(runtime.Items(groups, func(g group) int { return g.K }),
		runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[group]) node {
			kids := runtime.UseComputed(c, func() []int { return slices.Clone(it.Get().Kids) })
			return el("section", attr("data-g", itoa(it.Peek().K)),
				el("ol", runtime.For(runtime.Items(kids, func(n int) int { return n }),
					runtime.PropsComponent(func(c *ctx, k runtime.ReadSignal[int]) node { return el("li", bound(k)) }))))
		}))))
}

func permutations(base []int) [][]int {
	var out [][]int
	var rec func(i int, cur []int)
	rec = func(i int, cur []int) {
		if i == len(cur) {
			out = append(out, slices.Clone(cur))
			return
		}
		for j := i; j < len(cur); j++ {
			cur[i], cur[j] = cur[j], cur[i]
			rec(i+1, cur)
			cur[i], cur[j] = cur[j], cur[i]
		}
	}
	rec(0, slices.Clone(base))
	slices.SortFunc(out, slices.Compare[[]int])
	return out
}

func forPermutations() scenario {
	var all []stepFn
	for _, p := range permutations([]int{1, 2, 3, 4}) {
		all = append(all, setRows(p...))
	}
	return scenario{file: "for_permutations", description: "reordering moves units with mv and never recreates them", cases: []scenarioCase{
		bagCase("named_permutations", rowsInit(1, 2, 3, 4, 5, 6), listView(),
			setRows(2, 3, 4, 5, 6, 1), setRows(1, 6, 5, 4, 3, 2), setRows(2, 3, 4, 5, 6, 1), setRows(1, 3, 2, 4, 5, 6),
			setRows(6, 2, 3, 4, 5, 1), setRows(1, 4, 2, 5, 3, 6), setRows(6, 5, 4, 3, 2, 1), setRows(1, 2, 3, 4, 5, 6)),
		bagCase("every_order_of_four", rowsInit(1, 2, 3, 4), listView(), all...),
		bagCase("reorder_with_insert_and_delete", rowsInit(1, 2, 3, 4, 5), listView(),
			setRows(5, 4, 3, 2, 1, 6), setRows(6, 2, 5, 1), setRows(1, 2, 5, 6, 7, 8), setRows(8, 7, 6, 5, 2, 1)),
	}}
}

func forDuplicate() scenario {
	dupKey := func(r row) int { return r.K / 10 }
	view := func(c *ctx, b *bag) node {
		list := use[[]row](c, b, "rows")
		return doc(el("ul", runtime.For(runtime.Items(list, dupKey), liRow)))
	}
	return scenario{file: "for_duplicate", description: "duplicate keys: production keeps the first occurrence", cases: []scenarioCase{
		prodBagCase("first_wins", rowsInit(11), view,
			setRows(11, 12, 25), setRows(25, 11, 12), setRows(25, 31, 32, 11), setRows(31), setRows()),
	}}
}

type feedModel struct {
	items []row
	feed  *runtime.Feed[row, int]
}

func feedInstance(exit bool, initial []row, ops []func(m *feedModel, a *runtimetest.App)) instance {
	m := &feedModel{items: slices.Clone(initial)}
	var opts []runtime.ForOpt
	if exit {
		opts = append(opts, runtime.Exit(50*time.Millisecond))
	}
	view := func(f *runtime.Feed[row, int]) node {
		return doc(el("ul", runtime.For(f, runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[row]) node {
			r := it.Peek()
			return el("li", attr("data-k", itoa(r.K)), txt(r.V), el("b", txt("static")))
		}), opts...)))
	}
	inst := instance{
		root: func(c *ctx) node {
			f := runtime.NewFeed(rowKey, func() []row { return slices.Clone(m.items) })
			if m.feed == nil {
				m.feed = f
			}
			return view(f)
		},
		fresh: func() func(c *ctx) node {
			snap := slices.Clone(m.items)
			return func(c *ctx) node {
				return view(runtime.NewFeed(rowKey, func() []row { return snap }))
			}
		},
	}
	for _, op := range ops {
		op := op
		inst.steps = append(inst.steps, func(a *runtimetest.App) { op(m, a) })
	}
	return inst
}

func feedOp(fn func(m *feedModel)) func(m *feedModel, a *runtimetest.App) {
	return func(m *feedModel, a *runtimetest.App) { a.Do(func() { fn(m) }) }
}

func appendOp(keys ...int) func(m *feedModel, a *runtimetest.App) {
	return feedOp(func(m *feedModel) {
		m.items = append(m.items, rows(keys...)...)
		m.feed.Append(rows(keys...)...)
	})
}

func prependOp(keys ...int) func(m *feedModel, a *runtimetest.App) {
	return feedOp(func(m *feedModel) {
		m.items = append(rows(keys...), m.items...)
		m.feed.Prepend(rows(keys...)...)
	})
}

func insertBeforeOp(before int, keys ...int) func(m *feedModel, a *runtimetest.App) {
	return feedOp(func(m *feedModel) {
		i := slices.IndexFunc(m.items, func(r row) bool { return r.K == before })
		m.items = slices.Insert(m.items, i, rows(keys...)...)
		m.feed.InsertBefore(before, rows(keys...)...)
	})
}

func replaceOp(k int, v string) func(m *feedModel, a *runtimetest.App) {
	return feedOp(func(m *feedModel) {
		i := slices.IndexFunc(m.items, func(r row) bool { return r.K == k })
		m.items[i] = row{k, v}
		m.feed.Replace(row{k, v})
	})
}

func deleteOp(k int) func(m *feedModel, a *runtimetest.App) {
	return feedOp(func(m *feedModel) {
		m.items = slices.DeleteFunc(m.items, func(r row) bool { return r.K == k })
		m.feed.Delete(k)
	})
}

func reloadOp(keys ...int) func(m *feedModel, a *runtimetest.App) {
	return feedOp(func(m *feedModel) {
		m.items = rows(keys...)
		m.feed.Reload()
	})
}

func advanceOp(d time.Duration) func(m *feedModel, a *runtimetest.App) {
	return func(m *feedModel, a *runtimetest.App) { a.Advance(d) }
}

func feedCase(name string, exit bool, initial []row, ops ...func(m *feedModel, a *runtimetest.App)) scenarioCase {
	return scenarioCase{name: name, build: func() instance { return feedInstance(exit, initial, ops) }}
}

func feedScenario() scenario {
	return scenario{file: "feed", description: "a Feed source: append, prepend, insert before, replace, delete and reload", cases: []scenarioCase{
		feedCase("append_prepend", false, rows(2, 3), appendOp(4), prependOp(1), appendOp(5, 6), prependOp(8, 7)),
		feedCase("insert_before", false, rows(1, 5), insertBeforeOp(5, 2, 3, 4), insertBeforeOp(1, 0), insertBeforeOp(3, 9)),
		feedCase("replace_and_delete", false, rows(1, 2, 3, 4), replaceOp(2, "two"), deleteOp(1), replaceOp(4, ""), deleteOp(3), deleteOp(2), deleteOp(4)),
		feedCase("reload", false, rows(1, 2, 3), reloadOp(3, 2, 1, 9), reloadOp(), reloadOp(4), appendOp(5)),
		feedCase("exit_delete", true, rows(1, 2, 3), deleteOp(2), advanceOp(20*time.Millisecond), advanceOp(time.Second), appendOp(4)),
		feedCase("exit_delete_then_reappend", true, rows(1, 2), deleteOp(1), advanceOp(10*time.Millisecond), appendOp(7), advanceOp(time.Second)),
	}}
}

func exitScenario() scenario {
	showView := func(c *ctx, b *bag) node {
		on := use[bool](c, b, "on")
		return doc(el("div", runtime.Show(on, func() node { return el("p", cls("arm"), txt("shown")) },
			runtime.Exit(50*time.Millisecond), runtime.Else(func() node { return el("p", txt("hidden")) }))))
	}
	advance := func(d time.Duration) stepFn {
		return func(a *runtimetest.App, b *bag) { a.Advance(d) }
	}
	return scenario{file: "exit", description: "exit timers keep a unit with data-exiting and inert until it is dropped", cases: []scenarioCase{
		bagCase("show_exit_then_drop", map[string]any{"on": true}, showView,
			set("on", false), advance(20*time.Millisecond), advance(time.Second), set("on", true)),
		bagCase("show_exit_then_return", map[string]any{"on": true}, showView,
			set("on", false), advance(10*time.Millisecond), set("on", true), advance(time.Second)),
		bagCase("for_exit_rows", rowsInit(1, 2, 3), listView(runtime.Exit(50*time.Millisecond)),
			setRows(1, 3), advance(20*time.Millisecond), setRows(1, 2, 3), advance(time.Second), setRows(3), advance(time.Second)),
	}}
}
