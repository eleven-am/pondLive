package runtime

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"

	h "github.com/eleven-am/liveui/pkg/liveui/html"
)

type (
	// Meta captures document metadata rendered into the SSR head element.
	Meta struct {
		Title       string
		Description string
		Meta        []h.MetaTag
		Links       []h.LinkTag
		Scripts     []h.ScriptTag
	}

	// RenderResult wraps a rendered body node with optional metadata.
	RenderResult struct {
		h.Node
		Meta *Meta
	}
)

type metaResult interface {
	BodyNode() h.Node
	Metadata() *Meta
}

// WithMetadata couples body content with document metadata for component renders.
func WithMetadata(body h.Node, meta *Meta) *RenderResult {
	if body == nil {
		body = h.Fragment()
	}
	return &RenderResult{Node: body, Meta: meta}
}

// BodyNode returns the body node contained in the render result.
func (r *RenderResult) BodyNode() h.Node {
	if r == nil || r.Node == nil {
		return h.Fragment()
	}
	return r.Node
}

// Metadata exposes the metadata captured by the render result.
func (r *RenderResult) Metadata() *Meta {
	if r == nil {
		return nil
	}
	return r.Meta
}

// CloneMeta creates a deep copy of meta.
func CloneMeta(meta *Meta) *Meta {
	if meta == nil {
		return nil
	}
	clone := *meta
	if len(meta.Meta) > 0 {
		clone.Meta = make([]h.MetaTag, len(meta.Meta))
		for i, tag := range meta.Meta {
			clone.Meta[i] = cloneMetaTag(tag)
		}
	} else {
		clone.Meta = nil
	}
	if len(meta.Links) > 0 {
		clone.Links = make([]h.LinkTag, len(meta.Links))
		for i, tag := range meta.Links {
			clone.Links[i] = cloneLinkTag(tag)
		}
	} else {
		clone.Links = nil
	}
	if len(meta.Scripts) > 0 {
		clone.Scripts = make([]h.ScriptTag, len(meta.Scripts))
		for i, tag := range meta.Scripts {
			clone.Scripts[i] = cloneScriptTag(tag)
		}
	} else {
		clone.Scripts = nil
	}
	return &clone
}

func cloneMetaTag(tag h.MetaTag) h.MetaTag {
	cloned := tag
	if len(tag.Attrs) > 0 {
		cloned.Attrs = make(map[string]string, len(tag.Attrs))
		for k, v := range tag.Attrs {
			cloned.Attrs[k] = v
		}
	} else {
		cloned.Attrs = nil
	}
	return cloned
}

func cloneLinkTag(tag h.LinkTag) h.LinkTag {
	cloned := tag
	if len(tag.Attrs) > 0 {
		cloned.Attrs = make(map[string]string, len(tag.Attrs))
		for k, v := range tag.Attrs {
			cloned.Attrs[k] = v
		}
	} else {
		cloned.Attrs = nil
	}
	return cloned
}

func cloneScriptTag(tag h.ScriptTag) h.ScriptTag {
	cloned := tag
	if len(tag.Attrs) > 0 {
		cloned.Attrs = make(map[string]string, len(tag.Attrs))
		for k, v := range tag.Attrs {
			cloned.Attrs[k] = v
		}
	} else {
		cloned.Attrs = nil
	}
	return cloned
}

// MergeMeta combines base with overrides, preferring non-empty override fields
// and appending link, meta, and script collections.
func MergeMeta(base *Meta, overrides ...*Meta) *Meta {
	merged := CloneMeta(base)
	for _, override := range overrides {
		if override == nil {
			continue
		}
		if merged == nil {
			merged = &Meta{}
		}
		if override.Title != "" {
			merged.Title = override.Title
		}
		if override.Description != "" {
			merged.Description = override.Description
		}
		if len(override.Meta) > 0 {
			merged.Meta = append(merged.Meta, cloneMetaSlice(override.Meta)...)
		}
		if len(override.Links) > 0 {
			merged.Links = append(merged.Links, cloneLinkSlice(override.Links)...)
		}
		if len(override.Scripts) > 0 {
			merged.Scripts = append(merged.Scripts, cloneScriptSlice(override.Scripts)...)
		}
	}
	return merged
}

func cloneMetaSlice(src []h.MetaTag) []h.MetaTag {
	out := make([]h.MetaTag, len(src))
	for i, tag := range src {
		out[i] = cloneMetaTag(tag)
	}
	return out
}

func cloneLinkSlice(src []h.LinkTag) []h.LinkTag {
	out := make([]h.LinkTag, len(src))
	for i, tag := range src {
		out[i] = cloneLinkTag(tag)
	}
	return out
}

func cloneScriptSlice(src []h.ScriptTag) []h.ScriptTag {
	out := make([]h.ScriptTag, len(src))
	for i, tag := range src {
		out[i] = cloneScriptTag(tag)
	}
	return out
}

// EqualMeta reports whether two Meta values are identical in content.
func EqualMeta(a, b *Meta) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Title != b.Title || a.Description != b.Description {
		return false
	}
	if !equalMetaTags(a.Meta, b.Meta) {
		return false
	}
	if !equalLinkTags(a.Links, b.Links) {
		return false
	}
	if !equalScriptTags(a.Scripts, b.Scripts) {
		return false
	}
	return true
}

func equalMetaTags(a, b []h.MetaTag) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Name != b[i].Name ||
			a[i].Content != b[i].Content ||
			a[i].Property != b[i].Property ||
			a[i].Charset != b[i].Charset ||
			a[i].HTTPEquiv != b[i].HTTPEquiv ||
			a[i].ItemProp != b[i].ItemProp ||
			!equalStringMap(a[i].Attrs, b[i].Attrs) {
			return false
		}
	}
	return true
}

