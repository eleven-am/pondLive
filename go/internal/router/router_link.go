package router

import (
	"strconv"

	h "github.com/eleven-am/pondlive/go/internal/html"
)

type LinkProps struct {
	To      string
	Replace bool
}

func RouterLink(ctx Ctx, p LinkProps, children ...h.Item) h.Node {
	state := requireRouterState(ctx)
	base := state.getLoc()
	target := resolveHref(base, p.To)
	href := BuildHref(target.Path, target.Query, target.Hash)

	replaceAttr := strconv.FormatBool(p.Replace)
	encodedQuery := encodeQuery(target.Query)
	items := make([]h.Item, 0, len(children)+2)
	items = append(items, h.Href(href))
	items = append(items, h.RouterMeta(target.Path, encodedQuery, target.Hash, replaceAttr))
	items = append(items, children...)
	return h.El(h.HTMLDivElement{}, "div", items...)
}
