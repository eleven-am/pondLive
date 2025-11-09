package html

import "sort"

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
func MetaTags(tags ...MetaTag) []Node {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]Node, 0, len(tags))
	for _, tag := range tags {
		items := make([]Item, 0, 6+len(tag.Attrs))
		if tag.Name != "" {
			items = append(items, Name(tag.Name))
		}
		if tag.Content != "" {
			items = append(items, Attr("content", tag.Content))
		}
		if tag.Property != "" {
			items = append(items, Attr("property", tag.Property))
		}
		if tag.Charset != "" {
			items = append(items, Attr("charset", tag.Charset))
		}
		if tag.HTTPEquiv != "" {
			items = append(items, Attr("http-equiv", tag.HTTPEquiv))
		}
		if tag.ItemProp != "" {
			items = append(items, Attr("itemprop", tag.ItemProp))
		}
		if len(tag.Attrs) > 0 {
			keys := make([]string, 0, len(tag.Attrs))
			for k := range tag.Attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				items = append(items, Attr(k, tag.Attrs[k]))
			}
		}
		nodes = append(nodes, El(HTMLMetaElement{}, "meta", items...))
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
func LinkTags(tags ...LinkTag) []Node {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]Node, 0, len(tags))
	for _, tag := range tags {
		items := make([]Item, 0, 8+len(tag.Attrs))
		if tag.Rel != "" {
			items = append(items, Rel(tag.Rel))
		}
		if tag.Href != "" {
			items = append(items, Href(tag.Href))
		}
		if tag.Type != "" {
			items = append(items, Type(tag.Type))
		}
		if tag.As != "" {
			items = append(items, Attr("as", tag.As))
		}
		if tag.Media != "" {
			items = append(items, Attr("media", tag.Media))
		}
		if tag.HrefLang != "" {
			items = append(items, Attr("hreflang", tag.HrefLang))
		}
		if tag.Title != "" {
			items = append(items, Title(tag.Title))
		}
		if tag.CrossOrigin != "" {
			items = append(items, Attr("crossorigin", tag.CrossOrigin))
		}
		if tag.Integrity != "" {
			items = append(items, Attr("integrity", tag.Integrity))
		}
		if tag.ReferrerPolicy != "" {
			items = append(items, Attr("referrerpolicy", tag.ReferrerPolicy))
		}
		if tag.Sizes != "" {
			items = append(items, Attr("sizes", tag.Sizes))
		}
		if len(tag.Attrs) > 0 {
			keys := make([]string, 0, len(tag.Attrs))
			for k := range tag.Attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				items = append(items, Attr(k, tag.Attrs[k]))
			}
		}
		nodes = append(nodes, El(HTMLLinkElement{}, "link", items...))
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
func ScriptTags(tags ...ScriptTag) []Node {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]Node, 0, len(tags))
	for _, tag := range tags {
		items := make([]Item, 0, 10+len(tag.Attrs))
		if tag.Src != "" {
			items = append(items, Src(tag.Src))
		}
		if tag.Type != "" {
			items = append(items, Type(tag.Type))
		}
		if tag.Async {
			items = append(items, Attr("async", "async"))
		}
		if tag.Defer {
			items = append(items, Attr("defer", "defer"))
		}
		if tag.Module {
			items = append(items, Attr("type", "module"))
		}
		if tag.NoModule {
			items = append(items, Attr("nomodule", "nomodule"))
		}
		if tag.CrossOrigin != "" {
			items = append(items, Attr("crossorigin", tag.CrossOrigin))
		}
		if tag.Integrity != "" {
			items = append(items, Attr("integrity", tag.Integrity))
		}
		if tag.ReferrerPolicy != "" {
			items = append(items, Attr("referrerpolicy", tag.ReferrerPolicy))
		}
		if tag.Nonce != "" {
			items = append(items, Attr("nonce", tag.Nonce))
		}
		if len(tag.Attrs) > 0 {
			keys := make([]string, 0, len(tag.Attrs))
			for k := range tag.Attrs {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				items = append(items, Attr(k, tag.Attrs[k]))
			}
		}
		script := El(HTMLScriptElement{}, "script", items...)
		if tag.Inner != "" {
			script.Unsafe = &tag.Inner
		}
		nodes = append(nodes, script)
	}
	return nodes
}
