package headers

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
)

func TestRequestStateGetCookieWithMutation(t *testing.T) {
	headers := http.Header{}
	headers.Set("Cookie", "session=original")

	info := NewRequestInfoFromHeaders(headers)
	state := NewRequestState(info)

	val, ok := state.GetCookie("session")
	if !ok || val != "original" {
		t.Errorf("Expected 'original', got %q (ok=%v)", val, ok)
	}

	state.MutateCookie("session", "mutated")

	val, ok = state.GetCookie("session")
	if !ok || val != "mutated" {
		t.Errorf("Expected 'mutated' after mutation, got %q (ok=%v)", val, ok)
	}
}

func TestRequestStateDeleteCookieMutation(t *testing.T) {
	headers := http.Header{}
	headers.Set("Cookie", "session=value")

	info := NewRequestInfoFromHeaders(headers)
	state := NewRequestState(info)

	state.DeleteCookieMutation("session")

	val, ok := state.GetCookie("session")
	if ok || val != "" {
		t.Errorf("Expected cookie to appear deleted, got %q (ok=%v)", val, ok)
	}
}

func TestRequestStateResponseHeaders(t *testing.T) {
	state := NewRequestState(nil)

	state.SetResponseHeader("X-Custom", "value1")
	state.AddResponseHeader("X-Multi", "a")
	state.AddResponseHeader("X-Multi", "b")

	headers := state.ResponseHeaders()

	if headers.Get("X-Custom") != "value1" {
		t.Errorf("Expected X-Custom='value1', got %q", headers.Get("X-Custom"))
	}

	multi := headers.Values("X-Multi")
	if len(multi) != 2 || multi[0] != "a" || multi[1] != "b" {
		t.Errorf("Expected X-Multi=['a','b'], got %v", multi)
	}
}

func TestRequestStateRedirect(t *testing.T) {
	state := NewRequestState(nil)

	url, code, has := state.Redirect()
	if has {
		t.Error("Expected no redirect initially")
	}

	state.SetRedirect("/login", http.StatusFound)

	url, code, has = state.Redirect()
	if !has {
		t.Error("Expected redirect to be set")
	}
	if url != "/login" {
		t.Errorf("Expected URL '/login', got %q", url)
	}
	if code != http.StatusFound {
		t.Errorf("Expected code %d, got %d", http.StatusFound, code)
	}
}

func TestRequestStateIsLive(t *testing.T) {
	state := NewRequestState(nil)

	if state.IsLive() {
		t.Error("Expected IsLive=false initially")
	}

	state.SetIsLive(true)

	if !state.IsLive() {
		t.Error("Expected IsLive=true after SetIsLive")
	}
}

func TestRequestStateCookieMutations(t *testing.T) {
	state := NewRequestState(nil)

	state.MutateCookie("auth", "token123")
	state.MutateCookie("theme", "dark")
	state.DeleteCookieMutation("old")

	mutations := cookieMutationsOf(state)

	if len(mutations) != 3 {
		t.Errorf("Expected 3 mutations, got %d", len(mutations))
	}

	if mutations["auth"].value != "token123" || mutations["auth"].deleted {
		t.Error("auth mutation incorrect")
	}

	if mutations["theme"].value != "dark" || mutations["theme"].deleted {
		t.Error("theme mutation incorrect")
	}

	if !mutations["old"].deleted {
		t.Error("old should be marked deleted")
	}
}

func TestRequestInfoClone(t *testing.T) {
	info := &RequestInfo{
		Method: "GET",
		Path:   "/test",
		Query:  map[string][]string{"a": {"1"}},
		Headers: http.Header{
			"X-Test": {"value"},
		},
	}

	clone := info.Clone()

	info.Method = "POST"
	info.Query["a"] = []string{"2"}
	info.Headers.Set("X-Test", "modified")

	if clone.Method != "GET" {
		t.Error("Clone method was modified")
	}
	if clone.Query["a"][0] != "1" {
		t.Error("Clone query was modified")
	}
	if clone.Headers.Get("X-Test") != "value" {
		t.Error("Clone headers were modified")
	}
}

