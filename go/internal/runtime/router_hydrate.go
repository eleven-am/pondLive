package runtime

// InternalSeedSessionLocation primes a session with a router location during SSR.
// For internal use by the LiveUI server stack.
func InternalSeedSessionLocation(sess *ComponentSession, loc Location) {
	if sess == nil {
		return
	}
	canon := canonicalizeLocation(loc)
	storeSessionLocation(sess, canon)
	if entry := ensureSessionEntry(sess); entry != nil {
		entry.mu.Lock()
		entry.navigation.seed = canon
		entry.navigation.hasSeed = true
		assign := entry.handlers.assign
		active := entry.render.active
		entry.mu.Unlock()
		if assign != nil && !active {
			assign(canon)
		}
	}
}

func consumeSeed(sess *ComponentSession) (Location, bool) {
	if sess == nil {
		return Location{}, false
	}
	if entry := loadSessionEntry(sess); entry != nil {
		entry.mu.Lock()
		if entry.navigation.hasSeed {
			loc := entry.navigation.seed
			entry.navigation.hasSeed = false
			entry.mu.Unlock()
			return canonicalizeLocation(loc), true
		}
		entry.mu.Unlock()
	}
	return Location{}, false
}
