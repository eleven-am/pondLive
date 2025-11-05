package runtime

import (
	"sync"
)

var sessionSeeds sync.Map // *ComponentSession -> Location

// InternalSeedSessionLocation primes a session with a router location during SSR.
// For internal use by the LiveUI server stack.
func InternalSeedSessionLocation(sess *ComponentSession, loc Location) {
	if sess == nil {
		return
	}
	canon := canonicalizeLocation(loc)
	sessionSeeds.Store(sess, canon)
	storeSessionLocation(sess, canon)
}

func consumeSeed(sess *ComponentSession) (Location, bool) {
	if sess == nil {
		return Location{}, false
	}
	if v, ok := sessionSeeds.LoadAndDelete(sess); ok {
		if loc, okCast := v.(Location); okCast {
			return canonicalizeLocation(loc), true
		}
	}
	return Location{}, false
}
