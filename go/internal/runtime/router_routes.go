package runtime

import (
	"fmt"
	"strings"
	"sync"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type RouteProps struct {
	Path      string
	Component Component[Match]
}

type routeNode struct {
	*h.FragmentNode
	entry routeEntry
}

type routeEntry struct {
	pattern   string
	component Component[Match]
	children  []h.Node
}

func Route(ctx Ctx, p RouteProps, children ...h.Node) h.Node {
	pattern := p.Path
	if pattern == "" {
		pattern = "/"
	}
	normalized := pattern
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}
	entry := routeEntry{
		pattern:   normalizePath(normalized),
		component: p.Component,
		children:  children,
	}
	return &routeNode{FragmentNode: h.Fragment(), entry: entry}
}

type routesNode struct {
	*h.FragmentNode
	entries []routeEntry
}

var routesPlaceholders sync.Map // *h.FragmentNode -> *routesNode

func Routes(ctx Ctx, children ...h.Node) h.Node {
	entries := collectRouteEntries(children)
	if !sessionRendering(ctx.Session()) {
		return newRoutesPlaceholder(entries)
	}
	state := routerStateCtx.Use(ctx)
	if state.getLoc == nil || state.setLoc == nil {
		return newRoutesPlaceholder(entries)
	}
	return renderRoutes(ctx, entries)
}

func newRoutesPlaceholder(entries []routeEntry) *routesNode {
	node := &routesNode{FragmentNode: h.Fragment(), entries: entries}
	routesPlaceholders.Store(node.FragmentNode, node)
	return node
}

func consumeRoutesPlaceholder(f *h.FragmentNode) (*routesNode, bool) {
	if f == nil {
		return nil, false
	}
	if value, ok := routesPlaceholders.LoadAndDelete(f); ok {
		if node, okCast := value.(*routesNode); okCast {
			return node, true
		}
	}
	return nil, false
}

type routeMatch struct {
	entry routeEntry
	match Match
}

func collectRouteEntries(nodes []h.Node) []routeEntry {
	if len(nodes) == 0 {
		return nil
	}
	var entries []routeEntry
	for _, node := range nodes {
		switch v := node.(type) {
		case *routeNode:
			entries = append(entries, v.entry)
		case *h.FragmentNode:
			entries = append(entries, collectRouteEntries(v.Children)...)
		case *routesNode:
			entries = append(entries, v.entries...)
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
	if sess != nil {
		if value, ok := sessionEntries.Load(sess); ok {
			entry = value.(*sessionEntry)
			entry.mu.Lock()
			entry.routeDepth++
			depth = entry.routeDepth
			entry.mu.Unlock()
			defer func() {
				entry.mu.Lock()
				entry.routeDepth--
				entry.mu.Unlock()
			}()
		}
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
		return Routes(ctx, chosen.entry.children...)
	}
	key := chosen.entry.pattern
	if key == "" {
		key = "/"
	}
	if entry != nil && depth == 1 {
		entry.mu.Lock()
		prevPattern := entry.pattern
		entry.pattern = chosen.entry.pattern
		entry.mu.Unlock()
		if prevPattern != "" && prevPattern != chosen.entry.pattern {
			requestTemplateReset(sess)
		}
	}
	return ParamsCtx.Provide(ctx, params, func() h.Node {
		return outletCtx.Provide(ctx, outletRender, func() h.Node {
			node := Render(ctx, chosen.entry.component, chosen.match, WithKey(key))
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

var outletCtx = NewContext(func() h.Node { return h.Fragment() })

func Outlet(ctx Ctx) h.Node {
	render := outletCtx.Use(ctx)
	if render == nil {
		return h.Fragment()
	}
	return render()
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
	case nil:
		return h.Span(h.Key(key))
	default:
		return h.Span(h.Key(key), node)
	}
}
