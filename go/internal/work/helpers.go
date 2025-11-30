package work

import "reflect"

var OnMapNonNodeDrop func(itemType string, index int)

// Conditional rendering helpers

type noopItem struct{}

func (noopItem) ApplyTo(*Element) {}

// If includes the node when cond is true; otherwise it contributes nothing.
func If(cond bool, node Item) Item {
	if cond {
		return node
	}
	return noopItem{}
}

// IfFn evaluates fn when cond is true.
// Note: fn returns Node, but nodes implement Item via ApplyTo
func IfFn(cond bool, fn func() Item) Item {
	if cond && fn != nil {
		return fn()
	}
	return noopItem{}
}

// Ternary returns whenTrue when cond is true, otherwise whenFalse.
func Ternary(cond bool, whenTrue, whenFalse Item) Item {
	if cond && whenTrue != nil {
		return whenTrue
	} else if whenFalse != nil {
		return whenFalse
	}
	return noopItem{}
}

// TernaryFn evaluates the matching branch when cond is true or false.
func TernaryFn(cond bool, whenTrue, whenFalse func() Item) Item {
	if cond && whenTrue != nil {
		return whenTrue()
	} else if whenFalse != nil {
		return whenFalse()
	}
	return noopItem{}
}

// List rendering helpers

// Map renders a slice into a fragment using render.
func Map[T any](xs []T, render func(T) Item) *Fragment {
	if len(xs) == 0 || render == nil {
		return NewFragment()
	}
	children := make([]Node, 0, len(xs))
	for i, v := range xs {
		child := render(v)
		if child == nil {
			continue
		}

		if node, ok := child.(Node); ok {
			children = append(children, node)
		} else if OnMapNonNodeDrop != nil {
			OnMapNonNodeDrop(reflect.TypeOf(child).String(), i)
		}
	}
	return &Fragment{Children: children}
}

// MapIdx renders a slice with index-aware render function.
func MapIdx[T any](xs []T, render func(int, T) Item) *Fragment {
	if len(xs) == 0 || render == nil {
		return NewFragment()
	}
	children := make([]Node, 0, len(xs))
	for i, v := range xs {
		child := render(i, v)
		if child == nil {
			continue
		}

		if node, ok := child.(Node); ok {
			children = append(children, node)
		} else if OnMapNonNodeDrop != nil {
			OnMapNonNodeDrop(reflect.TypeOf(child).String(), i)
		}
	}
	return &Fragment{Children: children}
}
