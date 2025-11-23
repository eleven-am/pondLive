package router

import (
	"net/url"
	"strings"
)

// Navigate navigates to a new location (pushState).
// Updates RequestController and syncs browser history.
//
// Usage:
//
//	router.Navigate(ctx, "/about")
//	router.Navigate(ctx, "/users/123")
//	router.Navigate(ctx, "#top")
func Navigate(ctx Ctx, href string) {
	performNavigation(ctx, href, false)
}

// Replace replaces the current location (replaceState).
// Updates RequestController and syncs browser history without adding a new entry.
//
// Usage:
//
//	router.Replace(ctx, "/login")
func Replace(ctx Ctx, href string) {
	performNavigation(ctx, href, true)
}

// performNavigation handles the actual navigation logic.
// Updates RequestController and calls ctx.EnqueueNavigation for browser sync.
func performNavigation(ctx Ctx, href string, replace bool) {
	controller := useRouterController(ctx)
	if controller == nil || controller.requestController == nil {
		return
	}

	current := controller.GetLocation()

	target := resolveHref(current, href)

	target = canonicalizeLocation(target)
	if locationEqual(current, target) {
		return
	}

	controller.requestController.SetCurrentLocation(target.Path, target.Query, target.Hash)

	recordNavigation(ctx, target, replace)
}

// recordNavigation syncs browser history via ctx.EnqueueNavigation.
func recordNavigation(ctx Ctx, loc Location, replace bool) {
	href := buildHref(loc.Path, loc.Query, loc.Hash)
	ctx.EnqueueNavigation(href, replace)
}

// resolveHref resolves a potentially relative href against a base location.
// Supports:
// - Absolute paths: "/about" -> {Path: "/about"}
// - Hash-only: "#top" -> {Path: base.Path, Hash: "top"}
// - Relative paths: "../users" -> resolved against base.Path
func resolveHref(base Location, href string) Location {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return base
	}

	if strings.HasPrefix(trimmed, "#") {
		next := base
		next.Hash = normalizeHash(trimmed)
		return next
	}

	if strings.HasPrefix(trimmed, "/") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return base
		}
		return Location{
			Path:  parsed.Path,
			Query: parsed.Query(),
			Hash:  parsed.Fragment,
		}
	}

	baseURL := &url.URL{
		Path:     base.Path,
		RawQuery: base.Query.Encode(),
		Fragment: base.Hash,
	}
	parsed, err := baseURL.Parse(trimmed)
	if err != nil {
		return base
	}

	return Location{
		Path:  parsed.Path,
		Query: parsed.Query(),
		Hash:  parsed.Fragment,
	}
}

// NavigateWithSearch updates query parameters while keeping the current path.
// The patch function receives current query values and returns updated values.
//
// Usage:
//
//	router.NavigateWithSearch(ctx, func(q url.Values) url.Values {
//	    q.Set("page", "2")
//	    return q
//	})
func NavigateWithSearch(ctx Ctx, patch func(url.Values) url.Values) {
	updateSearchWithNavigation(ctx, patch, false)
}

// ReplaceWithSearch updates query parameters with replace instead of push.
func ReplaceWithSearch(ctx Ctx, patch func(url.Values) url.Values) {
	updateSearchWithNavigation(ctx, patch, true)
}

// updateSearchWithNavigation applies a query update function.
func updateSearchWithNavigation(ctx Ctx, patch func(url.Values) url.Values, replace bool) {
	controller := useRouterController(ctx)
	if controller == nil || controller.requestController == nil {
		return
	}

	current := controller.GetLocation()

	nextQuery := cloneValues(current.Query)
	if patch != nil {
		nextQuery = patch(nextQuery)
	}

	next := current
	next.Query = canonicalizeValues(nextQuery)

	performLocationUpdate(ctx, next, replace)
}

// performLocationUpdate is a helper for updating location.
func performLocationUpdate(ctx Ctx, target Location, replace bool) {
	controller := useRouterController(ctx)
	if controller == nil || controller.requestController == nil {
		return
	}

	current := controller.GetLocation()
	canon := canonicalizeLocation(target)

	if locationEqual(current, canon) {
		return
	}

	controller.requestController.SetCurrentLocation(canon.Path, canon.Query, canon.Hash)

	recordNavigation(ctx, canon, replace)
}
