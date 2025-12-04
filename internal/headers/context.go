package headers

import (
	"github.com/eleven-am/pondlive/internal/runtime"
)

var requestCtx = runtime.CreateContext[*RequestState](nil)

func UseRequestState(ctx *runtime.Ctx) *RequestState {
	return requestCtx.UseContextValue(ctx)
}
