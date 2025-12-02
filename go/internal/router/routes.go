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
				slot:      defaultSlotName,
			},
		},
	}
}

func routes(ctx *runtime.Ctx, items []work.Item) work.Node {
	children := work.ItemsToNodes(items)

	loc := locationCtx.UseContextValue(ctx)
	base := routeBaseCtx.UseContextValue(ctx)
	parentMatch := matchCtx.UseContextValue(ctx)

	pathToMatch := loc.Path
	if parentMatch != nil && parentMatch.Matched && parentMatch.Rest != "" && parentMatch.Path == loc.Path {
		pathToMatch = parentMatch.Rest
	}

	allSlots := runtime.UseMemo(ctx, func() []slotEntry {
		routeEntries := collectRouteEntries(children, base)
		slotEntries := collectSlotEntries(children, base)

		if len(routeEntries) > 0 {
			slotEntries = append(slotEntries, slotEntry{
				name:   defaultSlotName,
				routes: routeEntries,
			})
		}

		return slotEntries
	}, fingerprintChildren(children), fingerprintSlots(children), base)

	var slots map[string]outletRenderer
	if pathToMatch != "" {
		slots = make(map[string]outletRenderer)

		for _, slot := range allSlots {
			trie := newRouterTrie()
			for _, e := range slot.routes {
				trie.Insert(e.fullPath, e)
			}

			matchResult := trie.Match(pathToMatch)
			if matchResult == nil || matchResult.Entry == nil {
				if slot.fallback != nil {
					slots[slot.name] = slot.fallback
				}
				continue
			}

			entry := matchResult.Entry
			childRoutes := entry.children
			childSlots := map[string]outletRenderer{}
			if len(childRoutes) > 0 {
				capturedChildren := work.NodesToItems(childRoutes)
				childSlots[defaultSlotName] = func(cctx *runtime.Ctx) work.Node {
					return routes(cctx, capturedChildren)
				}
			}

			capturedMatch := Match{
				Pattern:  entry.fullPath,
				Path:     loc.Path,
				Params:   matchResult.Params,
				Query:    loc.Query,
				RawQuery: loc.Query.Encode(),
				Hash:     loc.Hash,
				Rest:     matchResult.Rest,
			}
			capturedMatchState := &MatchState{
				Matched: true,
				Pattern: entry.fullPath,
				Path:    loc.Path,
				Params:  matchResult.Params,
				Rest:    matchResult.Rest,
			}

			basePath := trimWildcardSuffix(entry.fullPath)

			componentKey := slot.name + ":" + entry.fullPath
			if strings.Contains(entry.fullPath, ":") || strings.Contains(entry.fullPath, "*") {
				componentKey += "|" + loc.Path
			}

			renderer := func(ictx *runtime.Ctx) work.Node {
				return routeMount(ictx, routeMountProps{
					match:        capturedMatch,
					matchState:   capturedMatchState,
					base:         basePath,
					childSlots:   childSlots,
					component:    entry.component,
					componentKey: componentKey,
				})
			}

			slots[slot.name] = renderer
		}
	}

	hasDefaultSlot := false
	for _, slot := range allSlots {
		if slot.name == defaultSlotName {
			hasDefaultSlot = true
			break
		}
	}

	if hasDefaultSlot {
		if slots == nil || slots[defaultSlotName] == nil {
			_, setMatch := matchCtx.UseProvider(ctx, &MatchState{Matched: false})
			_, setSlots := slotsCtx.UseProvider(ctx, nil)
			_, setBase := routeBaseCtx.UseProvider(ctx, base)
			setMatch(&MatchState{Matched: false})
			setSlots(nil)
			setBase(base)
			return &work.Fragment{}
		}
		return slots[defaultSlotName](ctx)
	}

	_, setSlots := slotsCtx.UseProvider(ctx, slots)
	setSlots(slots)
	return &work.Fragment{}
}

func collectRouteEntries(nodes []work.Node, base string) []routeEntry {
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
						entry.pattern = strings.TrimSpace(entry.pattern)
						entry.fullPath = resolveRoutePattern(entry.pattern, base)
						if len(entry.children) > 0 && !strings.HasSuffix(entry.fullPath, "*") {
							if entry.fullPath == "/" {
								entry.fullPath = "/*"
							} else {
								entry.fullPath = entry.fullPath + "/*"
							}
						}
						entries = append(entries, entry)
						continue
					}
				}
				if _, ok := frag.Metadata[slotMetadataKey]; ok {
					continue
				}
			}

			if len(frag.Children) > 0 {
				entries = append(entries, collectRouteEntries(frag.Children, base)...)
			}
		}
	}

	return entries
}

func collectSlotEntries(nodes []work.Node, base string) []slotEntry {
	if len(nodes) == 0 {
		return nil
	}

	var entries []slotEntry
	for _, node := range nodes {
		if node == nil {
			continue
		}

		if frag, ok := node.(*work.Fragment); ok {
			if frag.Metadata != nil {
				if meta, ok := frag.Metadata[slotMetadataKey]; ok {
					if entry, ok := meta.(slotEntry); ok {
						for i := range entry.routes {
							entry.routes[i].fullPath = resolveRoutePattern(entry.routes[i].pattern, base)
							if len(entry.routes[i].children) > 0 && !strings.HasSuffix(entry.routes[i].fullPath, "*") {
								if entry.routes[i].fullPath == "/" {
									entry.routes[i].fullPath = "/*"
								} else {
									entry.routes[i].fullPath = entry.routes[i].fullPath + "/*"
								}
							}
						}
						entries = append(entries, entry)
						continue
					}
				}
			}

			if len(frag.Children) > 0 {
				entries = append(entries, collectSlotEntries(frag.Children, base)...)
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

func fingerprintSlots(children []work.Node) string {
	var parts []string
	for _, child := range children {
		if frag, ok := child.(*work.Fragment); ok && frag.Metadata != nil {
			if meta, ok := frag.Metadata[slotMetadataKey]; ok {
				if entry, ok := meta.(slotEntry); ok {
					slotPart := entry.name + ":"
					for _, route := range entry.routes {
						slotPart += route.pattern + ","
					}
					parts = append(parts, slotPart)
				}
			}
		}
	}
	return strings.Join(parts, "|")
}

var Routes = runtime.Component(routes)