func TestNilSafety(t *testing.T) {
	var info *RequestInfo
	var state *RequestState

	if _, ok := info.Get("test"); ok {
		t.Error("Expected false for nil info.Get")
	}
	if _, ok := info.GetCookie("test"); ok {
		t.Error("Expected false for nil info.GetCookie")
	}
	if info.Clone() != nil {
		t.Error("Expected nil for nil info.Clone")
	}

	if state.Info() != nil {
		t.Error("Expected nil for nil state.Info")
	}
	if _, ok := state.Get("test"); ok {
		t.Error("Expected false for nil state.Get")
	}
	if _, ok := state.GetCookie("test"); ok {
		t.Error("Expected false for nil state.GetCookie")
	}
	if got := state.URL(); got.Path != "" || got.Query != nil || got.Hash != "" {
		t.Errorf("Expected zero URL for nil state, got %+v", got)
	}
	if state.IsLive() {
		t.Error("Expected false for nil state.IsLive")
	}
	if state.ResponseHeaders() != nil {
		t.Error("Expected nil for nil state.ResponseHeaders")
	}
	if _, _, has := state.Redirect(); has {
		t.Error("Expected no redirect for nil state")
	}
	if cookieMutationsOf(state) != nil {
		t.Error("Expected nil for nil state.CookieMutations")
	}

	state.SetIsLive(true)
	state.SetResponseHeader("X", "Y")
	state.AddResponseHeader("X", "Z")
	state.SetRedirect("/", 302)
	state.MutateCookie("a", "b")
	state.DeleteCookieMutation("c")
}

func TestNewRequestInfo_NilURL(t *testing.T) {
	r := &http.Request{
		Method:     "GET",
		Host:       "example.com",
		RemoteAddr: "127.0.0.1:8080",
		URL:        nil,
		Header:     http.Header{"User-Agent": {"value"}},
	}

	info := NewRequestInfo(r)

	if info.Method != "GET" {
		t.Errorf("Expected Method='GET', got %q", info.Method)
	}
	if info.Host != "example.com" {
		t.Errorf("Expected Host='example.com', got %q", info.Host)
	}
	if info.Path != "" {
		t.Errorf("Expected Path='', got %q", info.Path)
	}
	if info.Query == nil {
		t.Error("Expected Query to be initialized, got nil")
	}
	if info.Hash != "" {
		t.Errorf("Expected Hash='', got %q", info.Hash)
	}
	if info.Headers.Get("User-Agent") != "value" {
		t.Errorf("Expected header User-Agent='value', got %q", info.Headers.Get("User-Agent"))
	}
}

func TestNewRequestInfo_NilHeader(t *testing.T) {
	r := &http.Request{
		Method: "POST",
		Header: nil,
	}

	info := NewRequestInfo(r)

	if info.Headers == nil {
		t.Error("Expected Headers to be initialized, got nil")
	}
}

func TestNewRequestInfo_NilRequest(t *testing.T) {
	info := NewRequestInfo(nil)

	if info == nil {
		t.Fatal("Expected non-nil RequestInfo for nil request")
	}
	if info.Query == nil {
		t.Error("Expected Query to be initialized")
	}
	if info.Headers == nil {
		t.Error("Expected Headers to be initialized")
	}
}

func TestRequestStateQuery_ReturnsClone(t *testing.T) {
	info := &RequestInfo{
		Query: map[string][]string{
			"key": {"value1", "value2"},
		},
	}
	state := NewRequestState(info)

	query := state.URL().Query
	query["key"] = []string{"modified"}
	query["new"] = []string{"added"}

	original := state.URL().Query
	if len(original["key"]) != 2 || original["key"][0] != "value1" {
		t.Error("Original query was modified")
	}
	if _, exists := original["new"]; exists {
		t.Error("New key should not exist in original")
	}
}

func TestRequestStateQuery_MultiValue(t *testing.T) {
	info := &RequestInfo{
		Query: map[string][]string{
			"tags": {"go", "rust", "python"},
		},
	}
	state := NewRequestState(info)

	query := state.URL().Query

	if len(query["tags"]) != 3 {
		t.Errorf("Expected 3 values for tags, got %d", len(query["tags"]))
	}
	if query["tags"][0] != "go" || query["tags"][1] != "python" || query["tags"][2] != "rust" {
		t.Errorf("Query values mismatch: %v", query["tags"])
	}
}

func TestRequestStateMergeHeadersKeepsPageAndMutations(t *testing.T) {
	state := NewRequestState(&RequestInfo{Method: "GET", Path: "/original", Query: map[string][]string{"a": {"1"}}, Hash: "h"})
	state.SetResponseHeader("X-Old", "value")
	state.MutateCookie("old_cookie", "value")

	state.MergeHeaders(http.Header{"Cookie": {"s=1"}, "User-Agent": {"ua"}})

	info := state.Info()
	if info.Path != "/original" || info.Method != "GET" || info.Hash != "h" || info.Query["a"][0] != "1" {
		t.Errorf("page facts changed: %+v", info)
	}
	if state.ResponseHeaders().Get("X-Old") != "value" {
		t.Error("response headers were cleared")
	}
	if v, ok := cookieMutationsOf(state)["old_cookie"]; !ok || v.value != "value" {
		t.Error("cookie mutations were cleared")
	}
	if v, _ := state.GetCookie("s"); v != "1" {
		t.Errorf("merged cookie missing, got %q", v)
	}
}

