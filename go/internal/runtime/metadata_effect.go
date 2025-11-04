package runtime

import (
	"strings"

	h "github.com/eleven-am/liveui/pkg/liveui/html"
)

// MetadataEffect describes incremental updates to document metadata.
type MetadataEffect struct {
	Type             string               `json:"type"`
	Title            *string              `json:"title,omitempty"`
	Description      *string              `json:"description,omitempty"`
	ClearDescription bool                 `json:"clearDescription,omitempty"`
	MetaAdd          []MetadataTagPayload `json:"metaAdd,omitempty"`
	MetaRemove       []string             `json:"metaRemove,omitempty"`
	LinkAdd          []LinkTagPayload     `json:"linkAdd,omitempty"`
	LinkRemove       []string             `json:"linkRemove,omitempty"`
	ScriptAdd        []ScriptTagPayload   `json:"scriptAdd,omitempty"`
	ScriptRemove     []string             `json:"scriptRemove,omitempty"`
}

// MetadataTagPayload serializes html.MetaTag for client reconciliation.
type MetadataTagPayload struct {
	Key       string            `json:"key"`
	Name      string            `json:"name,omitempty"`
	Content   string            `json:"content,omitempty"`
	Property  string            `json:"property,omitempty"`
	Charset   string            `json:"charset,omitempty"`
	HTTPEquiv string            `json:"httpEquiv,omitempty"`
	ItemProp  string            `json:"itemProp,omitempty"`
	Attrs     map[string]string `json:"attrs,omitempty"`
}

// LinkTagPayload serializes html.LinkTag for client reconciliation.
type LinkTagPayload struct {
	Key            string            `json:"key"`
	Rel            string            `json:"rel,omitempty"`
	Href           string            `json:"href,omitempty"`
	Type           string            `json:"type,omitempty"`
	As             string            `json:"as,omitempty"`
	Media          string            `json:"media,omitempty"`
	HrefLang       string            `json:"hreflang,omitempty"`
	Title          string            `json:"title,omitempty"`
	CrossOrigin    string            `json:"crossorigin,omitempty"`
	Integrity      string            `json:"integrity,omitempty"`
	ReferrerPolicy string            `json:"referrerpolicy,omitempty"`
	Sizes          string            `json:"sizes,omitempty"`
	Attrs          map[string]string `json:"attrs,omitempty"`
}

// ScriptTagPayload serializes html.ScriptTag for client reconciliation.
type ScriptTagPayload struct {
	Key            string            `json:"key"`
	Src            string            `json:"src,omitempty"`
	Type           string            `json:"type,omitempty"`
	Async          bool              `json:"async,omitempty"`
	Defer          bool              `json:"defer,omitempty"`
	Module         bool              `json:"module,omitempty"`
	NoModule       bool              `json:"noModule,omitempty"`
	CrossOrigin    string            `json:"crossorigin,omitempty"`
	Integrity      string            `json:"integrity,omitempty"`
	ReferrerPolicy string            `json:"referrerpolicy,omitempty"`
	Nonce          string            `json:"nonce,omitempty"`
	Attrs          map[string]string `json:"attrs,omitempty"`
	Inner          string            `json:"inner,omitempty"`
}

func buildMetadataDiff(prev, next *Meta) (*MetadataEffect, bool) {
	effect := &MetadataEffect{Type: "metadata"}
	changed := false

	prevTitle := ""
	if prev != nil {
		prevTitle = prev.Title
	}
	nextTitle := ""
	if next != nil {
		nextTitle = next.Title
	}
	if prevTitle != nextTitle {
		effect.Title = stringPtr(nextTitle)
		changed = true
	}

	prevDesc := ""
	if prev != nil {
		prevDesc = prev.Description
	}
	nextDesc := ""
	if next != nil {
		nextDesc = next.Description
	}
	if prevDesc != nextDesc {
		if stringsTrim(nextDesc) == "" {
			effect.ClearDescription = prevDesc != ""
			effect.Description = nil
		} else {
			effect.Description = stringPtr(nextDesc)
			effect.ClearDescription = false
		}
		changed = true
	}

	metaChanged := buildMetaDiff(prev, next, effect)
	linkChanged := buildLinkDiff(prev, next, effect)
	scriptChanged := buildScriptDiff(prev, next, effect)

	changed = changed || metaChanged || linkChanged || scriptChanged
	if !changed {
		return nil, false
	}
	return effect, true
}

