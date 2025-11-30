package metatags

import (
	"sort"

	"github.com/eleven-am/pondlive/go/internal/work"
)

type MetaTag struct {
	Name      string
	Content   string
	Property  string
	Charset   string
	HTTPEquiv string
	ItemProp  string
	Attrs     map[string]string
}

func (m MetaTag) ToNode() work.Node {
	attrs := make(map[string][]string)
	if m.Name != "" {
		attrs["name"] = []string{m.Name}
	}
	if m.Content != "" {
		attrs["content"] = []string{m.Content}
	}
	if m.Property != "" {
		attrs["property"] = []string{m.Property}
	}
	if m.Charset != "" {
		attrs["charset"] = []string{m.Charset}
	}
	if m.HTTPEquiv != "" {
		attrs["http-equiv"] = []string{m.HTTPEquiv}
	}
	if m.ItemProp != "" {
		attrs["itemprop"] = []string{m.ItemProp}
	}
	if len(m.Attrs) > 0 {
		keys := make([]string, 0, len(m.Attrs))
		for k := range m.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			attrs[k] = []string{m.Attrs[k]}
		}
	}
	return &work.Element{
		Tag:   "meta",
		Attrs: attrs,
	}
}

func MetaTags(tags ...MetaTag) []work.Node {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]work.Node, 0, len(tags))
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

func (l LinkTag) ToNode() work.Node {
	attrs := make(map[string][]string)
	if l.Rel != "" {
		attrs["rel"] = []string{l.Rel}
	}
	if l.Href != "" {
		attrs["href"] = []string{l.Href}
	}
	if l.Type != "" {
		attrs["type"] = []string{l.Type}
	}
	if l.As != "" {
		attrs["as"] = []string{l.As}
	}
	if l.Media != "" {
		attrs["media"] = []string{l.Media}
	}
	if l.HrefLang != "" {
		attrs["hreflang"] = []string{l.HrefLang}
	}
	if l.Title != "" {
		attrs["title"] = []string{l.Title}
	}
	if l.CrossOrigin != "" {
		attrs["crossorigin"] = []string{l.CrossOrigin}
	}
	if l.Integrity != "" {
		attrs["integrity"] = []string{l.Integrity}
	}
	if l.ReferrerPolicy != "" {
		attrs["referrerpolicy"] = []string{l.ReferrerPolicy}
	}
	if l.Sizes != "" {
		attrs["sizes"] = []string{l.Sizes}
	}
	if len(l.Attrs) > 0 {
		keys := make([]string, 0, len(l.Attrs))
		for k := range l.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			attrs[k] = []string{l.Attrs[k]}
		}
	}
	return &work.Element{
		Tag:   "link",
		Attrs: attrs,
	}
}

func LinkTags(tags ...LinkTag) []work.Node {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]work.Node, 0, len(tags))
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

func (s ScriptTag) ToNode() work.Node {
	attrs := make(map[string][]string)
	if s.Src != "" {
		attrs["src"] = []string{s.Src}
	}
	if s.Type != "" && !s.Module {
		attrs["type"] = []string{s.Type}
	}
	if s.Async {
		attrs["async"] = []string{""}
	}
	if s.Defer {
		attrs["defer"] = []string{""}
	}
	if s.Module {
		attrs["type"] = []string{"module"}
	}
	if s.NoModule {
		attrs["nomodule"] = []string{""}
	}
	if s.CrossOrigin != "" {
		attrs["crossorigin"] = []string{s.CrossOrigin}
	}
	if s.Integrity != "" {
		attrs["integrity"] = []string{s.Integrity}
	}
	if s.ReferrerPolicy != "" {
		attrs["referrerpolicy"] = []string{s.ReferrerPolicy}
	}
	if s.Nonce != "" {
		attrs["nonce"] = []string{s.Nonce}
	}
	if len(s.Attrs) > 0 {
		keys := make([]string, 0, len(s.Attrs))
		for k := range s.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			attrs[k] = []string{s.Attrs[k]}
		}
	}

	el := &work.Element{
		Tag:   "script",
		Attrs: attrs,
	}
	if s.Inner != "" {
		el.UnsafeHTML = s.Inner
	}
	return el
}

func ScriptTags(tags ...ScriptTag) []work.Node {
	if len(tags) == 0 {
		return nil
	}
	nodes := make([]work.Node, 0, len(tags))
	for _, tag := range tags {
		nodes = append(nodes, tag.ToNode())
	}
	return nodes
}
