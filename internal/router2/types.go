package router2

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// Location represents a URL location with path, query parameters, and hash.
type Location struct {
	Path  string
	Query url.Values
	Hash  string
}

// MatchState holds the current matched route information (stored in context).
type MatchState struct {
	Matched bool              // Whether any route matched
	Pattern string            // Matched route pattern
	Params  map[string]string // Route parameters
	Path    string            // Matched path
	Rest    string            // Remaining path for wildcard matches
}

// Match contains full match info passed to route components as props.
type Match struct {
	Pattern  string            // The pattern that matched (e.g., "/users/:id")
	Path     string            // The actual path that was matched (e.g., "/users/123")
	Params   map[string]string // Extracted route parameters (e.g., {"id": "123"})
	Query    url.Values        // Query parameters from the location
	RawQuery string            // Encoded query string
	Hash     string            // Hash fragment from the location
	Rest     string            // Remaining path for wildcard matches
}

// RouteProps defines properties for a single route.
type RouteProps struct {
	Path      string                                            // Route pattern (e.g., "/users/:id" or "./settings")
	Component func(*runtime2.Ctx, Match, []work.Node) work.Node // Component to render when matched
}

// RoutesProps defines properties for a Routes group.
type RoutesProps struct {
	Outlet string // Which outlet these routes belong to (default: "default")
}

// LinkProps defines properties for the Link component.
type LinkProps struct {
	To      string // Target href
	Replace bool   // Use replaceState instead of pushState
}

// NavPayload for Bus communication (inbound navigation requests).
type NavPayload struct {
	Path    string `json:"path"`
	Query   string `json:"query"`
	Hash    string `json:"hash"`
	Replace bool   `json:"replace"`
}

// ToLocation converts NavPayload to Location.
func (n NavPayload) ToLocation() *Location {
	query, _ := url.ParseQuery(n.Query)
	return &Location{
		Path:  n.Path,
		Query: query,
		Hash:  n.Hash,
	}
}

// NavResponse for Bus communication (outbound confirmation).
type NavResponse struct {
	Path    string `json:"path"`
	Query   string `json:"query"`
	Hash    string `json:"hash"`
	Replace bool   `json:"replace"`
}

// routeEntry is an internal representation of a route.
// Used during collection and trie building.
type routeEntry struct {
	pattern   string
	component func(*runtime2.Ctx, Match, []work.Node) work.Node
	children  []work.Node
}

// Metadata keys used to mark route nodes.
const (
	routeMetadataKey = "router:entry"
)