func TestRequestStateClone(t *testing.T) {
	info := &RequestInfo{
		Method: "GET",
		Path:   "/test",
		Query:  map[string][]string{"a": {"1"}},
	}
	state := NewRequestState(info)
	state.SetIsLive(true)
	state.SetResponseHeader("X-Test", "value")
	state.MutateCookie("session", "abc")
	state.SetRedirect("/login", http.StatusFound)

	clone := state.Clone()

	if clone == state {
		t.Error("Clone should be a different instance")
	}
	if clone.URL().Path != "/test" {
		t.Errorf("Expected path '/test', got %q", clone.URL().Path)
	}
	if !clone.IsLive() {
		t.Error("Expected IsLive=true in clone")
	}
	if clone.ResponseHeaders().Get("X-Test") != "value" {
		t.Error("Expected response header in clone")
	}
	if m := cookieMutationsOf(clone); m["session"] == nil || m["session"].value != "abc" {
		t.Error("Expected cookie mutation in clone")
	}
	url, code, has := clone.Redirect()
	if !has || url != "/login" || code != http.StatusFound {
		t.Error("Expected redirect in clone")
	}

	state.SetResponseHeader("X-Test", "modified")
	if clone.ResponseHeaders().Get("X-Test") == "modified" {
		t.Error("Clone should not be affected by original modification")
	}
}

func TestRequestStateNotifyChange_NilState(t *testing.T) {
	var state *RequestState
	state.NotifyChange()
}

func TestRequestStateNotifyChange_NilSetState(t *testing.T) {
	state := NewRequestState(nil)
	state.NotifyChange()
}

func TestRequestStateGetCookie_NoCookieHeader(t *testing.T) {
	headers := http.Header{}
	info := NewRequestInfoFromHeaders(headers)
	state := NewRequestState(info)

	val, ok := state.GetCookie("session")
	if ok || val != "" {
		t.Errorf("Expected no cookie, got %q (ok=%v)", val, ok)
	}
}

func TestRequestStateGetCookie_MutationOverridesPriority(t *testing.T) {
	headers := http.Header{}
	headers.Set("Cookie", "session=original")
	info := NewRequestInfoFromHeaders(headers)
	state := NewRequestState(info)

	state.MutateCookie("session", "mutated1")
	state.MutateCookie("session", "mutated2")

	val, ok := state.GetCookie("session")
	if !ok || val != "mutated2" {
		t.Errorf("Expected 'mutated2', got %q (ok=%v)", val, ok)
	}
}

func TestRequestStateInfo_WithNilInfo(t *testing.T) {
	state := NewRequestState(nil)
	info := state.Info()
	if info == nil {
		t.Error("Expected non-nil info (defaults created)")
	}
	if info.Query == nil {
		t.Error("Expected Query to be initialized")
	}
	if info.Headers == nil {
		t.Error("Expected Headers to be initialized")
	}
}

func TestRequestStateGetCookie_NilInfoNoMutation(t *testing.T) {
	state := &RequestState{
		info:            nil,
		cookieMutations: make(map[string]*cookieMutation),
	}

	val, ok := state.GetCookie("session")
	if ok || val != "" {
		t.Errorf("Expected empty/false for nil info, got %q (ok=%v)", val, ok)
	}
}

func TestRequestStateGet_InfoNil(t *testing.T) {
	state := NewRequestState(nil)
	val, ok := state.Get("X-Test")
	if ok || val != "" {
		t.Errorf("Expected empty, got %q (ok=%v)", val, ok)
	}
}

func TestRequestStatePath_NilInfo(t *testing.T) {
	state := NewRequestState(nil)
	if state.URL().Path != "/" {
		t.Error("Expected root path for nil info")
	}
}

func TestRequestStateHash_NilInfo(t *testing.T) {
	state := NewRequestState(nil)
	if state.URL().Hash != "" {
		t.Error("Expected empty hash for nil info")
	}
}

func TestRequestInfoGet_NonExistent(t *testing.T) {
	headers := http.Header{}
	headers.Set("X-Existing", "value")
	info := NewRequestInfoFromHeaders(headers)

	val, ok := info.Get("X-NotExisting")
	if ok || val != "" {
		t.Errorf("Expected empty for non-existent header, got %q (ok=%v)", val, ok)
	}
}

func TestRequestInfoGet_Exists(t *testing.T) {
	headers := http.Header{}
	headers.Set("User-Agent", "value123")
	info := NewRequestInfoFromHeaders(headers)

	val, ok := info.Get("User-Agent")
	if !ok || val != "value123" {
		t.Errorf("Expected 'value123', got %q (ok=%v)", val, ok)
	}
}

