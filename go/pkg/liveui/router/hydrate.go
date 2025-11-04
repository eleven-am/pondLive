package router

import (
	"sync"

	ui "github.com/eleven-am/liveui/pkg/liveui"
)

var sessionSeeds sync.Map // *ui.Session -> Location

// InternalSeedSessionLocation primes a session with a router location during SSR.
// For internal use by the LiveUI server stack.
func InternalSeedSessionLocation(sess *ui.Session, loc Location) {
	if sess == nil {
		return
	}
	canon := canonicalizeLocation(loc)
	sessionSeeds.Store(sess, canon)
	storeSessionLocation(sess, canon)
}

func consumeSeed(sess *ui.Session) (Location, bool) {
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
