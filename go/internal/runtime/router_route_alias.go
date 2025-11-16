package runtime

import "github.com/eleven-am/pondlive/go/internal/route"

type (
	// Match represents parsed routing information surfaced to components.
	Match = route.Match
)

// RouteProps configures a route definition for router implementations.
type RouteProps struct {
	Path      string
	Component Component[Match]
}
