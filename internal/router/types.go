package router

import (
	"errors"
	"net/url"

	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

var (
	ErrParamNotFound      = errors.New("router: param not found")
	ErrQueryParamNotFound = errors.New("router: query param not found")
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
	params   map[string]string
	query    url.Values
	RawQuery string
	Hash     string
	Rest     string
}

func (m Match) Param(key string) (string, error) {
	if m.params == nil {
		return "", ErrParamNotFound
	}
	val, ok := m.params[key]
	if !ok {
		return "", ErrParamNotFound
	}
	return val, nil
}

func (m Match) QueryParam(key string) (string, error) {
	if m.query == nil {
		return "", ErrQueryParamNotFound
	}
	val := m.query.Get(key)
	if val == "" {
		if _, ok := m.query[key]; !ok {
			return "", ErrQueryParamNotFound
		}
	}
	return val, nil
}

func (m Match) QueryValues(key string) ([]string, error) {
	if m.query == nil {
		return nil, ErrQueryParamNotFound
	}
	vals, ok := m.query[key]
	if !ok {
		return nil, ErrQueryParamNotFound
	}
	return vals, nil
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
