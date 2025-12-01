package router

import (
	"strings"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

func Route(ctx *runtime.Ctx, props RouteProps, children ...work.Node) work.Node {
	pattern := strings.TrimSpace(props.Path)
	if pattern == "" {
		pattern = "/"
	}

	return &work.Fragment{
		Metadata: map[string]any{
			routeMetadataKey: routeEntry{
				pattern:   pattern,
				component: props.Component,
				children:  children,
			},
		},
	}
}

var Routes = runtime.PropsComponent(func(ctx *runtime.Ctx, props RoutesProps, children []work.Node) work.Node {
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

	childRoutesCtx.UseProvider(ctx, matchResult.Entry.children)

	component := matchResult.Entry.component(ctx, match)

	outletSlotCtx.SetSlot(ctx, outlet, component)

	return &work.Fragment{}
})

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
