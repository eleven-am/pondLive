package runtime_test

import (
	"errors"
	"math/rand"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func randomString(rng *rand.Rand) string {
	pieces := []string{"a", "b", " ", "<", ">", "&", "\"", "'", "</", "-->", "<!--", "&amp;", "&lt;", "é", "\n", "\t", "</style", "</script", "</title", "</textarea", "x=1", " "}
	var sb strings.Builder
	for i, n := 0, rng.Intn(8); i < n; i++ {
		sb.WriteString(pieces[rng.Intn(len(pieces))])
	}
	return sb.String()
}

func parseFragment(t *testing.T, markup string) []*html.Node {
	t.Helper()
	ctx := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(strings.NewReader(markup), ctx)
	if err != nil {
		t.Fatal(err)
	}
	return nodes
}

func textOf(n *html.Node) string {
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode {
			sb.WriteString(c.Data)
		}
	}
	return sb.String()
}

func TestEscapingRoundTripThroughTheHTMLParser(t *testing.T) {
	rng := rand.New(rand.NewSource(11))
	tags := []string{"div", "p", "span", "title", "textarea", "style", "script"}
	for i := 0; i < 600; i++ {
		tag := tags[rng.Intn(len(tags))]
		text := randomString(rng)
		attrValue := randomString(rng)
		s := runtime.NewSession(runtime.Config{Root: func(ctx *Ctx) Node {
			return el(tag, attr("data-v", attrValue), txt(text))
		}})
		snap, err := s.RenderSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		nodes := parseFragment(t, snap.HTML)
		if tag == "title" || tag == "style" || tag == "script" || tag == "textarea" {
			nodes = parseHead(t, snap.HTML)
		}
		var got *html.Node
		for _, n := range nodes {
			if n.Type == html.ElementNode && n.Data == tag {
				got = n
			}
		}
		if got == nil {
			t.Fatalf("%s: element lost in %q", tag, snap.HTML)
		}
		if v, _ := attrFor(got, "data-v"); v != attrValue {
			t.Fatalf("attribute round trip: got %q want %q in %q", v, attrValue, snap.HTML)
		}
		want := text
		if tag == "style" {
			want = strings.NewReplacer("</style", "<\\/style", "</STYLE", "<\\/STYLE").Replace(text)
		}
		if tag == "script" {
			want = strings.NewReplacer("</script", "<\\/script").Replace(text)
		}
		if !utf8.ValidString(text) {
			continue
		}
		if g := textOf(got); g != want {
			t.Fatalf("%s text round trip:\n got  %q\n want %q\n html %q", tag, g, want, snap.HTML)
		}
	}
}

func parseHead(t *testing.T, markup string) []*html.Node {
	t.Helper()
	ctx := &html.Node{Type: html.ElementNode, Data: "head", DataAtom: atom.Head}
	nodes, err := html.ParseFragment(strings.NewReader(markup), ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) == 0 || nodes[0].Data == "html" {
		body := parseFragment(t, markup)
		return body
	}
	return nodes
}

func attrFor(n *html.Node, key string) (string, bool) {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val, true
		}
	}
	return "", false
}

func TestBoundTextEscapesAndSeparatesFromStaticNeighbours(t *testing.T) {
	var v *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		v = runtime.UseSignal(ctx, "<b>")
		return el("p", bound(v), txt("tail"), bound(v), bound(v))
	})
	requireEqual(t, app.SSR, `<p><!--t:1-->&lt;b&gt;<!---->tail<!--t:2-->&lt;b&gt;<!--t:3-->&lt;b&gt;</p>`)

	app.Do(func() { v.Set("") })
	app.Do(func() { v.Set("x&y") })

	requireEqual(t, app.Normalized(), `<p>x&amp;y<!---->tailx&amp;yx&amp;y</p>`)
}

func TestEmptyBoundTextBeforeStaticTextStaysSeparate(t *testing.T) {
	var v *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		v = runtime.UseSignal(ctx, "")
		return el("p", bound(v), txt("tail"))
	})

	app.Do(func() { v.Set("head-") })

	requireEqual(t, app.Normalized(), `<p>head-<!---->tail</p>`)
}

func TestReservedAttributeNames(t *testing.T) {
	body := func(ctx *Ctx) Node {
		return el("div", attr("data-pi", "7"), attr("data-pon", "[]"), attr("title", "ok"))
	}
	dev, _ := rawStart(t, true, body)
	if dev.html != "" {
		t.Fatalf("dev must refuse reserved names, got %q", dev.html)
	}

	prod, err := rawStart(t, false, body)
	if err != nil {
		t.Fatal(err)
	}
	requireEqual(t, prod.html, `<div title="ok"></div>`)
	if !strings.Contains(prod.phases(), "attribute:reserved") {
		t.Fatalf("diagnostics = %s", prod.phases())
	}
}

