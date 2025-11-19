package session

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
)

func newRequest(target string) *http.Request {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		trimmed = "/"
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		panic(err)
	}
	base := &url.URL{
		Path:     u.Path,
		RawQuery: u.RawQuery,
	}
	req := httptest.NewRequest(http.MethodGet, base.String(), nil)
	req.URL.Fragment = u.Fragment
	return req
}
