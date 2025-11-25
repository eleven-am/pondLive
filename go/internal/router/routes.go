package router

import (
	"strings"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// Route defines a single route with a pattern and component.
// Returns a fragment node with route metadata for collection by Routes.
//
// Usage:
//
//	router.Route(ctx, router.RouteProps{
//	    Path: "/users/:id",
//	    ComponentNode: UserDetailPage,
//	})
func Route(ctx *runtime.Ctx, props RouteProps) work.Node {
	pattern := strings.TrimSpace(props.Path)
	if pattern == "" {
		pattern = "/"
	}

	return &work.Fragment{
		Metadata: map[string]any{
			routeMetadataKey: routeEntry{
				pattern:   pattern,
				component: props.Component,
			},
		},
	}
}

// Routes collects route definitions, matches against current location,
// and provides the matched component as slot content for the specified outlet.
//
// Usage:
//
//	router.Routes(ctx, router.RoutesProps{Outlet: "main"},
//	    router.Route(ctx, router.RouteProps{Path: "/", ComponentNode: HomePage}),
//	    router.Route(ctx, router.RouteProps{Path: "/about", ComponentNode: AboutPage}),
//	)
func Routes(ctx *runtime.Ctx, props RoutesProps, children []work.Node) work.Node {
	outlet := props.Outlet
	if outlet == "" {
		outlet = "default"
	}

	location := LocationContext.UseContextValue(ctx)
	if location == nil {
		return &work.Fragment{}
	}

	_, setMatch := MatchContext.UseProvider(ctx, nil)

	entries := runtime.UseMemo(ctx, func() []routeEntry {
		return collectRouteEntries(children)
	}, fingerprintChildren(children))

	trie := runtime.UseMemo(ctx, func() *RouterTrie {
		t := NewRouterTrie()
		for _, e := range entries {
			t.Insert(e.pattern, e)
		}
		return t
	}, entries)

	matchResult := runtime.UseMemo(ctx, func() *MatchResult {
		return trie.Match(location.Path)
	}, location.Path, trie)

	if matchResult == nil || matchResult.Entry == nil {
		if outlet == "default" {
			runtime.UseEffect(ctx, func() func() {
				setMatch(&MatchState{Matched: false})
				return nil
			}, location)
		}
		return &work.Fragment{}
	}

	if outlet == "default" {
		setMatch(&MatchState{
			Matched: true,
			Pattern: matchResult.Entry.pattern,
			Params:  matchResult.Params,
			Path:    location.Path,
			Rest:    matchResult.Rest,
		})
	}

	match := Match{
		Pattern:  matchResult.Entry.pattern,
		Path:     location.Path,
		Params:   matchResult.Params,
		Query:    location.Query,
		RawQuery: location.Query.Encode(),
		Hash:     location.Hash,
		Rest:     matchResult.Rest,
	}

	component := matchResult.Entry.component(ctx, match, matchResult.Entry.children)

	outletSlotCtx.SetSlot(ctx, outlet, component)

	return &work.Fragment{}
}

// collectRouteEntries scans children nodes for route metadata.
// Returns a flat list of all route entries found.
func collectRouteEntries(nodes []work.Node) []routeEntry {
	if len(nodes) == 0 {
		return nil
	}

	var entries []routeEntry
	for _, node := range nodes {
		if node == nil {
			continue
		}

		if frag, ok := node.(*work.Fragment); ok {

			if frag.Metadata != nil {
				if meta, ok := frag.Metadata[routeMetadataKey]; ok {
					if entry, ok := meta.(routeEntry); ok {
						entries = append(entries, entry)
						continue
					}
				}
			}

			if len(frag.Children) > 0 {
				entries = append(entries, collectRouteEntries(frag.Children)...)
			}
		}
	}

	return entries
}

// fingerprintChildren creates a stable fingerprint of children for memoization.
func fingerprintChildren(children []work.Node) string {
	var parts []string
	for _, child := range children {
		if frag, ok := child.(*work.Fragment); ok && frag.Metadata != nil {
			if meta, ok := frag.Metadata[routeMetadataKey]; ok {
				if entry, ok := meta.(routeEntry); ok {
					parts = append(parts, entry.pattern)
				}
			}
		}
	}
	return strings.Join(parts, "|")
}
