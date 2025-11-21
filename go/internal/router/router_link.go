package router

import (
	"strconv"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

type LinkProps struct {
	To      string
	Replace bool
}

func Link(ctx Ctx, p LinkProps, children ...*dom.StructuredNode) *dom.StructuredNode {
	controller := UseRouterState(ctx)
	state := controller.Get()
	base := state.Location
	target := resolveHref(base, p.To)
	href := BuildHref(target.Path, target.Query, target.Hash)

	replaceAttr := strconv.FormatBool(p.Replace)
	encodedQuery := encodeQuery(target.Query)

	link := dom.ElementNode("div")
	link.Attrs = map[string][]string{
		"href": {href},
	}
	link.Router = &dom.RouterMeta{
		PathValue: target.Path,
		Query:     encodedQuery,
		Hash:      target.Hash,
		Replace:   replaceAttr,
	}
	link.WithChildren(children...)
	return link
}
