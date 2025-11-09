package runtime

import (
	"fmt"
	"strings"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type RouteProps struct {
	Path      string
	Component Component[Match]
}

type routeNode struct {
	*h.FragmentNode
	entry      routeEntry
	rawPattern string
}

type routeEntry struct {
	pattern   string
	component Component[Match]
	children  []h.Node
}

func Route(ctx Ctx, p RouteProps, children ...h.Node) h.Node {
	pattern := strings.TrimSpace(p.Path)
	if pattern == "" {
		pattern = "/"
	}
	entry := routeEntry{
		component: p.Component,
		children:  children,
	}
	return &routeNode{FragmentNode: h.Fragment(), entry: entry, rawPattern: pattern}
}

type routesNode struct {
	*h.FragmentNode
	entries  []routeEntry
	children []h.Node
}

func Routes(ctx Ctx, children ...h.Node) h.Node {
	base := routeBaseCtx.Use(ctx)
	entries := collectRouteEntries(children, base)
	sess := ctx.Session()
	if !sessionRendering(sess) {
		return newRoutesPlaceholder(sess, entries, children)
	}
	state := routerStateCtx.Use(ctx)
	if state.getLoc == nil || state.setLoc == nil {
		return newRoutesPlaceholder(sess, entries, children)
	}
	return renderRoutes(ctx, entries)
}

func newRoutesPlaceholder(sess *ComponentSession, entries []routeEntry, children []h.Node) *routesNode {
	node := &routesNode{FragmentNode: h.Fragment(), entries: entries, children: append([]h.Node(nil), children...)}
	if sess != nil {
		sess.storeRoutesPlaceholder(node.FragmentNode, node)
	}
	return node
}

func consumeRoutesPlaceholder(sess *ComponentSession, f *h.FragmentNode) (*routesNode, bool) {
	if sess == nil {
		return nil, false
	}
	return sess.takeRoutesPlaceholder(f)
}

type routeMatch struct {
	entry routeEntry
	match Match
}

func collectRouteEntries(nodes []h.Node, base string) []routeEntry {
	if len(nodes) == 0 {
		return nil
	}
	var entries []routeEntry
	for _, node := range nodes {
		switch v := node.(type) {
		case *routeNode:
			entry := v.entry
			entry.pattern = resolveRoutePattern(v.rawPattern, base)
			entries = append(entries, entry)
		case *h.FragmentNode:
			entries = append(entries, collectRouteEntries(v.Children, base)...)
		case *routesNode:
			if len(v.children) > 0 {
				entries = append(entries, collectRouteEntries(v.children, base)...)
			} else {
				entries = append(entries, v.entries...)
			}
		}
	}
	return entries
}

func renderRoutes(ctx Ctx, entries []routeEntry) h.Node {
	if len(entries) == 0 {
		return h.Fragment()
	}

	sess := ctx.Session()
	var (
		entry *sessionEntry
		depth int
	)
	if entry = sess.loadRouterEntry(); entry != nil {
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

	state := routerStateCtx.Use(ctx)
	var loc Location
	if state.getLoc != nil {
		loc = state.getLoc()
	} else {
		loc = currentSessionLocation(ctx.Session())
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
		return h.Fragment()
	}

	params := copyParams(chosen.match.Params)
	storeSessionParams(ctx.Session(), params)
	outletRender := func() h.Node {
		if len(chosen.entry.children) == 0 {
			return h.Fragment()
		}
		return routeBaseCtx.Provide(ctx, chosen.entry.pattern, func() h.Node {
			return Routes(ctx, chosen.entry.children...)
		})
	}
	key := chosen.entry.pattern
	if key == "" {
		key = "/"
	}
	if entry != nil && depth == 1 {
		entry.mu.Lock()
		prevPattern := entry.render.currentRoute
		entry.render.currentRoute = chosen.entry.pattern
		entry.mu.Unlock()
		if prevPattern != "" && prevPattern != chosen.entry.pattern {
			requestTemplateReset(sess)
		}
	}
	return ParamsCtx.Provide(ctx, params, func() h.Node {
		return routeBaseCtx.Provide(ctx, chosen.entry.pattern, func() h.Node {
			return outletCtx.Provide(ctx, outletRender, func() h.Node {
				node := Render(ctx, chosen.entry.component, chosen.match, WithKey(key))
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

var outletCtx = NewContext(func() h.Node { return h.Fragment() })
var routeBaseCtx = NewContext("/")

func Outlet(ctx Ctx) h.Node {
	render := outletCtx.Use(ctx)
	if render == nil {
		return h.Fragment()
	}
	return render()
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
	case *h.ComponentNode:
		if v.Child == nil {
			return node
		}
		child := ensureRouteKey(v.Child, key)
		if child == v.Child {
			return node
		}
		clone := *v
		clone.Child = child
		return &clone
	case nil:
		return h.Fragment()
	default:
		return node
	}
}
