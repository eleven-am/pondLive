package contract

import (
	"github.com/eleven-am/pondlive/internal/css"
	"github.com/eleven-am/pondlive/internal/head"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func headPage(body node) node {
	return head.Provider(head.ProviderProps{Children: head.HTML(head.ElementProps{Children: []work.Item{
		el("head", head.Render()),
		head.Body(head.ElementProps{Children: []work.Item{body}}),
	}})})
}

func headScenario() scenario {
	view := func(c *ctx, b *bag) node {
		title := use[string](c, b, "title")
		lang := use[string](c, b, "lang")
		bodyClass := use[string](c, b, "bodyClass")
		extra := use[bool](c, b, "extra")
		child := runtime.Component(func(c *ctx) node {
			head.UseHead(c, head.Head{
				Title: runtime.Const("child title"),
				Meta:  []head.MetaTag{{Name: "robots", Content: "noindex"}},
				Links: []head.LinkTag{{Rel: "stylesheet", Href: "/child.css"}},
			})
			return el("span", txt("child"))
		})
		page := runtime.Component(func(c *ctx) node {
			head.UseHead(c, head.Head{Title: title, HTMLLang: lang, BodyClass: bodyClass, Meta: []head.MetaTag{{Name: "author", Content: "me"}}})
			return el("main", runtime.Show(extra, func() node { return child() }))
		})
		return headPage(page())
	}
	init := map[string]any{"title": "first", "lang": "", "bodyClass": "", "extra": false}
	return scenario{file: "head_title", description: "title, html attributes and head entries follow the component tree; empty html attributes are absent", cases: []scenarioCase{
		bagCase("title_changes", init, view, set("title", "second"), set("title", ""), set("title", "<third> & co")),
		bagCase("html_lang_absent_when_empty", init, view, set("lang", "fr"), set("lang", ""), set("lang", "de"), set("lang", "")),
		bagCase("body_class_absent_when_empty", init, view, set("bodyClass", "dark"), set("bodyClass", ""), set("bodyClass", "a b")),
		bagCase("child_entries_mount_and_unmount", init, view, set("extra", true), set("title", "other"), set("extra", false), set("extra", true)),
	}}
}

func useScoped(c *ctx, raw string) *css.Scoped {
	scoped := css.Scope(c.ComponentName()+":"+raw, raw)
	head.UseHead(c, head.Head{Styles: []head.StyleTag{{ID: scoped.Hash, CSS: scoped.Text}}})
	return scoped
}

func styledView(c *ctx, b *bag) node {
	a, bb := use[bool](c, b, "a"), use[bool](c, b, "b")
	first := runtime.Component(func(c *ctx) node {
		st := useScoped(c, ".x { color: red; }")
		return el("p", work.Class(st.Class("x")), txt("first"))
	})
	second := runtime.Component(func(c *ctx) node {
		st := useScoped(c, ".y { margin: 0; }")
		return el("p", work.Class(st.Class("y")), txt("second"))
	})
	return headPage(el("main",
		runtime.Show(a, func() node { return first() }),
		runtime.Show(bb, func() node { return second() })))
}
