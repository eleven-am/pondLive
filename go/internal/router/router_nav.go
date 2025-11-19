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
	state := requireRouterState(ctx)
	current := state.getLoc()
	nextQuery := cloneValues(current.Query)
	if patch != nil {
		nextQuery = patch(nextQuery)
	}
	next := current
	next.Query = canonicalizeValues(nextQuery)
	performLocationUpdate(ctx, next, replace, true)
}

func applyNavigation(ctx runtime.Ctx, href string, replace bool) {
	state := requireRouterState(ctx)
	current := state.getLoc()
	target := resolveHref(current, href)
	performLocationUpdate(ctx, target, replace, true)
}

func performLocationUpdate(ctx runtime.Ctx, target Location, replace bool, record bool) {
	state := requireRouterState(ctx)
	current := state.getLoc()
	canon := canonicalizeLocation(target)
	if LocEqual(current, canon) {
		return
	}
	state.setLoc(canon)
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

func parseQuery(raw string) url.Values {
	if raw == "" {
		return url.Values{}
	}
	vals, err := url.ParseQuery(raw)
	if err != nil {
		return url.Values{}
	}
	return canonicalizeValues(vals)
}

func currentSessionLocation(ctx runtime.Ctx) Location {
	if entry := loadSessionRouterEntry(ctx); entry != nil {
		entry.mu.Lock()
		defer entry.mu.Unlock()
		return cloneLocation(entry.navigation.loc)
	}
	return canonicalizeLocation(Location{Path: "/"})
}

func recordNavigation(ctx runtime.Ctx, loc Location, replace bool) {
	msg := NavMsg{
		T:    "nav",
		Path: loc.Path,
		Q:    encodeQuery(loc.Query),
		Hash: loc.Hash,
	}
	if replace {
		msg.T = "replace"
	}
	if entry := loadSessionRouterEntry(ctx); entry != nil {
		entry.mu.Lock()
		entry.navigation.history = append(entry.navigation.history, msg)
		entry.navigation.pending = append(entry.navigation.pending, msg)
		entry.mu.Unlock()
	}
}
