package router

import (
	"net/url"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func RouterNavigate(ctx runtime.Ctx, href string) {
	applyNavigation(ctx, href, false)
}

func RouterReplace(ctx runtime.Ctx, href string) {
	applyNavigation(ctx, href, true)
}

func RouterNavigateWithSearch(ctx runtime.Ctx, patch func(url.Values) url.Values) {
	updateSearchWithNavigation(ctx, patch, false)
}

func RouterReplaceWithSearch(ctx runtime.Ctx, patch func(url.Values) url.Values) {
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

type locationMessagePayload struct {
	path     string
	rawQuery string
	hash     string
}

func handleLocationMessage(sess *runtime.ComponentSession, payload locationMessagePayload) {
	target := Location{
		Path:  payload.path,
		Query: parseQuery(payload.rawQuery),
		Hash:  payload.hash,
	}
	setSessionLocation(sess, target)
}

// InternalHandleNav applies a navigation message to the session. Internal use only.
func InternalHandleNav(sess *runtime.ComponentSession, msg NavMsg) {
	handleLocationMessage(sess, locationMessagePayload{
		path:     msg.Path,
		rawQuery: msg.Q,
		hash:     msg.Hash,
	})
}

// InternalHandlePop applies a popstate message to the session. Internal use only.
func InternalHandlePop(sess *runtime.ComponentSession, msg PopMsg) {
	handleLocationMessage(sess, locationMessagePayload{
		path:     msg.Path,
		rawQuery: msg.Q,
		hash:     msg.Hash,
	})
}

func currentLocation(sess *runtime.ComponentSession) Location {
	return currentSessionLocation(sess)
}

func navHistory(sess *runtime.ComponentSession) []NavMsg {
	if sess == nil {
		return nil
	}
	if entry := loadSessionRouterEntry(sess); entry != nil {
		entry.mu.Lock()
		defer entry.mu.Unlock()
		navs := entry.navigation.history
		if len(navs) == 0 {
			return nil
		}
		out := make([]NavMsg, len(navs))
		copy(out, navs)
		return out
	}
	return nil
}

func clearNavHistory(sess *runtime.ComponentSession) {
	if sess == nil {
		return
	}
	if entry := loadSessionRouterEntry(sess); entry != nil {
		entry.mu.Lock()
		entry.navigation.history = nil
		entry.navigation.pending = nil
		entry.mu.Unlock()
	}
}

func consumePendingNavigation(sess *runtime.ComponentSession) (NavMsg, bool) {
	if sess == nil {
		return NavMsg{}, false
	}
	if entry := loadSessionRouterEntry(sess); entry != nil {
		entry.mu.Lock()
		defer entry.mu.Unlock()
		pending := entry.navigation.pending
		if len(pending) == 0 {
			return NavMsg{}, false
		}
		last := pending[len(pending)-1]
		entry.navigation.pending = nil
		return last, true
	}
	return NavMsg{}, false
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
		recordNavigation(ctx.Session(), canon, replace)
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

func setSessionLocation(sess *runtime.ComponentSession, target Location) {
	if sess == nil {
		return
	}
	canon := canonicalizeLocation(target)
	if entry := loadSessionRouterEntry(sess); entry != nil {
		entry.mu.Lock()
		setter := entry.handlers.set
		current := entry.navigation.loc
		entry.mu.Unlock()
		if LocEqual(current, canon) {
			return
		}
		if setter != nil {
			setter(canon)
			return
		}
		storeSessionLocation(sess, canon)
	}
}

func recordNavigation(sess *runtime.ComponentSession, loc Location, replace bool) {
	if sess == nil {
		return
	}
	msg := NavMsg{
		T:    "nav",
		Path: loc.Path,
		Q:    encodeQuery(loc.Query),
		Hash: loc.Hash,
	}
	if replace {
		msg.T = "replace"
	}
	if entry := ensureSessionRouterEntry(sess); entry != nil {
		entry.mu.Lock()
		entry.navigation.history = append(entry.navigation.history, msg)
		entry.navigation.pending = append(entry.navigation.pending, msg)
		entry.mu.Unlock()
	}
}

func buildNavURL(msg NavMsg) string {
	path := msg.Path
	if path == "" {
		path = "/"
	}
	if msg.Q != "" {
		path = path + "?" + msg.Q
	}
	if msg.Hash != "" {
		path = path + "#" + msg.Hash
	}
	return path
}
