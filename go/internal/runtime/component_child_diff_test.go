package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

func TestComponentWrapperDiff(t *testing.T) {
	prevSpan := dom.ElementNode("span").WithChildren(dom.TextNode("old"))
	nextSpan := dom.ElementNode("span").WithChildren(dom.TextNode("new"))

	t.Logf("prevSpan ptr: %p, text: %q", prevSpan, prevSpan.Text)
	t.Logf("nextSpan ptr: %p, text: %q", nextSpan, nextSpan.Text)

	prevWrapper := &dom.StructuredNode{
		ComponentID: "child1",
		Children:    []*dom.StructuredNode{prevSpan},
	}

	nextWrapper := &dom.StructuredNode{
		ComponentID: "child1",
		Children:    []*dom.StructuredNode{nextSpan},
	}

	t.Logf("prevWrapper.Children[0] ptr: %p", prevWrapper.Children[0])
	t.Logf("nextWrapper.Children[0] ptr: %p", nextWrapper.Children[0])

	prevDiv := &dom.StructuredNode{
		Tag:      "div",
		Children: []*dom.StructuredNode{prevWrapper},
	}

	nextDiv := &dom.StructuredNode{
		Tag:      "div",
		Children: []*dom.StructuredNode{nextWrapper},
	}

	patches := dom2diff.Diff(prevDiv, nextDiv)

	t.Logf("Patches: %d", len(patches))
	for i, p := range patches {
		t.Logf("  [%d] %+v", i, p)
	}

	spanPatches := dom2diff.Diff(prevSpan, nextSpan)
	t.Logf("Direct span diff patches: %d", len(spanPatches))

	if len(patches) == 0 {
		t.Fatal("expected patches for text change, got 0")
	}
}
