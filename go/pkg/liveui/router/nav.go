package router

import (
	"net/url"
	"strings"

	ui "github.com/eleven-am/liveui/pkg/liveui"
)

func Navigate(ctx ui.Ctx, href string) {
	applyNavigation(ctx, href, false)
}

func Replace(ctx ui.Ctx, href string) {
	applyNavigation(ctx, href, true)
}

func NavigateWithSearch(ctx ui.Ctx, patch func(url.Values) url.Values) {
	state := requireRouterState(ctx)
	current := state.getLoc()
	nextQuery := cloneValues(current.Query)
	if patch != nil {
		nextQuery = patch(nextQuery)
	}
	next := current
	next.Query = canonicalizeValues(nextQuery)
	performLocationUpdate(ctx, next, false, true)
}

func ReplaceWithSearch(ctx ui.Ctx, patch func(url.Values) url.Values) {
	state := requireRouterState(ctx)
	current := state.getLoc()
	nextQuery := cloneValues(current.Query)
	if patch != nil {
		nextQuery = patch(nextQuery)
	}
	next := current
	next.Query = canonicalizeValues(nextQuery)
	performLocationUpdate(ctx, next, true, true)
}

// InternalHandleNav applies a navigation message to the session. Internal use only.
func InternalHandleNav(sess *ui.Session, msg NavMsg) {
	target := Location{
		Path:  msg.Path,
		Query: parseQuery(msg.Q),
		Hash:  msg.Hash,
	}
	setSessionLocation(sess, target)
}

// InternalHandlePop applies a popstate message to the session. Internal use only.
func InternalHandlePop(sess *ui.Session, msg PopMsg) {
	target := Location{
		Path:  msg.Path,
		Query: parseQuery(msg.Q),
		Hash:  msg.Hash,
	}
	setSessionLocation(sess, target)
}

func currentLocation(sess *ui.Session) Location {
	return currentSessionLocation(sess)
}

func navHistory(sess *ui.Session) []NavMsg {
	if sess == nil {
		return nil
	}
	if v, ok := sessionEntries.Load(sess); ok {
		entry := v.(*sessionEntry)
		entry.mu.Lock()
		defer entry.mu.Unlock()
		if len(entry.navs) == 0 {
			return nil
		}
		out := make([]NavMsg, len(entry.navs))
		copy(out, entry.navs)
		return out
	}
	return nil
}

func clearNavHistory(sess *ui.Session) {
	if sess == nil {
		return
	}
	if v, ok := sessionEntries.Load(sess); ok {
		entry := v.(*sessionEntry)
		entry.mu.Lock()
		entry.navs = nil
		entry.mu.Unlock()
	}
}

func applyNavigation(ctx ui.Ctx, href string, replace bool) {
	state := requireRouterState(ctx)
	current := state.getLoc()
	target := resolveHref(current, href)
	performLocationUpdate(ctx, target, replace, true)
}

func performLocationUpdate(ctx ui.Ctx, target Location, replace bool, record bool) {
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
	if strings.TrimSpace(href) == "" {
		return base
	}
	trimmed := strings.TrimSpace(href)
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

func setSessionLocation(sess *ui.Session, target Location) {
	if sess == nil {
		return
	}
	canon := canonicalizeLocation(target)
	if v, ok := sessionEntries.Load(sess); ok {
		entry := v.(*sessionEntry)
		entry.mu.Lock()
		setter := entry.set
		current := entry.loc
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

func recordNavigation(sess *ui.Session, loc Location, replace bool) {
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
	if v, ok := sessionEntries.Load(sess); ok {
		entry := v.(*sessionEntry)
		entry.mu.Lock()
		entry.navs = append(entry.navs, msg)
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
