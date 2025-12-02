package pkg

import (
	"github.com/eleven-am/pondlive/go/internal/css"
	"github.com/eleven-am/pondlive/go/internal/work"
)

func If(cond bool, node Item) Item {
	return work.If(cond, node)
}

func IfFn(cond bool, fn func() Item) Item {
	return work.IfFn(cond, fn)
}

func Ternary(cond bool, whenTrue, whenFalse Item) Item {
	return work.Ternary(cond, whenTrue, whenFalse)
}

func TernaryFn(cond bool, whenTrue, whenFalse func() Item) Item {
	return work.TernaryFn(cond, whenTrue, whenFalse)
}

func Map[T any](xs []T, render func(T) Item) Node {
	return work.Map(xs, render)
}

func MapIdx[T any](xs []T, render func(int, T) Item) Node {
	return work.MapIdx(xs, render)
}

func CN(classes ...string) string {
	return css.CN(classes...)
}