func TestRequestInfoGetCookie_NotFound(t *testing.T) {
	headers := http.Header{}
	headers.Set("Cookie", "other=value")
	info := NewRequestInfoFromHeaders(headers)

	val, ok := info.GetCookie("session")
	if ok || val != "" {
		t.Errorf("Expected empty for non-existent cookie, got %q (ok=%v)", val, ok)
	}
}

func TestNewRequestInfo_WithHashQuery(t *testing.T) {
	r := &http.Request{
		Method: "GET",
		Host:   "example.com",
		URL:    &url.URL{Path: "/test", RawQuery: "_hash=section&page=1&_hash=ignored"},
		Header: http.Header{},
	}

	info := NewRequestInfo(r)

	if info.Hash != "section" {
		t.Errorf("Expected Hash='section', got %q", info.Hash)
	}
	if _, hasHash := info.Query["_hash"]; hasHash {
		t.Error("_hash should not be in Query map")
	}
	if info.Query.Get("page") != "1" {
		t.Errorf("Expected page=1, got %q", info.Query.Get("page"))
	}
}

func TestNewRequestInfo_WithFullURL(t *testing.T) {
	r := &http.Request{
		Method:     "POST",
		Host:       "api.example.com",
		RemoteAddr: "192.168.1.1:12345",
		URL:        &url.URL{Path: "/api/users", RawQuery: "sort=asc&filter=active"},
		Header:     http.Header{"Authorization": {"Bearer token123"}, "Content-Type": {"application/json"}},
	}

	info := NewRequestInfo(r)

	if info.Method != "POST" {
		t.Errorf("Expected Method='POST', got %q", info.Method)
	}
	if info.Host != "api.example.com" {
		t.Errorf("Expected Host='api.example.com', got %q", info.Host)
	}
	if info.RemoteAddr != "192.168.1.1:12345" {
		t.Errorf("Expected RemoteAddr='192.168.1.1:12345', got %q", info.RemoteAddr)
	}
	if info.Path != "/api/users" {
		t.Errorf("Expected Path='/api/users', got %q", info.Path)
	}
	if info.Query.Get("sort") != "asc" {
		t.Errorf("Expected sort=asc, got %q", info.Query.Get("sort"))
	}
	if info.Headers.Get("Authorization") != "" {
		t.Errorf("Authorization must not be retained, got %q", info.Headers.Get("Authorization"))
	}
}

func TestRequestStateMergeHeaders_NilState(t *testing.T) {
	var state *RequestState
	state.MergeHeaders(nil)
}

func TestRequestStateClone_NilState(t *testing.T) {
	var state *RequestState
	clone := state.Clone()
	if clone != nil {
		t.Error("Expected nil clone from nil state")
	}
}

func TestProviderStateStoreCookie(t *testing.T) {
	pState := &providerState{holder: &tokenHolder{}}

	pState.storeCookie("token123", &pendingCookie{name: "session", value: "abc"})

	tokens := pState.holder.get()
	if len(tokens.pending.slots) != 1 {
		t.Errorf("Expected 1 pending cookie, got %d", len(tokens.pending.slots))
	}
	if tokens.pending.slots["token123"].cookie.name != "session" {
		t.Error("Cookie not stored correctly")
	}
}

func TestProviderStateConsumeCookie(t *testing.T) {
	tokens := newCookieTokens()
	tokens.store("token123", &pendingCookie{name: "session", value: "abc"})

	cookie := tokens.consume("token123")
	if cookie == nil {
		t.Fatal("Expected cookie to be returned")
	}
	if cookie.name != "session" || cookie.value != "abc" {
		t.Error("Cookie data incorrect")
	}
	if len(tokens.pending.slots) != 0 {
		t.Error("Cookie should be removed after consume")
	}
	if tokens.consume("token123") != nil {
		t.Error("Token must be single use")
	}
}

func TestProviderStateConsumeCookie_NotFound(t *testing.T) {
	if newCookieTokens().consume("nonexistent") != nil {
		t.Error("Expected nil for non-existent cookie")
	}
}

