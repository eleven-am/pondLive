package session

import "net/url"

// Location represents a browser location (path, query, hash).
type Location struct {
	Path  string
	Query url.Values
	Hash  string
}

func cloneQuery(q url.Values) url.Values {
	if q == nil {
		return url.Values{}
	}
	clone := make(url.Values, len(q))
	for k, v := range q {
		clone[k] = append([]string(nil), v...)
	}
	return clone
}
