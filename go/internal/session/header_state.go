package session

import (
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const CookieEndpointPath = "/pondlive/cookie"

type HeaderState interface {
	GetHeader(name string) (string, bool)
	SetHeader(name, value string)
	DeleteHeader(name string)
	AllHeaders() http.Header

	GetCookie(name string) (*http.Cookie, bool)
	SetCookie(cookie *http.Cookie)
	DeleteCookie(name string)
	AllCookies() []*http.Cookie

	RequestID() string

	SetMeta(key string, value any)
	Meta(key string) (any, bool)
}

type headerState struct {
	mu        sync.RWMutex
	headers   http.Header
	cookies   map[string]*http.Cookie
	meta      map[string]any
	requestID string

	pending struct {
		setCookies []*http.Cookie
		delCookies []string
		setHeaders http.Header
		delHeaders []string
	}
}

func newHeaderState() *headerState {
	hs := &headerState{
		headers: make(http.Header),
		cookies: make(map[string]*http.Cookie),
		meta:    make(map[string]any),
	}
	hs.pending.setHeaders = make(http.Header)
	return hs
}

func (s *headerState) clone() *headerState {
	if s == nil {
		return newHeaderState()
	}
	cp := newHeaderState()
	s.mu.RLock()
	for key, values := range s.headers {
		cp.headers[key] = append([]string(nil), values...)
	}
	for name, cookie := range s.cookies {
		cp.cookies[name] = cloneCookie(cookie)
	}
	for key, value := range s.meta {
		cp.meta[key] = value
	}
	cp.requestID = s.requestID
	s.mu.RUnlock()
	return cp
}

func (s *headerState) mergeRequest(r *http.Request) {
	if s == nil || r == nil {
		return
	}
	s.mu.Lock()
	for key, values := range r.Header {
		s.headers[key] = append([]string(nil), values...)
	}
	cookies := r.Cookies()
	if len(cookies) > 0 {
		for _, cookie := range cookies {
			if cookie == nil || cookie.Name == "" {
				continue
			}
			s.cookies[cookie.Name] = cloneCookie(cookie)
		}
	}
	if rid := strings.TrimSpace(r.Header.Get("X-Request-Id")); rid != "" {
		s.requestID = rid
	} else if rid := strings.TrimSpace(r.Header.Get("X-Request-ID")); rid != "" {
		s.requestID = rid
	}
	s.mu.Unlock()
}

func (s *headerState) mergeHeaders(headers http.Header) {
	if s == nil || len(headers) == 0 {
		return
	}
	s.mu.Lock()
	for key, values := range headers {
		s.headers[key] = append(s.headers[key], values...)
	}
	s.mu.Unlock()
}

func (s *headerState) mergeCookies(cookies []*http.Cookie) {
	if s == nil || len(cookies) == 0 {
		return
	}
	s.mu.Lock()
	for _, cookie := range cookies {
		if cookie == nil || cookie.Name == "" {
			continue
		}
		s.cookies[cookie.Name] = cloneCookie(cookie)
	}
	s.mu.Unlock()
}

func (s *headerState) GetHeader(name string) (string, bool) {
	if s == nil || name == "" {
		return "", false
	}
	canonical := http.CanonicalHeaderKey(name)
	s.mu.RLock()
	defer s.mu.RUnlock()
	values := s.headers[canonical]
	if len(values) == 0 {
		return "", false
	}
	return values[0], true
}

func (s *headerState) SetHeader(name, value string) {
	if s == nil || name == "" {
		return
	}
	canonical := http.CanonicalHeaderKey(name)
	s.mu.Lock()
	if s.headers == nil {
		s.headers = make(http.Header)
	}
	s.headers[canonical] = []string{value}
	if s.pending.setHeaders == nil {
		s.pending.setHeaders = make(http.Header)
	}
	s.pending.setHeaders[canonical] = []string{value}
	removeString(&s.pending.delHeaders, canonical)
	s.mu.Unlock()
}

func (s *headerState) DeleteHeader(name string) {
	if s == nil || name == "" {
		return
	}
	canonical := http.CanonicalHeaderKey(name)
	s.mu.Lock()
	delete(s.headers, canonical)
	removeString(&s.pending.delHeaders, canonical)
	s.pending.delHeaders = append(s.pending.delHeaders, canonical)
	if s.pending.setHeaders != nil {
		delete(s.pending.setHeaders, canonical)
	}
	s.mu.Unlock()
}

func (s *headerState) AllHeaders() http.Header {
	if s == nil {
		return http.Header{}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	copyMap := make(http.Header, len(s.headers))
	for key, values := range s.headers {
		copyMap[key] = append([]string(nil), values...)
	}
	return copyMap
}

func (s *headerState) GetCookie(name string) (*http.Cookie, bool) {
	if s == nil || name == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	cookie, ok := s.cookies[name]
	if !ok || cookie == nil {
		return nil, false
	}
	return cloneCookie(cookie), true
}

func (s *headerState) SetCookie(cookie *http.Cookie) {
	if s == nil || cookie == nil || cookie.Name == "" {
		return
	}
	clone := cloneCookie(cookie)
	clone.HttpOnly = true
	if clone.Path == "" {
		clone.Path = "/"
	}
	s.mu.Lock()
	if s.cookies == nil {
		s.cookies = make(map[string]*http.Cookie)
	}
	s.cookies[clone.Name] = clone
	s.pending.setCookies = append(s.pending.setCookies, cloneCookie(clone))
	removeString(&s.pending.delCookies, clone.Name)
	s.mu.Unlock()
}

func (s *headerState) DeleteCookie(name string) {
	if s == nil || name == "" {
		return
	}
	s.mu.Lock()
	delete(s.cookies, name)
	removeString(&s.pending.delCookies, name)
	s.pending.delCookies = append(s.pending.delCookies, name)
	if len(s.pending.setCookies) > 0 {
		filtered := s.pending.setCookies[:0]
		for _, ck := range s.pending.setCookies {
			if ck != nil && ck.Name == name {
				continue
			}
			filtered = append(filtered, ck)
		}
		s.pending.setCookies = filtered
	}
	s.mu.Unlock()
}

func (s *headerState) AllCookies() []*http.Cookie {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.cookies) == 0 {
		return nil
	}
	names := make([]string, 0, len(s.cookies))
	for name := range s.cookies {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]*http.Cookie, 0, len(names))
	for _, name := range names {
		if ck := s.cookies[name]; ck != nil {
			out = append(out, cloneCookie(ck))
		}
	}
	return out
}

