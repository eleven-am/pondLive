package pkg

import (
	"time"

	"github.com/eleven-am/pondlive/internal/portal"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func Component(fn func(ctx *Ctx) Node) func() Node {
	return runtime.Component(fn)
}

func PropsComponent[P any](fn func(ctx *Ctx, props P) Node) func(P) Node {
	return runtime.PropsComponent(fn)
}

func El(tag string, items ...Item) Node {
	return work.BuildElement(tag, items...)
}

func Text[T any](v T) Node {
	return work.TextOf(v)
}

func Textf(format string, args ...any) Node {
	return work.NewTextf(format, args...)
}

func Comment(value string) Node {
	return work.NewComment(value)
}

func Fragment(children ...Item) Node {
	return work.NewFragment(children...)
}

func Portal(children ...Item) Node {
	return portal.Portal(children...)
}

func Show(cond ReadSignal[bool], then func() Node, opts ...ShowOpt) Node {
	return runtime.Show(cond, then, opts...)
}

func Else(fn func() Node) ShowOpt {
	return runtime.Else(fn)
}

func Switch[K comparable](sel ReadSignal[K], args ...SwitchArg) Node {
	return runtime.Switch(sel, args...)
}

func Case[K comparable](k K, fn func() Node) SwitchArg {
	return runtime.Case(k, fn)
}

func Default(fn func() Node) SwitchArg {
	return runtime.Default(fn)
}

func WaitExit() SwitchArg {
	return runtime.WaitExit()
}

func For[T any](src Source[T], row func(item ReadSignal[T]) Node, opts ...ForOpt) Node {
	return runtime.For(src, row, opts...)
}

func Items[T any, K comparable](items ReadSignal[[]T], key func(T) K) Source[T] {
	return runtime.Items(items, key)
}

func NewFeed[T any, K comparable](key func(T) K, load func() []T) *Feed[T, K] {
	return runtime.NewFeed(key, load)
}

func Exit(d time.Duration) RegionOpt {
	return runtime.Exit(d)
}
