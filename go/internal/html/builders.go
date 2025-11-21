package html

import (
	"fmt"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

// Text creates an escaped text node.
func Text(s string) *dom.StructuredNode { return dom.TextNode(s) }

// Textf formats according to fmt.Sprintf and wraps result in a text node.
func Textf(format string, args ...any) *dom.StructuredNode {
	return dom.TextNode(fmt.Sprintf(format, args...))
}

// Fragment constructs a fragment node from children.
func Fragment(children ...dom.Item) *dom.StructuredNode { return dom.FragmentNode(children...) }

// Comment creates an HTML comment node.
func Comment(value string) *dom.StructuredNode { return dom.CommentNode(value) }

// WrapComponent wraps a component subtree so render passes can attach metadata.
func WrapComponent(id string, child dom.Item) *dom.StructuredNode {
	comp := dom.ComponentNode(id)
	if child != nil {
		child.ApplyTo(comp)
	}
	return comp
}

// If includes the node when cond is true; otherwise it contributes nothing.
func If(cond bool, node dom.Item) dom.Item {
	if cond {
		return node
	}
	return noopNode{}
}

// IfFn evaluates fn when cond is true.
func IfFn(cond bool, fn func() dom.Item) dom.Item {
	if cond && fn != nil {
		return fn()
	}
	return noopNode{}
}

// Ternary returns whenTrue when cond is true, otherwise whenFalse.
// Missing branches fall back to a noop node.
func Ternary(cond bool, whenTrue, whenFalse dom.Item) dom.Item {
	if cond {
		if whenTrue != nil {
			return whenTrue
		}
	} else if whenFalse != nil {
		return whenFalse
	}
	return noopNode{}
}

// TernaryFn evaluates the matching branch when cond is true or false.
func TernaryFn(cond bool, whenTrue, whenFalse func() dom.Item) dom.Item {
	if cond {
		if whenTrue != nil {
			return whenTrue()
		}
	} else if whenFalse != nil {
		return whenFalse()
	}
	return noopNode{}
}

type noopNode struct{}

func (noopNode) ApplyTo(*dom.StructuredNode) {}
func (noopNode) ToHTML() string              { return "" }
func (noopNode) Validate() error             { return nil }

// Map renders a slice into a fragment using render.
func Map[T any](xs []T, render func(T) dom.Item) *dom.StructuredNode {
	if len(xs) == 0 || render == nil {
		return Fragment()
	}
	children := make([]dom.Item, 0, len(xs))
	for _, v := range xs {
		child := render(v)
		if child == nil {
			continue
		}
		children = append(children, child)
	}

	return Fragment(children...)
}

// MapIdx renders a slice with index-aware render function.
func MapIdx[T any](xs []T, render func(int, T) dom.Item) *dom.StructuredNode {
	if len(xs) == 0 || render == nil {
		return Fragment()
	}

	children := make([]dom.Item, 0, len(xs))

	for i, v := range xs {
		child := render(i, v)
		if child == nil {
			continue
		}

		children = append(children, child)
	}

	return Fragment(children...)
}
