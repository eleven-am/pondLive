package html

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

type ComponentNode = runtime.ComponentWrapper

type PropsComponentNode[P any] = runtime.PropsComponentWrapper[P]

var Component = runtime.Component

func PropsComponent[P any](fn func(ctx *Ctx, props P, children []Node) Node) PropsComponentNode[P] {
	return runtime.PropsComponent(fn)
}
