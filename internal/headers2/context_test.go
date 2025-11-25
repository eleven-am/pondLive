package headers2

import (
	"net/http"
	"testing"
)

// TestRequestStateGetCookieWithMutation tests that mutations are reflected in GetCookie.
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

// TestRequestStateDeleteCookieMutation tests cookie deletion mutation.
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

// TestRequestStateResponseHeaders tests response header accumulation.
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

// TestRequestStateRedirect tests redirect functionality.
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

// TestRequestStateIsLive tests live mode flag.
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

// TestRequestStateCookieMutations tests CookieMutations returns correct data.
func TestRequestStateCookieMutations(t *testing.T) {
	state := NewRequestState(nil)

	state.MutateCookie("auth", "token123")
	state.MutateCookie("theme", "dark")
	state.DeleteCookieMutation("old")

	mutations := state.CookieMutations()

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

// TestRequestInfoClone tests that Clone creates an independent copy.
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

// TestNilSafety tests that nil receivers don't panic.
func TestNilSafety(t *testing.T) {
	var info *RequestInfo
	var state *RequestState

	if _, ok := info.Get("test"); ok {
		t.Error("Expected false for nil info.Get")
	}
	if _, ok := info.GetCookie("test"); ok {
		t.Error("Expected false for nil info.GetCookie")
	}
	if info.Cookies() != nil {
		t.Error("Expected nil for nil info.Cookies")
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
	if state.Path() != "" {
		t.Error("Expected empty for nil state.Path")
	}
	if state.Query() != nil {
		t.Error("Expected nil for nil state.Query")
	}
	if state.Hash() != "" {
		t.Error("Expected empty for nil state.Hash")
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
	if state.CookieMutations() != nil {
		t.Error("Expected nil for nil state.CookieMutations")
	}

	state.SetIsLive(true)
	state.SetResponseHeader("X", "Y")
	state.AddResponseHeader("X", "Z")
	state.SetRedirect("/", 302)
	state.MutateCookie("a", "b")
	state.DeleteCookieMutation("c")
}
