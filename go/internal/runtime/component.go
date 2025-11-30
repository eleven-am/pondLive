package runtime

import (
	"github.com/eleven-am/pondlive/go/internal/work"
)

type Node = work.Node

type ComponentWrapper = func(ctx *Ctx, items ...work.Item) Node

type PropsComponentWrapper[P any] = func(ctx *Ctx, props P, items ...work.Item) Node

func Component(fn func(ctx *Ctx, children []work.Node) work.Node) ComponentWrapper {
	wrappedFn := func(ctx *Ctx, _ any, workChildren []work.Node) work.Node {
		return fn(ctx, workChildren)
	}

	return func(ctx *Ctx, items ...work.Item) Node {
		return work.Component(wrappedFn, items...)
	}
}

func PropsComponent[P any](fn func(ctx *Ctx, props P, children []work.Node) work.Node) PropsComponentWrapper[P] {
	wrappedFn := func(ctx *Ctx, propsAny any, workChildren []work.Node) work.Node {
		p, ok := propsAny.(P)
		if !ok {
			var zero P
			p = zero
		}
		return fn(ctx, p, workChildren)
	}

	return func(ctx *Ctx, props P, items ...work.Item) Node {
		return work.PropsComponent(wrappedFn, props, items...)
	}
}
