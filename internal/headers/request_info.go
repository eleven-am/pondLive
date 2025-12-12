package headers

import (
	"log"
	"net/http"
	"net/url"
	"sync"
)

type RequestInfo struct {
	Method     string
	Host       string
	RemoteAddr string

	Path    string
	Query   url.Values
	Hash    string
	Headers http.Header
}

func NewRequestInfo(r *http.Request) *RequestInfo {
	if r == nil {
		return &RequestInfo{
			Query:   make(url.Values),
			Headers: make(http.Header),
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

	headers := make(http.Header)
	if r.Header != nil {
		headers = r.Header.Clone()
	}

	return &RequestInfo{
		Method:     r.Method,
		Host:       r.Host,
		RemoteAddr: r.RemoteAddr,
		Path:       path,
		Query:      query,
		Hash:       hash,
		Headers:    headers,
	}
}

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

type RequestState struct {
	mu sync.RWMutex

	info *RequestInfo

	isLive bool

	responseHeaders http.Header
	redirectURL     string
	redirectCode    int

	cookieMutations map[string]*cookieMutation

	setState func(*RequestState)
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
		info:            info,
		responseHeaders: make(http.Header),
		cookieMutations: make(map[string]*cookieMutation),
	}
}

func (s *RequestState) Info() *RequestInfo {
	if s == nil {
		return nil
	}
	return s.info
}

func (s *RequestState) Get(name string) (string, bool) {
	if s == nil || s.info == nil {
		return "", false
	}
	return s.info.Get(name)
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

	if s.info == nil {
		return "", false
	}
	return s.info.GetCookie(name)
}

func (s *RequestState) Path() string {
	if s == nil || s.info == nil {
		return ""
	}
	return s.info.Path
}

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

func (s *RequestState) Hash() string {
	if s == nil || s.info == nil {
		return ""
	}
	return s.info.Hash
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

func (s *RequestState) ReplaceInfo(info *RequestInfo) {
	if s == nil {
		return
	}

	s.mu.Lock()
	s.info = info
	s.responseHeaders = make(http.Header)
	s.cookieMutations = make(map[string]*cookieMutation)
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
		info:            s.info.Clone(),
		isLive:          s.isLive,
		responseHeaders: responseHeaders,
		redirectURL:     s.redirectURL,
		redirectCode:    s.redirectCode,
		cookieMutations: cookieMutations,
		setState:        s.setState,
	}
}

func (s *RequestState) NotifyChange() {
	if s == nil || s.setState == nil {
		log.Printf("[NotifyChange] skipping - s=%v setState=%v", s != nil, s != nil && s.setState != nil)
		return
	}
	log.Printf("[NotifyChange] calling setState isLive=%v", s.IsLive())
	clone := s.Clone()
	s.setState(clone)
}
