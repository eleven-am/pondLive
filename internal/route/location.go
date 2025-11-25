package route

import (
	"errors"
	"net/url"
	"strings"
)

// Location represents a router location with path, query parameters, and hash fragment.
type Location struct {
	Path  string
	Query url.Values
	Hash  string
}

// ErrMissingRouter is returned when router operations are attempted without an active router.
var ErrMissingRouter = errors.New("router: missing router context")

// ParseHref parses a URL string into a Location.
func ParseHref(href string) Location {
	u, err := url.Parse(href)
	if err != nil {
		return Location{Path: "/"}
	}
	return Location{
		Path:  u.Path,
		Query: u.Query(),
		Hash:  u.Fragment,
	}
}

// BuildHref constructs a URL string from a Location.
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

// SetSearch replaces all query parameters with the provided values.
func SetSearch(loc Location, values url.Values) Location {
	loc.Query = cloneValues(values)
	return loc
}

// AddSearch adds or updates query parameters.
func AddSearch(loc Location, key string, values ...string) Location {
	if loc.Query == nil {
		loc.Query = url.Values{}
	}
	for _, v := range values {
		loc.Query.Add(key, v)
	}
	return loc
}

// DelSearch removes a query parameter.
func DelSearch(loc Location, key string) Location {
	if loc.Query != nil {
		loc.Query.Del(key)
	}
	return loc
}

// MergeSearch merges query parameters into the location.
func MergeSearch(loc Location, values url.Values) Location {
	if loc.Query == nil {
		loc.Query = url.Values{}
	}
	for key, vals := range values {
		for _, v := range vals {
			loc.Query.Add(key, v)
		}
	}
	return loc
}

// ClearSearch removes all query parameters.
func ClearSearch(loc Location) Location {
	loc.Query = url.Values{}
	return loc
}

// Clone returns a deep copy of the location.
func (loc Location) Clone() Location {
	return Location{
		Path:  loc.Path,
		Hash:  loc.Hash,
		Query: cloneValues(loc.Query),
	}
}

// LocEqual compares two Location values for equality.
func LocEqual(a, b Location) bool {
	if a.Path != b.Path || a.Hash != b.Hash {
		return false
	}
	if len(a.Query) != len(b.Query) {
		return false
	}
	for key, valsA := range a.Query {
		valsB, ok := b.Query[key]
		if !ok || len(valsA) != len(valsB) {
			return false
		}
		for i, val := range valsA {
			if val != valsB[i] {
				return false
			}
		}
	}
	return true
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