func TestProviderStateConcurrentAccess(t *testing.T) {
	pState := &providerState{holder: &tokenHolder{}}

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(id int) {
			token := "token" + string(rune('0'+id))
			pState.storeCookie(token, &pendingCookie{name: token, value: "value"})
			pState.holder.get().consume(token)
			pState.holder.get().acknowledge(token)
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSendCookieViaScript_NilProviderState(t *testing.T) {
	sendCookieViaScript(nil, "test", "value", nil)
}

func newAckFixture(t *testing.T) (*providerState, *RequestState) {
	t.Helper()
	page := httptest.NewRequest(http.MethodGet, "/dashboard?tab=2&_hash=top", nil)
	page.Header.Set("Cookie", "session=old")
	state := NewRequestState(NewRequestInfo(page))
	return &providerState{requestState: state, holder: &tokenHolder{}}, state
}

func tokenBody(token string) *strings.Reader {
	return strings.NewReader(`{"token":"` + token + `"}`)
}

func runCookieSet(p *providerState, token string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	p.serveCookieSet(rec, httptest.NewRequest(http.MethodPost, "/_handlers/set", tokenBody(token)))
	return rec
}

func runCookieAck(p *providerState, token, cookie string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/_handlers/ack", tokenBody(token))
	req.Header.Set("Cookie", cookie)
	rec := httptest.NewRecorder()
	p.serveCookieAck(rec, req)
	return rec
}

func TestCookieAckDoesNotReplacePageLocation(t *testing.T) {
	p, state := newAckFixture(t)
	state.MutateCookie("pending", "v")
	state.SetResponseHeader("X-Keep", "1")
	p.storeCookie("tok", &pendingCookie{name: "theme", value: "dark"})
	if rec := runCookieSet(p, "tok"); rec.Code != http.StatusNoContent {
		t.Fatalf("set status %d", rec.Code)
	}

	rec := runCookieAck(p, "tok", "session=new; theme=dark")
	if rec.Code != http.StatusNoContent {
		t.Fatalf("ack status %d", rec.Code)
	}

	info := state.Info()
	if info.Path != "/dashboard" || info.Method != http.MethodGet || info.Hash != "top" || info.Query.Get("tab") != "2" {
		t.Fatalf("page location replaced: %+v", info)
	}
	if v, _ := state.GetCookie("theme"); v != "dark" {
		t.Fatalf("ack cookies not merged, theme=%q", v)
	}
	if _, ok := cookieMutationsOf(state)["pending"]; !ok {
		t.Fatal("pending cookie mutation was wiped by ack")
	}
	if state.ResponseHeaders().Get("X-Keep") != "1" {
		t.Fatal("response headers wiped by ack")
	}
}

func TestCookieAckRequiresConsumedToken(t *testing.T) {
	p, state := newAckFixture(t)
	p.storeCookie("tok", &pendingCookie{name: "theme", value: "dark"})

	if rec := runCookieAck(p, "tok", "theme=forged"); rec.Code != http.StatusForbidden {
		t.Fatalf("ack before set: status %d, want 403", rec.Code)
	}
	if rec := runCookieAck(p, "never-issued", "theme=forged"); rec.Code != http.StatusForbidden {
		t.Fatalf("ack with unknown token: status %d, want 403", rec.Code)
	}
	if v, _ := state.GetCookie("theme"); v == "forged" {
		t.Fatal("forged cookies were merged")
	}

	runCookieSet(p, "tok")
	if rec := runCookieAck(p, "tok", "theme=dark"); rec.Code != http.StatusNoContent {
		t.Fatalf("ack after set: status %d", rec.Code)
	}
	if rec := runCookieAck(p, "tok", "theme=replay"); rec.Code != http.StatusForbidden {
		t.Fatalf("ack replay: status %d, want 403", rec.Code)
	}
}

func TestCookieSetBodyIsBounded(t *testing.T) {
	p, _ := newAckFixture(t)
	p.storeCookie("tok", &pendingCookie{name: "theme", value: "dark"})

	body := `{"token":"tok","pad":"` + strings.Repeat("a", 4096) + `"}`
	rec := httptest.NewRecorder()
	p.serveCookieSet(rec, httptest.NewRequest(http.MethodPost, "/_handlers/set", strings.NewReader(body)))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("set status %d, want 413", rec.Code)
	}

	rec = httptest.NewRecorder()
	p.serveCookieAck(rec, httptest.NewRequest(http.MethodPost, "/_handlers/ack", strings.NewReader(body)))
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("ack status %d, want 413", rec.Code)
	}
}

func TestRequestInfoRetainsOnlyAllowlistedHeaders(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer secret")
	r.Header.Set("Proxy-Authorization", "secret")
	r.Header.Set("X-Api-Key", "secret")
	r.Header.Set("Cookie", "a=b")
	r.Header.Set("User-Agent", "ua")
	r.Header.Set("Accept-Language", "en")

	info := NewRequestInfo(r)
	for _, name := range []string{"Authorization", "Proxy-Authorization", "X-Api-Key"} {
		if v := info.Headers.Get(name); v != "" {
			t.Errorf("%s retained: %q", name, v)
		}
	}
	for name, want := range map[string]string{"Cookie": "a=b", "User-Agent": "ua", "Accept-Language": "en"} {
		if got := info.Headers.Get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}

	extra := NewRequestInfo(r, "x-api-key")
	if extra.Headers.Get("X-Api-Key") != "secret" {
		t.Error("configured extra header not retained")
	}
	if extra.Headers.Get("Authorization") != "" {
		t.Error("Authorization retained with extras")
	}

	upgraded := NewRequestInfoFromHeaders(r.Header)
	if upgraded.Headers.Get("Authorization") != "" {
		t.Error("upgrade headers retained Authorization")
	}
}

