package router

import (
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/work"
)

type Location = route.Location

type MatchState struct {
	Matched bool
	Pattern string
	Path    string
	Params  map[string]string
	Rest    string
}

type LinkProps struct {
	To       string
	Replace  bool
	Children work.Node
}

type NavLinkProps struct {
	To          string
	Replace     bool
	ClassName   string
	ActiveClass string
	End         bool
	Children    work.Node
}

type RedirectProps struct {
	To     string
	Push   bool
	Status int
}

type ProviderProps struct {
	Children work.Node
}
