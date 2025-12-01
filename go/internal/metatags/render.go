package metatags

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

var Render = runtime.Component(func(ctx *runtime.Ctx, children []work.Node) work.Node {
	state := metaCtx.UseContextValue(ctx)
	metaData := getMergedMeta(state.entries)

	items := make([]work.Node, 0)

	if metaData.Title != "" {
		items = append(items, &work.Element{
			Tag:      "title",
			Children: []work.Node{&work.Text{Value: metaData.Title}},
		})
	}

	if metaData.Description != "" {
		descMeta := MetaTags(MetaTag{
			Name:    "description",
			Content: metaData.Description,
		})
		items = append(items, descMeta...)
	}

	items = append(items, MetaTags(metaData.Meta...)...)
	items = append(items, LinkTags(metaData.Links...)...)
	items = append(items, ScriptTags(metaData.Scripts...)...)

	return &work.Fragment{Children: items}
})

func getMergedMeta(entries map[string]metaEntry) *Meta {
	if len(entries) == 0 {
		return defaultMeta
	}

	var deepestTitle *metaEntry
	var deepestDescription *metaEntry
	metaMap := make(map[string]metaEntry)
	linkMap := make(map[string]metaEntry)
	scriptMap := make(map[string]scriptEntry)

	for componentID, entry := range entries {
		if entry.meta == nil {
			continue
		}

		if entry.meta.Title != "" {
			if deepestTitle == nil || entry.depth > deepestTitle.depth {
				deepestTitle = &entry
			}
		}

		if entry.meta.Description != "" {
			if deepestDescription == nil || entry.depth > deepestDescription.depth {
				deepestDescription = &entry
			}
		}

		for _, mt := range entry.meta.Meta {
			key := mt.Name
			if key == "" {
				key = mt.Property
			}
			if existing, ok := metaMap[key]; !ok || entry.depth > existing.depth {
				metaMap[key] = entry
			}
		}

		for _, link := range entry.meta.Links {
			key := link.Rel + ":" + link.Href
			if existing, ok := linkMap[key]; !ok || entry.depth > existing.depth {
				linkMap[key] = entry
			}
		}

		for i, script := range entry.meta.Scripts {
			key := script.Src
			if key == "" {
				key = inlineScriptKey(componentID, entry.depth, i, script)
			}
			if existing, ok := scriptMap[key]; !ok || entry.depth > existing.depth {
				scriptMap[key] = scriptEntry{script: script, depth: entry.depth}
			}
		}
	}

	result := &Meta{}

	if deepestTitle != nil {
		result.Title = deepestTitle.meta.Title
	} else {
		result.Title = defaultMeta.Title
	}

	if deepestDescription != nil {
		result.Description = deepestDescription.meta.Description
	} else {
		result.Description = defaultMeta.Description
	}

	for key, entry := range metaMap {
		for _, mt := range entry.meta.Meta {
			mtKey := mt.Name
			if mtKey == "" {
				mtKey = mt.Property
			}
			if mtKey == key {
				result.Meta = append(result.Meta, mt)
				break
			}
		}
	}

	for key, entry := range linkMap {
		for _, link := range entry.meta.Links {
			linkKey := link.Rel + ":" + link.Href
			if linkKey == key {
				result.Links = append(result.Links, link)
				break
			}
		}
	}

	for _, entry := range scriptMap {
		result.Scripts = append(result.Scripts, entry.script)
	}

	return result
}
