package headers

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// requestCtx is the context for providing request state to child components.
var requestCtx = runtime.CreateContext[*RequestState](nil)

// UseRequestState returns the request state from context.
// Returns nil if not within a RequestState provider.
func UseRequestState(ctx *runtime.Ctx) *RequestState {
	return requestCtx.UseContextValue(ctx)
}

// UseProvideRequestState provides a RequestState to child components.
// Returns the state and a setter function (setter is typically unused since state is mutated directly).
func UseProvideRequestState(ctx *runtime.Ctx, state *RequestState) (*RequestState, func(*RequestState)) {
	return requestCtx.UseProvider(ctx, state)
}
