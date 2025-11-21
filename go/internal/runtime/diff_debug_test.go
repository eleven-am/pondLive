package runtime

import (
	"fmt"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

func TestDiffDebug(t *testing.T) {
	var setChildText func(string)

	child := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		value, set := UseState(ctx, "old")
		setChildText = set
		return &dom.StructuredNode{Tag: "span", Text: value()}
	}

	parent := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		childNode := Render(ctx, child, struct{}{})
		return &dom.StructuredNode{
			Tag:      "div",
			Children: []*dom.StructuredNode{childNode},
		}
	}

	sess := NewSession(parent, struct{}{})

	var prevTree, nextTree *dom.StructuredNode
	flushCount := 0
	sess.SetPatchSender(func(patches []dom2diff.Patch) error {
		flushCount++
		fmt.Printf("Flush %d - Patches: %d\n", flushCount, len(patches))
		for i, p := range patches {
			fmt.Printf("  [%d] %+v\n", i, p)
		}
		return nil
	})

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}
	prevTree = sess.prevTree
	fmt.Printf("After first flush, prevTree: %p\n", prevTree)
	if prevTree != nil && len(prevTree.Children) > 0 && len(prevTree.Children[0].Children) > 0 {
		spanNode := prevTree.Children[0].Children[0]
		fmt.Printf("First flush span text: %q (ptr: %p)\n", spanNode.Text, spanNode)
	}

	setChildText("new")
	fmt.Printf("\n=== Second Flush ===\n")

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after child state change failed: %v", err)
	}
	fmt.Printf("Second flush completed\n")
	nextTree = sess.prevTree
	fmt.Printf("After second flush, nextTree: %p\n", nextTree)
	fmt.Printf("prevTree == nextTree: %v\n", prevTree == nextTree)

	if prevTree != nil && len(prevTree.Children) > 0 && len(prevTree.Children[0].Children) > 0 {
		spanNode := prevTree.Children[0].Children[0]
		fmt.Printf("After second flush, prevTree span text: %q (ptr: %p)\n", spanNode.Text, spanNode)
	}
	if nextTree != nil && len(nextTree.Children) > 0 && len(nextTree.Children[0].Children) > 0 {
		spanNode := nextTree.Children[0].Children[0]
		fmt.Printf("After second flush, nextTree span text: %q (ptr: %p)\n", spanNode.Text, spanNode)
	}

	if prevTree != nil && nextTree != nil {
		fmt.Printf("prevTree.Children[0]: %p\n", prevTree.Children[0])
		fmt.Printf("nextTree.Children[0]: %p\n", nextTree.Children[0])
		fmt.Printf("Same wrapper? %v\n", prevTree.Children[0] == nextTree.Children[0])

		fmt.Printf("\nPrevTree structure:\n")
		printTree(prevTree, 0)
		fmt.Printf("\nNextTree structure:\n")
		printTree(nextTree, 0)

		fmt.Printf("\n=== Manual Diff ===\n")
		manualPatches := dom2diff.Diff(prevTree, nextTree)
		fmt.Printf("Manual diff patches: %d\n", len(manualPatches))
		for i, p := range manualPatches {
			fmt.Printf("  [%d] %+v\n", i, p)
		}
	}
}

func printTree(n *dom.StructuredNode, depth int) {
	if n == nil {
		return
	}
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	if n.Tag != "" {
		fmt.Printf("%s<%s> text=%q componentID=%q\n", indent, n.Tag, n.Text, n.ComponentID)
	} else if n.ComponentID != "" {
		fmt.Printf("%s<component id=%q>\n", indent, n.ComponentID)
	} else if n.Text != "" {
		fmt.Printf("%sText: %q\n", indent, n.Text)
	}
	for _, child := range n.Children {
		printTree(child, depth+1)
	}
}