func TestInvalidAttributeAndTagNamesAreRefused(t *testing.T) {
	for _, bad := range []string{`x" onclick="1`, "a b", "", "<x"} {
		app, err := rawStart(t, false, func(ctx *Ctx) Node { return el("div", attr(bad, "v")) })
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(app.html, "onclick") || strings.Contains(app.html, bad) && bad != "" {
			t.Fatalf("name %q leaked into %q", bad, app.html)
		}
	}
	app, err := rawStart(t, false, func(ctx *Ctx) Node { return el("div onclick=1", txt("x")) })
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(app.html, "onclick") {
		t.Fatalf("tag name leaked: %q", app.html)
	}
}

func TestUserCommentsCannotImpersonateMarkers(t *testing.T) {
	app, err := rawStart(t, false, func(ctx *Ctx) Node {
		return el("div", work.NewComment("t:5"), work.NewComment("/r:3"), work.NewComment("a--b-"), work.NewComment(">x"))
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(app.html, "<!--t:5-->") || strings.Contains(app.html, "<!--/r:3-->") {
		t.Fatalf("marker spoofed: %s", app.html)
	}
	nodes := parseFragment(t, app.html)
	for c := nodes[0].FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.CommentNode {
			t.Fatalf("comment broke out into %v", c.Type)
		}
	}
}

func TestParityCheckFlagsTableTr(t *testing.T) {
	_, err := rawStart(t, true, func(ctx *Ctx) Node {
		return el("table", el("tr", el("td", txt("x"))))
	})
	var pe *runtime.ParityError
	if !errors.As(err, &pe) {
		t.Fatalf("err = %v, want ParityError", err)
	}
}

func TestParityCheckFlagsPInsideDiv(t *testing.T) {
	_, err := rawStart(t, true, func(ctx *Ctx) Node {
		return el("p", el("div", txt("x")))
	})
	var pe *runtime.ParityError
	if !errors.As(err, &pe) {
		t.Fatalf("err = %v, want ParityError", err)
	}
}

func TestParityCheckAcceptsExplicitTbody(t *testing.T) {
	if _, err := rawStart(t, true, func(ctx *Ctx) Node {
		return el("table", el("tbody", el("tr", el("td", txt("x")))))
	}); err != nil {
		t.Fatal(err)
	}
}

func TestParityCheckOnInsertedUnit(t *testing.T) {
	var sig *runtime.Signal[bool]
	app, err := rawStart(t, true, func(ctx *Ctx) Node {
		sig = runtime.UseSignal(ctx, false)
		return el("table", el("tbody", runtime.Show(sig, func() Node { return el("tr", el("td", txt("x"))) })))
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := app.s.Ask(1e9, func() error { sig.Set(true); return app.s.SettleIfNeeded() }); err != nil {
		t.Fatal(err)
	}
	for _, d := range app.diags {
		if strings.Contains(d.Message, "reshape") {
			t.Fatalf("a valid tbody row was flagged: %s", d.Message)
		}
	}
}

func TestRawCannotSpoofIDsOrCloseTheHost(t *testing.T) {
	hostile := `<b data-pi="99" data-pon="[]" data-pscript="{}">x</b><!--t:5--><!--r:6--></div><div data-praw>`
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		return el("section", el("div", work.UnsafeHTML(hostile)), el("footer", txt("after")))
	})

	if strings.Contains(app.SSR, `data-pi="99"`) || strings.Contains(app.SSR, "<!--t:5-->") || strings.Contains(app.SSR, "<!--r:6-->") {
		t.Fatalf("raw content leaked reserved names: %s", app.SSR)
	}
	if !strings.Contains(app.SSR, "<footer>after</footer></section>") {
		t.Fatalf("raw content closed its host: %s", app.SSR)
	}
}

func TestDynamicRawUsesHostEdit(t *testing.T) {
	var v *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		v = runtime.UseSignal(ctx, "<b>one</b>")
		return el("div", work.UnsafeHTML(v))
	})
	requireEqual(t, app.SSR, `<div data-pi="1" data-praw=""><b>one</b></div>`)

	edits := app.Do(func() { v.Set(`<i data-pi="5">two</i>`) })

	if len(edits) != 1 || edits[0].Value != "<i>two</i>" {
		t.Fatalf("edits = %v", edits)
	}
}
