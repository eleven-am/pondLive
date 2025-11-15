package router2

import (
	"net/url"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

// LinkProps configures router-aware links.
type LinkProps struct {
	To      string
	Replace bool
}

// RenderLink builds an <a> element targeting the provided href relative to the
// store's current location. Event wiring happens later in the runtime layer;
// this function focuses on canonical href construction.
func RenderLink(store *RouterStore, props LinkProps, children ...dom.Node) *dom.Element {
	target := Location{Path: props.To}
	if store != nil {
		base := store.Location()
		target = resolveHref(base, props.To)
	}
	href := buildHref(target)
	el := &dom.Element{Tag: "a", Attrs: map[string]string{"href": href}}
	if len(children) > 0 {
		el.Children = make([]dom.Node, len(children))
		copy(el.Children, children)
	}
	return el
}

// LinkNode constructs a placeholder node that resolves into an anchor using the
// router store.
func LinkNode(props LinkProps, children ...dom.Node) dom.Node {
	return &linkNode{props: props, children: children}
}

type linkNode struct {
	props    LinkProps
	children []dom.Node
}

func (n *linkNode) ApplyTo(e *dom.Element) { e.Children = append(e.Children, n) }
func (*linkNode) isNode()                  {}
func (*linkNode) privateNodeTag()          {}

func (n *linkNode) resolve(store *RouterStore) dom.Node {
	return RenderLink(store, n.props, n.children...)
}

func resolveHref(base Location, href string) Location {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return base
	}
	if strings.HasPrefix(trimmed, "./") {
		next := base
		next.Path = resolvePattern(trimmed, base.Path)
		return canonicalizeLocation(next)
	}
	if strings.HasPrefix(trimmed, "#") {
		next := base
		next.Hash = normalizeHash(trimmed, "")
		return canonicalizeLocation(next)
	}
	baseURL := &url.URL{
		Path:     base.Path,
		RawQuery: encodeQuery(base.Query),
		Fragment: base.Hash,
	}
	parsed, err := baseURL.Parse(trimmed)
	if err != nil {
		return base
	}
	return locationFromURL(parsed)
}

func locationFromURL(u *url.URL) Location {
	if u == nil {
		return canonicalizeLocation(Location{Path: "/"})
	}
	loc := Location{
		Path:  u.Path,
		Query: u.Query(),
		Hash:  u.Fragment,
	}
	return canonicalizeLocation(loc)
}

func buildHref(loc Location) string {
	canon := canonicalizeLocation(loc)
	var builder strings.Builder
	builder.WriteString(canon.Path)
	if encoded := encodeQuery(canon.Query); encoded != "" {
		builder.WriteByte('?')
		builder.WriteString(encoded)
	}
	if canon.Hash != "" {
		builder.WriteByte('#')
		builder.WriteString(canon.Hash)
	}
	return builder.String()
}
