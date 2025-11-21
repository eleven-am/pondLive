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
	out := canonicalizeValues(q)
	if len(values) == 0 {
		delete(out, key)
		return out
	}
	out[key] = canonicalizeList(values)
	return out
}

func AddSearch(q url.Values, key string, values ...string) url.Values {
	out := canonicalizeValues(q)
	if len(values) == 0 {
		return out
	}
	combined := make([]string, 0, len(out[key])+len(values))
	combined = append(combined, out[key]...)
	combined = append(combined, values...)
	out[key] = canonicalizeList(combined)
	return out
}

func DelSearch(q url.Values, key string) url.Values {
	out := canonicalizeValues(q)
	delete(out, key)
	return out
}

func MergeSearch(q url.Values, other url.Values) url.Values {
	out := canonicalizeValues(q)
	if len(other) == 0 {
		return out
	}
	for key, values := range other {
		if len(values) == 0 {
			delete(out, key)
			continue
		}
		out[key] = canonicalizeList(values)
	}
	return out
}
