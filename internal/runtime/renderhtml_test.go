package runtime_test

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestRenderHTMLIsAStaticRenderOfAnyNode(t *testing.T) {
	node := work.BuildElement("svg", work.Attribute("viewBox", "0 0 10 10"),
		work.BuildElement("path", work.Attribute("d", "M0 0L1 1"), work.Class("a", "a", "b")),
		work.BuildElement("text", work.NewText("a < b & c")),
	)
	got := runtime.RenderHTML(node)
	want := `<svg viewBox="0 0 10 10"><path d="M0 0L1 1" class="a b"></path><text>a &lt; b &amp; c</text></svg>`
	if got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
	if strings.Contains(got, "data-pi") {
		t.Fatalf("a static render carries no ids: %s", got)
	}
}
