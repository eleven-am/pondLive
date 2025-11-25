package headers2

import (
	"github.com/eleven-am/pondlive/go/internal/runtime2"
)

// requestCtx is the context for providing request state to child components.
var requestCtx = runtime2.CreateContext[*RequestState](nil)

// UseRequestState returns the request state from context.
// Returns nil if not within a RequestState provider.
func UseRequestState(ctx *runtime2.Ctx) *RequestState {
	return requestCtx.UseContextValue(ctx)
}

// ProvideRequestState provides a RequestState to child components.
// Returns the state and a setter function (setter is typically unused since state is mutated directly).
func ProvideRequestState(ctx *runtime2.Ctx, state *RequestState) (*RequestState, func(*RequestState)) {
	return requestCtx.UseProvider(ctx, state)
}