func buildMetaDiff(prev, next *Meta, effect *MetadataEffect) bool {
	prevEntries := metaEntries(nil)
	if prev != nil {
		prevEntries = metaEntries(prev.Meta)
	}
	nextEntries := metaEntries(nil)
	if next != nil {
		nextEntries = metaEntries(next.Meta)
	}

	prevMap := make(map[string]h.MetaTag, len(prevEntries))
	for _, entry := range prevEntries {
		prevMap[entry.key] = entry.tag
	}

	changed := false
	for _, entry := range prevEntries {
		if _, ok := findEntry(nextEntries, entry.key); !ok {
			effect.MetaRemove = append(effect.MetaRemove, entry.key)
			changed = true
		}
	}

	for _, entry := range nextEntries {
		prevTag, ok := prevMap[entry.key]
		if ok && metaTagsEqual(prevTag, entry.tag) {
			continue
		}
		effect.MetaAdd = append(effect.MetaAdd, metaPayload(entry))
		changed = true
	}

	return changed
}

func buildLinkDiff(prev, next *Meta, effect *MetadataEffect) bool {
	prevEntries := linkEntries(nil)
	if prev != nil {
		prevEntries = linkEntries(prev.Links)
	}
	nextEntries := linkEntries(nil)
	if next != nil {
		nextEntries = linkEntries(next.Links)
	}

	prevMap := make(map[string]h.LinkTag, len(prevEntries))
	for _, entry := range prevEntries {
		prevMap[entry.key] = entry.tag
	}

	changed := false
	for _, entry := range prevEntries {
		if _, ok := findLinkEntry(nextEntries, entry.key); !ok {
			effect.LinkRemove = append(effect.LinkRemove, entry.key)
			changed = true
		}
	}

	for _, entry := range nextEntries {
		prevTag, ok := prevMap[entry.key]
		if ok && linkTagsEqual(prevTag, entry.tag) {
			continue
		}
		effect.LinkAdd = append(effect.LinkAdd, linkPayload(entry))
		changed = true
	}

	return changed
}

func buildScriptDiff(prev, next *Meta, effect *MetadataEffect) bool {
	prevEntries := scriptEntries(nil)
	if prev != nil {
		prevEntries = scriptEntries(prev.Scripts)
	}
	nextEntries := scriptEntries(nil)
	if next != nil {
		nextEntries = scriptEntries(next.Scripts)
	}

	prevMap := make(map[string]h.ScriptTag, len(prevEntries))
	for _, entry := range prevEntries {
		prevMap[entry.key] = entry.tag
	}

	changed := false
	for _, entry := range prevEntries {
		if _, ok := findScriptEntry(nextEntries, entry.key); !ok {
			effect.ScriptRemove = append(effect.ScriptRemove, entry.key)
			changed = true
		}
	}

	for _, entry := range nextEntries {
		prevTag, ok := prevMap[entry.key]
		if ok && scriptTagsEqual(prevTag, entry.tag) {
			continue
		}
		effect.ScriptAdd = append(effect.ScriptAdd, scriptPayload(entry))
		changed = true
	}

	return changed
}

type metaEntry struct {
	key string
	tag h.MetaTag
}

func metaEntries(tags []h.MetaTag) []metaEntry {
	if len(tags) == 0 {
		return nil
	}
	keys := MetaTagKeys(tags)
	entries := make([]metaEntry, len(tags))
	for i, tag := range tags {
		entries[i] = metaEntry{key: keys[i], tag: tag}
	}
	return entries
}

type linkEntry struct {
	key string
	tag h.LinkTag
}

func linkEntries(tags []h.LinkTag) []linkEntry {
	if len(tags) == 0 {
		return nil
	}
	keys := LinkTagKeys(tags)
	entries := make([]linkEntry, len(tags))
	for i, tag := range tags {
		entries[i] = linkEntry{key: keys[i], tag: tag}
	}
	return entries
}

type scriptEntry struct {
	key string
	tag h.ScriptTag
}

func scriptEntries(tags []h.ScriptTag) []scriptEntry {
	if len(tags) == 0 {
		return nil
	}
	keys := ScriptTagKeys(tags)
	entries := make([]scriptEntry, len(tags))
	for i, tag := range tags {
		entries[i] = scriptEntry{key: keys[i], tag: tag}
	}
	return entries
}

