package pkg

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

type ComponentNode = runtime.ComponentWrapper

type PropsComponentNode[P any] = runtime.PropsComponentWrapper[P]

func Component(fn func(ctx *Ctx, children []Node) Node) ComponentNode {
	return runtime.Component(fn)
}

func PropsComponent[P any](fn func(ctx *Ctx, props P, children []Node) Node) PropsComponentNode[P] {
	return runtime.PropsComponent(fn)
}
