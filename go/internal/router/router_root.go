package router

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Type aliases for convenience
type (
	Ctx     = runtime.Ctx
	Cleanup = runtime.Cleanup
)

// Old router state types removed - now using consolidated RouterState in state.go

type routerProps struct {
	Children []*dom.StructuredNode
}

// Legacy Router component removed - router state now provided at documentRoot

func Router(ctx Ctx, children ...*dom.StructuredNode) *dom.StructuredNode {

	return renderRouterChildren(ctx, children...)
}
