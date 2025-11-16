package router

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/route"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type LinkProps struct {
	To      string
	Replace bool
}

// Link renders a router-aware anchor that resolves relative URLs via the active store.
func Link(ctx runtime.Ctx, props LinkProps, children ...h.Item) h.Node {

	store := routerStoreCtx.Use(ctx)
	if store != nil {

		return renderLinkWithStore(ctx, store, props, children)
	}

	return &linkNode{props: props, children: children}
}

// linkNode is a placeholder for Link that gets resolved when RouterStore becomes available
type linkNode struct {
	props    LinkProps
	children []h.Item
}

func (n *linkNode) ApplyTo(e *dom.Element) { e.Children = append(e.Children, n) }
func (*linkNode) isNode()                  {}
func (*linkNode) privateNodeTag()          {}

func (n *linkNode) resolve(ctx runtime.Ctx, store *RouterStore) dom.Node {
	if n == nil || store == nil {
		return &dom.FragmentNode{}
	}
	return renderLinkWithStore(ctx, store, n.props, n.children)
}

func renderLinkWithStore(ctx runtime.Ctx, store *RouterStore, props LinkProps, children []h.Item) h.Node {
	target := resolveHref(store.Location(), props.To)
	href := buildHref(target)

	handler := func(ev h.Event) h.Updates {
		if ev.Mods.Ctrl || ev.Mods.Meta || ev.Mods.Shift || ev.Mods.Alt || ev.Mods.Button != 0 {
			return nil
		}
		current := store.Location()
		next := extractTargetFromEvent(ev.Payload, current)
		if next.Path == "" {
			return nil
		}
		if route.LocEqual(toRuntimeLocation(current), toRuntimeLocation(next)) {
			return nil
		}
		if props.Replace {
			store.RecordNavigation(NavKindReplace, next)
		} else {
			store.RecordNavigation(NavKindPush, next)
		}
		return nil
	}

	items := []h.Item{
		h.Href(href),
		h.OnWith("click", h.EventOptions{Props: []string{"currentTarget.href"}}, handler),
	}
	if len(children) > 0 {
		items = append(items, children...)
	}
	link := h.A(items...)
	attachRouterMetaNode(link, target, props.Replace)
	return link
}

func attachRouterMetaNode(node h.Node, target Location, replace bool) {
	el, ok := node.(*h.Element)
	if !ok {
		return
	}
	meta := dom.RouterMeta{
		Path:    target.Path,
		Query:   encodeQuery(target.Query),
		Hash:    target.Hash,
		Replace: strconv.FormatBool(replace),
	}
	el.RouterMeta = &meta
}

func extractTargetFromEvent(payload map[string]any, base Location) Location {
	href := ""
	if raw, ok := payload["currentTarget.href"].(string); ok {
		href = raw
	}
	if href == "" {
		return Location{}
	}
	return resolveHref(base, href)
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
