package runtime

import (
	"sync/atomic"

	"github.com/eleven-am/pondlive/go/internal/route"
)

// Location is an alias to route.Location for router integration.
type Location = route.Location

// routerEntry holds the router callback handlers registered by router implementations.
type routerEntry struct {
	getLoc    atomic.Pointer[func() Location]
	setLoc    atomic.Pointer[func(Location)]
	assignLoc atomic.Pointer[func(Location)]
}

func (s *ComponentSession) ensureRouterEntry() *routerEntry {
	if entry := s.loadRouterEntry(); entry != nil {
		return entry
	}
	entry := &routerEntry{}
	s.storeRouterEntry(entry)
	return entry
}

func (s *ComponentSession) loadRouterEntry() *routerEntry {
	ptr := s.routerEntry.Load()
	if ptr == nil {
		return nil
	}
	return ptr
}

func (s *ComponentSession) storeRouterEntry(entry *routerEntry) {
	s.routerEntry.Store(entry)
}

// Bridge functions implementation

func initialLocation(sess *ComponentSession) Location {
	loc := sess.initLoc.Load()
	if loc == nil {
		return Location{Path: "/"}
	}
	return *loc
}

func storeSessionLocation(sess *ComponentSession, loc Location) {
	sess.cachedLoc.Store(&loc)
}

func registerSessionEntry(sess *ComponentSession, get func() Location, set func(Location), assign func(Location)) {
	entry := sess.ensureRouterEntry()
	entry.getLoc.Store(&get)
	entry.setLoc.Store(&set)
	entry.assignLoc.Store(&assign)
}

func currentLocation(sess *ComponentSession) Location {
	if entry := sess.loadRouterEntry(); entry != nil {
		if get := entry.getLoc.Load(); get != nil {
			fn := *get
			return fn()
		}
	}
	if cached := sess.cachedLoc.Load(); cached != nil {
		return *cached
	}
	return initialLocation(sess)
}

func navHistory(sess *ComponentSession) []NavMsg {
	sess.navHistoryMu.Lock()
	defer sess.navHistoryMu.Unlock()
	result := make([]NavMsg, len(sess.navHistory))
	copy(result, sess.navHistory)
	return result
}

func clearNavHistory(sess *ComponentSession) {
	sess.navHistoryMu.Lock()
	defer sess.navHistoryMu.Unlock()
	sess.navHistory = nil
}

// Context keys for router integration
var (
	LocationCtx = NewContext(Location{})
	ParamsCtx   = NewContext(map[string]string{})
)

// InternalSeedSessionLocation seeds the initial location for testing.
func InternalSeedSessionLocation(sess *ComponentSession, loc Location) {
	sess.initLoc.Store(&loc)
	sess.cachedLoc.Store(&loc)
}
