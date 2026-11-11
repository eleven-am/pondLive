package head

import (
	"sort"
	"strings"

	"github.com/eleven-am/pondlive/internal/work"
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

type StyleTag struct {
	ID    string
	Media string
	CSS   string
}

type pair struct{ name, value string }

func collect(pairs []pair, extra map[string]string) []pair {
	if len(extra) == 0 {
		return pairs
	}
	keys := make([]string, 0, len(extra))
	for k := range extra {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		pairs = append(pairs, pair{k, extra[k]})
	}
	return pairs
}

func build(tag string, pairs []pair, presence []string, inner string) work.Node {
	items := make([]work.Item, 0, len(pairs)+len(presence)+1)
	for _, p := range pairs {
		if p.value != "" {
			items = append(items, work.Attribute(p.name, p.value))
		}
	}
	for _, name := range presence {
		items = append(items, work.Attribute(name, true))
	}
	if inner != "" {
		items = append(items, work.UnsafeHTML(inner))
	}
	return work.BuildElement(tag, items...)
}

func (m MetaTag) pairs() []pair {
	return collect([]pair{
		{"name", m.Name}, {"content", m.Content}, {"property", m.Property},
		{"charset", m.Charset}, {"http-equiv", m.HTTPEquiv}, {"itemprop", m.ItemProp},
	}, m.Attrs)
}

func (m MetaTag) ToNode() work.Node { return build("meta", m.pairs(), nil, "") }

func (l LinkTag) pairs() []pair {
	return collect([]pair{
		{"rel", l.Rel}, {"href", l.Href}, {"type", l.Type}, {"as", l.As}, {"media", l.Media},
		{"hreflang", l.HrefLang}, {"title", l.Title}, {"crossorigin", l.CrossOrigin},
		{"integrity", l.Integrity}, {"referrerpolicy", l.ReferrerPolicy}, {"sizes", l.Sizes},
	}, l.Attrs)
}

func (l LinkTag) ToNode() work.Node { return build("link", l.pairs(), nil, "") }

func (s ScriptTag) pairs() ([]pair, []string) {
	typ := s.Type
	if s.Module {
		typ = "module"
	}
	pairs := collect([]pair{
		{"src", s.Src}, {"type", typ}, {"crossorigin", s.CrossOrigin}, {"integrity", s.Integrity},
		{"referrerpolicy", s.ReferrerPolicy}, {"nonce", s.Nonce},
	}, s.Attrs)
	var presence []string
	if s.Async {
		presence = append(presence, "async")
	}
	if s.Defer {
		presence = append(presence, "defer")
	}
	if s.NoModule {
		presence = append(presence, "nomodule")
	}
	return pairs, presence
}

func (s ScriptTag) ToNode() work.Node {
	pairs, presence := s.pairs()
	return build("script", pairs, presence, s.Inner)
}

func signature(kind string, pairs []pair, presence []string, inner string) string {
	var b strings.Builder
	b.WriteString(kind)
	for _, p := range pairs {
		if p.value == "" {
			continue
		}
		b.WriteByte('|')
		b.WriteString(p.name)
		b.WriteByte('=')
		b.WriteString(p.value)
	}
	for _, name := range presence {
		b.WriteByte('|')
		b.WriteString(name)
	}
	if inner != "" {
		b.WriteString("|inner=")
		b.WriteString(inner)
	}
	return b.String()
}

func (s StyleTag) pairs() []pair { return []pair{{"media", s.Media}} }

func (s StyleTag) ToNode() work.Node {
	items := []work.Item{work.TextOf(s.CSS)}
	if s.Media != "" {
		items = append([]work.Item{work.Attribute("media", s.Media)}, items...)
	}
	return work.BuildElement("style", items...)
}

func (s StyleTag) signature() string { return signature("style", s.pairs(), nil, s.CSS) }

func (s StyleTag) identity() string {
	if s.ID != "" {
		return "id:" + s.ID
	}
	return "sig:" + s.signature()
}

func (m MetaTag) signature() string { return signature("meta", m.pairs(), nil, "") }
func (l LinkTag) signature() string { return signature("link", l.pairs(), nil, "") }
func (s ScriptTag) signature() string {
	pairs, presence := s.pairs()
	return signature("script", pairs, presence, s.Inner)
}

func (m MetaTag) identity() string {
	switch {
	case m.Name != "":
		return "name:" + m.Name
	case m.Property != "":
		return "property:" + m.Property
	case m.HTTPEquiv != "":
		return "http-equiv:" + m.HTTPEquiv
	case m.Charset != "":
		return "charset"
	case m.ItemProp != "":
		return "itemprop:" + m.ItemProp
	}
	return "sig:" + m.signature()
}

func (l LinkTag) identity() string { return l.Rel + ":" + l.Href }

func (s ScriptTag) identity() string {
	if s.Src != "" {
		return "src:" + s.Src
	}
	return "sig:" + s.signature()
}
