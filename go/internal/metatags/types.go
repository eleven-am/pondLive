package metatags

import (
	"crypto/sha256"
	"fmt"
)

type Meta struct {
	Title       string
	Description string
	Meta        []MetaTag
	Links       []LinkTag
	Scripts     []ScriptTag
}

type metaEntry struct {
	meta        *Meta
	depth       int
	componentID string
}

type scriptEntry struct {
	script ScriptTag
	depth  int
}

type Controller struct {
	get    func() map[string]metaEntry
	set    func(componentID string, entry metaEntry)
	remove func(componentID string)
}

func (c *Controller) Get() *Meta {
	if c == nil || c.get == nil {
		return defaultMeta
	}

	entries := c.get()
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

func inlineScriptKey(componentID string, depth, idx int, script ScriptTag) string {
	data := script.Inner + "|" + script.Nonce + "|" + script.Type + "|" + script.Src
	return fmt.Sprintf("inline:%s:%d:%d:%x", componentID, depth, idx, sha256.Sum256([]byte(data)))
}

func (c *Controller) Set(componentID string, depth int, meta *Meta) {
	if c != nil && c.set != nil {
		c.set(componentID, metaEntry{meta: meta, depth: depth, componentID: componentID})
	}
}

func (c *Controller) Remove(componentID string) {
	if c != nil && c.remove != nil {
		c.remove(componentID)
	}
}

var defaultMeta = &Meta{
	Title:       "PondLive Application",
	Description: "A PondLive application",
}
