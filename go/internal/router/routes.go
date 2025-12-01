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

func Routes(ctx *runtime.Ctx, children ...work.Node) work.Node {
	loc := locationCtx.UseContextValue(ctx)
	base := routeBaseCtx.UseContextValue(ctx)
	parentMatch := matchCtx.UseContextValue(ctx)

	pathToMatch := loc.Path
	if parentMatch != nil && parentMatch.Rest != "" {
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

	slots := runtime.UseMemo(ctx, func() map[string]outletRenderer {
		if pathToMatch == "" {
			return nil
		}

		result := make(map[string]outletRenderer)

		for _, slot := range allSlots {
			trie := newRouterTrie()
			for _, e := range slot.routes {
				trie.Insert(e.pattern, e)
			}

			matchResult := trie.Match(pathToMatch)
			if matchResult == nil || matchResult.Entry == nil {
				if slot.fallback != nil {
					result[slot.name] = slot.fallback
				}
				continue
			}

			entry := matchResult.Entry
			childRoutes := entry.children
			capturedMatch := Match{
				Pattern:  entry.pattern,
				Path:     loc.Path,
				Params:   matchResult.Params,
				Query:    loc.Query,
				RawQuery: loc.Query.Encode(),
				Hash:     loc.Hash,
				Rest:     matchResult.Rest,
			}
			capturedMatchState := &MatchState{
				Matched: true,
				Pattern: entry.pattern,
				Path:    loc.Path,
				Params:  matchResult.Params,
				Rest:    matchResult.Rest,
			}
			capturedBase := entry.pattern

			result[slot.name] = func(ictx *runtime.Ctx) work.Node {
				matchCtx.UseProvider(ictx, capturedMatchState)
				routeBaseCtx.UseProvider(ictx, capturedBase)

				childRender := func(cctx *runtime.Ctx) work.Node {
					if len(childRoutes) == 0 {
						return &work.Fragment{}
					}
					return Routes(cctx, childRoutes...)
				}
				slotsCtx.UseProvider(ictx, map[string]outletRenderer{defaultSlotName: childRender})

				return entry.component(ictx, capturedMatch)
			}
		}

		return result
	}, pathToMatch, allSlots, loc)

	hasDefaultSlot := false
	for _, slot := range allSlots {
		if slot.name == defaultSlotName {
			hasDefaultSlot = true
			break
		}
	}

	if hasDefaultSlot {
		if slots == nil || slots[defaultSlotName] == nil {
			matchCtx.UseProvider(ctx, &MatchState{Matched: false})
			slotsCtx.UseProvider(ctx, nil)
			routeBaseCtx.UseProvider(ctx, base)
			return &work.Fragment{}
		}

		return slots[defaultSlotName](ctx)
	}

	slotsCtx.UseProvider(ctx, slots)
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
						entry.pattern = resolveRoutePattern(entry.pattern, base)
						if len(entry.children) > 0 && !strings.HasSuffix(entry.pattern, "*") {
							if entry.pattern == "/" {
								entry.pattern = "/*"
							} else {
								entry.pattern = entry.pattern + "/*"
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
							entry.routes[i].pattern = resolveRoutePattern(entry.routes[i].pattern, base)
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

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	return path
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
