package headers2

import (
	"net/http"
	"net/url"
	"sync"
)

// RequestInfo captures HTTP request information from the originating request.
// For SSR, this is the request being served.
// For WebSocket, this is the handshake request (updated on reconnect).
//
// This struct is immutable after creation - create a new one on reconnect.
type RequestInfo struct {
	// Request metadata
	Method     string
	Host       string
	RemoteAddr string

	// URL components
	Path  string
	Query url.Values
	Hash  string // Fragment (typically from client, not in server request)

	// Headers from the request
	Headers http.Header
}

// NewRequestInfo creates a RequestInfo from an http.Request.
func NewRequestInfo(r *http.Request) *RequestInfo {
	if r == nil {
		return &RequestInfo{
			Query:   make(url.Values),
			Headers: make(http.Header),
		}
	}

	hash := ""
	query := make(url.Values)
	if r.URL != nil {
		for k, v := range r.URL.Query() {
			if k == "_hash" && len(v) > 0 {
				hash = v[0]
			} else {
				query[k] = append([]string(nil), v...)
			}
		}
	}

	return &RequestInfo{
		Method:     r.Method,
		Host:       r.Host,
		RemoteAddr: r.RemoteAddr,
		Path:       r.URL.Path,
		Query:      query,
		Hash:       hash,
		Headers:    r.Header.Clone(),
	}
}

// NewRequestInfoFromHeaders creates a RequestInfo from headers.
// This is useful for pondsocket and other WebSocket libraries that provide
// headers (including cookies) rather than an *http.Request.
func NewRequestInfoFromHeaders(headers http.Header) *RequestInfo {
	h := make(http.Header)
	if headers != nil {
		for k, v := range headers {
			h[k] = append([]string(nil), v...)
		}
	}

	return &RequestInfo{
		Query:   make(url.Values),
		Headers: h,
	}
}

// Get retrieves a header value by name.
// Returns the value and true if found, empty string and false otherwise.
func (r *RequestInfo) Get(name string) (string, bool) {
	if r == nil || r.Headers == nil {
		return "", false
	}
	val := r.Headers.Get(name)
	return val, val != ""
}

// GetCookie retrieves a cookie value by name from the request headers.
// Returns the value and true if found, empty string and false otherwise.
func (r *RequestInfo) GetCookie(name string) (string, bool) {
	if r == nil || r.Headers == nil {
		return "", false
	}

	cookieHeader := r.Headers.Get("Cookie")
	if cookieHeader == "" {
		return "", false
	}

	header := http.Header{}
	header.Set("Cookie", cookieHeader)
	req := &http.Request{Header: header}

	cookie, err := req.Cookie(name)
	if err != nil {
		return "", false
	}

	return cookie.Value, true
}

// Cookies returns all cookies from the request.
func (r *RequestInfo) Cookies() []*http.Cookie {
	if r == nil || r.Headers == nil {
		return nil
	}

	cookieHeader := r.Headers.Get("Cookie")
	if cookieHeader == "" {
		return nil
	}

	header := http.Header{}
	header.Set("Cookie", cookieHeader)
	req := &http.Request{Header: header}

	return req.Cookies()
}

// Clone creates a deep copy of the RequestInfo.
func (r *RequestInfo) Clone() *RequestInfo {
	if r == nil {
		return nil
	}

	query := make(url.Values)
	for k, v := range r.Query {
		query[k] = append([]string(nil), v...)
	}

	return &RequestInfo{
		Method:     r.Method,
		Host:       r.Host,
		RemoteAddr: r.RemoteAddr,
		Path:       r.Path,
		Query:      query,
		Hash:       r.Hash,
		Headers:    r.Headers.Clone(),
	}
}

// RequestState extends RequestInfo with mutable response state.
// Used during rendering to collect response headers and redirects.
type RequestState struct {
	mu sync.RWMutex

	// Immutable request info
	info *RequestInfo

	// Connection state
	isLive bool

	// Response state (written during render, read by SSR handler after)
	responseHeaders http.Header
	redirectURL     string
	redirectCode    int

	// Cookie mutations (for optimistic updates)
	cookieMutations map[string]*cookieMutation
}