func TestUpgradeCookiesMergeIntoPageRequest(t *testing.T) {
	page := httptest.NewRequest(http.MethodGet, "/inbox?folder=sent&_hash=m1", nil)
	page.Host = "example.com"
	page.Header.Set("Cookie", "session=ssr")
	state := NewRequestState(NewRequestInfo(page))
	state.MutateCookie("draft", "1")
	state.AddResponseHeader("Set-Cookie", "draft=1")

	upgrade := http.Header{"Cookie": {"session=ws; theme=dark"}, "User-Agent": {"ws-agent"}, "Authorization": {"Bearer x"}}
	state.MergeHeaders(upgrade)

	info := state.Info()
	if info.Path != "/inbox" || info.Query.Get("folder") != "sent" || info.Hash != "m1" || info.Host != "example.com" || info.Method != http.MethodGet {
		t.Fatalf("page facts lost on upgrade: %+v", info)
	}
	if v, _ := state.GetCookie("session"); v != "ws" {
		t.Errorf("session cookie = %q, want ws", v)
	}
	if v, _ := state.GetCookie("theme"); v != "dark" {
		t.Errorf("theme cookie = %q, want dark", v)
	}
	if v, _ := state.GetCookie("draft"); v != "1" {
		t.Errorf("pending mutation lost, draft=%q", v)
	}
	if state.ResponseHeaders().Get("Set-Cookie") != "draft=1" {
		t.Error("response headers lost on upgrade")
	}
	if info.Headers.Get("Authorization") != "" {
		t.Error("Authorization retained from upgrade")
	}
	if info.Headers.Get("User-Agent") != "ws-agent" {
		t.Error("allowlisted upgrade header not merged")
	}
}

func TestCookieTokensExpire(t *testing.T) {
	now := time.Unix(1000, 0)
	tokens := newCookieTokens()
	tokens.now = func() time.Time { return now }

	tokens.store("stale", &pendingCookie{name: "a", value: "1"})
	tokens.store("consumed", &pendingCookie{name: "b", value: "2"})
	tokens.consume("consumed")

	now = now.Add(cookieTokenTTL + time.Second)
	if tokens.consume("stale") != nil {
		t.Error("expired pending token was consumable")
	}
	if tokens.acknowledge("consumed") != nil {
		t.Error("expired ack token was accepted")
	}

	tokens.store("fresh", &pendingCookie{name: "c", value: "3"})
	if len(tokens.pending.slots) != 1 || len(tokens.awaiting.slots) != 0 {
		t.Errorf("expired tokens retained: pending=%d awaiting=%d", len(tokens.pending.slots), len(tokens.awaiting.slots))
	}
}

func cookieMutationsOf(s *RequestState) map[string]*cookieMutation {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]*cookieMutation, len(s.cookieMutations))
	for k, v := range s.cookieMutations {
		out[k] = &cookieMutation{value: v.value, deleted: v.deleted}
	}
	return out
}

func TestCookieAckAppliesRecordedMutationNotRequestCookies(t *testing.T) {
	p, state := newAckFixture(t)
	state.MergeHeaders(http.Header{"Cookie": {"session=keep; other=x"}})

	p.storeCookie("tok", &pendingCookie{name: "theme", value: "dark", options: &CookieOptions{Path: "/app"}})
	runCookieSet(p, "tok")

	req := httptest.NewRequest(http.MethodPost, "/_handlers/ack", tokenBody("tok"))
	rec := httptest.NewRecorder()
	p.serveCookieAck(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("ack without Cookie header: status %d", rec.Code)
	}
	for name, want := range map[string]string{"session": "keep", "other": "x", "theme": "dark"} {
		if got, _ := state.GetCookie(name); got != want {
			t.Errorf("cookie %s = %q, want %q", name, got, want)
		}
	}
}

func TestCookieAckAppliesRecordedDelete(t *testing.T) {
	p, state := newAckFixture(t)
	state.MergeHeaders(http.Header{"Cookie": {"session=keep; theme=dark"}})

	p.storeCookie("tok", &pendingCookie{name: "theme", options: &CookieOptions{MaxAge: -1}})
	runCookieSet(p, "tok")
	runCookieAck(p, "tok", "")

	if _, ok := state.Info().GetCookie("theme"); ok {
		t.Error("deleted cookie still present in request cookies")
	}
	if v, _ := state.Info().GetCookie("session"); v != "keep" {
		t.Errorf("unrelated cookie = %q, want keep", v)
	}
}

