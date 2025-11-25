package router2

import (
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/route"
)

// canonicalizeLocation normalizes a location's path, query, and hash.
// This ensures consistent location comparison and matching.
func canonicalizeLocation(loc *Location) *Location {
	if loc == nil {
		return &Location{Path: "/", Query: url.Values{}}
	}

	parts := route.NormalizeParts(loc.Path)

	var canonQuery url.Values
	if len(loc.Query) > 0 {
		canonQuery = canonicalizeValues(loc.Query)
	} else {
		canonQuery = url.Values{}
	}

	canon := &Location{
		Path:  parts.Path,
		Query: canonQuery,
		Hash:  normalizeHash(loc.Hash),
	}

	if canon.Hash == "" && parts.Hash != "" {
		canon.Hash = route.NormalizeHash(parts.Hash)
	}

	return canon
}

// cloneLocation creates a deep copy of a location.
func cloneLocation(loc *Location) *Location {
	if loc == nil {
		return &Location{Path: "/", Query: url.Values{}}
	}
	return &Location{
		Path:  loc.Path,
		Query: cloneValues(loc.Query),
		Hash:  loc.Hash,
	}
}

// locationEqual compares two locations for equality.
func locationEqual(a, b *Location) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Path != b.Path {
		return false
	}
	if a.Hash != b.Hash {
		return false
	}
	return valuesEqual(a.Query, b.Query)
}

// normalizeHash removes leading # prefix from hash.
func normalizeHash(hash string) string {
	return route.NormalizeHash(hash)
}

// cloneValues creates a deep copy of url.Values.
func cloneValues(q url.Values) url.Values {
	if len(q) == 0 {
		return url.Values{}
	}
	out := make(url.Values, len(q))
	for k, values := range q {
		cp := make([]string, len(values))
		copy(cp, values)
		out[k] = cp
	}
	return out
}

// canonicalizeValues sorts and normalizes url.Values for consistent comparison.
func canonicalizeValues(q url.Values) url.Values {
	if len(q) == 0 {
		return url.Values{}
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(url.Values, len(keys))
	for _, key := range keys {
		out[key] = canonicalizeList(q[key])
	}
	return out
}

// canonicalizeList sorts and trims string values.
func canonicalizeList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	cleaned := make([]string, 0, len(values))
	for _, v := range values {
		cleaned = append(cleaned, strings.TrimSpace(v))
	}
	sort.Strings(cleaned)
	return cleaned
}

// valuesEqual compares two url.Values for equality.
// Uses encoded representation for efficient comparison.
func valuesEqual(a, b url.Values) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	return encodeQuery(a) == encodeQuery(b)
}

// encodeQuery creates a canonical encoded representation of url.Values.
func encodeQuery(q url.Values) string {
	if len(q) == 0 {
		return ""
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var builder strings.Builder
	first := true
	for _, key := range keys {
		values := q[key]
		if len(values) == 0 {
			if !first {
				builder.WriteByte('&')
			}
			builder.WriteString(url.QueryEscape(key))
			builder.WriteString("=")
			first = false
			continue
		}
		for _, v := range values {
			if !first {
				builder.WriteByte('&')
			}
			builder.WriteString(url.QueryEscape(key))
			builder.WriteByte('=')
			builder.WriteString(url.QueryEscape(v))
			first = false
		}
	}
	return builder.String()
}

// buildHref constructs a URL string from path, query, and hash.
func buildHref(path string, query url.Values, hash string) string {
	if path == "" {
		path = "/"
	}

	href := path

	if len(query) > 0 {
		encoded := query.Encode()
		if encoded != "" {
			href += "?" + encoded
		}
	}

	if hash != "" {
		if !strings.HasPrefix(hash, "#") {
			href += "#" + hash
		} else {
			href += hash
		}
	}

	return href
}

// resolveHref resolves an href relative to the current location.
// Supports absolute paths ("/about"), hash-only ("#section"), and relative ("./edit").
func resolveHref(current *Location, href string) *Location {
	if href == "" {
		return cloneLocation(current)
	}

	if strings.HasPrefix(href, "/") {
		parsed, err := url.Parse(href)
		if err != nil {
			return cloneLocation(current)
		}
		return &Location{
			Path:  parsed.Path,
			Query: parsed.Query(),
			Hash:  normalizeHash(parsed.Fragment),
		}
	}

	if strings.HasPrefix(href, "#") {
		return &Location{
			Path:  current.Path,
			Query: cloneValues(current.Query),
			Hash:  normalizeHash(strings.TrimPrefix(href, "#")),
		}
	}

	if strings.HasPrefix(href, "./") || strings.HasPrefix(href, "../") {
		basePath := current.Path
		if !strings.HasSuffix(basePath, "/") {

			if idx := strings.LastIndex(basePath, "/"); idx >= 0 {
				basePath = basePath[:idx+1]
			}
		}

		resolved := basePath + href
		parsed, err := url.Parse(resolved)
		if err != nil {
			return cloneLocation(current)
		}

		cleanPath := path.Clean(parsed.Path)

		return &Location{
			Path:  cleanPath,
			Query: parsed.Query(),
			Hash:  normalizeHash(parsed.Fragment),
		}
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return cloneLocation(current)
	}

	path := parsed.Path
	if path == "" {
		path = current.Path
	}

	return &Location{
		Path:  path,
		Query: parsed.Query(),
		Hash:  normalizeHash(parsed.Fragment),
	}
}
