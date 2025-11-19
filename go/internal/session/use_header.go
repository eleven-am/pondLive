package session

import "github.com/eleven-am/pondlive/go/internal/runtime"

// UseHeader returns the header state from the nearest HeaderContext provider.
func UseHeader(ctx runtime.Ctx) HeaderState {
	return HeaderContext.Use(ctx)
}
