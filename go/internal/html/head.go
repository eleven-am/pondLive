package html

import "sort"

type MetaTag struct {
	Name      string
	Content   string
	Property  string
	Charset   string
	HTTPEquiv string
	ItemProp  string
	Attrs     map[string]string
}

func (m MetaTag) ToNode() Node {
	items := make([]Item, 0, 6+len(m.Attrs))
	if m.Name != "" {
		items = append(items, Name(m.Name))
	}
	if m.Content != "" {
		items = append(items, Attr("content", m.Content))
	}
	if m.Property != "" {
		items = append(items, Attr("property", m.Property))
	}
	if m.Charset != "" {
		items = append(items, Attr("charset", m.Charset))
	}
	if m.HTTPEquiv != "" {
		items = append(items, Attr("http-equiv", m.HTTPEquiv))
	}
	if m.ItemProp != "" {
		items = append(items, Attr("itemprop", m.ItemProp))
	}
	if len(m.Attrs) > 0 {
		keys := make([]string, 0, len(m.Attrs))
		for k := range m.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			items = append(items, Attr(k, m.Attrs[k]))
		}
	}
	return Meta(items...)
}

func MetaTags(tags ...MetaTag) []Node {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]Node, 0, len(tags))
	for _, tag := range tags {
		nodes = append(nodes, tag.ToNode())
	}
	return nodes
}

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

func (l LinkTag) ToNode() Node {
	items := make([]Item, 0, 11+len(l.Attrs))
	if l.Rel != "" {
		items = append(items, Rel(l.Rel))
	}
	if l.Href != "" {
		items = append(items, Href(l.Href))
	}
	if l.Type != "" {
		items = append(items, Type(l.Type))
	}
	if l.As != "" {
		items = append(items, Attr("as", l.As))
	}
	if l.Media != "" {
		items = append(items, Attr("media", l.Media))
	}
	if l.HrefLang != "" {
		items = append(items, Attr("hreflang", l.HrefLang))
	}
	if l.Title != "" {
		items = append(items, Title(l.Title))
	}
	if l.CrossOrigin != "" {
		items = append(items, Attr("crossorigin", l.CrossOrigin))
	}
	if l.Integrity != "" {
		items = append(items, Attr("integrity", l.Integrity))
	}
	if l.ReferrerPolicy != "" {
		items = append(items, Attr("referrerpolicy", l.ReferrerPolicy))
	}
	if l.Sizes != "" {
		items = append(items, Attr("sizes", l.Sizes))
	}
	if len(l.Attrs) > 0 {
		keys := make([]string, 0, len(l.Attrs))
		for k := range l.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			items = append(items, Attr(k, l.Attrs[k]))
		}
	}
	return Link(items...)
}

func LinkTags(tags ...LinkTag) []Node {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]Node, 0, len(tags))
	for _, tag := range tags {
		nodes = append(nodes, tag.ToNode())
	}
	return nodes
}

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

func (s ScriptTag) ToNode() Node {
	items := make([]Item, 0, 10+len(s.Attrs))
	if s.Src != "" {
		items = append(items, Src(s.Src))
	}
	if s.Type != "" && !s.Module {
		items = append(items, Type(s.Type))
	}
	if s.Async {
		items = append(items, boolAttrItem{name: "async"})
	}
	if s.Defer {
		items = append(items, boolAttrItem{name: "defer"})
	}
	if s.Module {
		items = append(items, Type("module"))
	}
	if s.NoModule {
		items = append(items, boolAttrItem{name: "nomodule"})
	}
	if s.CrossOrigin != "" {
		items = append(items, Attr("crossorigin", s.CrossOrigin))
	}
	if s.Integrity != "" {
		items = append(items, Attr("integrity", s.Integrity))
	}
	if s.ReferrerPolicy != "" {
		items = append(items, Attr("referrerpolicy", s.ReferrerPolicy))
	}
	if s.Nonce != "" {
		items = append(items, Attr("nonce", s.Nonce))
	}
	if len(s.Attrs) > 0 {
		keys := make([]string, 0, len(s.Attrs))
		for k := range s.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			items = append(items, Attr(k, s.Attrs[k]))
		}
	}
	if s.Inner != "" {
		items = append(items, UnsafeHTML(s.Inner))
	}
	return ScriptEl(items...)
}

func ScriptTags(tags ...ScriptTag) []Node {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]Node, 0, len(tags))
	for _, tag := range tags {
		nodes = append(nodes, tag.ToNode())
	}
	return nodes
}
