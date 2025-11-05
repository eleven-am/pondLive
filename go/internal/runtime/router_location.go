package runtime

import (
	"net/url"
	"sort"
	"strings"
)

func canonicalizeLocation(loc Location) Location {
	canon := Location{
		Path:  normalizePath(loc.Path),
		Query: canonicalizeValues(loc.Query),
		Hash:  normalizeHash(loc.Hash),
	}
	return canon
}

func cloneLocation(loc Location) Location {
	canon := canonicalizeLocation(loc)
	canon.Query = cloneValues(canon.Query)
	return canon
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	trimmed := path
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if idx := strings.Index(trimmed, "#"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	parts := strings.Split(trimmed, "/")
	segs := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		segs = append(segs, part)
	}
	if len(segs) == 0 {
		return "/"
	}
	return "/" + strings.Join(segs, "/")
}

func normalizeHash(hash string) string {
	if hash == "" {
		return ""
	}
	return strings.TrimPrefix(hash, "#")
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
		values := q[key]
		if len(values) == 0 {
			out[key] = []string{}
			continue
		}
		cleaned := make([]string, 0, len(values))
		for _, v := range values {
			cleaned = append(cleaned, strings.TrimSpace(v))
		}
		sort.Strings(cleaned)
		out[key] = cleaned
	}
	return out
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
