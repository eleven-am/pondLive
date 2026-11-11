package head

import (
	"slices"
	"strings"

	"github.com/eleven-am/pondlive/internal/htmlutil"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

const (
	DefaultTitle       = "PondLive Application"
	DefaultDescription = "A PondLive application"
)

type Head struct {
	Title          runtime.ReadSignal[string]
	Description    runtime.ReadSignal[string]
	HTMLLang       runtime.ReadSignal[string]
	HTMLDir        runtime.ReadSignal[string]
	HTMLClass      runtime.ReadSignal[string]
	BodyClass      runtime.ReadSignal[string]
	Meta           []MetaTag
	Links          []LinkTag
	HeadScripts    []ScriptTag
	Styles         []StyleTag
	Icon           work.Node
	IconBackground string
	IconColor      string
}

type entry struct {
	depth int
	seq   uint64
	h     Head
}

type iconInfo struct {
	node       work.Node
	background string
	color      string
}

type merged struct {
	title       string
	description string
	lang        string
	dir         string
	htmlClass   string
	bodyClass   string
	meta        []MetaTag
	links       []LinkTag
	scripts     []ScriptTag
	styles      []StyleTag
	icon        *iconInfo
}

func read(s runtime.ReadSignal[string]) string {
	if s == nil {
		return ""
	}
	return s.Get()
}

func ordered(entries []*entry) []*entry {
	out := slices.Clone(entries)
	slices.SortFunc(out, func(a, b *entry) int {
		if a.depth != b.depth {
			return a.depth - b.depth
		}
		switch {
		case a.seq < b.seq:
			return -1
		case a.seq > b.seq:
			return 1
		}
		return 0
	})
	return out
}

func mergeList[T any](keys *[]string, values map[string]T, items []T, identity func(T) string) {
	for _, item := range items {
		k := identity(item)
		if _, ok := values[k]; !ok {
			*keys = append(*keys, k)
		}
		values[k] = item
	}
}

func collectList[T any](keys []string, values map[string]T) []T {
	out := make([]T, 0, len(keys))
	for _, k := range keys {
		out = append(out, values[k])
	}
	return out
}

func mergeEntries(entries []*entry) *merged {
	m := &merged{title: DefaultTitle, description: DefaultDescription}
	var htmlClasses, bodyClasses []string
	var metaKeys, linkKeys, scriptKeys, styleKeys []string
	metas, links, scripts, styles := map[string]MetaTag{}, map[string]LinkTag{}, map[string]ScriptTag{}, map[string]StyleTag{}

	for _, e := range ordered(entries) {
		h := e.h
		if v := read(h.Title); v != "" {
			m.title = v
		}
		if v := read(h.Description); v != "" {
			m.description = v
		}
		if v := read(h.HTMLLang); v != "" {
			m.lang = v
		}
		if v := read(h.HTMLDir); v != "" {
			m.dir = v
		}
		htmlClasses = append(htmlClasses, read(h.HTMLClass))
		bodyClasses = append(bodyClasses, read(h.BodyClass))
		mergeList(&metaKeys, metas, h.Meta, MetaTag.identity)
		mergeList(&linkKeys, links, h.Links, LinkTag.identity)
		mergeList(&scriptKeys, scripts, h.HeadScripts, ScriptTag.identity)
		mergeList(&styleKeys, styles, h.Styles, StyleTag.identity)
		if h.Icon != nil {
			m.icon = &iconInfo{node: h.Icon, background: h.IconBackground, color: h.IconColor}
		}
	}
	m.htmlClass = htmlutil.UniqueClasses(strings.Join(htmlClasses, " "))
	m.bodyClass = htmlutil.UniqueClasses(strings.Join(bodyClasses, " "))
	m.meta = collectList(metaKeys, metas)
	m.links = collectList(linkKeys, links)
	m.scripts = collectList(scriptKeys, scripts)
	m.styles = collectList(styleKeys, styles)
	return m
}

func equalBy[T any](sig func(T) string) func(a, b []T) bool {
	return func(a, b []T) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if sig(a[i]) != sig(b[i]) {
				return false
			}
		}
		return true
	}
}

func iconEqual(a, b *iconInfo) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.node == b.node && a.background == b.background && a.color == b.color
}
