package contract

import (
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func strs(kv ...string) map[string]any {
	out := map[string]any{}
	for i := 0; i < len(kv); i += 2 {
		out[kv[i]] = kv[i+1]
	}
	return out
}

func textAdjacent() scenario {
	return scenario{file: "text_adjacent", description: "adjacent bound and static text stay separate text slots", cases: []scenarioCase{
		bagCase("static_around_bound", strs("x", "1", "y", "2"), func(c *ctx, b *bag) node {
			x, y := use[string](c, b, "x"), use[string](c, b, "y")
			return doc(el("p", txt("a"), bound(x), bound(y), txt("b")))
		}, set("x", "10"), set("y", ""), set("x", ""), set("x", "  "), set("y", "<&>"), set("x", "1"), set("y", "2")),
		bagCase("only_bound", strs("x", "a", "y", "b"), func(c *ctx, b *bag) node {
			x, y := use[string](c, b, "x"), use[string](c, b, "y")
			return doc(el("div", bound(x), bound(y), bound(x)))
		}, set("x", "c"), set("y", ""), set("x", ""), set("y", "d")),
		bagCase("bound_then_static", strs("x", "a"), func(c *ctx, b *bag) node {
			x := use[string](c, b, "x")
			return doc(el("p", bound(x), txt("tail"), el("i", txt("e")), bound(x)))
		}, set("x", "b"), set("x", ""), set("x", "c")),
	}}
}

func textEmpty() scenario {
	return scenario{file: "text_empty", description: "empty text is created by the client and survives updates", cases: []scenarioCase{
		bagCase("sole_child_toggles", strs("x", ""), func(c *ctx, b *bag) node {
			return doc(el("span", bound(use[string](c, b, "x"))))
		}, set("x", "v"), set("x", ""), set("x", "w"), set("x", "")),
		bagCase("empty_between_elements", strs("x", ""), func(c *ctx, b *bag) node {
			return doc(el("p", el("i", txt("a")), bound(use[string](c, b, "x")), el("b", txt("c")), txt("")))
		}, set("x", "mid"), set("x", ""), set("x", "again")),
		bagCase("empty_static_text", strs("x", "k"), func(c *ctx, b *bag) node {
			return doc(el("p", txt(""), bound(use[string](c, b, "x")), txt("")))
		}, set("x", ""), set("x", "z")),
	}}
}

func tableRows(c *ctx, b *bag) node {
	list := use[[]row](c, b, "rows")
	return doc(el("table",
		el("thead", el("tr", el("th", txt("k")), el("th", txt("v")))),
		el("tbody", runtime.For(runtime.Items(list, rowKey), runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[row]) node {
			v := runtime.UseComputed(c, func() string { return it.Get().V })
			return el("tr", el("td", txt(itoa(it.Peek().K))), el("td", bound(v)))
		})))))
}

func tableScenario() scenario {
	return scenario{file: "table", description: "rows inserted into tbody, select and Show arms parse in the parent's context", cases: []scenarioCase{
		bagCase("tbody_rows", rowsInit(1, 2), tableRows,
			setRows(1, 2, 3), setRows(0, 1, 2, 3), setRows(3, 2, 1, 0), setRows(2), setRows(), setRows(5, 4)),
		bagCase("show_row_in_tbody", map[string]any{"on": false}, func(c *ctx, b *bag) node {
			on := use[bool](c, b, "on")
			return doc(el("table", el("tbody", el("tr", el("td", txt("fixed"))),
				runtime.Show(on, func() node { return el("tr", el("td", txt("extra"))) }))))
		}, set("on", true), set("on", false), set("on", true)),
		bagCase("select_options", rowsInit(1, 2), func(c *ctx, b *bag) node {
			list := use[[]row](c, b, "rows")
			return doc(el("select", runtime.For(runtime.Items(list, rowKey), runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[row]) node {
				return el("option", attr("value", itoa(it.Peek().K)), txt(it.Peek().V))
			}))))
		}, setRows(1, 2, 3), setRows(3, 1), setRows(), setRows(2, 4)),
		bagCase("colgroup_and_caption", map[string]any{"on": false}, func(c *ctx, b *bag) node {
			on := use[bool](c, b, "on")
			return doc(el("table", runtime.Show(on, func() node { return el("caption", txt("cap")) }), el("tbody", el("tr", el("td", txt("x"))))))
		}, set("on", true), set("on", false)),
	}}
}

