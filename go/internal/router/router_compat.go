package router

import (
	"net/url"

	h "github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// API compatibility wrappers for v2 router API.
// These functions wrap the router implementations to provide the same API surface
// as the previous router package.

// Link wraps RouterLink to provide v2 API compatibility.
func Link(ctx Ctx, props LinkProps, children ...h.Item) h.Node {
	return RouterLink(ctx, props, children...)
}

// Navigate wraps RouterNavigate to provide v2 API compatibility.
func Navigate(ctx runtime.Ctx, href string) {
	RouterNavigate(ctx, href)
}

// Replace wraps RouterReplace to provide v2 API compatibility.
func Replace(ctx runtime.Ctx, href string) {
	RouterReplace(ctx, href)
}

// NavigateWithSearch wraps RouterNavigateWithSearch to provide v2 API compatibility.
func NavigateWithSearch(ctx runtime.Ctx, patch func(url.Values) url.Values) {
	RouterNavigateWithSearch(ctx, patch)
}

// ReplaceWithSearch wraps RouterReplaceWithSearch to provide v2 API compatibility.
func ReplaceWithSearch(ctx runtime.Ctx, patch func(url.Values) url.Values) {
	RouterReplaceWithSearch(ctx, patch)
}

// Redirect wraps RouterRedirect to provide v2 API compatibility.
func Redirect(ctx runtime.Ctx, to string) h.Node {
	return RouterRedirect(ctx, to)
}
