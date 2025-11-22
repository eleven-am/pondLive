package headers

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// requestCtx is the context for providing request controller to child components.
var requestCtx = runtime.CreateContext[*RequestController](nil)

// UseRequestController returns the request controller from context.
func UseRequestController(ctx runtime.Ctx) *RequestController {
	return requestCtx.Use(ctx)
}

// ProvideRequestController provides the request controller to child components.
func ProvideRequestController(ctx runtime.Ctx, controller *RequestController, render func(runtime.Ctx) *dom.StructuredNode) *dom.StructuredNode {
	return requestCtx.Provide(ctx, controller, render)
}
