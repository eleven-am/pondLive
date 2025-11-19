package router

import (
	"fmt"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

type RouteProps struct {
	Path      string
	Component runtime.Component[Match]
}

type routeEntry struct {
	pattern   string
	component runtime.Component[Match]
	children  []*dom2.StructuredNode
}

const (
	routeMetadataKey = "router:entry"
	routeChildrenKey = "router:children"
)

func Route(ctx Ctx, p RouteProps, children ...*dom2.StructuredNode) *dom2.StructuredNode {
	pattern := strings.TrimSpace(p.Path)
	if pattern == "" {
		pattern = "/"
	}
	children = expandRouteChildren(children)
	node := dom2.FragmentNode()
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

func Routes(ctx Ctx, children ...*dom2.StructuredNode) *dom2.StructuredNode {
	base := routeBaseCtx.Use(ctx)
	entries := collectRouteEntries(children, base)
	node := renderRoutes(ctx, entries)
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

func collectRouteEntries(nodes []*dom2.StructuredNode, base string) []routeEntry {
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

func expandRouteChildren(nodes []*dom2.StructuredNode) []*dom2.StructuredNode {
	if len(nodes) == 0 {
		return nil
	}
	expanded := make([]*dom2.StructuredNode, 0, len(nodes))
	for _, node := range nodes {
		if node == nil {
			continue
		}
		if node.Metadata != nil {
			if raw, ok := node.Metadata[routeChildrenKey]; ok {
				if nested, ok := raw.([]*dom2.StructuredNode); ok {
					expanded = append(expanded, nested...)
					continue
				}
			}
		}
		expanded = append(expanded, node)
	}
	return expanded
}

func renderRoutes(ctx Ctx, entries []routeEntry) *dom2.StructuredNode {
	if len(entries) == 0 {
		return dom2.FragmentNode()
	}

	var (
		entry *sessionEntry
		depth int
	)

	if entry = loadSessionRouterEntry(ctx); entry != nil {
		entry.mu.Lock()
		entry.render.depth++
		depth = entry.render.depth
		entry.mu.Unlock()
		defer func() {
			entry.mu.Lock()
			entry.render.depth--
			entry.mu.Unlock()
		}()
	}

	state := RouterStateCtx.Use(ctx)
	var loc Location
	if state.getLoc != nil {
		loc = state.getLoc()
	} else {
		loc = currentSessionLocation(ctx)
	}

	var chosen *routeMatch
	for _, entry := range entries {
		component := entry.component
		if component == nil {
			continue
		}
		rawQuery := loc.Query.Encode()
		match, err := Parse(entry.pattern, loc.Path, rawQuery)
		if err != nil {
			continue
		}
		if chosen == nil || Prefer(match, chosen.match) {
			chosen = &routeMatch{entry: entry, match: match}
		}
	}

	if chosen == nil {
		return dom2.FragmentNode()
	}

	params := copyParams(chosen.match.Params)
	storeSessionParams(ctx, params)
	outletRender := func(ictx runtime.Ctx) *dom2.StructuredNode {
		if len(chosen.entry.children) == 0 {
			return dom2.FragmentNode()
		}
		return routeBaseCtx.Provide(ictx, chosen.entry.pattern, func(childCtx runtime.Ctx) *dom2.StructuredNode {
			return Routes(childCtx, chosen.entry.children...)
		})
	}
	key := chosen.entry.pattern
	if key == "" {
		key = "/"
	}
	if entry != nil && depth == 1 {
		entry.mu.Lock()
		entry.render.currentRoute = chosen.entry.pattern
		entry.mu.Unlock()
	}
	return ParamsCtx.Provide(ctx, params, func(pctx runtime.Ctx) *dom2.StructuredNode {
		return routeBaseCtx.Provide(pctx, chosen.entry.pattern, func(rctx runtime.Ctx) *dom2.StructuredNode {
			return outletCtx.Provide(rctx, outletRender, func(ictx runtime.Ctx) *dom2.StructuredNode {
				node := runtime.Render(ictx, chosen.entry.component, chosen.match, runtime.WithKey(key))
				return ensureRouteKey(node, key)
			})
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

type outletRenderer func(runtime.Ctx) *dom2.StructuredNode

var outletCtx = runtime.CreateContext[outletRenderer](func(ctx runtime.Ctx) *dom2.StructuredNode {
	return dom2.FragmentNode()
})
var routeBaseCtx = runtime.CreateContext("/")

func Outlet(ctx Ctx) *dom2.StructuredNode {
	render := outletCtx.Use(ctx)
	if render == nil {
		return dom2.FragmentNode()
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

func ensureRouteKey(node *dom2.StructuredNode, key string) *dom2.StructuredNode {
	if key == "" {
		key = "/"
	}
	if node == nil {
		return dom2.FragmentNode()
	}
	if node.Key != "" {
		return node
	}
	if len(node.Children) == 0 {
		clone := *node
		clone.Key = key
		return &clone
	}
	updated := make([]*dom2.StructuredNode, len(node.Children))
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
