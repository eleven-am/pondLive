package router2

import "github.com/eleven-am/pondlive/go/internal/dom"

// RenderRoutes evaluates the provided route definitions against the store and
// returns the rendered DOM tree. When no route matches, an empty fragment is
// returned.
func RenderRoutes(store *RouterStore, defs ...*RouteDef) dom.Node {
	return renderRoutesWithBase(store, "/", defs...)
}

func renderRoutesWithBase(store *RouterStore, base string, defs ...*RouteDef) dom.Node {
	if store == nil || len(defs) == 0 {
		return &dom.FragmentNode{}
	}
	tree := compileRoutesCached(base, defs...)
	match := tree.Resolve(store)
	if match == nil {
		return &dom.FragmentNode{}
	}
	return match.Render()
}
