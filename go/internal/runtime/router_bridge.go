package runtime

import (
	"net/url"
)

// InternalInitialLocation exposes the initial location stored on the session for
// router integrations such as router.
func InternalInitialLocation(sess *ComponentSession) Location {
	return initialLocation(sess)
}

// InternalStoreLocation updates the session's cached router location so
// transport layers can hydrate or report it.
func InternalStoreLocation(sess *ComponentSession, loc Location) {
	storeSessionLocation(sess, loc)
}

// InternalRegisterRouterHandlers exposes the session entry registration hook so
// alternative router implementations can keep the legacy navigation plumbing in
// sync while they are wired in.
func InternalRegisterRouterHandlers(sess *ComponentSession, get func() Location, set func(Location), assign func(Location)) {
	registerSessionEntry(sess, get, set, assign)
}

// InternalCurrentLocation returns the current canonical session location.
func InternalCurrentLocation(sess *ComponentSession) Location {
	return currentLocation(sess)
}

// InternalNavHistory exposes the recorded navigation list for tests.
func InternalNavHistory(sess *ComponentSession) []NavMsg {
	return navHistory(sess)
}

// InternalClearNavHistory wipes recorded navigation events for deterministic tests.
func InternalClearNavHistory(sess *ComponentSession) {
	clearNavHistory(sess)
}

// InternalEnqueueNavMessage appends a navigation message to the session's history.
// This is used by router implementations to record navigation events.
func InternalEnqueueNavMessage(sess *ComponentSession, msg NavMsg) {
	if sess == nil {
		return
	}
	sess.navHistoryMu.Lock()
	defer sess.navHistoryMu.Unlock()
	sess.navHistory = append(sess.navHistory, msg)
}

// InternalHandleNav processes navigation messages from the client.
// This stores the new location; the caller should call Flush() to trigger a rerender.
func InternalHandleNav(sess *ComponentSession, msg NavMsg) {
	if sess == nil {
		return
	}

	loc := Location{
		Path: msg.Path,
		Hash: msg.Hash,
	}
	if msg.Q != "" {
		loc.Query, _ = url.ParseQuery(msg.Q)
	}
	storeSessionLocation(sess, loc)
}

// InternalHandlePop processes popstate messages from the client.
// This stores the new location; the caller should call Flush() to trigger a rerender.
func InternalHandlePop(sess *ComponentSession, msg PopMsg) {
	if sess == nil {
		return
	}

	loc := Location{
		Path: msg.Path,
		Hash: msg.Hash,
	}
	if msg.Q != "" {
		loc.Query, _ = url.ParseQuery(msg.Q)
	}
	storeSessionLocation(sess, loc)
}