func attrsForm() scenario {
	view := func(c *ctx, b *bag) node {
		value := use[string](c, b, "value")
		checked := use[bool](c, b, "checked")
		area := use[string](c, b, "area")
		return doc(el("form",
			el("input", attr("type", "text"), work.Bind(value)),
			el("input", attr("type", "checkbox"), work.Bind(checked)),
			el("textarea", work.Bind(area)),
			el("select", el("option", attr("value", "a"), txt("A")), el("option", attr("value", "b"), txt("B"))),
			el("input", attr("type", "text"), attr("disabled", checked), attr("placeholder", value))))
	}
	init := map[string]any{"value": "start", "checked": false, "area": "text"}
	return scenario{file: "attrs_form", description: "a sets the attribute and the form property only when it differs, through Bind", cases: []scenarioCase{
		bagCase("input_value", init, view, set("value", "next"), set("value", ""), set("value", "a \"quoted\" <b>"), set("value", "start")),
		bagCase("checked", init, view, set("checked", true), set("checked", false), set("checked", true)),
		bagCase("textarea_text", init, view, set("area", "line1\nline2"), set("area", ""), set("area", "<b>&"), set("area", "text")),
	}}
}

func attrsBool() scenario {
	view := func(c *ctx, b *bag) node {
		hidden := use[bool](c, b, "hidden")
		title := use[string](c, b, "title")
		mood := use[string](c, b, "mood")
		on := use[bool](c, b, "on")
		order := use[int](c, b, "order")
		return doc(el("div",
			el("p", attr("hidden", hidden), work.Attribute("title", title, work.OmitEmpty()), txt("p")),
			el("p", cls("base", classWhen{"on", on}, mood), work.Style("order", order), txt("q")),
			el("p", attr("data-static", "s"), attr("disabled", true), txt("r"))))
	}
	init := map[string]any{"hidden": false, "title": "", "mood": "", "on": false, "order": 0}
	return scenario{file: "attrs_bool", description: "presence attributes, absent-when-empty attributes, classes and styles use a, ra and st", cases: []scenarioCase{
		bagCase("presence_toggles", init, view, set("hidden", true), set("hidden", false), set("hidden", true)),
		bagCase("optional_attribute", init, view, set("title", "hi"), set("title", ""), set("title", "again"), set("title", "")),
		bagCase("class_parts", init, view, set("on", true), set("mood", "calm"), set("on", false), set("mood", ""), set("on", true), set("mood", "x y")),
		bagCase("style_property", init, view, set("order", 3), set("order", 0), set("order", 7)),
	}}
}

func raw() scenario {
	view := func(host string) func(c *ctx, b *bag) node {
		return func(c *ctx, b *bag) node {
			html := use[string](c, b, "html")
			if host == "tbody" {
				return doc(el("table", el("tbody", work.UnsafeHTML(html))))
			}
			return doc(el(host, work.UnsafeHTML(html)))
		}
	}
	return scenario{file: "raw", description: "raw hosts are replaced with h and parsed by the browser", cases: []scenarioCase{
		bagCase("div_host", strs("html", "<b>x</b>"), view("div"),
			set("html", "a &amp; b <i>z</i>"), set("html", ""), set("html", "<p>one</p><p>two</p>"), set("html", "<b>x</b>")),
		bagCase("ul_host", strs("html", "<li>a</li>"), view("ul"),
			set("html", "<li>a</li><li>b</li>"), set("html", "<li>c</li>"), set("html", "")),
		bagCase("tbody_host", strs("html", "<tr><td>1</td></tr>"), view("tbody"),
			set("html", "<tr><td>1</td></tr><tr><td>2</td></tr>"), set("html", "")),
		bagCase("sibling_stays_put", strs("html", "<b>x</b>", "tail", "t"), func(c *ctx, b *bag) node {
			html, tail := use[string](c, b, "html"), use[string](c, b, "tail")
			return doc(el("div", el("section", work.UnsafeHTML(html)), el("p", bound(tail))))
		}, set("html", "<i>y</i>"), set("tail", "u"), set("html", "<b>x</b><b>y</b>")),
	}}
}

