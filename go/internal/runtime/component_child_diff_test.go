package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom2/diff"
)

func TestComponentWrapperDiff(t *testing.T) {
	prevSpan := dom2.ElementNode("span").WithChildren(dom2.TextNode("old"))
	nextSpan := dom2.ElementNode("span").WithChildren(dom2.TextNode("new"))

	t.Logf("prevSpan ptr: %p, text: %q", prevSpan, prevSpan.Text)
	t.Logf("nextSpan ptr: %p, text: %q", nextSpan, nextSpan.Text)

	prevWrapper := &dom2.StructuredNode{
		ComponentID: "child1",
		Children:    []*dom2.StructuredNode{prevSpan},
	}

	nextWrapper := &dom2.StructuredNode{
		ComponentID: "child1",
		Children:    []*dom2.StructuredNode{nextSpan},
	}

	t.Logf("prevWrapper.Children[0] ptr: %p", prevWrapper.Children[0])
	t.Logf("nextWrapper.Children[0] ptr: %p", nextWrapper.Children[0])

	prevDiv := &dom2.StructuredNode{
		Tag:      "div",
		Children: []*dom2.StructuredNode{prevWrapper},
	}

	nextDiv := &dom2.StructuredNode{
		Tag:      "div",
		Children: []*dom2.StructuredNode{nextWrapper},
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
