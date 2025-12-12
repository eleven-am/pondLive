package headers

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
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

func TestNewRequestInfo_NilURL(t *testing.T) {
	r := &http.Request{
		Method:     "GET",
		Host:       "example.com",
		RemoteAddr: "127.0.0.1:8080",
		URL:        nil,
		Header:     http.Header{"X-Test": {"value"}},
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
	if info.Headers.Get("X-Test") != "value" {
		t.Errorf("Expected header X-Test='value', got %q", info.Headers.Get("X-Test"))
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

	query := state.Query()
	query["key"] = []string{"modified"}
	query["new"] = []string{"added"}

	original := state.Query()
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

	query := state.Query()

	if len(query["tags"]) != 3 {
		t.Errorf("Expected 3 values for tags, got %d", len(query["tags"]))
	}
	if query["tags"][0] != "go" || query["tags"][1] != "rust" || query["tags"][2] != "python" {
		t.Errorf("Query values mismatch: %v", query["tags"])
	}
}

func TestRequestInfoCookies(t *testing.T) {
	headers := http.Header{}
	headers.Set("Cookie", "session=abc123; theme=dark; lang=en")

	info := NewRequestInfoFromHeaders(headers)
	cookies := info.Cookies()

	if len(cookies) != 3 {
		t.Errorf("Expected 3 cookies, got %d", len(cookies))
	}

	found := make(map[string]string)
	for _, c := range cookies {
		found[c.Name] = c.Value
	}

	if found["session"] != "abc123" {
		t.Errorf("Expected session=abc123, got %q", found["session"])
	}
	if found["theme"] != "dark" {
		t.Errorf("Expected theme=dark, got %q", found["theme"])
	}
}

func TestRequestInfoCookies_EmptyCookieHeader(t *testing.T) {
	info := &RequestInfo{
		Headers: http.Header{},
	}

	cookies := info.Cookies()
	if cookies != nil {
		t.Errorf("Expected nil cookies for empty header, got %v", cookies)
	}
}

func TestRequestInfoCookies_NilHeaders(t *testing.T) {
	info := &RequestInfo{
		Headers: nil,
	}

	cookies := info.Cookies()
	if cookies != nil {
		t.Errorf("Expected nil cookies for nil headers, got %v", cookies)
	}
}

func TestRequestStateReplaceInfo(t *testing.T) {
	originalInfo := &RequestInfo{
		Method: "GET",
		Path:   "/original",
	}
	state := NewRequestState(originalInfo)

	state.SetResponseHeader("X-Old", "value")
	state.MutateCookie("old_cookie", "value")

	newInfo := &RequestInfo{
		Method: "POST",
		Path:   "/new",
	}
	state.ReplaceInfo(newInfo)

	if state.Info().Path != "/new" {
		t.Errorf("Expected path '/new', got %q", state.Info().Path)
	}

	if len(state.ResponseHeaders()) != 0 {
		t.Error("Expected response headers to be cleared")
	}

	if len(state.CookieMutations()) != 0 {
		t.Error("Expected cookie mutations to be cleared")
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
	if clone.Path() != "/test" {
		t.Errorf("Expected path '/test', got %q", clone.Path())
	}
	if !clone.IsLive() {
		t.Error("Expected IsLive=true in clone")
	}
	if clone.ResponseHeaders().Get("X-Test") != "value" {
		t.Error("Expected response header in clone")
	}
	if m := clone.CookieMutations(); m["session"] == nil || m["session"].value != "abc" {
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

func TestRequestStateNotifyChange_WithSetState(t *testing.T) {
	state := NewRequestState(nil)
	called := false
	state.setState = func(s *RequestState) {
		called = true
		if s == nil {
			t.Error("setState should receive a clone")
		}
	}

	state.NotifyChange()

	if !called {
		t.Error("Expected setState to be called")
	}
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
	if state.Path() != "" {
		t.Error("Expected empty path for nil info")
	}
}

func TestRequestStateHash_NilInfo(t *testing.T) {
	state := NewRequestState(nil)
	if state.Hash() != "" {
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
	headers.Set("X-Test", "value123")
	info := NewRequestInfoFromHeaders(headers)

	val, ok := info.Get("X-Test")
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
	if info.Headers.Get("Authorization") != "Bearer token123" {
		t.Errorf("Expected Authorization header, got %q", info.Headers.Get("Authorization"))
	}
}

func TestRequestStateReplaceInfo_NilState(t *testing.T) {
	var state *RequestState
	state.ReplaceInfo(nil)
}

func TestRequestStateClone_NilState(t *testing.T) {
	var state *RequestState
	clone := state.Clone()
	if clone != nil {
		t.Error("Expected nil clone from nil state")
	}
}

func TestProviderStateStoreCookie(t *testing.T) {
	pState := &providerState{
		pendingCookiesRef: &runtime.Ref[map[string]*pendingCookie]{Current: make(map[string]*pendingCookie)},
	}

	pState.storeCookie("token123", &pendingCookie{
		name:  "session",
		value: "abc",
	})

	if len(pState.pendingCookiesRef.Current) != 1 {
		t.Errorf("Expected 1 pending cookie, got %d", len(pState.pendingCookiesRef.Current))
	}
	if pState.pendingCookiesRef.Current["token123"].name != "session" {
		t.Error("Cookie not stored correctly")
	}
}

func TestProviderStateConsumeCookie(t *testing.T) {
	pState := &providerState{
		pendingCookiesRef: &runtime.Ref[map[string]*pendingCookie]{Current: make(map[string]*pendingCookie)},
	}

	pState.pendingCookiesRef.Current["token123"] = &pendingCookie{
		name:  "session",
		value: "abc",
	}

	cookie := pState.consumeCookie("token123")
	if cookie == nil {
		t.Fatal("Expected cookie to be returned")
	}
	if cookie.name != "session" || cookie.value != "abc" {
		t.Error("Cookie data incorrect")
	}
	if len(pState.pendingCookiesRef.Current) != 0 {
		t.Error("Cookie should be removed after consume")
	}
}

func TestProviderStateConsumeCookie_NotFound(t *testing.T) {
	pState := &providerState{
		pendingCookiesRef: &runtime.Ref[map[string]*pendingCookie]{Current: make(map[string]*pendingCookie)},
	}

	cookie := pState.consumeCookie("nonexistent")
	if cookie != nil {
		t.Error("Expected nil for non-existent cookie")
	}
}

func TestProviderStateConcurrentAccess(t *testing.T) {
	pState := &providerState{
		pendingCookiesRef: &runtime.Ref[map[string]*pendingCookie]{Current: make(map[string]*pendingCookie)},
	}

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(id int) {
			token := "token" + string(rune('0'+id))
			pState.storeCookie(token, &pendingCookie{name: token, value: "value"})
			pState.consumeCookie(token)
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

func TestUseRequestState(t *testing.T) {
	t.Run("returns nil when no provider", func(t *testing.T) {
		var capturedState *RequestState

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				capturedState = UseRequestState(ctx)
				return &work.Text{Value: "test"}
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedState != nil {
			t.Error("expected nil state when no provider")
		}
	})

	t.Run("returns state when provider exists", func(t *testing.T) {
		var capturedState *RequestState

		inputState := NewRequestState(&RequestInfo{
			Method: "GET",
			Path:   "/test",
		})

		childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			capturedState = UseRequestState(ctx)
			return &work.Text{Value: "test"}
		}

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx, inputState, work.Component(childFn))
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedState == nil {
			t.Fatal("expected non-nil state")
		}
		if capturedState.Path() != "/test" {
			t.Errorf("expected path /test, got %s", capturedState.Path())
		}
	})
}

func TestUseHeaders(t *testing.T) {
	t.Run("returns nil when no provider", func(t *testing.T) {
		var capturedHeaders http.Header

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				capturedHeaders = UseHeaders(ctx)
				return &work.Text{Value: "test"}
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedHeaders != nil {
			t.Error("expected nil headers when no provider")
		}
	})

	t.Run("returns headers clone when provider exists", func(t *testing.T) {
		var capturedHeaders http.Header

		headers := http.Header{}
		headers.Set("X-Test", "value123")
		inputState := NewRequestState(NewRequestInfoFromHeaders(headers))

		childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			capturedHeaders = UseHeaders(ctx)
			return &work.Text{Value: "test"}
		}

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx, inputState, work.Component(childFn))
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedHeaders == nil {
			t.Fatal("expected non-nil headers")
		}
		if capturedHeaders.Get("X-Test") != "value123" {
			t.Errorf("expected header X-Test=value123, got %s", capturedHeaders.Get("X-Test"))
		}
	})
}

func TestUseIsLive(t *testing.T) {
	t.Run("returns false when no provider", func(t *testing.T) {
		var capturedIsLive bool

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				capturedIsLive = UseIsLive(ctx)
				return &work.Text{Value: "test"}
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedIsLive {
			t.Error("expected false when no provider")
		}
	})

	t.Run("returns IsLive state from provider", func(t *testing.T) {
		var capturedIsLive bool

		inputState := NewRequestState(nil)
		inputState.SetIsLive(true)

		childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			capturedIsLive = UseIsLive(ctx)
			return &work.Text{Value: "test"}
		}

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx, inputState, work.Component(childFn))
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if !capturedIsLive {
			t.Error("expected true when provider has IsLive=true")
		}
	})
}

func TestUseCookie(t *testing.T) {
	t.Run("returns empty value and no-op setter when no provider", func(t *testing.T) {
		var capturedValue string
		var capturedSetter func(string, *CookieOptions)

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				capturedValue, capturedSetter = UseCookie(ctx, "session")
				return &work.Text{Value: "test"}
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedValue != "" {
			t.Errorf("expected empty value, got %s", capturedValue)
		}
		if capturedSetter == nil {
			t.Fatal("expected non-nil setter")
		}
		capturedSetter("test", nil)
	})

	t.Run("returns cookie value from provider", func(t *testing.T) {
		var capturedValue string

		headers := http.Header{}
		headers.Set("Cookie", "session=abc123")
		inputState := NewRequestState(NewRequestInfoFromHeaders(headers))

		childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			capturedValue, _ = UseCookie(ctx, "session")
			return &work.Text{Value: "test"}
		}

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx, inputState, work.Component(childFn))
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedValue != "abc123" {
			t.Errorf("expected session=abc123, got %s", capturedValue)
		}
	})

	t.Run("setter mutates cookie in SSR mode", func(t *testing.T) {
		var capturedSetter func(string, *CookieOptions)
		inputState := NewRequestState(nil)

		childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			_, capturedSetter = UseCookie(ctx, "session")
			capturedSetter("newvalue", nil)
			return &work.Text{Value: "test"}
		}

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx, inputState, work.Component(childFn))
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		mutations := inputState.CookieMutations()
		if mutations["session"] == nil || mutations["session"].value != "newvalue" {
			t.Error("expected cookie mutation")
		}
	})

	t.Run("setter deletes cookie with negative MaxAge", func(t *testing.T) {
		var capturedSetter func(string, *CookieOptions)

		headers := http.Header{}
		headers.Set("Cookie", "session=abc123")
		inputState := NewRequestState(NewRequestInfoFromHeaders(headers))

		childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			_, capturedSetter = UseCookie(ctx, "session")
			capturedSetter("", &CookieOptions{MaxAge: -1})
			return &work.Text{Value: "test"}
		}

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx, inputState, work.Component(childFn))
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		mutations := inputState.CookieMutations()
		if mutations["session"] == nil || !mutations["session"].deleted {
			t.Error("expected cookie to be marked deleted")
		}
	})

	t.Run("setter adds Set-Cookie header with options", func(t *testing.T) {
		var capturedSetter func(string, *CookieOptions)
		inputState := NewRequestState(nil)

		childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			_, capturedSetter = UseCookie(ctx, "session")
			capturedSetter("value", &CookieOptions{
				Path:     "/app",
				Domain:   "example.com",
				Secure:   true,
				HttpOnly: true,
				SameSite: http.SameSiteStrictMode,
			})
			return &work.Text{Value: "test"}
		}

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx, inputState, work.Component(childFn))
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		respHeaders := inputState.ResponseHeaders()
		setCookie := respHeaders.Get("Set-Cookie")
		if setCookie == "" {
			t.Fatal("expected Set-Cookie header")
		}
	})
}

