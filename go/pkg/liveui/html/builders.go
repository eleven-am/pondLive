package html

import "fmt"

// Text creates an escaped text node.
func Text(s string) *TextNode { return &TextNode{Value: s} }

// Textf formats according to fmt.Sprintf and wraps result in a text node.
func Textf(format string, args ...any) *TextNode {
	return &TextNode{Value: fmt.Sprintf(format, args...)}
}

// Fragment constructs a fragment node from children.
func Fragment(children ...Node) *FragmentNode { return &FragmentNode{Children: children} }

// If includes the item when cond is true; otherwise it contributes nothing.
func If(cond bool, item Item) Item {
	if cond {
		return item
	}
	return noopItem{}
}

// IfFn evaluates fn when cond is true.
func IfFn(cond bool, fn func() Item) Item {
	if cond && fn != nil {
		return fn()
	}
	return noopItem{}
}

type noopItem struct{}

func (noopItem) applyTo(*Element) {}

// Map renders a slice into a fragment using render.
func Map[T any](xs []T, render func(T) Node) Item {
	if len(xs) == 0 || render == nil {
		return Fragment()
	}
	children := make([]Node, 0, len(xs))
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
func MapIdx[T any](xs []T, render func(int, T) Node) Item {
	if len(xs) == 0 || render == nil {
		return Fragment()
	}
	children := make([]Node, 0, len(xs))
	for i, v := range xs {
		child := render(i, v)
		if child == nil {
			continue
		}
		children = append(children, child)
	}
	return Fragment(children...)
}
