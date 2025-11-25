package html

import (
	"github.com/eleven-am/pondlive/go/internal/work"
)

// ComponentNode is a function that takes context and children and returns a node
type ComponentNode = func(ctx *Ctx, children ...Node) Node

// PropsComponentNode is a function that takes context, props, and children and returns a node
type PropsComponentNode[P any] = func(ctx *Ctx, props P, children ...Node) Node

// Component creates a component node with no props.
// fn is the component function with signature: func(ctx *runtime.Ctx, props any, children []work.Node) work.Node
// Returns a function that when called with (ctx, children), creates a work.Component
func Component(fn func(ctx *Ctx, children []work.Node) work.Node) ComponentNode {
	return func(ctx *Ctx, children ...Node) Node {
		return work.Component(fn, children...)
	}
}

// PropsComponent creates a component node with props.
// fn is the component function with signature: func(ctx *runtime.Ctx, props P, children []work.Node) work.Node
// Returns a function that when called with (ctx, props, children), creates a work.Component
func PropsComponent[P any](fn func(ctx *Ctx, props P, children []work.Node) work.Node) PropsComponentNode[P] {
	return func(ctx *Ctx, props P, children ...Node) Node {
		return work.PropsComponent(fn, props, children...)
	}
}
