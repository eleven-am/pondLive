package headers

import (
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/eleven-am/pondlive/internal/route"
)

type RequestInfo struct {
	Method     string
	Scheme     string
	Host       string
	RemoteAddr string

	Path    string
	Query   url.Values
	Hash    string
	Headers http.Header

	retained retainSet
}

type retainSet map[string]struct{}

var defaultRetained = buildRetainSet(nil)

func newRetainSet(extra []string) retainSet {
	if len(extra) == 0 {
		return defaultRetained
	}
	return buildRetainSet(extra)
}

func buildRetainSet(extra []string) retainSet {
	names := []string{
		"Cookie", "User-Agent", "Accept", "Accept-Language", "Referer",
		"X-Forwarded-For", "X-Forwarded-Proto", "X-Forwarded-Host", "X-Real-Ip",
	}
	set := make(retainSet, len(names)+len(extra))
	for _, n := range names {
		set[http.CanonicalHeaderKey(n)] = struct{}{}
	}
	for _, n := range extra {
		set[http.CanonicalHeaderKey(n)] = struct{}{}
	}
	return set
}

func (r *RequestInfo) retainSet() retainSet {
	if r == nil || r.retained == nil {
		return defaultRetained
	}
	return r.retained
}

func retainedHeaders(src http.Header, set retainSet) http.Header {
	matches, values := 0, 0
	for name := range set {
		if n := len(src[name]); n > 0 {
			matches++
			values += n
		}
	}
	out := make(http.Header, matches)
	if matches == 0 {
		return out
	}
	backing := make([]string, 0, values)
	for name := range set {
		if vals := src[name]; len(vals) > 0 {
			start := len(backing)
			backing = append(backing, vals...)
			out[name] = backing[start:len(backing):len(backing)]
		}
	}
	return out
}

func NewRequestInfo(r *http.Request, extraRetained ...string) *RequestInfo {
	set := newRetainSet(extraRetained)
	if r == nil {
		return &RequestInfo{
			Query:    make(url.Values),
			Headers:  make(http.Header),
			retained: set,
		}
	}

	hash := ""
	query := make(url.Values)
	path := ""

	if r.URL != nil {
		path = r.URL.Path
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
		Scheme:     requestScheme(r),
		Host:       r.Host,
		RemoteAddr: r.RemoteAddr,
		Path:       path,
		Query:      query,
		Hash:       hash,
		Headers:    retainedHeaders(r.Header, set),
		retained:   set,
	}
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := strings.ToLower(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0])); proto == "https" {
		return "https"
	}
	return "http"
}

func (r *RequestInfo) scheme() string {
	if r == nil || r.Scheme == "" {
		return "http"
	}
	return strings.ToLower(r.Scheme)
}

func NewRequestInfoFromHeaders(headers http.Header, extraRetained ...string) *RequestInfo {
	set := newRetainSet(extraRetained)
	return &RequestInfo{
		Query:    make(url.Values),
		Headers:  retainedHeaders(headers, set),
		retained: set,
	}
}

func (r *RequestInfo) Get(name string) (string, bool) {
	if r == nil || r.Headers == nil {
		return "", false
	}
	val := r.Headers.Get(name)
	return val, val != ""
}

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
		Scheme:     r.Scheme,
		Host:       r.Host,
		RemoteAddr: r.RemoteAddr,
		Path:       r.Path,
		Query:      query,
		Hash:       r.Hash,
		Headers:    r.Headers.Clone(),
		retained:   r.retained,
	}
}

func (r *RequestInfo) withCookie(c *http.Cookie, deleted bool) *RequestInfo {
	next := r.Clone()
	if next.Headers == nil {
		next.Headers = make(http.Header)
	}

	var parts []string
	for _, line := range next.Headers.Values("Cookie") {
		for _, part := range strings.Split(line, ";") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if n, _, ok := strings.Cut(part, "="); ok && strings.TrimSpace(n) == c.Name {
				continue
			}
			parts = append(parts, part)
		}
	}
	if pair := cookiePair(c); !deleted && pair != "" {
		parts = append(parts, pair)
	}

	if len(parts) == 0 {
		next.Headers.Del("Cookie")
	} else {
		next.Headers.Set("Cookie", strings.Join(parts, "; "))
	}
	return next
}

func cookiePair(c *http.Cookie) string {
	return (&http.Cookie{Name: c.Name, Value: c.Value}).String()
}

