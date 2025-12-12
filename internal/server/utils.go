package server

import (
	"net/http"
)

func cloneHeader(h http.Header) http.Header {
	if h == nil {
		return http.Header{}
	}
	clone := make(http.Header, len(h))
	for k, v := range h {
		clone[k] = append([]string(nil), v...)
	}
	return clone
}
