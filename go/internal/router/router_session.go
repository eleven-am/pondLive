package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Session helper functions removed - router state now managed via Controller

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
