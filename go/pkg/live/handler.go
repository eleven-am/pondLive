package live

import "github.com/eleven-am/pondlive/go/internal/runtime"

type (
	HandlerFunc   = runtime.HandlerFunc
	HandlerHandle = runtime.HandlerHandle
)

// UseHandler registers an ephemeral HTTP handler scoped to the current component and session.
// Example: handle := UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {...})
func UseHandler(ctx Ctx, method string, chain ...HandlerFunc) HandlerHandle {
	return runtime.UseHandler(ctx, method, chain...)
}