type cookieMutation struct {
	value   string
	deleted bool
}

// NewRequestState creates a RequestState from a RequestInfo.
func NewRequestState(info *RequestInfo) *RequestState {
	if info == nil {
		info = &RequestInfo{
			Query:   make(url.Values),
			Headers: make(http.Header),
		}
	}
	return &RequestState{
		info:            info,
		responseHeaders: make(http.Header),
		cookieMutations: make(map[string]*cookieMutation),
	}
}

// Info returns the underlying RequestInfo.
func (s *RequestState) Info() *RequestInfo {
	if s == nil {
		return nil
	}
	return s.info
}

// Get retrieves a request header value.
func (s *RequestState) Get(name string) (string, bool) {
	if s == nil || s.info == nil {
		return "", false
	}
	return s.info.Get(name)
}

// GetCookie retrieves a cookie value, respecting any mutations made during render.
func (s *RequestState) GetCookie(name string) (string, bool) {
	if s == nil {
		return "", false
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if mut, ok := s.cookieMutations[name]; ok {
		if mut.deleted {
			return "", false
		}
		return mut.value, true
	}

	if s.info == nil {
		return "", false
	}
	return s.info.GetCookie(name)
}

// Path returns the request path.
func (s *RequestState) Path() string {
	if s == nil || s.info == nil {
		return ""
	}
	return s.info.Path
}

// Query returns the request query parameters.
func (s *RequestState) Query() url.Values {
	if s == nil || s.info == nil {
		return nil
	}

	query := make(url.Values)
	for k, v := range s.info.Query {
		query[k] = append([]string(nil), v...)
	}
	return query
}

// Hash returns the URL hash/fragment.
func (s *RequestState) Hash() string {
	if s == nil || s.info == nil {
		return ""
	}
	return s.info.Hash
}

// IsLive returns whether the session is in live (WebSocket) mode.
func (s *RequestState) IsLive() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isLive
}

// SetIsLive sets the live status.
func (s *RequestState) SetIsLive(live bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isLive = live
}

// SetResponseHeader sets a response header (for SSR).
func (s *RequestState) SetResponseHeader(name, value string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseHeaders.Set(name, value)
}

// AddResponseHeader adds a response header value (for SSR).
func (s *RequestState) AddResponseHeader(name, value string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseHeaders.Add(name, value)
}

// ResponseHeaders returns all response headers set during render.
// Returns a copy to prevent mutation.
func (s *RequestState) ResponseHeaders() http.Header {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	headers := make(http.Header, len(s.responseHeaders))
	for k, v := range s.responseHeaders {
		headers[k] = append([]string(nil), v...)
	}
	return headers
}

// SetRedirect sets a redirect for SSR mode.
func (s *RequestState) SetRedirect(url string, code int) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.redirectURL = url
	s.redirectCode = code
}

// Redirect returns the redirect URL and status code if set.
// Returns ("", 0, false) if no redirect is set.
func (s *RequestState) Redirect() (url string, code int, hasRedirect bool) {
	if s == nil {
		return "", 0, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.redirectURL != "" {
		return s.redirectURL, s.redirectCode, true
	}
	return "", 0, false
}

// MutateCookie records a cookie mutation (for optimistic updates).
// This affects subsequent GetCookie calls but doesn't actually set the cookie.
func (s *RequestState) MutateCookie(name, value string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cookieMutations[name] = &cookieMutation{value: value, deleted: false}
}

// DeleteCookieMutation records a cookie deletion (for optimistic updates).
func (s *RequestState) DeleteCookieMutation(name string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cookieMutations[name] = &cookieMutation{deleted: true}
}

// CookieMutations returns all cookie mutations made during render.
// Used by cookie manager to know what to actually set.
func (s *RequestState) CookieMutations() map[string]*cookieMutation {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	mutations := make(map[string]*cookieMutation, len(s.cookieMutations))
	for k, v := range s.cookieMutations {
		mutations[k] = &cookieMutation{value: v.value, deleted: v.deleted}
	}
	return mutations
}
