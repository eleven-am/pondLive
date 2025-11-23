package slot

import (
	"fmt"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

const maxStripDepth = 1000 // Prevent stack overflow on pathological trees

// stripMarkerMetadata removes slot marker metadata from a node tree.
// This prevents marker metadata from leaking to rendered output.
// Has depth limit to prevent stack overflow on deeply nested trees.
func stripMarkerMetadata(node *dom.StructuredNode) {
	stripMarkerMetadataDepth(node, 0)
}

func stripMarkerMetadataDepth(node *dom.StructuredNode, depth int) {
	if node == nil {
		return
	}

	if depth > maxStripDepth {

		panic(fmt.Sprintf("slot: stripMarkerMetadata exceeded max depth %d (possible circular reference or pathological tree)", maxStripDepth))
	}

	if node.Metadata != nil {
		delete(node.Metadata, slotNameKey)
		delete(node.Metadata, scopedSlotNameKey)
		delete(node.Metadata, scopedSlotFuncKey)

		if len(node.Metadata) == 0 {
			node.Metadata = nil
		}
	}

	for _, child := range node.Children {
		stripMarkerMetadataDepth(child, depth+1)
	}
}

// renderSlotContent converts SlotContent to a renderable node.
// Returns an empty fragment if content is nil or empty.
// Clones nodes to prevent shared pointers and strips marker metadata.
func renderSlotContent(content *SlotContent) *dom.StructuredNode {
	if content == nil || len(content.nodes) == 0 {
		return dom.FragmentNode()
	}

	cloned := make([]*dom.StructuredNode, len(content.nodes))
	for i, node := range content.nodes {
		cloned[i] = runtime.CloneTree(node)

		stripMarkerMetadata(cloned[i])
	}

	if len(cloned) == 1 {
		return cloned[0]
	}

	fragment := dom.FragmentNode()
	fragment.Children = cloned
	return fragment
}

// nodeToSlotContent converts a StructuredNode to SlotContent.
func nodeToSlotContent(node *dom.StructuredNode) *SlotContent {
	if node == nil {
		return &SlotContent{nodes: nil}
	}

	if node.Fragment {
		return &SlotContent{nodes: node.Children}
	}

	return &SlotContent{nodes: []*dom.StructuredNode{node}}
}
