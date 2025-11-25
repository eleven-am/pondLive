package html2

import "github.com/eleven-am/pondlive/go/internal/work"

type noopItem struct{}

func (noopItem) ApplyTo(*work.Element) {}

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

func Map[T any](xs []T, render func(T) Node) Node {
	if len(xs) == 0 || render == nil {
		return work.NewFragment()
	}
	children := make([]Node, 0, len(xs))
	for _, v := range xs {
		child := render(v)
		if child == nil {
			continue
		}
		children = append(children, child)
	}
	return work.NewFragment(children...)
}

func MapIdx[T any](xs []T, render func(int, T) Node) Node {
	if len(xs) == 0 || render == nil {
		return work.NewFragment()
	}
	children := make([]Node, 0, len(xs))
	for i, v := range xs {
		child := render(i, v)
		if child == nil {
			continue
		}
		children = append(children, child)
	}
	return work.NewFragment(children...)
}
