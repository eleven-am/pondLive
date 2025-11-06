package runtime

import (
	"fmt"
	"strconv"
	"strings"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type LinkProps struct {
	To      string
	Replace bool
}

func RouterLink(ctx Ctx, p LinkProps, children ...h.Item) h.Node {
	state := routerStateCtx.Use(ctx)
	sess := ctx.Session()
	if sessionRendering(sess) && state.getLoc != nil {
		return renderLink(ctx, p, children...)
	}
	return newLinkPlaceholder(sess, p, children)
}

type linkNode struct {
	*h.FragmentNode
	props    LinkProps
	children []h.Item
}

func newLinkPlaceholder(sess *ComponentSession, p LinkProps, children []h.Item) *linkNode {
	node := &linkNode{FragmentNode: h.Fragment(), props: p, children: children}
	if sess != nil {
		if state := sess.ensureRouterState(); state != nil {
			state.linkPlaceholders.Store(node.FragmentNode, node)
		}
	}
	return node
}

func consumeLinkPlaceholder(sess *ComponentSession, f *h.FragmentNode) (*linkNode, bool) {
	if f == nil || sess == nil {
		return nil, false
	}
	if state := sess.loadRouterState(); state != nil {
		if value, ok := state.linkPlaceholders.LoadAndDelete(f); ok {
			if node, okCast := value.(*linkNode); okCast {
				return node, true
			}
		}
	}
	return nil, false
}

func renderLink(ctx Ctx, p LinkProps, children ...h.Item) h.Node {
	state := requireRouterState(ctx)
	base := state.getLoc()
	target := resolveHref(base, p.To)
	href := BuildHref(target.Path, target.Query, target.Hash)

	handler := func(ev h.Event) h.Updates {
		if ev.Mods.Ctrl || ev.Mods.Meta || ev.Mods.Shift || ev.Mods.Alt || ev.Mods.Button != 0 {
			return nil
		}
		state := requireRouterState(ctx)
		current := state.getLoc()
		next := extractTargetFromEvent(ev, current)
		if next.Path == "" {
			return nil
		}
		replace := strings.EqualFold(payloadString(ev.Payload, "target.dataset.routerReplace"), "true")

		if LocEqual(current, next) {
			return nil
		}
		performLocationUpdate(ctx, next, replace, true)
		return nil
	}

	replaceAttr := strconv.FormatBool(p.Replace)
	encodedQuery := encodeQuery(target.Query)
	items := []h.Item{
		h.Href(href),
		h.Data("router-path", target.Path),
		h.Data("router-query", encodedQuery),
		h.Data("router-hash", target.Hash),
		h.Data("router-replace", replaceAttr),
		h.OnWith("click", h.EventOptions{Props: []string{
			"currentTarget.dataset.routerPath",
			"currentTarget.dataset.routerQuery",
			"currentTarget.dataset.routerHash",
			"currentTarget.dataset.routerReplace",
			"currentTarget.href",
		}}, handler),
	}
	if len(children) > 0 {
		items = append(items, children...)
	}
	return h.A(items...)
}

func extractTargetFromEvent(ev h.Event, base Location) Location {
	datasetPath := payloadString(ev.Payload, "currentTarget.dataset.routerPath")
	datasetQuery := payloadString(ev.Payload, "currentTarget.dataset.routerQuery")
	datasetHash := payloadString(ev.Payload, "currentTarget.dataset.routerHash")

	if datasetPath != "" {
		target := Location{
			Path:  datasetPath,
			Query: parseQuery(datasetQuery),
			Hash:  normalizeHash(datasetHash),
		}
		return canonicalizeLocation(target)
	}

	href := payloadString(ev.Payload, "currentTarget.href")
	if href == "" {
		return Location{}
	}
	return resolveHref(base, href)
}

func payloadString(payload map[string]any, key string) string {
	if len(payload) == 0 {
		return ""
	}
	raw, ok := payload[key]
	if !ok || raw == nil {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return fmt.Sprint(v)
	}
}
