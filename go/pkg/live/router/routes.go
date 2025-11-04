package router

import (
	"strings"

	ui "github.com/eleven-am/pondlive/go/pkg/live"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type RouteProps struct {
	Path      string
	Component ui.Component[Match]
}

type routeNode struct {
	*h.FragmentNode
	entry routeEntry
}

type routeEntry struct {
	pattern   string
	component ui.Component[Match]
	children  []ui.Node
}

func Route(ctx ui.Ctx, p RouteProps, children ...ui.Node) ui.Node {
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

func Routes(ctx ui.Ctx, children ...ui.Node) ui.Node {
	entries := collectRouteEntries(children)
	if !sessionRendering(ctx.Session()) {
		return &routesNode{FragmentNode: h.Fragment(), entries: entries}
	}
	state := routerStateCtx.Use(ctx)
	if state.getLoc == nil || state.setLoc == nil {
		return &routesNode{FragmentNode: h.Fragment(), entries: entries}
	}
	return renderRoutes(ctx, entries)
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

func renderRoutes(ctx ui.Ctx, entries []routeEntry) ui.Node {
	if len(entries) == 0 {
		return h.Fragment()
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
	_ = ParamsCtx.Provide(ctx, params)
	outletRender := func() ui.Node {
		if len(chosen.entry.children) == 0 {
			return h.Fragment()
		}
		return Routes(ctx, chosen.entry.children...)
	}
	_ = outletCtx.Provide(ctx, outletRender)
	return ui.Render(ctx, chosen.entry.component, chosen.match)
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

var outletCtx = ui.NewContext(func() ui.Node { return h.Fragment() })

func Outlet(ctx ui.Ctx) ui.Node {
	render := outletCtx.Use(ctx)
	if render == nil {
		return h.Fragment()
	}
	return render()
}
