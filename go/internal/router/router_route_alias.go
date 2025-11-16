package router

import "github.com/eleven-am/pondlive/go/internal/route"

type (
	// Match represents parsed routing information surfaced to components.
	Match = route.Match
)

var (
	// Parse extracts params and query values from a request path.
	Parse = route.Parse
	// NormalizePattern canonicalizes route patterns for matching.
	NormalizePattern = route.NormalizePattern
	// Prefer reports whether the candidate match is more specific.
	Prefer = route.Prefer
	// BestMatch selects the most specific match among patterns.
	BestMatch = route.BestMatch
)