func equalLinkTags(a, b []h.LinkTag) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Rel != b[i].Rel ||
			a[i].Href != b[i].Href ||
			a[i].Type != b[i].Type ||
			a[i].As != b[i].As ||
			a[i].Media != b[i].Media ||
			a[i].HrefLang != b[i].HrefLang ||
			a[i].Title != b[i].Title ||
			a[i].CrossOrigin != b[i].CrossOrigin ||
			a[i].Integrity != b[i].Integrity ||
			a[i].ReferrerPolicy != b[i].ReferrerPolicy ||
			a[i].Sizes != b[i].Sizes ||
			!equalStringMap(a[i].Attrs, b[i].Attrs) {
			return false
		}
	}
	return true
}

func equalScriptTags(a, b []h.ScriptTag) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Src != b[i].Src ||
			a[i].Type != b[i].Type ||
			a[i].Async != b[i].Async ||
			a[i].Defer != b[i].Defer ||
			a[i].Module != b[i].Module ||
			a[i].NoModule != b[i].NoModule ||
			a[i].CrossOrigin != b[i].CrossOrigin ||
			a[i].Integrity != b[i].Integrity ||
			a[i].ReferrerPolicy != b[i].ReferrerPolicy ||
			a[i].Nonce != b[i].Nonce ||
			a[i].Inner != b[i].Inner ||
			!equalStringMap(a[i].Attrs, b[i].Attrs) {
			return false
		}
	}
	return true
}

func equalStringMap(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			return false
		}
	}
	return true
}

// MetaTagKey returns a stable base key identifying the provided meta tag.
func MetaTagKey(tag h.MetaTag) string {
	if tag.Name != "" {
		return "meta:name:" + tag.Name
	}
	if tag.Property != "" {
		return "meta:property:" + tag.Property
	}
	if tag.Charset != "" {
		return "meta:charset:" + tag.Charset
	}
	if tag.HTTPEquiv != "" {
		return "meta:http-equiv:" + tag.HTTPEquiv
	}
	if tag.ItemProp != "" {
		return "meta:itemprop:" + tag.ItemProp
	}
	return "meta:hash:" + hashParts(
		tag.Content,
		tag.Charset,
		tag.HTTPEquiv,
		tag.ItemProp,
		flattenStringMap(tag.Attrs),
	)
}

// LinkTagKey returns a stable base key identifying the provided link tag.
func LinkTagKey(tag h.LinkTag) string {
	rel := strings.TrimSpace(tag.Rel)
	href := strings.TrimSpace(tag.Href)
	if rel != "" && href != "" {
		return "link:rel:" + rel + "|href:" + href
	}
	if href != "" {
		return "link:href:" + href
	}
	if rel != "" {
		return "link:rel:" + rel
	}
	return "link:hash:" + hashParts(
		tag.Type,
		tag.As,
		tag.Media,
		tag.HrefLang,
		tag.Title,
		tag.CrossOrigin,
		tag.Integrity,
		tag.ReferrerPolicy,
		tag.Sizes,
		flattenStringMap(tag.Attrs),
	)
}

// ScriptTagKey returns a stable base key identifying the provided script tag.
func ScriptTagKey(tag h.ScriptTag) string {
	src := strings.TrimSpace(tag.Src)
	if src != "" {
		return "script:src:" + src
	}
	if tag.Inner != "" {
		return "script:inline:" + hashParts(tag.Inner)
	}
	return "script:hash:" + hashParts(
		tag.Type,
		boolString(tag.Async),
		boolString(tag.Defer),
		boolString(tag.Module),
		boolString(tag.NoModule),
		tag.CrossOrigin,
		tag.Integrity,
		tag.ReferrerPolicy,
		tag.Nonce,
		flattenStringMap(tag.Attrs),
	)
}

// MetaTagKeys returns ordered unique keys for the supplied meta tags.
func MetaTagKeys(tags []h.MetaTag) []string {
	return assignKeys(len(tags), func(i int) string {
		return MetaTagKey(tags[i])
	})
}

// LinkTagKeys returns ordered unique keys for the supplied link tags.
func LinkTagKeys(tags []h.LinkTag) []string {
	return assignKeys(len(tags), func(i int) string {
		return LinkTagKey(tags[i])
	})
}

// ScriptTagKeys returns ordered unique keys for the supplied script tags.
func ScriptTagKeys(tags []h.ScriptTag) []string {
	return assignKeys(len(tags), func(i int) string {
		return ScriptTagKey(tags[i])
	})
}

func assignKeys(length int, base func(int) string) []string {
	if length == 0 {
		return nil
	}
	counts := make(map[string]int, length)
	out := make([]string, length)
	for i := 0; i < length; i++ {
		key := base(i)
		if key == "" {
			key = "key"
		}
		count := counts[key]
		counts[key] = count + 1
		if count > 0 {
			out[i] = key + "#" + strconv.Itoa(count)
		} else {
			out[i] = key
		}
	}
	return out
}

func hashParts(parts ...string) string {
	h := sha1.New()
	for _, part := range parts {
		h.Write([]byte(part))
		h.Write([]byte{0})
	}
	sum := hex.EncodeToString(h.Sum(nil))
	if len(sum) > 12 {
		return sum[:12]
	}
	return sum
}

func boolString(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func flattenStringMap(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys)*2)
	for _, k := range keys {
		parts = append(parts, k, m[k])
	}
	return strings.Join(parts, "|")
}