func (s *headerState) RequestID() string {
	if s == nil {
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.requestID
}

func (s *headerState) SetMeta(key string, value any) {
	if s == nil || strings.TrimSpace(key) == "" {
		return
	}
	normalized := strings.TrimSpace(key)
	s.mu.Lock()
	if value == nil {
		delete(s.meta, normalized)
	} else {
		if s.meta == nil {
			s.meta = make(map[string]any)
		}
		s.meta[normalized] = value
	}
	s.mu.Unlock()
}

func (s *headerState) Meta(key string) (any, bool) {
	if s == nil || strings.TrimSpace(key) == "" {
		return nil, false
	}
	normalized := strings.TrimSpace(key)
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.meta[normalized]
	return val, ok
}

func (s *headerState) hasCookieMutations() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	pending := len(s.pending.setCookies) > 0 || len(s.pending.delCookies) > 0
	s.mu.RUnlock()
	return pending
}

func (s *headerState) drainCookieMutations() CookieBatch {
	if s == nil {
		return CookieBatch{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	batch := CookieBatch{}
	if len(s.pending.setCookies) > 0 {
		for _, ck := range s.pending.setCookies {
			if ck == nil {
				continue
			}
			batch.Set = append(batch.Set, cloneCookie(ck))
		}
	}
	if len(s.pending.delCookies) > 0 {
		batch.Delete = append(batch.Delete, s.pending.delCookies...)
	}
	s.pending.setCookies = nil
	s.pending.delCookies = nil
	if len(s.pending.setHeaders) > 0 {
		s.pending.setHeaders = make(http.Header)
	}
	if len(s.pending.delHeaders) > 0 {
		s.pending.delHeaders = nil
	}
	return batch
}

// CookieBatch describes cookie mutations pending acknowledgement from the client.
type CookieBatch struct {
	Set    []*http.Cookie
	Delete []string
}

// Empty reports whether the batch carries any changes.
func (b CookieBatch) Empty() bool {
	return len(b.Set) == 0 && len(b.Delete) == 0
}

func cloneCookie(in *http.Cookie) *http.Cookie {
	if in == nil {
		return nil
	}
	cp := *in
	if in.Value != "" {
		cp.Value = in.Value
	}
	if len(in.Raw) > 0 {
		cp.Raw = in.Raw
	}
	if len(in.Unparsed) > 0 {
		cp.Unparsed = append([]string(nil), in.Unparsed...)
	}
	if in.Expires.IsZero() {
		cp.Expires = time.Time{}
	} else {
		cp.Expires = in.Expires
	}
	return &cp
}

func removeString(slice *[]string, target string) {
	if slice == nil || len(*slice) == 0 {
		return
	}
	out := (*slice)[:0]
	for _, item := range *slice {
		if item == target {
			continue
		}
		out = append(out, item)
	}
	*slice = out
}
