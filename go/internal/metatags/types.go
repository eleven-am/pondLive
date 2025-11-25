package metatags

import "github.com/eleven-am/pondlive/go/internal/html"

// Meta holds document metadata including title, description, and various head elements.
type Meta struct {
	Title       string
	Description string
	Meta        []html.MetaTag
	Links       []html.LinkTag
	Scripts     []html.ScriptTag
}

// metaEntry stores meta along with the component depth for priority merging.
type metaEntry struct {
	meta  *Meta
	depth int // component depth in the tree (deeper = higher priority)
}

// Controller provides get/set access to meta state.
type Controller struct {
	get    func() map[string]metaEntry // map of componentID -> metaEntry
	set    func(componentID string, entry metaEntry)
	remove func(componentID string)
}

// Get returns the current merged meta state.
// Merges all meta entries with deeper components taking priority.
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
	scriptMap := make(map[string]metaEntry)

	for _, entry := range entries {
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
				key = string(rune('0'+entry.depth)) + ":" + string(rune('0'+i))
			}
			if existing, ok := scriptMap[key]; !ok || entry.depth > existing.depth {
				scriptMap[key] = entry
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

	for key, entry := range scriptMap {
		for _, script := range entry.meta.Scripts {
			scriptKey := script.Src
			if scriptKey == key || scriptKey == "" {
				result.Scripts = append(result.Scripts, script)
				break
			}
		}
	}

	return result
}

// Set updates the meta for a specific component.
func (c *Controller) Set(componentID string, depth int, meta *Meta) {
	if c != nil && c.set != nil {
		c.set(componentID, metaEntry{meta: meta, depth: depth})
	}
}

// Remove removes the meta for a specific component.
func (c *Controller) Remove(componentID string) {
	if c != nil && c.remove != nil {
		c.remove(componentID)
	}
}

var defaultMeta = &Meta{
	Title:       "PondLive Application",
	Description: "A PondLive application",
}
