package runtime

import (
	"github.com/eleven-am/pondlive/internal/work"
)

type Node = work.Node

type ComponentWrapper = func(ctx *Ctx, items ...work.Item) Node

type PropsComponentWrapper[P any] = func(ctx *Ctx, props P, items ...work.Item) Node

func Component(fn func(ctx *Ctx, children []work.Item) work.Node) ComponentWrapper {
	name := captureComponentName(3)

	wrappedFn := func(ctx *Ctx, _ any, children []work.Item) work.Node {
		return fn(ctx, children)
	}

	return func(ctx *Ctx, items ...work.Item) Node {
		comp := work.Component(wrappedFn, items...)
		comp.Name = name
		return comp
	}
}

func PropsComponent[P any](fn func(ctx *Ctx, props P, children []work.Item) work.Node) PropsComponentWrapper[P] {
	name := captureComponentName(3)

	wrappedFn := func(ctx *Ctx, propsAny any, children []work.Item) work.Node {
		p, ok := propsAny.(P)
		if !ok {
			var zero P
			p = zero
		}
		return fn(ctx, p, children)
	}

	return func(ctx *Ctx, props P, items ...work.Item) Node {
		comp := work.PropsComponent(wrappedFn, props, items...)
		comp.Name = name
		return comp
	}
}
