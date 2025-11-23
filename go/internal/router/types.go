package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Ctx is the router context type used throughout router.
type Ctx = runtime.Ctx

// Location represents a URL location with path, query parameters, and hash.
type Location struct {
	Path  string
	Query url.Values
	Hash  string
}

// Match contains information about a matched route.
// Passed as props to route components.
type Match struct {
	Pattern  string            // The pattern that matched (e.g., "/users/:id")
	Path     string            // The actual path that was matched (e.g., "/users/123")
	Params   map[string]string // Extracted route parameters (e.g., {"id": "123"})
	Query    url.Values        // Query parameters from the location
	RawQuery string            // Encoded query string
	Hash     string            // Hash fragment from the location
	Rest     string            // Remaining path for wildcard matches
}

// MatchState holds the current matched route information.
// Stored in router state, separate from location (which lives in RequestController).
type MatchState struct {
	Matched bool              // Whether any route matched
	Pattern string            // Matched route pattern
	Params  map[string]string // Route parameters
	Path    string            // Matched path
}

// RouteProps defines properties for a single route.
type RouteProps struct {
	Path      string                   // Route pattern (e.g., "/users/:id" or "./settings")
	Component runtime.Component[Match] // Component to render when matched
}

// RoutesProps defines properties for a Routes group.
type RoutesProps struct {
	Outlet string // Which outlet these routes belong to (default: "default")
}

// Component is a type alias for route components.
type Component[T any] = runtime.Component[T]

// routeEntry is an internal representation of a route.
// Used during collection and trie building.
type routeEntry struct {
	pattern   string
	component runtime.Component[Match]
	children  []*dom.StructuredNode
}

// Metadata keys used to mark route nodes.
const (
	routeMetadataKey = "router:entry"
)
