package router2

import (
	"net/http"
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/headers2"
	"github.com/eleven-am/pondlive/go/internal/runtime2"
)

// Navigate triggers navigation to the specified href.
// In live mode, publishes to Bus for client-side navigation.
// In SSR mode, sets a redirect on the RequestState.
//
// Supports:
// - Absolute paths: "/about", "/users/123"
// - Hash-only: "#section"
// - Relative: "./edit", "../settings"
// - With query: "/search?q=foo"
func Navigate(ctx *runtime2.Ctx, href string) {
	navigate(ctx, href, false)
}

// Replace triggers navigation with history.replaceState semantics.
// Same as Navigate but doesn't create a new history entry.
func Replace(ctx *runtime2.Ctx, href string) {
	navigate(ctx, href, true)
}

// NavigateWithQuery navigates to a path with the given query parameters.
func NavigateWithQuery(ctx *runtime2.Ctx, path string, query url.Values) {
	href := buildHref(path, query, "")
	Navigate(ctx, href)
}

// ReplaceWithQuery replaces current location with path and query parameters.
func ReplaceWithQuery(ctx *runtime2.Ctx, path string, query url.Values) {
	href := buildHref(path, query, "")
	Replace(ctx, href)
}

// NavigateToHash navigates to a hash on the current path.
func NavigateToHash(ctx *runtime2.Ctx, hash string) {
	Navigate(ctx, "#"+hash)
}

// Back navigates in browser history.
// Only works in live mode.
func Back(ctx *runtime2.Ctx) {
	bus := getBus(ctx)
	if bus == nil {
		return
	}
	bus.Publish("router", "back", nil)
}

// Forward navigates in browser history.
// Only works in live mode.
func Forward(ctx *runtime2.Ctx) {
	bus := getBus(ctx)
	if bus == nil {
		return
	}
	bus.Publish("router", "forward", nil)
}

// navigate is the internal navigation implementation.
func navigate(ctx *runtime2.Ctx, href string, replace bool) {
	bus := getBus(ctx)

	if bus == nil {
		requestState := headers2.UseRequestState(ctx)
		if requestState != nil {

			currentLoc := &Location{
				Path:  requestState.Path(),
				Query: requestState.Query(),
				Hash:  requestState.Hash(),
			}
			target := resolveHref(currentLoc, href)
			redirectURL := buildHref(target.Path, target.Query, target.Hash)
			requestState.SetRedirect(redirectURL, http.StatusFound)
		}
		return
	}

	currentLoc := LocationContext.UseContextValue(ctx)
	if currentLoc == nil {
		currentLoc = &Location{Path: "/", Query: url.Values{}}
	}

	target := resolveHref(currentLoc, href)
	target = canonicalizeLocation(target)

	bus.Publish("router", "navigate", NavPayload{
		Path:    target.Path,
		Query:   target.Query.Encode(),
		Hash:    target.Hash,
		Replace: replace,
	})
}
