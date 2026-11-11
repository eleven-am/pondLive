package head_test

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/head"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func page(body work.Node) func(*runtime.Ctx) work.Node {
	return func(*runtime.Ctx) work.Node {
		return head.Provider(head.ProviderProps{Children: head.HTML(head.ElementProps{Children: []work.Item{
			work.BuildElement("head", head.Render()),
			head.Body(head.ElementProps{Children: []work.Item{body}}),
		}})})
	}
}

type toggles struct {
	show  *runtime.Signal[bool]
	title *runtime.Signal[string]
}

func TestUseHeadDeepestWinsAndUnmountRestores(t *testing.T) {
	var tg toggles
	child := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Title: runtime.Const("child")})
		return work.BuildElement("span")
	})
	parent := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		tg.show = runtime.UseSignal(ctx, true)
		head.UseHead(ctx, head.Head{Title: runtime.Const("parent")})
		return work.BuildElement("div", runtime.Show(tg.show, func() work.Node { return child() }))
	})
	app := runtimetest.New(t, page(parent()))
	if !strings.Contains(app.SSR, "<title") || !strings.Contains(app.SSR, ">child</title>") {
		t.Fatalf("ssr title: %s", app.SSR)
	}
	edits := app.Do(func() { tg.show.Set(false) })
	if !strings.Contains(app.Normalized(), ">parent</title>") {
		t.Fatalf("title not restored: %s", app.Normalized())
	}
	if len(edits) == 0 {
		t.Fatal("no edits for the restore")
	}
	app.Do(func() { tg.show.Set(true) })
	if !strings.Contains(app.Normalized(), ">child</title>") {
		t.Fatalf("title not reapplied: %s", app.Normalized())
	}
}

func TestUseHeadSignalTitleUpdatesInPlace(t *testing.T) {
	var title *runtime.Signal[string]
	comp := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		title = runtime.UseSignal(ctx, "one")
		head.UseHead(ctx, head.Head{Title: title})
		return work.NewText("x")
	})
	app := runtimetest.New(t, page(comp()))
	edits := app.Do(func() { title.Set("two") })
	if len(edits) != 1 || edits[0].Op != "t" {
		t.Fatalf("edits = %v, want a single t edit", edits)
	}
	if !strings.Contains(app.Normalized(), ">two</title>") {
		t.Fatalf("title = %s", app.Normalized())
	}
}

func TestUseHeadLaterMountBreaksATie(t *testing.T) {
	a := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Title: runtime.Const("first")})
		return work.NewText("a")
	})
	b := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Title: runtime.Const("second")})
		return work.NewText("b")
	})
	app := runtimetest.New(t, page(work.BuildElement("div", a(), b())))
	if !strings.Contains(app.SSR, ">second</title>") {
		t.Fatalf("ssr: %s", app.SSR)
	}
}