func svg() scenario {
	view := func(c *ctx, b *bag) node {
		r := use[int](c, b, "r")
		on := use[bool](c, b, "on")
		list := use[[]row](c, b, "rows")
		return doc(el("svg", attr("viewBox", "0 0 10 10"),
			el("circle", attr("r", r)),
			runtime.Show(on, func() node {
				return el("g", el("title", txt("tip")), el("rect", attr("width", "3")))
			}),
			runtime.For(runtime.Items(list, rowKey), runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[row]) node {
				return el("circle", attr("cx", itoa(it.Peek().K)))
			}))))
	}
	init := map[string]any{"r": 1, "on": false, "rows": rows(1, 2)}
	math := func(c *ctx, b *bag) node {
		on := use[bool](c, b, "on")
		return doc(el("math", el("mi", txt("x")), runtime.Show(on, func() node { return el("mrow", el("mn", txt("1"))) })))
	}
	return scenario{file: "svg", description: "SVG and MathML content inserted by edits lands in the right namespace", cases: []scenarioCase{
		bagCase("show_group_with_title", init, view, set("on", true), set("r", 4), set("on", false), set("on", true)),
		bagCase("for_circles", init, view, set("rows", rows(1, 2, 3)), set("rows", rows(3, 1)), set("rows", rows()), set("rows", rows(5, 4))),
		bagCase("mathml_arm", map[string]any{"on": false}, math, set("on", true), set("on", false)),
	}}
}

func refsScenario() scenario {
	build := func() instance {
		var ref *runtime.Ref
		var rowRef *runtime.Ref
		b := newBag(map[string]any{"on": true, "label": "a", "rows": rows(1, 2)})
		view := func(c *ctx, b *bag) node {
			on := use[bool](c, b, "on")
			label := use[string](c, b, "label")
			list := use[[]row](c, b, "rows")
			r := useRef(c)
			ref = r
			return doc(el("main",
				el("button", runtime.AttachItem(r), bound(label)),
				runtime.Show(on, func() node { return el("p", txt("shown")) }),
				runtime.For(runtime.Items(list, rowKey), runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[row]) node {
					rr := useRef(c)
					rowRef = rr
					return el("li", runtime.AttachItem(rr), txt(it.Peek().V))
				}))))
		}
		check := func(a *runtimetest.App) {
			a.T.Helper()
			var id uint32
			a.On(func() { id = ref.ID() })
			n := a.Model.Node(id)
			if id == 0 || n == nil || n.Data != "button" {
				a.T.Fatalf("the button ref id %d does not resolve to the button in %s", id, a.Model.HTML())
			}
			a.On(func() { id = rowRef.ID() })
			if n := a.Model.Node(id); id == 0 || n == nil || n.Data != "li" {
				a.T.Fatalf("the row ref id %d does not resolve to an li in %s", id, a.Model.HTML())
			}
		}
		do := func(fn func(b *bag)) func(a *runtimetest.App) {
			return func(a *runtimetest.App) {
				a.Do(func() { fn(b) })
				check(a)
			}
		}
		return instance{
			root: func(c *ctx) node { return view(c, b) },
			fresh: func() func(c *ctx) node {
				fb := b.clone()
				return func(c *ctx) node { return view(c, fb) }
			},
			steps: []func(a *runtimetest.App){
				do(func(b *bag) { put(b, "label", "b") }),
				do(func(b *bag) { put(b, "on", false) }),
				do(func(b *bag) { put(b, "rows", rows(2, 3, 1)) }),
				do(func(b *bag) { put(b, "on", true); put(b, "rows", rows(3)) }),
			},
		}
	}
	return scenario{file: "refs", description: "element refs are node ids and stay valid across edits", cases: []scenarioCase{
		{name: "ref_ids_resolve", build: build},
	}}
}

