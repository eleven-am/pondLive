package router

import (
	"net/http"
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Navigate triggers navigation to the specified href.
// In live mode, publishes to Bus for client-side navigation.
// In SSR mode, sets a redirect on the requestState.
//
// Supports:
// - Absolute paths: "/about", "/users/123"
// - Hash-only: "#section"
// - Relative: "./edit", "../settings"
// - With query: "/search?q=foo"
func Navigate(ctx *runtime.Ctx, href string) {
	navigate(ctx, href, false)
}

// Replace triggers navigation with history.replaceState semantics.
// Same as Navigate but doesn't create a new history entry.
func Replace(ctx *runtime.Ctx, href string) {
	navigate(ctx, href, true)
}

// NavigateWithQuery navigates to a path with the given query parameters.
func NavigateWithQuery(ctx *runtime.Ctx, path string, query url.Values) {
	href := buildHref(path, query, "")
	Navigate(ctx, href)
}

// ReplaceWithQuery replaces current location with path and query parameters.
func ReplaceWithQuery(ctx *runtime.Ctx, path string, query url.Values) {
	href := buildHref(path, query, "")
	Replace(ctx, href)
}

// NavigateToHash navigates to a hash on the current path.
func NavigateToHash(ctx *runtime.Ctx, hash string) {
	Navigate(ctx, "#"+hash)
}

// Back navigates in browser history.
// Only works in live mode.
func Back(ctx *runtime.Ctx) {
	bus := getBus(ctx)
	if bus == nil {
		return
	}
	bus.PublishRouterBack()
}

// Forward navigates in browser history.
// Only works in live mode.
func Forward(ctx *runtime.Ctx) {
	bus := getBus(ctx)
	if bus == nil {
		return
	}
	bus.PublishRouterForward()
}

// navigate is the internal navigation implementation.
func navigate(ctx *runtime.Ctx, href string, replace bool) {
	requestState := headers.UseRequestState(ctx)
	currentLoc, setLocation := LocationContext.UseContext(ctx)
	bus := getBus(ctx)

	if bus == nil || requestState == nil || !requestState.IsLive() {
		if requestState != nil {
			currentLoc := &Location{
				Path:  requestState.Path(),
				Query: requestState.Query(),
				Hash:  requestState.Hash(),
			}

			target := resolveHref(currentLoc, href)
			redirectURL := buildHref(target.Path, target.Query, target.Hash)
			status := http.StatusFound
			if replace {
				status = http.StatusSeeOther
			}
			requestState.SetRedirect(redirectURL, status)
		}

		return
	}

	if currentLoc == nil {
		currentLoc = &Location{Path: "/", Query: url.Values{}}
	}

	target := resolveHref(currentLoc, href)
	target = canonicalizeLocation(target)

	if setLocation != nil {
		setLocation(target)
	}

	payload := protocol.RouterNavPayload{
		Path:    target.Path,
		Query:   target.Query.Encode(),
		Hash:    target.Hash,
		Replace: replace,
	}

	if replace {
		bus.PublishRouterReplace(payload)
	} else {
		bus.PublishRouterPush(payload)
	}
}