func TestCookieTokensAreCappedAndLinear(t *testing.T) {
	now := time.Unix(1000, 0)
	tokens := newCookieTokens()
	tokens.now = func() time.Time { return now }

	for i := 0; i < 100000; i++ {
		token := "t" + strconv.Itoa(i)
		tokens.store(token, &pendingCookie{name: "a", value: "1"})
		if i%2 == 0 {
			tokens.consume(token)
		}
		now = now.Add(time.Millisecond)
	}

	if n := len(tokens.pending.slots); n > cookieTokenCap {
		t.Errorf("pending tokens = %d, cap %d", n, cookieTokenCap)
	}
	if n := len(tokens.awaiting.slots); n > cookieTokenCap {
		t.Errorf("awaiting tokens = %d, cap %d", n, cookieTokenCap)
	}
	if n := len(tokens.pending.order) + len(tokens.awaiting.order); n > 10*cookieTokenCap {
		t.Errorf("queues grew to %d entries", n)
	}
	if tokens.consume("t1") != nil {
		t.Error("evicted token was consumable")
	}
	if tokens.consume("t99999") == nil {
		t.Error("newest token was evicted")
	}
}

func TestRequestStateInfoReturnsCopy(t *testing.T) {
	state := NewRequestState(NewRequestInfo(httptest.NewRequest(http.MethodGet, "/p?q=1", nil)))
	info := state.Info()
	info.Path = "/mutated"
	info.Query.Set("q", "mutated")
	info.Headers.Set("Cookie", "x=y")

	if state.URL().Path != "/p" || state.URL().Query.Get("q") != "1" {
		t.Fatal("mutating Info() result changed shared state")
	}
	if _, ok := state.GetCookie("x"); ok {
		t.Fatal("mutating Info() headers changed shared state")
	}
}

func TestRetainedHeadersSizedToMatches(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("X-Other", "1")
	if n := len(NewRequestInfo(r).Headers); n != 0 {
		t.Fatalf("headers = %d, want 0", n)
	}
	allocs := testing.AllocsPerRun(100, func() { retainedHeaders(r.Header, defaultRetained) })
	if allocs > 1 {
		t.Errorf("allocs with no matches = %v, want <= 1", allocs)
	}
}

func TestApplyCookiePreservesUnparseablePairs(t *testing.T) {
	for _, bad := range []string{"flag", `j={"a":1}`, "=x", ";;"} {
		state := NewRequestState(&RequestInfo{Headers: http.Header{"Cookie": {"keep=1; " + bad + "; other=2"}}})

		state.ApplyCookie(&http.Cookie{Name: "theme", Value: "dark"}, false)

		for name, want := range map[string]string{"keep": "1", "other": "2", "theme": "dark"} {
			if got, _ := state.GetCookie(name); got != want {
				t.Errorf("%q: cookie %s = %q, want %q", bad, name, got, want)
			}
		}
		if bad != ";;" && !strings.Contains(state.Info().Headers.Get("Cookie"), bad) {
			t.Errorf("%q: unparseable pair not preserved in %q", bad, state.Info().Headers.Get("Cookie"))
		}
	}
}

func TestApplyCookieReplacesAndDeletesByName(t *testing.T) {
	state := NewRequestState(&RequestInfo{Headers: http.Header{"Cookie": {"a=1; theme=old; b=2"}}})
	state.ApplyCookie(&http.Cookie{Name: "theme", Value: "new"}, false)
	if got, _ := state.GetCookie("theme"); got != "new" {
		t.Errorf("theme = %q, want new", got)
	}
	state.ApplyCookie(&http.Cookie{Name: "theme"}, true)
	if _, ok := state.GetCookie("theme"); ok {
		t.Error("theme still present after delete")
	}
	if a, _ := state.GetCookie("a"); a != "1" {
		t.Errorf("a = %q", a)
	}
}

func TestApplyCookieRecordsWhatTheBrowserReceives(t *testing.T) {
	p, state := newAckFixture(t)
	p.storeCookie("tok", &pendingCookie{name: "theme", value: "x; admin=1"})
	set := runCookieSet(p, "tok")
	if set.Code != http.StatusNoContent {
		t.Fatalf("set status %d", set.Code)
	}
	runCookieAck(p, "tok", "")

	if _, ok := state.Info().GetCookie("admin"); ok {
		t.Fatal("value injected a second cookie into request state")
	}

	sent := (&http.Response{Header: set.Header()}).Cookies()
	if len(sent) != 1 {
		t.Fatalf("browser received %d cookies", len(sent))
	}
	got, _ := state.Info().GetCookie("theme")
	if got != sent[0].Value {
		t.Errorf("recorded theme %q, browser received %q", got, sent[0].Value)
	}
}