func TestUseHeadMergesListsByKey(t *testing.T) {
	var show *runtime.Signal[bool]
	child := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{
			Meta:  []head.MetaTag{{Name: "viewport", Content: "child"}, {Name: "robots", Content: "none"}},
			Links: []head.LinkTag{{Rel: "stylesheet", Href: "/b.css"}},
		})
		return work.BuildElement("span")
	})
	parent := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		show = runtime.UseSignal(ctx, true)
		head.UseHead(ctx, head.Head{
			Meta:        []head.MetaTag{{Name: "viewport", Content: "parent"}, {Name: "author", Content: "me"}},
			Links:       []head.LinkTag{{Rel: "stylesheet", Href: "/a.css"}},
			HeadScripts: []head.ScriptTag{{Src: "/app.js", Defer: true}},
		})
		return work.BuildElement("div", runtime.Show(show, func() work.Node { return child() }))
	})
	app := runtimetest.New(t, page(parent()))
	out := app.SSR
	if strings.Count(out, `name="viewport"`) != 1 || !strings.Contains(out, `content="child"`) || strings.Contains(out, `content="parent"`) {
		t.Fatalf("viewport not overridden by the deeper entry: %s", out)
	}
	for _, want := range []string{`name="author"`, `name="robots"`, `href="/a.css"`, `href="/b.css"`, `src="/app.js"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %s in %s", want, out)
		}
	}
	app.Do(func() { show.Set(false) })
	n := app.Normalized()
	if !strings.Contains(n, `content="parent"`) || strings.Contains(n, `content="child"`) || strings.Contains(n, "robots") || strings.Contains(n, "/b.css") {
		t.Fatalf("unmount did not restore the parent entries: %s", n)
	}
}

func TestUseHeadHTMLAndBodyAttributes(t *testing.T) {
	var cls *runtime.Signal[string]
	child := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		cls = runtime.UseSignal(ctx, "dark")
		head.UseHead(ctx, head.Head{HTMLLang: runtime.Const("fr"), HTMLClass: cls, BodyClass: runtime.Const("page")})
		return work.BuildElement("span")
	})
	parent := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{HTMLLang: runtime.Const("en"), HTMLDir: runtime.Const("rtl"), HTMLClass: runtime.Const("base"), BodyClass: runtime.Const("shell")})
		return work.BuildElement("div", child())
	})
	app := runtimetest.New(t, page(parent()))
	for _, want := range []string{`lang="fr"`, `dir="rtl"`, `class="base dark"`, `class="shell page"`} {
		if !strings.Contains(app.SSR, want) {
			t.Fatalf("missing %s in %s", want, app.SSR)
		}
	}
	app.Do(func() { cls.Set("light") })
	if !strings.Contains(app.Normalized(), `class="base light"`) {
		t.Fatalf("html class not updated: %s", app.Normalized())
	}
}

func TestHTMLAndBodyOmitEmptyAttributes(t *testing.T) {
	var lang *runtime.Signal[string]
	child := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		lang = runtime.UseSignal(ctx, "")
		head.UseHead(ctx, head.Head{HTMLLang: lang})
		return work.BuildElement("span")
	})
	app := runtimetest.New(t, page(child()))
	for _, bad := range []string{`lang=`, `dir=`, `class=`} {
		if strings.Contains(app.SSR, bad) {
			t.Fatalf("empty %s rendered: %s", bad, app.SSR)
		}
	}
	app.Do(func() { lang.Set("de") })
	if !strings.Contains(app.Normalized(), `lang="de"`) {
		t.Fatalf("lang not set: %s", app.Normalized())
	}
	edits := app.Do(func() { lang.Set("") })
	if len(edits) != 1 || edits[0].Op != protocol.OpRemoveAttr {
		t.Fatalf("edits = %v", edits)
	}
	if strings.Contains(app.Normalized(), `lang=`) {
		t.Fatalf("lang not removed: %s", app.Normalized())
	}
}

func TestUseHeadWithoutProviderIsAnError(t *testing.T) {
	comp := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{})
		return work.NewText("x")
	})
	s := runtime.NewSession(runtime.Config{SessionID: "t", Root: func(*runtime.Ctx) work.Node { return comp() }, DevMode: true})
	runtimetest.Start(t, s)
	if err := s.Ask(5e9, func() error {
		snap, err := s.RenderSnapshot()
		if err == nil && !snap.Crashed {
			t.Error("expected a crash for a missing provider")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func TestEntryIsRemovedWhenOwnerDisposed(t *testing.T) {
	var show *runtime.Signal[bool]
	var reg *head.Registry
	child := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Title: runtime.Const("c")})
		return work.BuildElement("span")
	})
	parent := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		show = runtime.UseSignal(ctx, true)
		reg = head.UseRegistry(ctx)
		return work.BuildElement("div", runtime.Show(show, func() work.Node { return child() }))
	})
	app := runtimetest.New(t, page(parent()))
	if reg.EntryCount() != 1 {
		t.Fatalf("entries = %d", reg.EntryCount())
	}
	app.Do(func() { show.Set(false) })
	if reg.EntryCount() != 0 {
		t.Fatalf("entries after unmount = %d", reg.EntryCount())
	}
}

func TestIconRendersFaviconLinksAndServesPNG(t *testing.T) {
	icon := work.BuildElement("svg",
		work.Attribute("xmlns", "http://www.w3.org/2000/svg"),
		work.Attribute("viewBox", "0 0 24 24"),
		work.BuildElement("circle", work.Attribute("cx", "12"), work.Attribute("cy", "12"), work.Attribute("r", "8"), work.Attribute("fill", "#ff0000")),
	)
	comp := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Icon: icon, IconBackground: "#000000"})
		return work.BuildElement("span")
	})
	app := runtimetest.New(t, page(comp()))
	if !strings.Contains(app.SSR, `rel="icon"`) || !strings.Contains(app.SSR, `rel="apple-touch-icon"`) {
		t.Fatalf("favicon links missing: %s", app.SSR)
	}
	start := strings.Index(app.SSR, `rel="icon"`)
	hrefAt := strings.Index(app.SSR[start:], `href="`)
	href := app.SSR[start+hrefAt+6:]
	href = href[:strings.Index(href, `"`)]
	var rec *httptestRecorder
	rec = serve(app.S, href)
	if rec.code != 200 || rec.contentType != "image/png" || len(rec.body) < 8 || string(rec.body[1:4]) != "PNG" {
		t.Fatalf("icon response: code %d type %q len %d", rec.code, rec.contentType, len(rec.body))
	}
}

