package runtime

import (
	"fmt"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

func TestChildDebug(t *testing.T) {
	var setChildText func(string)

	child := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		value, set := UseState(ctx, "old")
		setChildText = set
		fmt.Printf("Child rendering with value: %s\n", value())
		return dom.ElementNode("span").WithChildren(dom.TextNode(value()))
	}

	parent := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		fmt.Println("Parent rendering")
		childNode := Render(ctx, child, struct{}{})
		return &dom.StructuredNode{
			Tag:      "div",
			Children: []*dom.StructuredNode{childNode},
		}
	}

	sess := NewSession(parent, struct{}{})

	var batches [][]dom2diff.Patch
	sess.SetPatchSender(func(patches []dom2diff.Patch) error {
		fmt.Printf("Patch batch %d: %d patches\n", len(batches)+1, len(patches))
		for i, p := range patches {
			fmt.Printf("  Patch %d: %+v\n", i, p)
		}
		copyBatch := append([]dom2diff.Patch(nil), patches...)
		batches = append(batches, copyBatch)
		return nil
	})

	fmt.Println("=== First Flush ===")
	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	fmt.Println("\n=== Set Child Text ===")
	setChildText("new")

	fmt.Println("\n=== Second Flush ===")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after child state change failed: %v", err)
	}

	fmt.Printf("\nTotal batches: %d\n", len(batches))
	if len(batches) < 2 {
		t.Fatalf("expected at least two patch batches, got %d", len(batches))
	}
}