func metaPayload(entry metaEntry) MetadataTagPayload {
	payload := MetadataTagPayload{
		Key:       entry.key,
		Name:      entry.tag.Name,
		Content:   entry.tag.Content,
		Property:  entry.tag.Property,
		Charset:   entry.tag.Charset,
		HTTPEquiv: entry.tag.HTTPEquiv,
		ItemProp:  entry.tag.ItemProp,
	}
	if len(entry.tag.Attrs) > 0 {
		payload.Attrs = copyHeadStringMap(entry.tag.Attrs)
	}
	return payload
}

func linkPayload(entry linkEntry) LinkTagPayload {
	payload := LinkTagPayload{
		Key:            entry.key,
		Rel:            entry.tag.Rel,
		Href:           entry.tag.Href,
		Type:           entry.tag.Type,
		As:             entry.tag.As,
		Media:          entry.tag.Media,
		HrefLang:       entry.tag.HrefLang,
		Title:          entry.tag.Title,
		CrossOrigin:    entry.tag.CrossOrigin,
		Integrity:      entry.tag.Integrity,
		ReferrerPolicy: entry.tag.ReferrerPolicy,
		Sizes:          entry.tag.Sizes,
	}
	if len(entry.tag.Attrs) > 0 {
		payload.Attrs = copyHeadStringMap(entry.tag.Attrs)
	}
	return payload
}

func scriptPayload(entry scriptEntry) ScriptTagPayload {
	payload := ScriptTagPayload{
		Key:            entry.key,
		Src:            entry.tag.Src,
		Type:           entry.tag.Type,
		Async:          entry.tag.Async,
		Defer:          entry.tag.Defer,
		Module:         entry.tag.Module,
		NoModule:       entry.tag.NoModule,
		CrossOrigin:    entry.tag.CrossOrigin,
		Integrity:      entry.tag.Integrity,
		ReferrerPolicy: entry.tag.ReferrerPolicy,
		Nonce:          entry.tag.Nonce,
		Inner:          entry.tag.Inner,
	}
	if len(entry.tag.Attrs) > 0 {
		payload.Attrs = copyHeadStringMap(entry.tag.Attrs)
	}
	return payload
}

func findEntry(entries []metaEntry, key string) (h.MetaTag, bool) {
	for _, entry := range entries {
		if entry.key == key {
			return entry.tag, true
		}
	}
	return h.MetaTag{}, false
}

func findLinkEntry(entries []linkEntry, key string) (h.LinkTag, bool) {
	for _, entry := range entries {
		if entry.key == key {
			return entry.tag, true
		}
	}
	return h.LinkTag{}, false
}

func findScriptEntry(entries []scriptEntry, key string) (h.ScriptTag, bool) {
	for _, entry := range entries {
		if entry.key == key {
			return entry.tag, true
		}
	}
	return h.ScriptTag{}, false
}

func metaTagsEqual(a, b h.MetaTag) bool {
	return a.Name == b.Name &&
		a.Content == b.Content &&
		a.Property == b.Property &&
		a.Charset == b.Charset &&
		a.HTTPEquiv == b.HTTPEquiv &&
		a.ItemProp == b.ItemProp &&
		equalStringMap(a.Attrs, b.Attrs)
}

func linkTagsEqual(a, b h.LinkTag) bool {
	return a.Rel == b.Rel &&
		a.Href == b.Href &&
		a.Type == b.Type &&
		a.As == b.As &&
		a.Media == b.Media &&
		a.HrefLang == b.HrefLang &&
		a.Title == b.Title &&
		a.CrossOrigin == b.CrossOrigin &&
		a.Integrity == b.Integrity &&
		a.ReferrerPolicy == b.ReferrerPolicy &&
		a.Sizes == b.Sizes &&
		equalStringMap(a.Attrs, b.Attrs)
}

func scriptTagsEqual(a, b h.ScriptTag) bool {
	return a.Src == b.Src &&
		a.Type == b.Type &&
		a.Async == b.Async &&
		a.Defer == b.Defer &&
		a.Module == b.Module &&
		a.NoModule == b.NoModule &&
		a.CrossOrigin == b.CrossOrigin &&
		a.Integrity == b.Integrity &&
		a.ReferrerPolicy == b.ReferrerPolicy &&
		a.Nonce == b.Nonce &&
		a.Inner == b.Inner &&
		equalStringMap(a.Attrs, b.Attrs)
}

func copyHeadStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dup := make(map[string]string, len(src))
	for k, v := range src {
		dup[k] = v
	}
	return dup
}

func stringPtr(v string) *string {
	return &v
}

func stringsTrim(s string) string { return strings.TrimSpace(s) }
