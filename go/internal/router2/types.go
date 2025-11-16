package router2

import runtime "github.com/eleven-am/pondlive/go/internal/runtime"

// Re-export runtime router types so router2 can remain API-compatible while we
// iterate on the internal implementation.
type (
	Match      = runtime.Match
	RouteProps = runtime.RouteProps
)
