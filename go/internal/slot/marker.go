package slot

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// slotMarker is a marker that tags content for a named slot.
// It implements dom.Item interface via ApplyTo method.
type slotMarker struct {
	name     string
	children []dom.Item
}

// Slot creates a named slot marker.
// Content provided in children will be rendered in the named slot.
//
// Example:
//
//	Slot("header", h.H2(h.Text("Title")), h.P(h.Text("Subtitle")))
func Slot(name string, children ...dom.Item) dom.Item {
	return slotMarker{
		name:     name,
		children: children,
	}
}

// ApplyTo implements the dom.Item interface.
// Creates a fragment node with slot metadata to mark the content.
func (s slotMarker) ApplyTo(parent *dom.StructuredNode) {

	fragment := dom.FragmentNode()

	if fragment.Metadata == nil {
		fragment.Metadata = make(map[string]any)
	}
	fragment.Metadata[slotNameKey] = s.name

	for _, child := range s.children {
		if child != nil {
			child.ApplyTo(fragment)
		}
	}

	parent.Children = append(parent.Children, fragment)
}

// scopedSlotMarker is a marker that provides a function to render slot content with data.
// It implements dom.Item interface via ApplyTo method.
type scopedSlotMarker[T any] struct {
	name string
	fn   func(T) *dom.StructuredNode
}

// ScopedSlot creates a scoped slot marker.
// The provided function receives data from the component and returns content to render.
//
// Example:
//
//	ScopedSlot("actions", func(row TableRow) *dom.StructuredNode {
//	    return h.Button(h.Text("Edit"))
//	})
func ScopedSlot[T any](name string, fn func(T) *dom.StructuredNode) dom.Item {
	return scopedSlotMarker[T]{
		name: name,
		fn:   fn,
	}
}

// ApplyTo implements the dom.Item interface.
// Creates a fragment node with scoped slot metadata.
func (s scopedSlotMarker[T]) ApplyTo(parent *dom.StructuredNode) {

	fragment := dom.FragmentNode()

	if fragment.Metadata == nil {
		fragment.Metadata = make(map[string]any)
	}
	fragment.Metadata[scopedSlotNameKey] = s.name
	fragment.Metadata[scopedSlotFuncKey] = s.fn

	parent.Children = append(parent.Children, fragment)
}
