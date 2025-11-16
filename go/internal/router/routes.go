package router

import (
	"fmt"
	"net/url"
	"reflect"
	"runtime"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/route"
	runtimepkg "github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

var outletCtx = runtimepkg.NewContext(func() h.Node { return h.Fragment() })

type routeEntryNode struct {
	props    RouteProps
	children []h.Node
}

func (n *routeEntryNode) ApplyTo(e *dom.Element) { e.Children = append(e.Children, n) }
func (*routeEntryNode) isNode()                  {}
func (*routeEntryNode) privateNodeTag()          {}

// Route describes a single route definition.
func Route(ctx runtimepkg.Ctx, props RouteProps, children ...h.Node) h.Node {
	entry := &routeEntryNode{props: props}
	entry.props.Path = strings.TrimSpace(entry.props.Path)
	if len(children) > 0 {
		entry.children = append([]h.Node(nil), children...)
	}
	return entry
}

// RouteRenderer produces an HTML node for the matched route.
type RouteRenderer func(runtimepkg.Ctx, RouteRenderContext) h.Node

// RouteRenderContext supplies the route match and a callback for rendering an
// outlet consisting of nested child routes.
type RouteRenderContext struct {
	Match        Match
	RenderOutlet func() h.Node
}

// RouteDef describes a route pattern, its renderer, and nested child routes.
type RouteDef struct {
	Pattern  string
	Render   RouteRenderer
	Children []*RouteDef
	Identity string
}

// Routes groups Route definitions under the current base path.
func Routes(ctx runtimepkg.Ctx, children ...h.Node) h.Node {
	base := routeBaseCtx.Use(ctx)
	if ctx.ComponentID() == "" {
		base = peekContextlessBase()
	}
	node := &routesNode{base: base}
	if len(children) > 0 {
		node.children = append([]h.Node(nil), children...)
	}
	return node
}

// Outlet renders nested routes within the active Route component.
func Outlet(ctx runtimepkg.Ctx) h.Node {
	if ctx.ComponentID() == "" {
		if render := peekContextlessOutlet(); render != nil {
			return render()
		}
		return h.Fragment()
	}
	render := outletCtx.Use(ctx)
	if render == nil {
		return h.Fragment()
	}
	return render()
}

// DefineRoute constructs a RouteDef with nested children for tests or manual resolution.
func DefineRoute(pattern string, render RouteRenderer, children ...*RouteDef) *RouteDef {
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
	Match Match
	Child *RouteMatch
}

// Render executes the renderer chain to produce the DOM subtree.
func (m *RouteMatch) Render(ctx runtimepkg.Ctx) dom.Node {
	if m == nil || m.route == nil {
		return &dom.FragmentNode{}
	}
	render := m.route.render
	if render == nil {
		return outletNode(ctx, m.Child)
	}
	rctx := RouteRenderContext{
		Match: m.Match,
		RenderOutlet: func() h.Node {
			return outletNode(ctx, m.Child)
		},
	}
	node := render(ctx, rctx)
	if node == nil {
		return &dom.FragmentNode{}
	}
	return node
}

func outletNode(ctx runtimepkg.Ctx, child *RouteMatch) dom.Node {
	if child == nil {
		return h.Fragment()
	}
	return child.Render(ctx)
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
func RoutesNode(base string, children ...h.Node) dom.Node {
	node := &routesNode{base: base}
	if len(children) > 0 {
		node.children = append([]h.Node(nil), children...)
	}
	return node
}

type routesNode struct {
	base     string
	children []h.Node
}

func (n *routesNode) ApplyTo(e *dom.Element) { e.Children = append(e.Children, n) }
func (*routesNode) isNode()                  {}
func (*routesNode) privateNodeTag()          {}

func (n *routesNode) resolve(ctx runtimepkg.Ctx, store *RouterStore) dom.Node {
	if n == nil {
		return &dom.FragmentNode{}
	}
	defs := collectRouteDefs(n.children, n.base)
	return renderRoutesWithBase(ctx, store, n.base, defs)
}

func collectRouteDefs(nodes []h.Node, base string) []*RouteDef {
	if len(nodes) == 0 {
		return nil
	}
	var defs []*RouteDef
	for _, node := range nodes {
		switch v := node.(type) {
		case *routeEntryNode:
			if v == nil || v.props.Component == nil {
				continue
			}
			pattern := resolvePattern(v.props.Path, base)
			childDefs := collectRouteDefs(v.children, pattern)
			defs = append(defs, &RouteDef{
				Pattern:  pattern,
				Render:   componentRenderer(pattern, v.props.Component),
				Children: childDefs,
				Identity: componentIdentity(v.props.Component),
			})
		case *dom.FragmentNode:
			defs = append(defs, collectRouteDefs(v.Children, base)...)
		case *routesNode:
			defs = append(defs, collectRouteDefs(v.children, base)...)
		}
	}
	return defs
}

func componentRenderer(pattern string, component runtimepkg.Component[Match]) RouteRenderer {
	if component == nil {
		return nil
	}
	key := pattern
	if key == "" {
		key = "/"
	}
	return func(ctx runtimepkg.Ctx, rctx RouteRenderContext) h.Node {
		outlet := rctx.RenderOutlet
		if outlet == nil {
			outlet = func() h.Node { return h.Fragment() }
		}
		if ctx.ComponentID() == "" {
			cleanup := pushContextlessOutlet(outlet)
			cleanupBase := pushContextlessBase(pattern)
			defer cleanupBase()
			defer cleanup()
			node := component(ctx, rctx.Match)
			return ensureRouteKey(node, key)
		}
		return routeBaseCtx.Provide(ctx, pattern, func() h.Node {
			return outletCtx.Provide(ctx, outlet, func() h.Node {
				node := renderRouteComponent(ctx, component, rctx.Match, key)
				return ensureRouteKey(node, key)
			})
		})
	}
}

func componentIdentity(component runtimepkg.Component[Match]) string {
	if component == nil {
		return ""
	}
	fn := runtime.FuncForPC(reflect.ValueOf(component).Pointer())
	if fn == nil {
		return ""
	}
	return fn.Name()
}

func renderRouteComponent(ctx runtimepkg.Ctx, component runtimepkg.Component[Match], match Match, key string) h.Node {
	sess := ctx.Session()
	if sess == nil || ctx.ComponentID() == "" {
		return component(ctx, match)
	}
	if key == "" {
		return runtimepkg.Render(ctx, component, match)
	}
	return runtimepkg.Render(ctx, component, match, runtimepkg.WithKey(key))
}

func renderRoutesWithBase(ctx runtimepkg.Ctx, store *RouterStore, base string, defs []*RouteDef) dom.Node {
	if store == nil || len(defs) == 0 {
		return h.Fragment()
	}
	tree := compileRoutesCached(base, defs...)
	match := tree.Resolve(store)
	if match == nil {
		return h.Fragment()
	}
	return match.Render(ctx)
}

func ensureRouteKey(node h.Node, key string) h.Node {
	if key == "" {
		key = "/"
	}
	switch v := node.(type) {
	case *h.Element:
		if v.Key != "" {
			return node
		}
		clone := *v
		clone.Key = key
		return &clone
	case *h.ComponentNode:
		if v.Key != "" {
			return node
		}
		clone := *v
		clone.Key = key
		return &clone
	case *h.FragmentNode:
		if len(v.Children) == 0 {
			return node
		}
		updated := make([]h.Node, len(v.Children))
		changed := false
		multi := len(v.Children) > 1
		for i, child := range v.Children {
			childKey := key
			if multi {
				childKey = fmt.Sprintf("%s#%d", key, i)
			}
			next := ensureRouteKey(child, childKey)
			if next != child {
				changed = true
			}
			updated[i] = next
		}
		if !changed {
			return node
		}
		clone := *v
		clone.Children = updated
		return &clone
	case nil:
		return h.Fragment()
	default:
		return node
	}
}
