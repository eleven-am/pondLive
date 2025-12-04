package pkg

import (
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type ComponentNode = runtime.ComponentWrapper

type PropsComponentNode[P any] = runtime.PropsComponentWrapper[P]

func Component(fn func(ctx *Ctx, children []Item) Node) ComponentNode {
	return runtime.Component(func(ctx *runtime.Ctx, children []work.Item) work.Node {
		items := make([]Item, len(children))
		for i, c := range children {
			items[i] = c
		}
		return fn(ctx, items)
	})
}

func PropsComponent[P any](fn func(ctx *Ctx, props P, children []Item) Node) PropsComponentNode[P] {
	return runtime.PropsComponent(func(ctx *runtime.Ctx, props P, children []work.Item) work.Node {
		items := make([]Item, len(children))
		for i, c := range children {
			items[i] = c
		}
		return fn(ctx, props, items)
	})
}
