package pkg_test

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/pkg"
	"github.com/eleven-am/pondlive/pkg/livetest"
)

func TestScopeCSSIsAPureFunctionOfItsKeyAndText(t *testing.T) {
	a := pkg.ScopeCSS("Card:.box{color:red}", ".box{color:red}")
	b := pkg.ScopeCSS("Card:.box{color:red}", ".box{color:red}")
	other := pkg.ScopeCSS("Other:.box{color:red}", ".box{color:red}")
	if a.Hash == "" || a.Hash != b.Hash || a.Text != b.Text {
		t.Fatalf("not deterministic: %+v %+v", a, b)
	}
	if a.Hash == other.Hash {
		t.Fatal("the key must scope the hash")
	}
	if got := a.Class("box"); got != "box-"+a.Hash {
		t.Fatalf("class = %q", got)
	}
	if !strings.Contains(a.Text, ".box-"+a.Hash) && !strings.Contains(a.Text, "box-"+a.Hash) {
		t.Fatalf("selectors were not scoped: %s", a.Text)
	}
}

func TestUseStylesLandsInTheFirstHTMLThroughUseHead(t *testing.T) {
	card := pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		st := pkg.UseStyles(ctx, ".box{color:red}")
		return pkg.Div(pkg.Class(st.Class("box")))
	})
	page := livetest.Mount(t, pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		return pkg.Div(card(), card())
	}))
	html := page.HTML()
	if strings.Count(html, "<style") != 1 || !strings.Contains(html, "color:red") {
		t.Fatalf("one scoped sheet expected: %s", html)
	}
}

func TestRenderHTMLRendersAnyNodeStatically(t *testing.T) {
	got := pkg.RenderHTML(pkg.Svg(pkg.Attr("viewBox", "0 0 1 1"), pkg.Path(pkg.Attr("d", "M0 0"))))
	if got != `<svg viewBox="0 0 1 1"><path d="M0 0"></path></svg>` {
		t.Fatalf("html = %s", got)
	}
}
