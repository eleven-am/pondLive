package route

import (
	"errors"
	"maps"
	"net/url"
	"slices"
	"sort"
	"strings"
)

type Location struct {
	Path  string     `json:"path"`
	Query url.Values `json:"query"`
	Hash  string     `json:"hash"`
}

var ErrMissingRouter = errors.New("router: missing router context")

func BuildHref(loc Location) string {
	path := loc.Path
	if path == "" {
		path = "/"
	}
	query := loc.Query.Encode()
	hash := loc.Hash

	href := path
	if query != "" {
		href += "?" + query
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

func SetSearch(loc Location, values url.Values) Location {
	loc.Query = cloneValues(values)
	return loc
}

func AddSearch(loc Location, key string, values ...string) Location {
	loc = loc.Clone()
	for _, v := range values {
		loc.Query.Add(key, v)
	}
	return loc
}

func DelSearch(loc Location, key string) Location {
	loc = loc.Clone()
	loc.Query.Del(key)
	return loc
}

func MergeSearch(loc Location, values url.Values) Location {
	loc = loc.Clone()
	for key, vals := range values {
		for _, v := range vals {
			loc.Query.Add(key, v)
		}
	}
	return loc
}

func ClearSearch(loc Location) Location {
	loc.Query = url.Values{}
	return loc
}

func (loc Location) Clone() Location {
	return Location{
		Path:  loc.Path,
		Hash:  loc.Hash,
		Query: cloneValues(loc.Query),
	}
}

func LocEqual(a, b Location) bool {
	return a.Path == b.Path && a.Hash == b.Hash && maps.EqualFunc(a.Query, b.Query, slices.Equal[[]string])
}

func cloneValues(values url.Values) url.Values {
	if values == nil {
		return url.Values{}
	}
	clone := make(url.Values, len(values))
	for k, v := range values {
		clone[k] = append([]string(nil), v...)
	}
	return clone
}

func Canonicalize(loc Location) Location {
	if loc.Path == "" {
		loc.Path = "/"
	}

	parts := NormalizeParts(loc.Path)

	canon := Location{
		Path:  parts.Path,
		Query: canonicalValues(loc.Query),
		Hash:  NormalizeHash(loc.Hash),
	}

	if canon.Hash == "" && parts.Hash != "" {
		canon.Hash = NormalizeHash(parts.Hash)
	}

	return canon
}

func canonicalValues(q url.Values) url.Values {
	out := make(url.Values, len(q))
	for key, values := range q {
		out[key] = canonicalList(values)
	}
	return out
}

func canonicalList(values []string) []string {
	cleaned := make([]string, 0, len(values))
	for _, v := range values {
		cleaned = append(cleaned, strings.TrimSpace(v))
	}
	sort.Strings(cleaned)
	return cleaned
}

func FromNavigation(path, rawQuery, hash string) Location {
	query, _ := url.ParseQuery(rawQuery)
	return Canonicalize(Location{Path: path, Query: query, Hash: hash})
}
