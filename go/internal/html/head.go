package html

import (
	"sort"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

// MetaTag describes a <meta> element.
type MetaTag struct {
	Name      string
	Content   string
	Property  string
	Charset   string
	HTTPEquiv string
	ItemProp  string
	Attrs     map[string]string
}

// MetaTags renders meta tag definitions into HTML nodes.
func MetaTags(tags ...MetaTag) []*dom.StructuredNode {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]*dom.StructuredNode, 0, len(tags))
	for _, tag := range tags {
		items := make([]dom.Item, 0, 6+len(tag.Attrs))
		if tag.Name != "" {
			items = append(items, dom.Name(tag.Name))
		}
		if tag.Content != "" {
			items = append(items, dom.Attr("content", tag.Content))
		}
		if tag.Property != "" {
			items = append(items, dom.Attr("property", tag.Property))
		}
		if tag.Charset != "" {
			items = append(items, dom.Attr("charset", tag.Charset))
		}
		if tag.HTTPEquiv != "" {
			items = append(items, dom.Attr("http-equiv", tag.HTTPEquiv))
		}
		if tag.ItemProp != "" {
			items = append(items, dom.Attr("itemprop", tag.ItemProp))
		}
		if len(tag.Attrs) > 0 {
			keys := make([]string, 0, len(tag.Attrs))
			for k := range tag.Attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				items = append(items, dom.Attr(k, tag.Attrs[k]))
			}
		}
		nodes = append(nodes, dom.El(HTMLMetaElement{}, items...))
	}
	return nodes
}

// LinkTag describes a <link> element.
type LinkTag struct {
	Rel            string
	Href           string
	Type           string
	As             string
	Media          string
	HrefLang       string
	Title          string
	CrossOrigin    string
	Integrity      string
	ReferrerPolicy string
	Sizes          string
	Attrs          map[string]string
}

// LinkTags renders link descriptors into HTML nodes.
func LinkTags(tags ...LinkTag) []*dom.StructuredNode {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]*dom.StructuredNode, 0, len(tags))
	for _, tag := range tags {
		items := make([]dom.Item, 0, 8+len(tag.Attrs))
		if tag.Rel != "" {
			items = append(items, dom.Rel(tag.Rel))
		}
		if tag.Href != "" {
			items = append(items, dom.Href(tag.Href))
		}
		if tag.Type != "" {
			items = append(items, dom.Type(tag.Type))
		}
		if tag.As != "" {
			items = append(items, dom.Attr("as", tag.As))
		}
		if tag.Media != "" {
			items = append(items, dom.Attr("media", tag.Media))
		}
		if tag.HrefLang != "" {
			items = append(items, dom.Attr("hreflang", tag.HrefLang))
		}
		if tag.Title != "" {
			items = append(items, dom.Title(tag.Title))
		}
		if tag.CrossOrigin != "" {
			items = append(items, dom.Attr("crossorigin", tag.CrossOrigin))
		}
		if tag.Integrity != "" {
			items = append(items, dom.Attr("integrity", tag.Integrity))
		}
		if tag.ReferrerPolicy != "" {
			items = append(items, dom.Attr("referrerpolicy", tag.ReferrerPolicy))
		}
		if tag.Sizes != "" {
			items = append(items, dom.Attr("sizes", tag.Sizes))
		}
		if len(tag.Attrs) > 0 {
			keys := make([]string, 0, len(tag.Attrs))
			for k := range tag.Attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				items = append(items, dom.Attr(k, tag.Attrs[k]))
			}
		}
		nodes = append(nodes, dom.El(HTMLLinkElement{}, items...))
	}
	return nodes
}

// ScriptTag describes a <script> element.
type ScriptTag struct {
	Src            string
	Type           string
	Async          bool
	Defer          bool
	Module         bool
	NoModule       bool
	CrossOrigin    string
	Integrity      string
	ReferrerPolicy string
	Nonce          string
	Attrs          map[string]string
	Inner          string
}

// ScriptTags renders script descriptors into HTML nodes.
func ScriptTags(tags ...ScriptTag) []*dom.StructuredNode {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]*dom.StructuredNode, 0, len(tags))
	for _, tag := range tags {
		items := make([]dom.Item, 0, 10+len(tag.Attrs))
		if tag.Src != "" {
			items = append(items, dom.Src(tag.Src))
		}
		if tag.Type != "" {
			items = append(items, dom.Type(tag.Type))
		}
		if tag.Async {
			items = append(items, dom.Attr("async", ""))
		}
		if tag.Defer {
			items = append(items, dom.Attr("defer", ""))
		}
		if tag.Module {
			items = append(items, dom.Attr("type", "module"))
		}
		if tag.NoModule {
			items = append(items, dom.Attr("nomodule", ""))
		}
		if tag.CrossOrigin != "" {
			items = append(items, dom.Attr("crossorigin", tag.CrossOrigin))
		}
		if tag.Integrity != "" {
			items = append(items, dom.Attr("integrity", tag.Integrity))
		}
		if tag.ReferrerPolicy != "" {
			items = append(items, dom.Attr("referrerpolicy", tag.ReferrerPolicy))
		}
		if tag.Nonce != "" {
			items = append(items, dom.Attr("nonce", tag.Nonce))
		}
		if len(tag.Attrs) > 0 {
			keys := make([]string, 0, len(tag.Attrs))
			for k := range tag.Attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				items = append(items, dom.Attr(k, tag.Attrs[k]))
			}
		}
		script := dom.El(HTMLScriptElement{}, items...)
		if tag.Inner != "" {
			script.UnsafeHTML = tag.Inner
		}
		nodes = append(nodes, script)
	}
	return nodes
}
