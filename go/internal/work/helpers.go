package work

import "reflect"

var OnMapNonNodeDrop func(itemType string, index int)

type noopItem struct{}

func (noopItem) ApplyTo(*Element) {}

func If(cond bool, node Item) Item {
	if cond {
		return node
	}
	return noopItem{}
}

func IfFn(cond bool, fn func() Item) Item {
	if cond && fn != nil {
		return fn()
	}
	return noopItem{}
}

func Ternary(cond bool, whenTrue, whenFalse Item) Item {
	if cond && whenTrue != nil {
		return whenTrue
	} else if whenFalse != nil {
		return whenFalse
	}
	return noopItem{}
}

func TernaryFn(cond bool, whenTrue, whenFalse func() Item) Item {
	if cond && whenTrue != nil {
		return whenTrue()
	} else if whenFalse != nil {
		return whenFalse()
	}
	return noopItem{}
}

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

func NodesToItems[T Node](nodes []T) []Item {
	items := make([]Item, len(nodes))
	for i, n := range nodes {
		items[i] = n
	}
	return items
}