func showSwitch() scenario {
	showView := func(c *ctx, b *bag) node {
		on := use[bool](c, b, "on")
		label := use[string](c, b, "label")
		return doc(el("div",
			el("h1", txt("title")),
			runtime.Show(on, func() node { return el("p", txt("then "), bound(label)) }, runtime.Else(func() node { return el("em", txt("else")) })),
			el("footer", txt("end"))))
	}
	switchView := func(c *ctx, b *bag) node {
		sel := use[int](c, b, "sel")
		label := use[string](c, b, "label")
		return doc(el("div", runtime.Switch(sel,
			runtime.Case(1, func() node { return el("p", txt("one "), bound(label)) }),
			runtime.Case(2, func() node { return el("section", el("i", txt("two"))) }),
			runtime.Default(func() node { return el("span", txt("other")) }))))
	}
	nested := func(c *ctx, b *bag) node {
		a1, a2 := use[bool](c, b, "a"), use[bool](c, b, "b")
		list := use[[]row](c, b, "rows")
		return doc(el("div", runtime.Show(a1, func() node {
			return el("section", runtime.Show(a2, func() node {
				return el("ul", runtime.For(runtime.Items(list, rowKey), liRow))
			}))
		})))
	}
	showInit := map[string]any{"on": false, "label": "x"}
	return scenario{file: "show_switch", description: "Show and Switch arms mount, swap and unmount as regions", cases: []scenarioCase{
		bagCase("show_toggle", showInit, showView, set("on", true), set("label", "y"), set("on", false), set("on", true), set("on", false)),
		bagCase("switch_arms", map[string]any{"sel": 0, "label": "x"}, switchView,
			set("sel", 1), set("label", "y"), set("sel", 2), set("sel", 3), set("sel", 1), set("sel", 0)),
		bagCase("nested_regions", map[string]any{"a": false, "b": false, "rows": rows(1, 2, 3)}, nested,
			set("a", true), set("b", true), set("rows", rows(3, 1)), set("a", false), set("a", true), set("rows", rows(4, 5)), set("b", false)),
	}}
}

func componentsScenario() scenario {
	card := runtime.PropsComponent(func(c *ctx, title string) node {
		n := runtime.UseSignal(c, 0)
		return el("article", el("h2", txt(title)), el("span", bound(n)))
	})
	static := runtime.Component(func(c *ctx) node { return el("hr") })
	view := func(c *ctx, b *bag) node {
		list := use[[]row](c, b, "rows")
		on := use[bool](c, b, "on")
		return doc(el("div", static(), runtime.Show(on, func() node { return card("shown") }),
			runtime.For(runtime.Items(list, rowKey), runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[row]) node { return card(it.Peek().V) }))))
	}
	return scenario{file: "components", description: "components inside regions get their own id blocks", cases: []scenarioCase{
		bagCase("components_in_regions", map[string]any{"rows": rows(1, 2), "on": false}, view,
			set("on", true), set("rows", rows(2, 3, 1)), set("on", false), set("rows", rows()), set("on", true)),
	}}
}

