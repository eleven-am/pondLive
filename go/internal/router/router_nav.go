package router

import (
	"net/url"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func Navigate(ctx runtime.Ctx, href string) {
	applyNavigation(ctx, href, false)
}

func Replace(ctx runtime.Ctx, href string) {
	applyNavigation(ctx, href, true)
}

func NavigateWithSearch(ctx runtime.Ctx, patch func(url.Values) url.Values) {
	updateSearchWithNavigation(ctx, patch, false)
}

func ReplaceWithSearch(ctx runtime.Ctx, patch func(url.Values) url.Values) {
	updateSearchWithNavigation(ctx, patch, true)
}

func updateSearchWithNavigation(ctx runtime.Ctx, patch func(url.Values) url.Values, replace bool) {
	controller := UseRouterState(ctx)
	state := controller.Get()
	current := state.Location
	nextQuery := cloneValues(current.Query)
	if patch != nil {
		nextQuery = patch(nextQuery)
	}
	next := current
	next.Query = canonicalizeValues(nextQuery)
	performLocationUpdate(ctx, next, replace, true)
}

func applyNavigation(ctx runtime.Ctx, href string, replace bool) {
	controller := UseRouterState(ctx)
	state := controller.Get()
	current := state.Location
	target := resolveHref(current, href)
	performLocationUpdate(ctx, target, replace, true)
}

func performLocationUpdate(ctx runtime.Ctx, target Location, replace bool, record bool) {
	controller := UseRouterState(ctx)
	state := controller.Get()
	current := state.Location
	canon := canonicalizeLocation(target)
	if LocEqual(current, canon) {
		return
	}
	controller.SetLocation(canon)
	if record {
		recordNavigation(ctx, canon, replace)
	}
}

func resolveHref(base Location, href string) Location {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return base
	}
	if strings.HasPrefix(trimmed, "#") {
		next := base
		next.Hash = normalizeHash(trimmed)
		return canonicalizeLocation(next)
	}

	if strings.HasPrefix(trimmed, "/") {
		parsed, err := url.Parse(trimmed)
		if err != nil {
			return base
		}
		return locationFromURL(parsed)
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

func recordNavigation(ctx runtime.Ctx, loc Location, replace bool) {
	href := BuildHref(loc.Path, loc.Query, loc.Hash)
	ctx.EnqueueNavigation(href, replace)
}
