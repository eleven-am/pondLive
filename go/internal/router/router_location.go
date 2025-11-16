package router

import (
	"net/url"
	"sort"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/route"
)

func canonicalizeLocation(loc Location) Location {
	parts := route.NormalizeParts(loc.Path)
	canon := Location{
		Path:  parts.Path,
		Query: canonicalizeValues(loc.Query),
		Hash:  normalizeHash(loc.Hash),
	}
	if canon.Hash == "" && parts.Hash != "" {
		canon.Hash = route.NormalizeHash(parts.Hash)
	}
	return canon
}

func cloneLocation(loc Location) Location {
	canon := canonicalizeLocation(loc)
	canon.Query = cloneValues(canon.Query)
	return canon
}

func normalizePath(path string) string {
	return route.NormalizeParts(path).Path
}

func normalizeHash(hash string) string {
	return route.NormalizeHash(hash)
}

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

func valuesEqual(a, b url.Values) bool {
	ca := canonicalizeValues(a)
	cb := canonicalizeValues(b)
	if len(ca) != len(cb) {
		return false
	}
	for key, av := range ca {
		bv, ok := cb[key]
		if !ok {
			return false
		}
		if len(av) != len(bv) {
			return false
		}
		for i := range av {
			if av[i] != bv[i] {
				return false
			}
		}
	}
	return true
}

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

func LocEqual(a, b Location) bool {
	if a.Path != b.Path {
		return false
	}
	if a.Hash != b.Hash {
		return false
	}
	return valuesEqual(a.Query, b.Query)
}
