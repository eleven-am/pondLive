package router

import (
	"net/url"
	"strings"
)

func BuildHref(path string, q url.Values, hash string) string {
	loc := canonicalizeLocation(Location{Path: path, Query: q, Hash: hash})
	var builder strings.Builder
	builder.WriteString(loc.Path)
	if encoded := encodeQuery(loc.Query); encoded != "" {
		builder.WriteByte('?')
		builder.WriteString(encoded)
	}
	if loc.Hash != "" {
		builder.WriteByte('#')
		builder.WriteString(url.PathEscape(loc.Hash))
	}
	return builder.String()
}

func SetSearch(q url.Values, key string, values ...string) url.Values {
	out := cloneValues(q)
	if out == nil {
		out = url.Values{}
	}
	if len(values) == 0 {
		delete(out, key)
		return canonicalizeValues(out)
	}
	cleaned := make([]string, 0, len(values))
	for _, v := range values {
		cleaned = append(cleaned, strings.TrimSpace(v))
	}
	out[key] = cleaned
	return canonicalizeValues(out)
}

func AddSearch(q url.Values, key string, values ...string) url.Values {
	if len(values) == 0 {
		return canonicalizeValues(q)
	}
	out := cloneValues(q)
	base := append([]string{}, out[key]...)
	for _, v := range values {
		base = append(base, strings.TrimSpace(v))
	}
	out[key] = base
	return canonicalizeValues(out)
}

func DelSearch(q url.Values, key string) url.Values {
	out := cloneValues(q)
	delete(out, key)
	return canonicalizeValues(out)
}

func MergeSearch(q url.Values, other url.Values) url.Values {
	out := cloneValues(q)
	if len(other) == 0 {
		return canonicalizeValues(out)
	}
	for key, values := range other {
		out = SetSearch(out, key, values...)
	}
	return canonicalizeValues(out)
}

func ClearSearch(url.Values) url.Values {
	return url.Values{}
}

func ParseHref(href string) Location {
	trimmed := strings.TrimSpace(href)
	if trimmed == "" {
		return canonicalizeLocation(Location{Path: "/"})
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return canonicalizeLocation(Location{Path: "/"})
	}
	return locationFromURL(parsed)
}
