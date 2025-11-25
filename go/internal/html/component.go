package html

import (
	"github.com/eleven-am/pondlive/go/internal/work"
)

// ComponentNode is a function that takes context and children and returns a node
type ComponentNode = func(ctx *Ctx, children ...Node) Node

// PropsComponentNode is a function that takes context, props, and children and returns a node
type PropsComponentNode[P any] = func(ctx *Ctx, props P, children ...Node) Node

// Component creates a component node with no props.
// fn is the component function with signature: func(ctx *runtime.Ctx, children []work.Node) work.Node
// Returns a function that when called with (ctx, children), creates a work.Component
func Component(fn func(ctx *Ctx, children []work.Node) work.Node) ComponentNode {
	wrappedFn := func(ctx *Ctx, _ any, workChildren []work.Node) work.Node {
		return fn(ctx, workChildren)
	}

	return func(ctx *Ctx, children ...Node) Node {
		return work.Component(wrappedFn, children...)
	}
}

// PropsComponent creates a component node with props.
// fn is the component function with signature: func(ctx *runtime.Ctx, props P, children []work.Node) work.Node
// Returns a function that when called with (ctx, props, children), creates a work.Component
func PropsComponent[P any](fn func(ctx *Ctx, props P, children []work.Node) work.Node) PropsComponentNode[P] {
	wrappedFn := func(ctx *Ctx, propsAny any, workChildren []work.Node) work.Node {
		p, ok := propsAny.(P)
		if !ok {
			var zero P
			p = zero
		}
		return fn(ctx, p, workChildren)
	}

	return func(ctx *Ctx, props P, children ...Node) Node {
		return work.PropsComponent(wrappedFn, props, children...)
	}
}
