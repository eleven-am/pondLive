package router

import runtime "github.com/eleven-am/pondlive/go/internal/runtime"

// InternalSeedSessionLocation primes a session with a router location during SSR.
// For internal use by the LiveUI server stack.
func InternalSeedSessionLocation(sess *runtime.ComponentSession, loc Location) {
	if sess == nil {
		return
	}
	canon := canonicalizeLocation(loc)
	storeSessionLocation(sess, canon)
	if entry := ensureSessionRouterEntry(sess); entry != nil {
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

func consumeSeed(sess *runtime.ComponentSession) (Location, bool) {
	if sess == nil {
		return Location{}, false
	}
	if entry := loadSessionRouterEntry(sess); entry != nil {
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
