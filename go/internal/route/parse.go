package route

import (
	"fmt"
	"net/url"
	"strings"
)

// Match represents the parsed route information passed to components.
type Match struct {
	Pattern  string
	Path     string
	Params   map[string]string
	Query    url.Values
	RawQuery string
	Rest     string
	Score    int
}

// NormalizePattern canonicalizes a route pattern ensuring it begins with a slash
// and collapses redundant separators. It mirrors the normalization applied by
// Parse, allowing callers to persist canonical patterns for matching.
func NormalizePattern(pattern string) string {
	return normalizePattern(pattern)
}

// Parse extracts params and query values from the provided request path using
// the supplied pattern. Patterns support segments such as ":id" or optional
// ":id?" parameters as well as "*rest" wildcards.
func Parse(pattern string, path string, rawQuery string) (Match, error) {
	normalizedPattern := normalizePattern(pattern)
	m := matchPath(path, normalizedPattern)
	if !m.ok {
		return Match{}, fmt.Errorf("route: path %q does not match pattern %q", path, pattern)
	}
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return Match{}, err
	}
	params := m.params
	if params == nil {
		params = map[string]string{}
	}
	return Match{
		Pattern:  normalizedPattern,
		Path:     normalizePath(path),
		Params:   params,
		Query:    values,
		RawQuery: rawQuery,
		Rest:     m.rest,
		Score:    m.score,
	}, nil
}

func matchPath(path, pattern string) matchResult {
	nPath := normalizePath(path)
	nPattern := normalizePattern(pattern)
	pathSegs := splitSegments(nPath)
	patSegs := splitSegments(nPattern)

	res := matchResult{params: map[string]string{}}
	pi, ti := 0, 0
	for pi < len(patSegs) {
		seg := patSegs[pi]
		if seg == "" {
			pi++
			continue
		}
		if isWildcard(seg) {
			name := strings.TrimPrefix(seg, "*")
			remainder := strings.Join(pathSegs[ti:], "/")
			if name == "" {
				if remainder != "" {
					res.params["*"] = remainder
				}
			} else {
				res.params[name] = remainder
			}
			if remainder != "" {
				res.rest = "/" + remainder
			}
			if remainder != "" {
				res.score++
			}
			break
		}
		name, optional := parseParam(seg)
		if name != "" {
			if ti >= len(pathSegs) {
				if optional {
					pi++
					continue
				}
				return matchResult{}
			}
			if optional {
				res.score++
			} else {
				res.score += 2
			}
			res.params[name] = pathSegs[ti]
			pi++
			ti++
			continue
		}
		if ti >= len(pathSegs) {
			return matchResult{}
		}
		if pathSegs[ti] != seg {
			return matchResult{}
		}
		res.score += 3
		pi++
		ti++
	}

	if ti < len(pathSegs) {
		if res.rest == "" {
			return matchResult{}
		}
	}

	for ; pi < len(patSegs); pi++ {
		seg := patSegs[pi]
		if seg == "" || isWildcard(seg) {
			continue
		}
		_, optional := parseParam(seg)
		if optional {
			continue
		}
		return matchResult{}
	}

	res.ok = true
	return res
}

// Prefer reports whether the candidate match should supersede the current one
// when considering specificity. Higher scores take precedence, followed by the
// shortest unmatched remainder to break ties.
func Prefer(candidate, current Match) bool {
	if current.Pattern == "" {
		return true
	}
	if candidate.Score != current.Score {
		return candidate.Score > current.Score
	}
	clen := len(strings.TrimPrefix(current.Rest, "/"))
	nlen := len(strings.TrimPrefix(candidate.Rest, "/"))
	return nlen < clen
}

// BestMatch selects the most specific pattern from the provided list that
// matches the supplied path. It returns the corresponding Match, the index of
// the winning pattern, and a boolean indicating whether a match was found.
func BestMatch(path string, rawQuery string, patterns []string) (Match, int, bool) {
	bestIdx := -1
	var best Match
	for idx, pattern := range patterns {
		match, err := Parse(pattern, path, rawQuery)
		if err != nil {
			continue
		}
		if bestIdx < 0 || Prefer(match, best) {
			best = match
			bestIdx = idx
		}
	}
	if bestIdx < 0 {
		return Match{}, -1, false
	}
	return best, bestIdx, true
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return trimmed
}

func normalizePattern(pattern string) string {
	if pattern == "" {
		return "/"
	}
	if !strings.HasPrefix(pattern, "/") {
		return "/" + pattern
	}
	return pattern
}

type matchResult struct {
	ok     bool
	params map[string]string
	rest   string
	score  int
}

func splitSegments(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func isWildcard(seg string) bool {
	return strings.HasPrefix(seg, "*")
}

func parseParam(seg string) (string, bool) {
	if !strings.HasPrefix(seg, ":") {
		return "", false
	}
	name := strings.TrimPrefix(seg, ":")
	optional := strings.HasSuffix(name, "?")
	if optional {
		name = strings.TrimSuffix(name, "?")
	}
	return name, optional
}