func TestUseProviderState(t *testing.T) {
	t.Run("returns nil when no provider", func(t *testing.T) {
		var capturedPState *providerState

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				capturedPState = useProviderState(ctx)
				return &work.Text{Value: "test"}
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedPState != nil {
			t.Error("expected nil provider state")
		}
	})

	t.Run("returns provider state when provider exists", func(t *testing.T) {
		var capturedPState *providerState
		inputState := NewRequestState(nil)

		childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			capturedPState = useProviderState(ctx)
			return &work.Text{Value: "test"}
		}

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx, inputState, work.Component(childFn))
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedPState == nil {
			t.Fatal("expected non-nil provider state")
		}
		if capturedPState.requestState != inputState {
			t.Error("expected same request state")
		}
	})
}

func TestUseCookieSetterInLiveMode(t *testing.T) {
	var capturedSetter func(string, *CookieOptions)
	inputState := NewRequestState(nil)
	inputState.SetIsLive(true)

	childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		_, capturedSetter = UseCookie(ctx, "livecookie")
		return &work.Text{Value: "test"}
	}

	root := &runtime.Instance{
		ID: "root",
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			return Provider(ctx, inputState, work.Component(childFn))
		},
		HookFrame: []runtime.HookSlot{},
		Children:  []*runtime.Instance{},
	}

	sess := &runtime.Session{
		Root:              root,
		Components:        map[string]*runtime.Instance{"root": root},
		Handlers:          make(map[string]work.Handler),
		MountedComponents: make(map[*runtime.Instance]struct{}),
		Bus:               protocol.NewBus(),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if capturedSetter == nil {
		t.Fatal("expected non-nil setter")
	}

	capturedSetter("livevalue", nil)

	mutations := inputState.CookieMutations()
	if mutations["livecookie"] == nil || mutations["livecookie"].value != "livevalue" {
		t.Error("expected cookie mutation even in live mode")
	}
}

func TestRender(t *testing.T) {
	t.Run("returns nil when no provider", func(t *testing.T) {
		var capturedNode work.Node

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				capturedNode = Render(ctx, nil)
				return &work.Text{Value: "test"}
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedNode != nil {
			t.Error("expected nil when no provider")
		}
	})

	t.Run("returns non-nil when provider exists", func(t *testing.T) {
		var capturedNode work.Node
		inputState := NewRequestState(nil)

		childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			capturedNode = Render(ctx)
			return &work.Fragment{Children: []work.Node{capturedNode}}
		}

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx, inputState, work.Component(childFn))
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedNode == nil {
			t.Fatal("expected non-nil node when provider exists")
		}
	})
}
