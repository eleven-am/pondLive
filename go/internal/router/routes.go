package router

import (
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/slot"
)

// Route defines a single route with a pattern and component.
// Returns a fragment node with route metadata for collection by Routes().
//
// Usage:
//
//	router.Route(ctx, router.RouteProps{
//	    Path: "/users/:id",
//	    Component: UserDetailPage,
//	})
func Route(ctx Ctx, props RouteProps, children ...*dom.StructuredNode) *dom.StructuredNode {
	pattern := strings.TrimSpace(props.Path)
	if pattern == "" {
		pattern = "/"
	}

	node := dom.FragmentNode()
	if node.Metadata == nil {
		node.Metadata = make(map[string]any)
	}
	node.Metadata[routeMetadataKey] = routeEntry{
		pattern:   pattern,
		component: props.Component,
		children:  children,
	}

	return node
}

// Routes collects route definitions, matches against current location,
// and provides the matched component as slot content for the specified outlet.
//
// Usage:
//
//	router.Routes(ctx, router.RoutesProps{Outlet: "main"},
//	    router.Route(ctx, router.RouteProps{Path: "/", Component: HomePage}),
//	    router.Route(ctx, router.RouteProps{Path: "/about", Component: AboutPage}),
//	)
func Routes(ctx Ctx, props RoutesProps, children ...*dom.StructuredNode) *dom.StructuredNode {
	outlet := props.Outlet
	if outlet == "" {
		outlet = "default"
	}

	controller := useRouterController(ctx)
	if controller == nil {
		return dom.FragmentNode()
	}

	loc := controller.GetLocation()

	entries := runtime.UseMemo(ctx, func() []routeEntry {
		collected := collectRouteEntries(children)
		return collected
	}, children)

	trie := runtime.UseMemo(ctx, func() *RouterTrie {
		t := NewRouterTrie()
		for _, e := range entries {
			t.Insert(e.pattern, e)
		}
		return t
	}, entries)

	matchResult := runtime.UseMemo(ctx, func() *MatchResult {
		result := trie.Match(loc.Path)
		return result
	}, loc.Path, trie)

	if matchResult == nil {
		if outlet == "default" {
			runtime.UseEffect(ctx, func() runtime.Cleanup {
				controller.ClearMatch()
				return nil
			}, loc)
		}
		return dom.FragmentNode()
	}

	if outlet == "default" {
		controller.SetMatch(
			matchResult.Entry.pattern,
			matchResult.Params,
			loc.Path,
		)
	}

	match := Match{
		Pattern:  matchResult.Entry.pattern,
		Path:     loc.Path,
		Params:   matchResult.Params,
		Query:    loc.Query,
		RawQuery: loc.Query.Encode(),
		Hash:     loc.Hash,
		Rest:     matchResult.Rest,
	}

	key := outlet + ":" + loc.Path
	if key == "" {
		key = outlet + ":/"
	}

	component := runtime.Render(ctx, matchResult.Entry.component, match, runtime.WithKey(key))

	fragment := dom.FragmentNode()
	slotItem := slot.Slot(outlet, component)
	slotItem.ApplyTo(fragment)
	return fragment
}

// collectRouteEntries scans children nodes for route metadata.
// Returns a flat list of all route entries found, including nested routes.
func collectRouteEntries(nodes []*dom.StructuredNode) []routeEntry {
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
				entries = append(entries, entry)
				if len(entry.children) > 0 {
					nested := collectRouteEntries(entry.children)
					entries = append(entries, nested...)
				}
				continue
			}
		}

		if len(node.Children) > 0 {
			entries = append(entries, collectRouteEntries(node.Children)...)
		}
	}

	return entries
}
