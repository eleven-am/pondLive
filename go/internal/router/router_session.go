package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Helper functions to access router session entry from context

func ensureSessionRouterEntry(ctx runtime.Ctx) *sessionEntry {
	entry := SessionEntryCtx.Use(ctx)
	if entry == nil {
		entry = &sessionEntry{}
	}
	return entry
}

func loadSessionRouterEntry(ctx runtime.Ctx) *sessionEntry {
	return SessionEntryCtx.Use(ctx)
}

// getInitialLocationFromSession gets the initial location from ComponentSession
func getInitialLocationFromSession(sess *runtime.ComponentSession) Location {
	if sess == nil {
		return canonicalizeLocation(Location{Path: "/"})
	}

	path, queryMap, hash, ok := sess.GetInitialLocation()
	if !ok {
		return canonicalizeLocation(Location{Path: "/"})
	}

	query := url.Values{}
	for k, v := range queryMap {
		query.Set(k, v)
	}

	return canonicalizeLocation(Location{
		Path:  path,
		Query: query,
		Hash:  hash,
	})
}
