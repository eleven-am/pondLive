package live

import "github.com/eleven-am/pondlive/go/internal/runtime"

type HeaderState = runtime.HeaderState

// UseHeader returns the always-on header state scoped to the current live session.
func UseHeader(ctx Ctx) HeaderState {
	return runtime.UseHeader(ctx)
}