func (r *RequestInfo) withHeaders(h http.Header) *RequestInfo {
	next := r.Clone()
	if next.Headers == nil {
		next.Headers = make(http.Header)
	}
	for name := range r.retainSet() {
		vals := h[name]
		switch {
		case len(vals) > 0:
			next.Headers[name] = append([]string(nil), vals...)
		case name == "Cookie":
			delete(next.Headers, name)
		}
	}
	return next
}

type RequestState struct {
	mu sync.RWMutex

	info *RequestInfo
	loc  route.Location

	isLive bool

	responseHeaders http.Header
	redirectURL     string
	redirectCode    int

	cookieMutations map[string]*cookieMutation

	onChange func()
}

type cookieMutation struct {
	value   string
	deleted bool
}

func NewRequestState(info *RequestInfo) *RequestState {
	if info == nil {
		info = &RequestInfo{
			Query:   make(url.Values),
			Headers: make(http.Header),
		}
	}
	return &RequestState{
		info:            info.Clone(),
		loc:             route.Canonicalize(route.Location{Path: info.Path, Query: info.Query, Hash: info.Hash}),
		responseHeaders: make(http.Header),
		cookieMutations: make(map[string]*cookieMutation),
	}
}

func (s *RequestState) Info() *RequestInfo {
	return s.snapshot().Clone()
}

func (s *RequestState) snapshot() *RequestInfo {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.info
}

func (s *RequestState) Get(name string) (string, bool) {
	return s.snapshot().Get(name)
}

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

	return s.info.GetCookie(name)
}

func (s *RequestState) effectiveHeaders() http.Header {
	if s == nil {
		return http.Header{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	info := s.info
	if info == nil {
		return http.Header{}
	}
	for name, m := range s.cookieMutations {
		info = info.withCookie(&http.Cookie{Name: name, Value: m.value}, m.deleted)
	}
	if info.Headers == nil {
		return http.Header{}
	}
	return info.Headers.Clone()
}

func (s *RequestState) URL() route.Location {
	if s == nil {
		return route.Location{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loc.Clone()
}

func (s *RequestState) setURL(loc route.Location) {
	s.mu.Lock()
	s.loc = loc.Clone()
	s.mu.Unlock()
}

func (s *RequestState) IsLive() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isLive
}

func (s *RequestState) SetIsLive(live bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.isLive = live
}

func (s *RequestState) SetResponseHeader(name, value string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseHeaders.Set(name, value)
}

func (s *RequestState) AddResponseHeader(name, value string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responseHeaders.Add(name, value)
}

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

func (s *RequestState) SetRedirect(url string, code int) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.redirectURL = url
	s.redirectCode = code
}

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

func (s *RequestState) MutateCookie(name, value string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cookieMutations[name] = &cookieMutation{value: value, deleted: false}
}

func (s *RequestState) DeleteCookieMutation(name string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cookieMutations[name] = &cookieMutation{deleted: true}
}

func (s *RequestState) MergeHeaders(h http.Header) {
	if s == nil {
		return
	}

	s.mu.Lock()
	if s.info == nil {
		s.info = NewRequestInfo(nil)
	}
	s.info = s.info.withHeaders(h)
	s.mu.Unlock()
	s.NotifyChange()
}

func (s *RequestState) ApplyCookie(c *http.Cookie, deleted bool) {
	if s == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.info == nil {
		s.info = NewRequestInfo(nil)
	}
	s.info = s.info.withCookie(c, deleted)
}

func (s *RequestState) bindNotify(fn func()) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.onChange = fn
	s.mu.Unlock()
}

func (s *RequestState) Clone() *RequestState {
	if s == nil {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	responseHeaders := make(http.Header, len(s.responseHeaders))
	for k, v := range s.responseHeaders {
		responseHeaders[k] = append([]string(nil), v...)
	}

	cookieMutations := make(map[string]*cookieMutation, len(s.cookieMutations))
	for k, v := range s.cookieMutations {
		cookieMutations[k] = &cookieMutation{value: v.value, deleted: v.deleted}
	}

	return &RequestState{
		info:            s.info,
		loc:             s.loc.Clone(),
		isLive:          s.isLive,
		responseHeaders: responseHeaders,
		redirectURL:     s.redirectURL,
		redirectCode:    s.redirectCode,
		cookieMutations: cookieMutations,
		onChange:        s.onChange,
	}
}

func (s *RequestState) NotifyChange() {
	if s == nil {
		return
	}
	s.mu.RLock()
	notify := s.onChange
	s.mu.RUnlock()
	if notify != nil {
		notify()
	}
}
