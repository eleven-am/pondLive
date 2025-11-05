package live

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type (
	Location   = runtime.Location
	RouteProps = runtime.RouteProps
	LinkProps  = runtime.LinkProps
	Match      = runtime.Match
	NavMsg     = runtime.NavMsg
	PopMsg     = runtime.PopMsg
)

var (
	Parse            = runtime.Parse
	NormalizePattern = runtime.NormalizePattern
	Prefer           = runtime.Prefer
	BestMatch        = runtime.BestMatch
	BuildHref        = runtime.BuildHref
	SetSearch        = runtime.SetSearch
	AddSearch        = runtime.AddSearch
	DelSearch        = runtime.DelSearch
	MergeSearch      = runtime.MergeSearch
	ClearSearch      = runtime.ClearSearch
	ParseHref        = runtime.ParseHref
	ErrMissingRouter = runtime.ErrMissingRouter
)

func Router(ctx Ctx, children ...Node) Node {
	return runtime.Router(ctx, children...)
}

func Routes(ctx Ctx, children ...Node) Node {
	return runtime.Routes(ctx, children...)
}

func Route(ctx Ctx, props RouteProps, children ...Node) Node {
	return runtime.Route(ctx, props, children...)
}

func Outlet(ctx Ctx) Node {
	return runtime.Outlet(ctx)
}

func Link(ctx Ctx, props LinkProps, children ...h.Item) Node {
	return runtime.RouterLink(ctx, props, children...)
}

func Navigate(ctx Ctx, href string) {
	runtime.RouterNavigate(ctx, href)
}

func Replace(ctx Ctx, href string) {
	runtime.RouterReplace(ctx, href)
}

func NavigateWithSearch(ctx Ctx, patch func(url.Values) url.Values) {
	runtime.RouterNavigateWithSearch(ctx, patch)
}

func ReplaceWithSearch(ctx Ctx, patch func(url.Values) url.Values) {
	runtime.RouterReplaceWithSearch(ctx, patch)
}

func Redirect(ctx Ctx, to string) Node {
	return runtime.RouterRedirect(ctx, to)
}

func UseLocation(ctx Ctx) Location {
	return runtime.UseLocation(ctx)
}

func UseParams(ctx Ctx) map[string]string {
	return runtime.UseParams(ctx)
}

func UseParam(ctx Ctx, key string) string {
	return runtime.UseParam(ctx, key)
}

func UseSearch(ctx Ctx) url.Values {
	return runtime.UseSearch(ctx)
}

func UseSearchParam(ctx Ctx, key string) (func() []string, func([]string)) {
	return runtime.UseSearchParam(ctx, key)
}

func LocEqual(a, b Location) bool {
	return runtime.LocEqual(a, b)
}

func UseMetadata(ctx Ctx, meta *Meta) {
	runtime.UseMetadata(ctx, meta)
}

func InternalSeedSessionLocation(sess *Session, loc Location) {
	runtime.InternalSeedSessionLocation(sess, loc)
}

func InternalSeedSessionParams(sess *Session, params map[string]string) {
	runtime.InternalSeedSessionParams(sess, params)
}

func InternalHandleNav(sess *Session, msg NavMsg) {
	runtime.InternalHandleNav(sess, msg)
}

func InternalHandlePop(sess *Session, msg PopMsg) {
	runtime.InternalHandlePop(sess, msg)
}
