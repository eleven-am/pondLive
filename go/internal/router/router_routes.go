package router

import (
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

type RouteProps struct {
	Path      string
	Component runtime.Component[Match]
}

type routeEntry struct {
	pattern   string
	component runtime.Component[Match]
	children  []*dom.StructuredNode
}

const (
	routeMetadataKey = "router:entry"
	routeChildrenKey = "router:children"
)

func Route(ctx Ctx, p RouteProps, children ...*dom.StructuredNode) *dom.StructuredNode {
	pattern := strings.TrimSpace(p.Path)
	if pattern == "" {
		pattern = "/"
	}
	children = expandRouteChildren(children)
	node := dom.FragmentNode()
	if node.Metadata == nil {
		node.Metadata = make(map[string]any)
	}
	node.Metadata[routeMetadataKey] = routeEntry{
		component: p.Component,
		children:  children,
		pattern:   pattern,
	}
	return node
}

func Routes(ctx Ctx, children ...*dom.StructuredNode) *dom.StructuredNode {
	// Get base pattern from router context (for nested routes)
	routerCtx := routerContextKey.Use(ctx)
	base := "/"
	if routerCtx != nil && routerCtx.Pattern != "" {
		base = routerCtx.Pattern
	}

	// Memoize route entries collection
	entries := runtime.UseMemo(ctx, func() []routeEntry {
		return collectRouteEntries(children, base)
	}, children, base)

	// Memoize trie construction
	trie := runtime.UseMemo(ctx, func() *RouterTrie {
		t := NewRouterTrie()
		for _, e := range entries {
			t.Insert(e.pattern, e)
		}
		return t
	}, entries)

	// Render the matched route
	node := renderRoutes(ctx, trie, children)
	if node.Metadata == nil {
		node.Metadata = make(map[string]any)
	}
	node.Metadata[routeChildrenKey] = children
	return node
}

type routeMatch struct {
	entry routeEntry
	match Match
}

func collectRouteEntries(nodes []*dom.StructuredNode, base string) []routeEntry {
	if len(nodes) == 0 {
		return nil
	}
	var entries []routeEntry
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if node.Metadata != nil {
			if meta, ok := node.Metadata[routeMetadataKey]; ok {
				entry := meta.(routeEntry)
				entry.pattern = resolveRoutePattern(entry.pattern, base)
				entries = append(entries, entry)
				continue
			}
		}
		if len(node.Children) > 0 {
			entries = append(entries, collectRouteEntries(node.Children, base)...)
		}
	}
	return entries
}

func expandRouteChildren(nodes []*dom.StructuredNode) []*dom.StructuredNode {
	if len(nodes) == 0 {
		return nil
	}
	expanded := make([]*dom.StructuredNode, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if node.Metadata != nil {
			if raw, ok := node.Metadata[routeChildrenKey]; ok {
				if nested, ok := raw.([]*dom.StructuredNode); ok {
					expanded = append(expanded, nested...)
					continue
				}
			}
		}
		expanded = append(expanded, node)
	}
	return expanded
}

func renderRoutes(ctx Ctx, trie *RouterTrie, children []*dom.StructuredNode) *dom.StructuredNode {
	controller := UseRouterState(ctx)
	if controller == nil {
		return dom.FragmentNode()
	}

	state := controller.Get()
	if state == nil {
		return dom.FragmentNode()
	}

	loc := state.Location

	// Memoize the match result to avoid re-matching on every render
	matchResult := runtime.UseMemo(ctx, func() *MatchResult {
		return trie.Match(loc.Path)
	}, loc.Path, trie)

	// If no route matches, render nothing
	if matchResult == nil {
		// Update controller to clear match state
		runtime.UseEffect(ctx, func() runtime.Cleanup {
			controller.SetLocation(loc)
			return nil
		}, loc)
		return dom.FragmentNode()
	}

	chosenEntry := matchResult.Entry
	params := matchResult.Params

	// Update match state synchronously.
	// SetMatch has built-in deduplication (see state.go:71-73), so it only triggers
	// a re-render if the pattern or path actually changed. Combined with UseMemo above,
	// this prevents re-render cascades while ensuring params are available immediately.
	controller.SetMatch(chosenEntry.pattern, params, loc.Path)

	// Create Match object for the component
	match := Match{
		Pattern:  chosenEntry.pattern,
		Path:     loc.Path,
		Params:   params,
		Query:    loc.Query,
		RawQuery: loc.Query.Encode(),
		Rest:     matchResult.Rest,
	}

	// Use the full path as the component key for stable identity
	// This is better than using pattern because:
	// 1. Different paths can match the same pattern (/users/1 vs /users/2)
	// 2. We want different component instances for different paths
	// 3. We DON'T want aggressive keying of children (let components handle their own keys)
	key := loc.Path
	if key == "" {
		key = "/"
	}

	// Provide router context with pattern and children for Outlet
	return routerContextKey.Provide(ctx, &RouterContextValue{
		Pattern:  chosenEntry.pattern,
		Children: chosenEntry.children,
	}, func(rctx runtime.Ctx) *dom.StructuredNode {
		// Render the route component with the match data
		return runtime.Render(rctx, chosenEntry.component, match, runtime.WithKey(key))
	})
}

// RouterContextValue holds all routing context information in a single structure.
// This replaces the previous dual-context approach (routeBaseCtx + outletCtx).
type RouterContextValue struct {
	// Pattern is the current matched route pattern (e.g., "/users/:id")
	Pattern string

	// Children are the nested route definitions for this route
	Children []*dom.StructuredNode
}

var routerContextKey = runtime.CreateContext[*RouterContextValue](nil)

// Outlet renders the nested child routes for the current route.
// It looks up the children from the router context and renders a nested Routes component.
func Outlet(ctx Ctx) *dom.StructuredNode {
	routerCtx := routerContextKey.Use(ctx)
	if routerCtx == nil || len(routerCtx.Children) == 0 {
		return dom.FragmentNode()
	}

	// Render child routes with the current pattern as their base
	return routerContextKey.Provide(ctx, &RouterContextValue{
		Pattern:  routerCtx.Pattern,
		Children: nil, // Children will be set by their Routes component
	}, func(childCtx runtime.Ctx) *dom.StructuredNode {
		return Routes(childCtx, routerCtx.Children...)
	})
}

func resolveRoutePattern(raw, base string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "/"
	}
	if strings.HasPrefix(trimmed, "./") {
		rel := strings.TrimPrefix(trimmed, ".")
		return normalizePath(joinRelativePath(base, rel))
	}
	return normalizePath(trimmed)
}

func joinRelativePath(base, rel string) string {
	rel = normalizePath(rel)
	base = normalizePath(base)
	base = trimWildcardSuffix(base)
	if base == "/" {
		return rel
	}
	if rel == "/" {
		return base
	}
	return normalizePath(strings.TrimSuffix(base, "/") + rel)
}

func trimWildcardSuffix(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	normalized := normalizePath(trimmed)
	if normalized == "/" {
		return "/"
	}
	segments := strings.Split(strings.Trim(normalized, "/"), "/")
	if len(segments) == 0 {
		return "/"
	}
	last := segments[len(segments)-1]
	if strings.HasPrefix(last, "*") {
		segments = segments[:len(segments)-1]
	}
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}

// copyParams creates a copy of the params map for trie matching.
// This is used during recursive route matching to avoid mutation.
func copyParams(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
