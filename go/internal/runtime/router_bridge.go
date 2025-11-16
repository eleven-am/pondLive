package runtime

// InternalInitialLocation exposes the initial location stored on the session for
// router integrations such as router2.
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
