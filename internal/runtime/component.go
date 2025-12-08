package runtime

import (
	"github.com/eleven-am/pondlive/internal/work"
)

type Node = work.Node

type ComponentWrapper = func(ctx *Ctx, items ...work.Item) Node

type PropsComponentWrapper[P any] = func(ctx *Ctx, props P, items ...work.Item) Node

func Component(fn func(ctx *Ctx, children []work.Item) work.Node) ComponentWrapper {
	wrappedFn := func(ctx *Ctx, _ any, children []work.Item) work.Node {
		return fn(ctx, children)
	}

	return func(ctx *Ctx, items ...work.Item) Node {
		return work.Component(wrappedFn, items...)
	}
}

func PropsComponent[P any](fn func(ctx *Ctx, props P, children []work.Item) work.Node) PropsComponentWrapper[P] {
	wrappedFn := func(ctx *Ctx, propsAny any, children []work.Item) work.Node {
		p, ok := propsAny.(P)
		if !ok {
			var zero P
			p = zero
		}
		return fn(ctx, p, children)
	}

	return func(ctx *Ctx, props P, items ...work.Item) Node {
		return work.PropsComponent(wrappedFn, props, items...)
	}
}