func TestUseHeadLandsInTheFirstHTMLAndSerializeIsByteIdentical(t *testing.T) {
	var reg *head.Registry
	child := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{
			Title:       runtime.Const("First paint title"),
			Description: runtime.Const("first paint description"),
			Meta:        []head.MetaTag{{Name: "robots", Content: "noindex"}},
		})
		return work.BuildElement("p", work.NewText("body"))
	})
	probe := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		reg = head.UseRegistry(ctx)
		return work.BuildElement("div", child())
	})
	app := runtimetest.New(t, page(probe()))

	headEnd := strings.Index(app.SSR, "</head>")
	bodyStart := strings.Index(app.SSR, "<body")
	if headEnd < 0 || bodyStart < headEnd {
		t.Fatalf("document layout unexpected: %s", app.SSR)
	}
	headHTML := app.SSR[:headEnd]
	for _, want := range []string{">First paint title</title>", `content="first paint description"`, `name="robots"`} {
		if !strings.Contains(headHTML, want) {
			t.Fatalf("the first HTML is missing %s in <head>: %s", want, app.SSR)
		}
	}

	for i := 0; i < 3; i++ {
		if again := app.ResetHTML(); again != app.SSR {
			t.Fatalf("serialize pass %d differs from the server-rendered HTML\nssr:       %s\nserialize: %s", i, app.SSR, again)
		}
	}
	if reg.EntryCount() != 1 {
		t.Fatalf("serialize passes registered entries: %d", reg.EntryCount())
	}
}

func TestUseHeadEntriesNestingStaysDeepestWinsWithTwoCallsInOneComponent(t *testing.T) {
	inner := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Title: runtime.Const("inner")})
		return work.NewText("i")
	})
	outer := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Title: runtime.Const("outer-1")})
		head.UseHead(ctx, head.Head{Title: runtime.Const("outer-2")})
		return work.BuildElement("div", inner())
	})
	app := runtimetest.New(t, page(outer()))
	if !strings.Contains(app.SSR, ">inner</title>") {
		t.Fatalf("ssr: %s", app.SSR)
	}
}

func TestUseHeadDeeperEntryBeatsALaterShallowerOne(t *testing.T) {
	inner := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Title: runtime.Const("a-inner")})
		return work.NewText("i")
	})
	a := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Title: runtime.Const("a-outer")})
		return work.BuildElement("div", inner())
	})
	b := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Title: runtime.Const("b")})
		return work.NewText("b")
	})
	app := runtimetest.New(t, page(work.BuildElement("div", a(), b())))
	if !strings.Contains(app.SSR, ">a-inner</title>") {
		t.Fatalf("ssr: %s", app.SSR)
	}
}

func TestStyleEntriesAreDeduplicatedByIDAndFollowTheirOwners(t *testing.T) {
	var flag *runtime.Signal[bool]
	card := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Styles: []head.StyleTag{{ID: "box-1", CSS: ".box-1{color:red}"}}})
		return work.BuildElement("div")
	})
	app := runtimetest.New(t, page(work.BuildElement("main", card(), runtime.Component(func(ctx *runtime.Ctx) work.Node {
		flag = runtime.UseSignal(ctx, true)
		return runtime.Show(flag, func() work.Node { return card() })
	})())))
	if strings.Count(app.SSR, "<style") != 1 || !strings.Contains(app.SSR, ".box-1{color:red}") {
		t.Fatalf("one shared sheet expected in the first HTML: %s", app.SSR)
	}
	app.Do(func() { flag.Set(false) })
	if strings.Count(app.Model.HTML(), "<style") != 1 {
		t.Fatalf("the sheet is still used by the first card: %s", app.Model.HTML())
	}
}

func TestStyleTextIsNotHTMLEscapedAndCannotCloseTheElement(t *testing.T) {
	app := runtimetest.New(t, page(runtime.Component(func(ctx *runtime.Ctx) work.Node {
		head.UseHead(ctx, head.Head{Styles: []head.StyleTag{{CSS: "a > b{content:'</style><script>x</script>'}"}}})
		return work.BuildElement("div")
	})()))
	if !strings.Contains(app.SSR, "a > b{") || strings.Contains(app.SSR, "</style><script>") {
		t.Fatalf("ssr = %s", app.SSR)
	}
}