func handlersScenario() scenario {
	counter := func(c *ctx, b *bag) node {
		n := use[int](c, b, "n")
		return doc(el("div", el("button", on("click", func(work.Event) { put(b, "n", b.vals["n"].(int)+1) }), txt("+")), el("span", bound(n))))
	}
	rowsView := func(c *ctx, b *bag) node {
		list := use[[]row](c, b, "rows")
		return doc(el("div",
			el("button", on("click", func(work.Event) {
				cur := b.vals["rows"].([]row)
				next := b.vals["next"].(int) + 1
				b.vals["next"] = next
				put(b, "rows", append([]row{{K: next, V: "new"}}, cur...))
			}), txt("add")),
			el("ul", runtime.For(runtime.Items(list, rowKey), runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[row]) node {
				k := it.Peek().K
				return el("li", txt(it.Peek().V), el("button", on("click", func(work.Event) {
					cur := b.vals["rows"].([]row)
					var next []row
					for _, r := range cur {
						if r.K != k {
							next = append(next, r)
						}
					}
					put(b, "rows", next)
				}), txt("x")))
			})))))
	}
	selfClosing := func(c *ctx, b *bag) node {
		open := use[bool](c, b, "open")
		return doc(el("div", el("button", on("click", func(work.Event) { put(b, "open", !b.vals["open"].(bool)) }), txt("toggle")),
			runtime.Show(open, func() node {
				return el("p", el("button", on("click", func(work.Event) { put(b, "open", false) }), txt("close")))
			})))
	}
	options := func(c *ctx, b *bag) node {
		return doc(el("div",
			el("input", work.OnWith("input", metadata.EventOptions{Prevent: true, Stop: true, Debounce: 300, Fields: []string{"value"}}, noop)),
			el("a", attr("href", "#"), work.OnWith("click", metadata.EventOptions{Prevent: true, Once: true, Capture: true}, noop), txt("link")),
			el("div", work.OnWith("scroll", metadata.EventOptions{Passive: true, Throttle: 100}, noop), work.On("click", noop), txt("s"))))
	}
	click := func(nth int) stepFn {
		return func(a *runtimetest.App, b *bag) {
			a.T.Helper()
			id := idOfTag(a.T, a, "button", nth)
			if err := a.Dispatch(id, "click", nil); err != nil {
				a.T.Fatalf("dispatch click on button %d (id %d): %v", nth, id, err)
			}
		}
	}
	return scenario{file: "handlers", description: "data-pon specs drive client listeners, and handlers run through DispatchHandler", cases: []scenarioCase{
		bagCase("counter_clicks", map[string]any{"n": 0}, counter, click(0), click(0), click(0)),
		bagCase("rows_add_and_remove", map[string]any{"rows": rows(1, 2), "next": 100}, rowsView, click(0), click(2), click(0), click(1), click(1)),
		bagCase("self_closing_arm", map[string]any{"open": false}, selfClosing, click(0), click(1), click(0), click(0)),
		bagCase("option_matrix", map[string]any{}, options, change(func(b *bag) {})),
	}}
}

func portalScenario() scenario {
	view := func(c *ctx, b *bag) node {
		open := use[bool](c, b, "open")
		label := use[string](c, b, "label")
		return doc(el("div",
			el("section", runtime.Show(open, func() node { return el("p", work.NewPortal(el("dialog", bound(label)))) })),
			el("footer", work.NewPortalTarget())))
	}
	return scenario{file: "portal", description: "portal entries are units inside the target region", cases: []scenarioCase{
		bagCase("open_close_and_relabel", map[string]any{"open": false, "label": "a"}, view,
			set("open", true), set("label", "b"), set("open", false), set("open", true)),
	}}
}

func styleScenario() scenario {
	return scenario{file: "styles", description: "scoped style sheets are static style units that follow component lifetime", cases: []scenarioCase{
		bagCase("sheets_follow_components", map[string]any{"a": false, "b": false}, styledView,
			set("a", true), set("b", true), set("a", false), set("b", false), set("a", true)),
	}}
}

func useRef(c *ctx) *runtime.Ref {
	return runtime.UseCell(c, func() *runtime.Ref { return runtime.NewRef(c, "ElementRef", nil) })
}
