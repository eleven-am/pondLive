package router

import (
	"fmt"
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
	base := routeBaseCtx.Use(ctx)

	entries := runtime.UseMemo(ctx, func() []routeEntry {
		return collectRouteEntries(children, base)
	}, children, base)

	trie := runtime.UseMemo(ctx, func() *RouterTrie {
		t := NewRouterTrie()
		for _, e := range entries {
			t.Insert(e.pattern, e)
		}
		return t
	}, entries)

	node := renderRoutes(ctx, trie)
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

func renderRoutes(ctx Ctx, trie *RouterTrie) *dom.StructuredNode {
	controller := UseRouterState(ctx)
	if controller == nil {
		return nil
	}

	state := controller.Get()
	if state == nil {
		return nil
	}

	loc := state.Location

	matchResult := trie.Match(loc.Path)
	if matchResult == nil {
		return dom.FragmentNode()
	}

	chosenEntry := matchResult.Entry
	params := matchResult.Params

	match := Match{
		Pattern:  chosenEntry.pattern,
		Path:     loc.Path,
		Params:   params,
		Query:    loc.Query,
		RawQuery: loc.Query.Encode(),
		Rest:     matchResult.Rest,
	}

	controller.SetMatch(chosenEntry.pattern, params, loc.Path)

	outletRender := func(ictx runtime.Ctx) *dom.StructuredNode {
		if len(chosenEntry.children) == 0 {
			return dom.FragmentNode()
		}
		return routeBaseCtx.Provide(ictx, chosenEntry.pattern, func(childCtx runtime.Ctx) *dom.StructuredNode {
			return Routes(childCtx, chosenEntry.children...)
		})
	}
	key := chosenEntry.pattern
	if key == "" {
		key = "/"
	}

	return routeBaseCtx.Provide(ctx, chosenEntry.pattern, func(rctx runtime.Ctx) *dom.StructuredNode {
		return outletCtx.Provide(rctx, outletRender, func(ictx runtime.Ctx) *dom.StructuredNode {
			node := runtime.Render(ictx, chosenEntry.component, match, runtime.WithKey(key))
			return ensureRouteKey(node, key)
		})
	})
}

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

type outletRenderer func(runtime.Ctx) *dom.StructuredNode

var outletCtx = runtime.CreateContext[outletRenderer](func(ctx runtime.Ctx) *dom.StructuredNode {
	return dom.FragmentNode()
})
var routeBaseCtx = runtime.CreateContext("/")

func Outlet(ctx Ctx) *dom.StructuredNode {
	render := outletCtx.Use(ctx)
	if render == nil {
		return dom.FragmentNode()
	}
	return render(ctx)
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

func ensureRouteKey(node *dom.StructuredNode, key string) *dom.StructuredNode {
	if key == "" {
		key = "/"
	}
	if node == nil {
		return dom.FragmentNode()
	}
	if node.Key != "" {
		return node
	}
	if len(node.Children) == 0 {
		clone := *node
		clone.Key = key
		return &clone
	}
	updated := make([]*dom.StructuredNode, len(node.Children))
	changed := false
	multi := len(node.Children) > 1
	for i, child := range node.Children {
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
	clone := *node
	clone.Children = updated
	return &clone
}