func TestNewRequestStateCopiesInfo(t *testing.T) {
	info := NewRequestInfo(httptest.NewRequest(http.MethodGet, "/p?q=1", nil))
	state := NewRequestState(info)
	info.Path = "/mutated"
	info.Query.Set("q", "mutated")
	if state.URL().Path != "/p" || state.URL().Query.Get("q") != "1" {
		t.Fatal("state shares the caller's RequestInfo")
	}
}

func TestCookieTokenEvictionReportsDiagnostic(t *testing.T) {
	bus := protocol.NewBus()
	got := make(chan protocol.Diagnostic, 4)
	bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) { got <- d })

	p := &providerState{holder: &tokenHolder{onEvict: evictionReporter(forwardTo(bus))}}
	for i := 0; i < cookieTokenCap; i++ {
		p.storeCookie("t"+strconv.Itoa(i), &pendingCookie{name: "a", value: "1"})
	}
	select {
	case d := <-got:
		t.Fatalf("diagnostic before the cap was exceeded: %+v", d)
	default:
	}

	p.storeCookie("overflow", &pendingCookie{name: "a", value: "1"})
	select {
	case d := <-got:
		if d.Phase != "headers:cookie-token-evicted" || d.Metadata["queue"] != "pending" {
			t.Errorf("unexpected diagnostic %+v", d)
		}
	case <-time.After(time.Second):
		t.Fatal("no diagnostic on eviction")
	}
}

func TestCookieAwaitingAckEvictionReportsDiagnostic(t *testing.T) {
	bus := protocol.NewBus()
	got := make(chan protocol.Diagnostic, 4)
	bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) { got <- d })

	tokens := newCookieTokens()
	tokens.onEvict = evictionReporter(forwardTo(bus))
	for i := 0; i < cookieTokenCap+1; i++ {
		token := "t" + strconv.Itoa(i)
		tokens.store(token, &pendingCookie{name: "a", value: "1"})
		if i < cookieTokenCap {
			tokens.consume(token)
		}
	}
	select {
	case d := <-got:
		t.Fatalf("diagnostic before the cap was exceeded: %+v", d)
	default:
	}

	tokens.consume("t" + strconv.Itoa(cookieTokenCap))
	select {
	case d := <-got:
		if d.Metadata["queue"] != "awaiting-ack" {
			t.Errorf("unexpected diagnostic %+v", d)
		}
	case <-time.After(time.Second):
		t.Fatal("no diagnostic on awaiting-ack eviction")
	}
}

func forwardTo(bus *protocol.Bus) func(runtime.Report) {
	return func(r runtime.Report) {
		bus.ReportDiagnostic(protocol.Diagnostic{Phase: r.Phase, Message: r.Message, StackTrace: r.Stack, Metadata: r.Meta})
	}
}

func TestSameOriginComparesSchemeHostAndPort(t *testing.T) {
	cases := []struct {
		name   string
		target string
		info   RequestInfo
		want   bool
	}{
		{"relative", "/a", RequestInfo{Host: "x.test"}, true},
		{"same host and scheme", "http://x.test/a", RequestInfo{Scheme: "http", Host: "x.test"}, true},
		{"host case and default port", "http://X.test:80/a", RequestInfo{Scheme: "http", Host: "x.test"}, true},
		{"https default port", "https://x.test:443/a", RequestInfo{Scheme: "https", Host: "x.test"}, true},
		{"scheme relative inherits the page scheme", "//x.test/a", RequestInfo{Scheme: "https", Host: "x.test"}, true},
		{"other scheme", "https://x.test/a", RequestInfo{Scheme: "http", Host: "x.test"}, false},
		{"other port", "http://x.test:8080/a", RequestInfo{Scheme: "http", Host: "x.test"}, false},
		{"other port on the page", "http://x.test/a", RequestInfo{Scheme: "http", Host: "x.test:8080"}, false},
		{"other host", "http://y.test/a", RequestInfo{Scheme: "http", Host: "x.test"}, false},
		{"no page host", "http://x.test/a", RequestInfo{Scheme: "http"}, false},
	}
	for _, c := range cases {
		u, err := url.Parse(c.target)
		if err != nil {
			t.Fatal(err)
		}
		if got := sameOrigin(u, &c.info); got != c.want {
			t.Errorf("%s: sameOrigin(%s) = %v, want %v", c.name, c.target, got, c.want)
		}
	}
}
