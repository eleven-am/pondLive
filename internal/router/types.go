package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

// Location is re-exported to mirror the public API of the existing router.
type Location = route.Location

type MatchState struct {
	Matched bool
	Pattern string
	Path    string
	Params  map[string]string
	Rest    string
}

type outletRenderer func(*runtime.Ctx) work.Node

type RouteProps struct {
	Path      string
	Component func(*runtime.Ctx, Match) work.Node
}

type Match struct {
	Pattern  string
	Path     string
	Params   map[string]string
	Query    url.Values
	RawQuery string
	Hash     string
	Rest     string
}

type routeEntry struct {
	pattern   string
	fullPath  string
	component func(*runtime.Ctx, Match) work.Node
	children  []work.Node
	slot      string
	key       string
}

const routeMetadataKey = "router:entry"
const slotMetadataKey = "router:slot"

type SlotProps struct {
	Name     string
	Fallback func(*runtime.Ctx) work.Node
}

type slotEntry struct {
	name     string
	fallback func(*runtime.Ctx) work.Node
	routes   []routeEntry
}

type LinkProps struct {
	To      string
	Replace bool
}

type NavLinkProps struct {
	To          string
	Replace     bool
	ClassName   string
	ActiveClass string
	End         bool
}

type RedirectProps struct {
	To      string
	Replace bool
}
