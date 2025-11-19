package router

import (
	"fmt"
	"strconv"

	"github.com/eleven-am/pondlive/go/internal/dom2"
)

type LinkProps struct {
	To      string
	Replace bool
}

func Link(ctx Ctx, p LinkProps, children ...*dom2.StructuredNode) *dom2.StructuredNode {
	state := requireRouterState(ctx)
	base := state.getLoc()
	target := resolveHref(base, p.To)
	href := BuildHref(target.Path, target.Query, target.Hash)

	replaceAttr := strconv.FormatBool(p.Replace)
	encodedQuery := encodeQuery(target.Query)

	link := dom2.ElementNode("a")
	link.Attrs = map[string][]string{
		"href": {href},
	}
	link.Router = &dom2.RouterMeta{
		PathValue: target.Path,
		Query:     encodedQuery,
		Hash:      target.Hash,
		Replace:   replaceAttr,
	}
	link.WithChildren(children...)
	return link
}

func datasetLocation(ev dom2.Event) (Location, bool) {
	path := eventString(ev, "currentTarget.dataset.routerPath")
	if path == "" {
		return Location{}, false
	}
	query := parseQuery(eventString(ev, "currentTarget.dataset.routerQuery"))
	hash := eventString(ev, "currentTarget.dataset.routerHash")
	loc := Location{
		Path:  normalizePath(path),
		Query: query,
		Hash:  normalizeHash(hash),
	}
	return canonicalizeLocation(loc), true
}

func eventString(ev dom2.Event, key string) string {
	if ev.Payload == nil {
		return ""
	}
	val, ok := ev.Payload[key]
	if !ok || val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	default:
		return ""
	}
}
