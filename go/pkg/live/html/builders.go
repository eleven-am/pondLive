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

// If includes the node when cond is true; otherwise it contributes nothing.
func If(cond bool, node Node) Node {
	if cond {
		return node
	}
	return noopNode{}
}

// IfFn evaluates fn when cond is true.
func IfFn(cond bool, fn func() Node) Node {
	if cond && fn != nil {
		return fn()
	}
	return noopNode{}
}

// Ternary returns whenTrue when cond is true, otherwise whenFalse.
// Missing branches fall back to a noop node.
func Ternary(cond bool, whenTrue, whenFalse Node) Node {
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
func TernaryFn(cond bool, whenTrue, whenFalse func() Node) Node {
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

func (noopNode) applyTo(*Element) {}
func (noopNode) isNode()          {}
func (noopNode) privateNodeTag()  {}

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
