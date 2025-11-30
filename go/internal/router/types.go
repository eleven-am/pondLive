package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type Location struct {
	Path  string
	Query url.Values
	Hash  string
}

type MatchState struct {
	Matched bool
	Pattern string
	Params  map[string]string
	Path    string
	Rest    string
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

type RouteProps struct {
	Path      string
	Component func(*runtime.Ctx, Match, []work.Node) work.Node
}

type RoutesProps struct {
	Outlet string
}

type LinkProps struct {
	To      string
	Replace bool
}

type routeEntry struct {
	pattern   string
	component func(*runtime.Ctx, Match, []work.Node) work.Node
	children  []work.Node
}

const (
	routeMetadataKey = "router:entry"
)
