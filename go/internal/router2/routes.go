package router2

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/route"
)

// RouteRenderer produces an HTML node for the matched route.
type RouteRenderer func(RouteRenderContext) dom.Node

// RouteRenderContext supplies the route match and a callback for rendering an
// outlet consisting of nested child routes.
type RouteRenderContext struct {
	Match        route.Match
	RenderOutlet func() dom.Node
}

// RouteDef describes a route pattern, its renderer, and nested child routes.
type RouteDef struct {
	Pattern  string
	Render   RouteRenderer
	Children []*RouteDef
}

// Route constructs a RouteDef with nested children.
func Route(pattern string, render RouteRenderer, children ...*RouteDef) *RouteDef {
	return &RouteDef{Pattern: pattern, Render: render, Children: children}
}

// RouteTree holds a compiled hierarchy of routes with resolved patterns.
type RouteTree struct {
	entries []*compiledRoute
}

type compiledRoute struct {
	pattern  string
	render   RouteRenderer
	children []*compiledRoute
}

// RouteMatch captures the resolved route stack for a location.
type RouteMatch struct {
	route *compiledRoute
	Match route.Match
	Child *RouteMatch
}

// Render executes the renderer chain to produce the DOM subtree.
func (m *RouteMatch) Render() dom.Node {
	if m == nil || m.route == nil {
		return &dom.FragmentNode{}
	}
	render := m.route.render
	if render == nil {
		return outletNode(m.Child)
	}
	ctx := RouteRenderContext{
		Match: m.Match,
		RenderOutlet: func() dom.Node {
			return outletNode(m.Child)
		},
	}
	node := render(ctx)
	if node == nil {
		return &dom.FragmentNode{}
	}
	return node
}

func outletNode(child *RouteMatch) dom.Node {
	if child == nil {
		return &dom.FragmentNode{}
	}
	return child.Render()
}

// Params flattens parameter maps along the matched chain.
func (m *RouteMatch) Params() map[string]string {
	if m == nil {
		return map[string]string{}
	}
	params := map[string]string{}
	accumulateParams(params, m)
	return params
}

func accumulateParams(dst map[string]string, node *RouteMatch) {
	if node == nil {
		return
	}
	if len(node.Match.Params) > 0 {
		for k, v := range node.Match.Params {
			dst[k] = v
		}
	}
	accumulateParams(dst, node.Child)
}

// CompileTree resolves relative patterns and prepares the route tree for matching.
func CompileTree(base string, defs ...*RouteDef) *RouteTree {
	entries := compileEntries(base, defs)
	return &RouteTree{entries: entries}
}

func compileEntries(base string, defs []*RouteDef) []*compiledRoute {
	if len(defs) == 0 {
		return nil
	}
	entries := make([]*compiledRoute, 0, len(defs))
	for _, def := range defs {
		if def == nil {
			continue
		}
		pattern := resolvePattern(def.Pattern, base)
		compiled := &compiledRoute{
			pattern:  pattern,
			render:   def.Render,
			children: compileEntries(pattern, def.Children),
		}
		entries = append(entries, compiled)
	}
	return entries
}

// Resolve walks the tree for the provided location using the given RouterStore.
func (t *RouteTree) Resolve(store *RouterStore) *RouteMatch {
	if t == nil || len(t.entries) == 0 || store == nil {
		return nil
	}
	loc := store.Location()
	rawQuery := encodeQuery(loc.Query)
	match := matchEntries(t.entries, loc, rawQuery)
	if match == nil {
		store.SetParams(nil)
		return nil
	}
	store.SetParams(match.Params())
	return match
}

func matchEntries(entries []*compiledRoute, loc Location, rawQuery string) *RouteMatch {
	if len(entries) == 0 {
		return nil
	}
	var chosen *RouteMatch
	for _, entry := range entries {
		parsed, err := route.Parse(entry.pattern, loc.Path, rawQuery)
		if err != nil {
			continue
		}
		if chosen == nil || route.Prefer(parsed, chosen.Match) {
			branch := &RouteMatch{route: entry, Match: parsed}
			if node := matchEntries(entry.children, loc, rawQuery); node != nil {
				branch.Child = node
			}
			chosen = branch
		}
	}
	return chosen
}

func encodeQuery(values url.Values) string {
	if len(values) == 0 {
		return ""
	}
	return values.Encode()
}

// RoutesNode constructs a placeholder node that resolves the provided route
// definitions using a RouterStore.
func RoutesNode(base string, defs ...*RouteDef) dom.Node {
	return &routesNode{base: base, defs: defs}
}

type routesNode struct {
	base string
	defs []*RouteDef
}

func (n *routesNode) ApplyTo(e *dom.Element) { e.Children = append(e.Children, n) }
func (*routesNode) isNode()                  {}
func (*routesNode) privateNodeTag()          {}

func (n *routesNode) resolve(store *RouterStore) dom.Node {
	if n == nil {
		return &dom.FragmentNode{}
	}
	return renderRoutesWithBase(store, n.base, n.defs...)
}
