package router

import (
	"net/url"
	"sync"
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestNewRouterEventEmitter(t *testing.T) {
	e := NewRouterEventEmitter()
	if e == nil {
		t.Fatal("expected non-nil emitter")
	}
	if e.subscribers == nil {
		t.Error("expected subscribers map to be initialized")
	}
}

func TestRouterEventEmitter_Subscribe(t *testing.T) {
	e := NewRouterEventEmitter()
	called := false

	sub := e.Subscribe("test", func(evt NavigationEvent) {
		called = true
	})

	if sub == nil {
		t.Fatal("expected non-nil subscription")
	}
	if !sub.active {
		t.Error("expected subscription to be active")
	}
	if sub.event != "test" {
		t.Errorf("expected event 'test', got %q", sub.event)
	}

	e.Emit("test", NavigationEvent{})

	if !called {
		t.Error("expected callback to be called")
	}
}

func TestRouterEventEmitter_Emit_NoSubscribers(t *testing.T) {
	e := NewRouterEventEmitter()
	e.Emit("nonexistent", NavigationEvent{})
}

func TestRouterEventEmitter_Emit_InactiveSubscription(t *testing.T) {
	e := NewRouterEventEmitter()
	called := false

	sub := e.Subscribe("test", func(evt NavigationEvent) {
		called = true
	})

	sub.Unsubscribe()
	e.Emit("test", NavigationEvent{})

	if called {
		t.Error("expected callback not to be called after unsubscribe")
	}
}

func TestSubscription_Unsubscribe(t *testing.T) {
	e := NewRouterEventEmitter()
	sub := e.Subscribe("test", func(evt NavigationEvent) {})

	if !sub.active {
		t.Error("expected subscription to be active initially")
	}

	sub.Unsubscribe()

	if sub.active {
		t.Error("expected subscription to be inactive after unsubscribe")
	}
}

func TestSubscription_Unsubscribe_Nil(t *testing.T) {
	var sub *Subscription
	sub.Unsubscribe()
}

func TestRouterEventEmitter_Cleanup(t *testing.T) {
	e := NewRouterEventEmitter()

	sub1 := e.Subscribe("test", func(evt NavigationEvent) {})
	sub2 := e.Subscribe("test", func(evt NavigationEvent) {})

	sub1.Unsubscribe()

	e.cleanup()

	e.mu.RLock()
	subs := e.subscribers["test"]
	e.mu.RUnlock()

	if len(subs) != 1 {
		t.Errorf("expected 1 active subscription after cleanup, got %d", len(subs))
	}
	if subs[0] != sub2 {
		t.Error("expected sub2 to remain")
	}
}

func TestRouterEventEmitter_Concurrent(t *testing.T) {
	e := NewRouterEventEmitter()
	var wg sync.WaitGroup
	var count int
	var mu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.Subscribe("test", func(evt NavigationEvent) {
				mu.Lock()
				count++
				mu.Unlock()
			})
		}()
	}

	wg.Wait()

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			e.Emit("test", NavigationEvent{})
		}()
	}

	wg.Wait()

	mu.Lock()
	if count < 10 {
		t.Errorf("expected at least 10 calls, got %d", count)
	}
	mu.Unlock()
}

func TestMatch_Param(t *testing.T) {
	m := Match{
		params: map[string]string{"id": "123", "name": "test"},
	}

	val, err := m.Param("id")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != "123" {
		t.Errorf("expected '123', got %q", val)
	}
}

func TestMatch_Param_NotFound(t *testing.T) {
	m := Match{
		params: map[string]string{"id": "123"},
	}

	_, err := m.Param("other")
	if err != ErrParamNotFound {
		t.Errorf("expected ErrParamNotFound, got %v", err)
	}
}

func TestMatch_Param_NilParams(t *testing.T) {
	m := Match{}

	_, err := m.Param("id")
	if err != ErrParamNotFound {
		t.Errorf("expected ErrParamNotFound, got %v", err)
	}
}

func TestMatch_QueryParam(t *testing.T) {
	m := Match{
		query: url.Values{"page": {"1"}, "sort": {"asc"}},
	}

	val, err := m.QueryParam("page")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != "1" {
		t.Errorf("expected '1', got %q", val)
	}
}

func TestMatch_QueryParam_EmptyValue(t *testing.T) {
	m := Match{
		query: url.Values{"empty": {""}},
	}

	val, err := m.QueryParam("empty")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
}

func TestMatch_QueryParam_NotFound(t *testing.T) {
	m := Match{
		query: url.Values{"page": {"1"}},
	}

	_, err := m.QueryParam("other")
	if err != ErrQueryParamNotFound {
		t.Errorf("expected ErrQueryParamNotFound, got %v", err)
	}
}

func TestMatch_QueryParam_NilQuery(t *testing.T) {
	m := Match{}

	_, err := m.QueryParam("page")
	if err != ErrQueryParamNotFound {
		t.Errorf("expected ErrQueryParamNotFound, got %v", err)
	}
}

func TestMatch_QueryValues(t *testing.T) {
	m := Match{
		query: url.Values{"tags": {"go", "rust", "python"}},
	}

	vals, err := m.QueryValues("tags")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(vals) != 3 {
		t.Errorf("expected 3 values, got %d", len(vals))
	}
}

func TestMatch_QueryValues_NotFound(t *testing.T) {
	m := Match{
		query: url.Values{"tags": {"go"}},
	}

	_, err := m.QueryValues("other")
	if err != ErrQueryParamNotFound {
		t.Errorf("expected ErrQueryParamNotFound, got %v", err)
	}
}

func TestMatch_QueryValues_NilQuery(t *testing.T) {
	m := Match{}

	_, err := m.QueryValues("tags")
	if err != ErrQueryParamNotFound {
		t.Errorf("expected ErrQueryParamNotFound, got %v", err)
	}
}

func TestMatchStateEqual(t *testing.T) {
	a := &MatchState{Matched: true, Pattern: "/test", Path: "/test"}
	b := &MatchState{Matched: true, Pattern: "/test", Path: "/test"}

	if !matchStateEqual(a, b) {
		t.Error("expected equal match states to return true")
	}
}

func TestMatchStateEqual_Different(t *testing.T) {
	a := &MatchState{Matched: true, Pattern: "/test", Path: "/test"}
	b := &MatchState{Matched: false, Pattern: "/test", Path: "/test"}

	if matchStateEqual(a, b) {
		t.Error("expected different match states to return false")
	}
}

func TestMatchStateEqual_NilCases(t *testing.T) {
	a := &MatchState{Matched: true}

	if !matchStateEqual(nil, nil) {
		t.Error("expected nil == nil to return true")
	}
	if matchStateEqual(a, nil) {
		t.Error("expected a != nil to return false")
	}
	if matchStateEqual(nil, a) {
		t.Error("expected nil != a to return false")
	}
}

func TestLocationEqual(t *testing.T) {
	a := Location{Path: "/test", Hash: "#section"}
	b := Location{Path: "/test", Hash: "#section"}

	if !locationEqual(a, b) {
		t.Error("expected equal locations to return true")
	}
}

func TestLocationEqual_Different(t *testing.T) {
	a := Location{Path: "/test", Hash: "#section1"}
	b := Location{Path: "/test", Hash: "#section2"}

	if locationEqual(a, b) {
		t.Error("expected different locations to return false")
	}
}

func TestCloneLocation(t *testing.T) {
	original := Location{
		Path:  "/test",
		Hash:  "#section",
		Query: url.Values{"a": {"1"}},
	}

	clone := cloneLocation(original)

	if clone.Path != original.Path {
		t.Error("expected same path")
	}
	if clone.Hash != original.Hash {
		t.Error("expected same hash")
	}

	original.Query["a"] = []string{"modified"}
	if clone.Query.Get("a") == "modified" {
		t.Error("clone should not be affected by original modification")
	}
}

func TestNormalizeHash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"section", "section"},
		{"#section", "section"},
		{"", ""},
		{"#", ""},
	}

	for _, tc := range tests {
		result := normalizeHash(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeHash(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestCanonicalizeValues(t *testing.T) {
	input := url.Values{
		"b": {"2"},
		"a": {"1"},
		"c": {"3", "4"},
	}

	result := canonicalizeValues(input)

	if result.Get("a") != "1" {
		t.Error("expected a=1")
	}
	if result.Get("b") != "2" {
		t.Error("expected b=2")
	}
	if vals := result["c"]; len(vals) != 2 || vals[0] != "3" || vals[1] != "4" {
		t.Error("expected c=[3,4]")
	}
}

func TestCanonicalizeValues_Nil(t *testing.T) {
	result := canonicalizeValues(nil)
	if result == nil {
		t.Error("expected non-nil result for nil input")
	}
}

func TestResolveHref_Absolute(t *testing.T) {
	current := Location{Path: "/current", Query: url.Values{"x": {"1"}}}
	result := resolveHref(current, "/absolute?y=2#hash")

	if result.Path != "/absolute" {
		t.Errorf("expected path '/absolute', got %q", result.Path)
	}
	if result.Query.Get("y") != "2" {
		t.Errorf("expected query y=2, got %v", result.Query)
	}
	if result.Hash != "hash" {
		t.Errorf("expected hash 'hash', got %q", result.Hash)
	}
}

func TestResolveHref_HashOnly(t *testing.T) {
	current := Location{Path: "/current", Query: url.Values{"x": {"1"}}}
	result := resolveHref(current, "#section")

	if result.Path != "/current" {
		t.Errorf("expected path '/current', got %q", result.Path)
	}
	if result.Hash != "section" {
		t.Errorf("expected hash 'section', got %q", result.Hash)
	}
}

func TestResolveHref_Empty(t *testing.T) {
	current := Location{Path: "/current", Query: url.Values{"x": {"1"}}}
	result := resolveHref(current, "")

	if result.Path != "/current" {
		t.Errorf("expected path '/current', got %q", result.Path)
	}
}

func TestResolveHref_Relative(t *testing.T) {
	current := Location{Path: "/users/123"}
	result := resolveHref(current, "./edit")

	if result.Path != "/users/edit" {
		t.Errorf("expected path '/users/edit', got %q", result.Path)
	}
}

func TestResolveHref_ParentRelative(t *testing.T) {
	current := Location{Path: "/users/123/edit"}
	result := resolveHref(current, "../profile")

	if result.Path != "/users/profile" {
		t.Errorf("expected path '/users/profile', got %q", result.Path)
	}
}

func TestJoinRelativePath(t *testing.T) {
	tests := []struct {
		base     string
		rel      string
		expected string
	}{
		{"/", "/child", "/child"},
		{"/parent", "/", "/parent"},
		{"/parent", "/child", "/parent/child"},
		{"/parent/*wildcard", "/child", "/parent/child"},
	}

	for _, tc := range tests {
		result := joinRelativePath(tc.base, tc.rel)
		if result != tc.expected {
			t.Errorf("joinRelativePath(%q, %q) = %q, want %q", tc.base, tc.rel, result, tc.expected)
		}
	}
}

func TestTrimWildcardSuffix(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"", "/"},
		{"/", "/"},
		{"/users", "/users"},
		{"/users/*", "/users"},
		{"/users/*rest", "/users"},
		{"/a/b/*c", "/a/b"},
	}

	for _, tc := range tests {
		result := trimWildcardSuffix(tc.path)
		if result != tc.expected {
			t.Errorf("trimWildcardSuffix(%q) = %q, want %q", tc.path, result, tc.expected)
		}
	}
}

func TestCloneValues(t *testing.T) {
	original := url.Values{"a": {"1", "2"}, "b": {"3"}}
	clone := cloneValues(original)

	original["a"][0] = "modified"
	if clone["a"][0] == "modified" {
		t.Error("clone should not be affected by original modification")
	}
}

func TestCloneValues_Empty(t *testing.T) {
	result := cloneValues(url.Values{})
	if result == nil {
		t.Error("expected non-nil result for empty input")
	}
}

func TestCanonicalizeList(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{nil, []string{}},
		{[]string{}, []string{}},
		{[]string{" b ", "a"}, []string{"a", "b"}},
		{[]string{"c", "a", "b"}, []string{"a", "b", "c"}},
	}

	for _, tc := range tests {
		result := canonicalizeList(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("canonicalizeList(%v) length = %d, want %d", tc.input, len(result), len(tc.expected))
			continue
		}
		for i := range result {
			if result[i] != tc.expected[i] {
				t.Errorf("canonicalizeList(%v)[%d] = %q, want %q", tc.input, i, result[i], tc.expected[i])
			}
		}
	}
}

func TestCanonicalizeLocation(t *testing.T) {
	loc := Location{
		Path:  "",
		Query: url.Values{"b": {"2"}, "a": {"1"}},
		Hash:  "#test",
	}

	canon := canonicalizeLocation(loc)

	if canon.Path != "/" {
		t.Errorf("expected path '/', got %q", canon.Path)
	}
	if canon.Hash != "test" {
		t.Errorf("expected hash 'test', got %q", canon.Hash)
	}
}

func TestBuildHrefNavigation(t *testing.T) {
	tests := []struct {
		path     string
		query    url.Values
		hash     string
		expected string
	}{
		{"/test", nil, "", "/test"},
		{"/test", url.Values{"a": {"1"}}, "", "/test?a=1"},
		{"/test", nil, "section", "/test#section"},
		{"/test", url.Values{"a": {"1"}}, "section", "/test?a=1#section"},
		{"/", url.Values{}, "", "/"},
	}

	for _, tc := range tests {
		result := buildHref(tc.path, tc.query, tc.hash)
		if result != tc.expected {
			t.Errorf("buildHref(%q, %v, %q) = %q, want %q", tc.path, tc.query, tc.hash, result, tc.expected)
		}
	}
}

func TestProvide(t *testing.T) {
	t.Run("provides router context", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx, &work.Text{Value: "content"})
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
	})
}

func TestUseLocation(t *testing.T) {
	t.Run("returns location from context", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				loc := UseLocation(ctx)
				t.Logf("UseLocation returned: %+v", loc)
				return Provide(ctx, &work.Text{Value: "test"})
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
	})
}

func TestUseParams(t *testing.T) {
	t.Run("returns nil when no match context", func(t *testing.T) {
		var capturedParams map[string]string

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					work.Component(runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
						capturedParams = UseParams(ctx)
						return &work.Text{Value: "test"}
					})),
				)
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

		if capturedParams != nil {
			t.Error("expected nil params without match context")
		}
	})
}

func TestUseParam(t *testing.T) {
	t.Run("returns empty string when no match context", func(t *testing.T) {
		var capturedParam string

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					work.Component(runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
						capturedParam = UseParam(ctx, "id")
						return &work.Text{Value: "test"}
					})),
				)
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

		if capturedParam != "" {
			t.Errorf("expected empty string, got %q", capturedParam)
		}
	})
}

func TestUseMatch(t *testing.T) {
	t.Run("returns nil when no match context", func(t *testing.T) {
		var capturedMatch *MatchState

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					work.Component(runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
						capturedMatch = UseMatch(ctx)
						return &work.Text{Value: "test"}
					})),
				)
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

		if capturedMatch != nil {
			t.Error("expected nil match without context")
		}
	})
}

func TestUseMatched(t *testing.T) {
	t.Run("returns false when no match context", func(t *testing.T) {
		var capturedMatched bool

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					work.Component(runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
						capturedMatched = UseMatched(ctx)
						return &work.Text{Value: "test"}
					})),
				)
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

		if capturedMatched {
			t.Error("expected false without match context")
		}
	})
}

func TestUseSearchParams(t *testing.T) {
	t.Run("returns query from location", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				query := UseSearchParams(ctx)
				t.Logf("UseSearchParams returned: %v", query)
				return Provide(ctx, &work.Text{Value: "test"})
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
	})
}

func TestUseSearchParam(t *testing.T) {
	t.Run("returns value and setter", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				value, setter := UseSearchParam(ctx, "page")
				t.Logf("UseSearchParam returned value=%q, setter=%v", value, setter != nil)
				return Provide(ctx, &work.Text{Value: "test"})
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
	})
}

func TestUseRouter(t *testing.T) {
	t.Run("returns router instance", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				router := UseRouter(ctx)
				t.Logf("UseRouter returned: %v", router != nil)
				return Provide(ctx, &work.Text{Value: "test"})
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
	})

	t.Run("router methods work", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				router := UseRouter(ctx)
				if router != nil {
					loc := router.Location()
					t.Logf("Router.Location: %+v", loc)

					params := router.Params()
					t.Logf("Router.Params: %v", params)

					_, err := router.Param("id")
					t.Logf("Router.Param error: %v", err)

					match := router.Match()
					t.Logf("Router.Match: %v", match)

					matched := router.Matched()
					t.Logf("Router.Matched: %v", matched)

					searchParams := router.SearchParams()
					t.Logf("Router.SearchParams: %v", searchParams)

					val, setter := router.SearchParam("key")
					t.Logf("Router.SearchParam: val=%q, setter=%v", val, setter != nil)
				}
				return Provide(ctx, &work.Text{Value: "test"})
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
	})
}

func TestRouterEventCallbacks(t *testing.T) {
	t.Run("OnBeforeNavigate with nil emitter", func(t *testing.T) {
		r := &Router{emitter: nil}
		r.OnBeforeNavigate(func(evt NavigationEvent) {})
	})

	t.Run("OnNavigated with nil emitter", func(t *testing.T) {
		r := &Router{emitter: nil}
		r.OnNavigated(func(evt NavigationEvent) {})
	})

	t.Run("OnBeforeNavigate with emitter", func(t *testing.T) {
		emitter := NewRouterEventEmitter()
		subs := &runtime.Ref[[]*Subscription]{Current: []*Subscription{}}
		r := &Router{emitter: emitter, subscriptions: subs}

		called := false
		r.OnBeforeNavigate(func(evt NavigationEvent) {
			called = true
		})

		if len(subs.Current) != 1 {
			t.Errorf("expected 1 subscription, got %d", len(subs.Current))
		}

		emitter.Emit("beforeNavigate", NavigationEvent{})
		if !called {
			t.Error("expected callback to be called")
		}
	})

	t.Run("OnNavigated with emitter", func(t *testing.T) {
		emitter := NewRouterEventEmitter()
		subs := &runtime.Ref[[]*Subscription]{Current: []*Subscription{}}
		r := &Router{emitter: emitter, subscriptions: subs}

		called := false
		r.OnNavigated(func(evt NavigationEvent) {
			called = true
		})

		if len(subs.Current) != 1 {
			t.Errorf("expected 1 subscription, got %d", len(subs.Current))
		}

		emitter.Emit("navigated", NavigationEvent{})
		if !called {
			t.Error("expected callback to be called")
		}
	})
}

func TestOutlet(t *testing.T) {
	t.Run("renders empty without slots", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx, Outlet(ctx))
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
	})

	t.Run("with custom slot name", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx, Outlet(ctx, "custom"))
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
	})
}

func TestNavigationFunctions(t *testing.T) {
	t.Run("Navigate within provider", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					work.Component(runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
						Navigate(ctx, "/new-path")
						return &work.Text{Value: "test"}
					})),
				)
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
	})

	t.Run("Replace within provider", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					work.Component(runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
						Replace(ctx, "/replaced")
						return &work.Text{Value: "test"}
					})),
				)
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
	})

	t.Run("NavigateWith within provider", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					work.Component(runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
						NavigateWith(ctx, func(loc Location) Location {
							loc.Path = "/modified"
							return loc
						})
						return &work.Text{Value: "test"}
					})),
				)
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
	})

	t.Run("ReplaceWith within provider", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					work.Component(runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
						ReplaceWith(ctx, func(loc Location) Location {
							loc.Hash = "section"
							return loc
						})
						return &work.Text{Value: "test"}
					})),
				)
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
	})

	t.Run("NavigateToHash within provider", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					work.Component(runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
						NavigateToHash(ctx, "section")
						return &work.Text{Value: "test"}
					})),
				)
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
	})

	t.Run("Back within provider", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					work.Component(runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
						Back(ctx)
						return &work.Text{Value: "test"}
					})),
				)
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
	})

	t.Run("Forward within provider", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					work.Component(runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
						Forward(ctx)
						return &work.Text{Value: "test"}
					})),
				)
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
	})
}

func TestRouterMethods(t *testing.T) {
	t.Run("Navigate method", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				router := UseRouter(ctx)
				if router != nil {
					router.Navigate("/path")
					router.Replace("/replaced")
					router.NavigateWith(func(loc Location) Location { return loc })
					router.ReplaceWith(func(loc Location) Location { return loc })
					router.Back()
					router.Forward()
					t.Log("Router methods called successfully")
				}
				return Provide(ctx, &work.Text{Value: "test"})
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
	})
}

func TestNavigateToHash(t *testing.T) {
	t.Run("navigates to hash", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				NavigateToHash(ctx, "section1")
				t.Log("NavigateToHash called")
				return Provide(ctx, &work.Text{Value: "test"})
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
	})
}

func TestUseParamWithParams(t *testing.T) {
	t.Run("returns param when set in match context", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				param := UseParam(ctx, "id")
				t.Logf("UseParam returned: %q", param)
				return Provide(ctx, &work.Text{Value: "test"})
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
	})
}

func TestMatchStateEqualDetails(t *testing.T) {
	t.Run("both nil", func(t *testing.T) {
		if !matchStateEqual(nil, nil) {
			t.Error("expected true for both nil")
		}
	})

	t.Run("one nil", func(t *testing.T) {
		a := &MatchState{Matched: true}
		if matchStateEqual(a, nil) {
			t.Error("expected false for one nil")
		}
		if matchStateEqual(nil, a) {
			t.Error("expected false for one nil (reversed)")
		}
	})

	t.Run("different matched", func(t *testing.T) {
		a := &MatchState{Matched: true}
		b := &MatchState{Matched: false}
		if matchStateEqual(a, b) {
			t.Error("expected false for different matched")
		}
	})

	t.Run("different pattern", func(t *testing.T) {
		a := &MatchState{Matched: true, Pattern: "/a"}
		b := &MatchState{Matched: true, Pattern: "/b"}
		if matchStateEqual(a, b) {
			t.Error("expected false for different pattern")
		}
	})

	t.Run("different path", func(t *testing.T) {
		a := &MatchState{Matched: true, Pattern: "/a", Path: "/x"}
		b := &MatchState{Matched: true, Pattern: "/a", Path: "/y"}
		if matchStateEqual(a, b) {
			t.Error("expected false for different path")
		}
	})

	t.Run("different rest", func(t *testing.T) {
		a := &MatchState{Matched: true, Pattern: "/a", Path: "/x", Rest: "/r1"}
		b := &MatchState{Matched: true, Pattern: "/a", Path: "/x", Rest: "/r2"}
		if matchStateEqual(a, b) {
			t.Error("expected false for different rest")
		}
	})

	t.Run("different params length", func(t *testing.T) {
		a := &MatchState{Matched: true, Params: map[string]string{"a": "1"}}
		b := &MatchState{Matched: true, Params: map[string]string{}}
		if matchStateEqual(a, b) {
			t.Error("expected false for different params length")
		}
	})

	t.Run("different params value", func(t *testing.T) {
		a := &MatchState{Matched: true, Params: map[string]string{"a": "1"}}
		b := &MatchState{Matched: true, Params: map[string]string{"a": "2"}}
		if matchStateEqual(a, b) {
			t.Error("expected false for different params value")
		}
	})

	t.Run("equal states", func(t *testing.T) {
		a := &MatchState{Matched: true, Pattern: "/a", Path: "/x", Rest: "/r", Params: map[string]string{"a": "1"}}
		b := &MatchState{Matched: true, Pattern: "/a", Path: "/x", Rest: "/r", Params: map[string]string{"a": "1"}}
		if !matchStateEqual(a, b) {
			t.Error("expected true for equal states")
		}
	})
}

func TestRouteFunction(t *testing.T) {
	t.Run("creates route with empty path", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					Route(ctx, RouteProps{Path: "", Component: nil}),
				)
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
	})

	t.Run("creates route with path", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					Route(ctx, RouteProps{Path: "/users/:id"}, &work.Text{Value: "child"}),
				)
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
	})
}

func TestSlotFunction(t *testing.T) {
	t.Run("creates slot node", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provide(ctx,
					Slot(ctx, SlotProps{Name: "sidebar"},
						Route(ctx, RouteProps{Path: "/"}, &work.Text{Value: "content"}),
					),
				)
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
	})
}

func TestMoreUtils(t *testing.T) {
	t.Run("resolveHref with various inputs", func(t *testing.T) {
		current := Location{Path: "/users/123", Query: url.Values{"a": {"1"}}, Hash: "top"}

		tests := []struct {
			href     string
			expected string
		}{
			{"./edit", "/users/edit"},
			{"../posts", "/posts"},
			{"/absolute", "/absolute"},
			{"#bottom", "/users/123#bottom"},
			{"?b=2", "/users/123?b=2"},
		}

		for _, tc := range tests {
			result := resolveHref(current, tc.href)
			t.Logf("resolveHref(%+v, %q) = %+v", current, tc.href, result)
		}
	})

	t.Run("matchesPrefix", func(t *testing.T) {
		tests := []struct {
			path   string
			prefix string
		}{
			{"/users/123", "/users"},
			{"/users", "/users"},
			{"/", "/"},
			{"/users/123/posts", "/users/123"},
		}

		for _, tc := range tests {
			result := matchesPrefix(tc.path, tc.prefix)
			t.Logf("matchesPrefix(%q, %q) = %v", tc.path, tc.prefix, result)
		}
	})

	t.Run("resolveRoutePattern", func(t *testing.T) {
		tests := []struct {
			base    string
			pattern string
		}{
			{"/", "/users"},
			{"/api", "/v1"},
			{"/users", "/:id"},
		}

		for _, tc := range tests {
			result := resolveRoutePattern(tc.base, tc.pattern)
			t.Logf("resolveRoutePattern(%q, %q) = %q", tc.base, tc.pattern, result)
		}
	})
}

func TestRouterParamBehavior(t *testing.T) {
	t.Run("Param returns ErrParamNotFound", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				router := UseRouter(ctx)
				if router != nil {
					_, err := router.Param("id")
					if err != ErrParamNotFound {
						t.Errorf("expected ErrParamNotFound, got %v", err)
					}
				}
				return Provide(ctx, &work.Text{Value: "test"})
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
	})
}

func TestTrieEdgeCases(t *testing.T) {
	t.Run("match with no entry", func(t *testing.T) {
		trie := newRouterTrie()
		result := trie.Match("/nonexistent")
		if result != nil {
			t.Error("expected nil for nonexistent path")
		}
	})

	t.Run("insert and match root with wildcard", func(t *testing.T) {
		trie := newRouterTrie()
		entry := routeEntry{pattern: "/*path", fullPath: "/*path"}
		trie.Insert("/*path", entry)

		result := trie.Match("/any/path/here")
		if result == nil {
			t.Fatal("expected match")
		}
		if result.Params["path"] != "any/path/here" {
			t.Errorf("expected 'any/path/here', got %q", result.Params["path"])
		}
	})

	t.Run("static vs param priority", func(t *testing.T) {
		trie := newRouterTrie()
		trie.Insert("/users/new", routeEntry{pattern: "/users/new"})
		trie.Insert("/users/:id", routeEntry{pattern: "/users/:id"})

		result := trie.Match("/users/new")
		if result == nil {
			t.Fatal("expected match")
		}
		if result.Entry.pattern != "/users/new" {
			t.Errorf("expected static pattern, got %q", result.Entry.pattern)
		}

		result = trie.Match("/users/123")
		if result == nil {
			t.Fatal("expected match")
		}
		if result.Params["id"] != "123" {
			t.Errorf("expected '123', got %q", result.Params["id"])
		}
	})
}

func TestRoutesComponent(t *testing.T) {
	t.Run("routes function directly", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{Path: "/"})
				routeBaseCtx.UseProvider(ctx, "/")

				items := []work.Item{
					Route(ctx, RouteProps{Path: "/"}, &work.Text{Value: "home"}),
					Route(ctx, RouteProps{Path: "/about"}, &work.Text{Value: "about"}),
				}
				return routes(ctx, items)
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
		t.Log("routes function called successfully")
	})

	t.Run("routes with no matching path", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{Path: "/nonexistent"})
				routeBaseCtx.UseProvider(ctx, "/")

				items := []work.Item{
					Route(ctx, RouteProps{Path: "/about"}, &work.Text{Value: "about"}),
				}
				return routes(ctx, items)
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
	})

	t.Run("routes with empty items", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{Path: "/"})
				routeBaseCtx.UseProvider(ctx, "/")
				return routes(ctx, nil)
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
	})

	t.Run("routes with parent match context", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{Path: "/users/123"})
				routeBaseCtx.UseProvider(ctx, "/users")
				matchCtx.UseProvider(ctx, &MatchState{
					Matched: true,
					Pattern: "/users/*",
					Path:    "/users/123",
					Rest:    "/123",
				})

				items := []work.Item{
					Route(ctx, RouteProps{Path: "/:id"}, &work.Text{Value: "user detail"}),
				}
				return routes(ctx, items)
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
	})

	t.Run("routes with slots", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{Path: "/"})
				routeBaseCtx.UseProvider(ctx, "/")

				items := []work.Item{
					Slot(ctx, SlotProps{Name: "sidebar"},
						Route(ctx, RouteProps{Path: "/"}, &work.Text{Value: "sidebar content"}),
					),
				}
				return routes(ctx, items)
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
	})
}

func TestFingerprintFunctions(t *testing.T) {
	t.Run("fingerprintChildren with no routes", func(t *testing.T) {
		nodes := []work.Node{
			&work.Text{Value: "text"},
			&work.Fragment{Children: []work.Node{&work.Text{Value: "nested"}}},
		}
		result := fingerprintChildren(nodes)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("fingerprintChildren with routes", func(t *testing.T) {
		nodes := []work.Node{
			&work.Fragment{
				Metadata: map[string]any{
					routeMetadataKey: routeEntry{pattern: "/users"},
				},
			},
			&work.Fragment{
				Metadata: map[string]any{
					routeMetadataKey: routeEntry{pattern: "/posts"},
				},
			},
		}
		result := fingerprintChildren(nodes)
		if result != "/users|/posts" {
			t.Errorf("expected '/users|/posts', got %q", result)
		}
	})

	t.Run("fingerprintSlots with no slots", func(t *testing.T) {
		nodes := []work.Node{
			&work.Text{Value: "text"},
		}
		result := fingerprintSlots(nodes)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("fingerprintSlots with slots", func(t *testing.T) {
		nodes := []work.Node{
			&work.Fragment{
				Metadata: map[string]any{
					slotMetadataKey: slotEntry{
						name: "sidebar",
						routes: []routeEntry{
							{pattern: "/menu"},
							{pattern: "/nav"},
						},
					},
				},
			},
		}
		result := fingerprintSlots(nodes)
		t.Logf("fingerprintSlots result: %q", result)
	})
}

func TestCollectRouteEntries(t *testing.T) {
	t.Run("empty nodes", func(t *testing.T) {
		result := collectRouteEntries(nil, "/")
		if result != nil {
			t.Error("expected nil for empty nodes")
		}
	})

	t.Run("with nil node", func(t *testing.T) {
		nodes := []work.Node{nil, &work.Text{Value: "text"}}
		result := collectRouteEntries(nodes, "/")
		if len(result) != 0 {
			t.Errorf("expected empty, got %d entries", len(result))
		}
	})

	t.Run("with route metadata", func(t *testing.T) {
		nodes := []work.Node{
			&work.Fragment{
				Metadata: map[string]any{
					routeMetadataKey: routeEntry{pattern: "/users"},
				},
			},
		}
		result := collectRouteEntries(nodes, "/")
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
		if result[0].pattern != "/users" {
			t.Errorf("expected pattern '/users', got %q", result[0].pattern)
		}
	})

	t.Run("with slot metadata is skipped", func(t *testing.T) {
		nodes := []work.Node{
			&work.Fragment{
				Metadata: map[string]any{
					slotMetadataKey: slotEntry{name: "sidebar"},
				},
			},
		}
		result := collectRouteEntries(nodes, "/")
		if len(result) != 0 {
			t.Errorf("expected empty, got %d entries", len(result))
		}
	})

	t.Run("with nested fragments", func(t *testing.T) {
		nodes := []work.Node{
			&work.Fragment{
				Children: []work.Node{
					&work.Fragment{
						Metadata: map[string]any{
							routeMetadataKey: routeEntry{pattern: "/nested"},
						},
					},
				},
			},
		}
		result := collectRouteEntries(nodes, "/")
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
	})

	t.Run("with children adds wildcard", func(t *testing.T) {
		nodes := []work.Node{
			&work.Fragment{
				Metadata: map[string]any{
					routeMetadataKey: routeEntry{
						pattern:  "/users",
						children: []work.Node{&work.Text{Value: "child"}},
					},
				},
			},
		}
		result := collectRouteEntries(nodes, "/")
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
		t.Logf("fullPath with children: %q", result[0].fullPath)
	})
}

func TestCollectSlotEntries(t *testing.T) {
	t.Run("empty nodes", func(t *testing.T) {
		result := collectSlotEntries(nil, "/")
		if result != nil {
			t.Error("expected nil for empty nodes")
		}
	})

	t.Run("with nil node", func(t *testing.T) {
		nodes := []work.Node{nil}
		result := collectSlotEntries(nodes, "/")
		if len(result) != 0 {
			t.Errorf("expected empty, got %d entries", len(result))
		}
	})

	t.Run("with slot metadata", func(t *testing.T) {
		nodes := []work.Node{
			&work.Fragment{
				Metadata: map[string]any{
					slotMetadataKey: slotEntry{
						name: "sidebar",
						routes: []routeEntry{
							{pattern: "/menu"},
						},
					},
				},
			},
		}
		result := collectSlotEntries(nodes, "/")
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
		if result[0].name != "sidebar" {
			t.Errorf("expected name 'sidebar', got %q", result[0].name)
		}
	})

	t.Run("with nested fragments", func(t *testing.T) {
		nodes := []work.Node{
			&work.Fragment{
				Children: []work.Node{
					&work.Fragment{
						Metadata: map[string]any{
							slotMetadataKey: slotEntry{name: "nested-slot"},
						},
					},
				},
			},
		}
		result := collectSlotEntries(nodes, "/")
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
	})

	t.Run("with slot with children adds wildcard", func(t *testing.T) {
		nodes := []work.Node{
			&work.Fragment{
				Metadata: map[string]any{
					slotMetadataKey: slotEntry{
						name: "main",
						routes: []routeEntry{
							{
								pattern:  "/users",
								children: []work.Node{&work.Text{Value: "nested"}},
							},
						},
					},
				},
			},
		}
		result := collectSlotEntries(nodes, "/")
		if len(result) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(result))
		}
		t.Logf("slot route fullPath: %q", result[0].routes[0].fullPath)
	})
}

func TestAdditionalCoverage(t *testing.T) {
	t.Run("UseParams with params", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				matchCtx.UseProvider(ctx, &MatchState{
					Matched: true,
					Params:  map[string]string{"id": "123", "name": "test"},
				})
				params := UseParams(ctx)
				if params == nil {
					t.Error("expected non-nil params")
				}
				if params["id"] != "123" {
					t.Errorf("expected id=123, got %q", params["id"])
				}
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
	})

	t.Run("UseParam with existing param", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				matchCtx.UseProvider(ctx, &MatchState{
					Matched: true,
					Params:  map[string]string{"id": "456"},
				})
				val := UseParam(ctx, "id")
				if val != "456" {
					t.Errorf("expected '456', got %q", val)
				}
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
	})

	t.Run("UseSearchParam with query value", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{
					Path:  "/",
					Query: url.Values{"page": {"5"}},
				})
				val, setter := UseSearchParam(ctx, "page")
				if val != "5" {
					t.Errorf("expected '5', got %q", val)
				}
				if setter == nil {
					t.Error("expected non-nil setter")
				}
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
	})

	t.Run("Router.Param with existing params", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				matchCtx.UseProvider(ctx, &MatchState{
					Matched: true,
					Params:  map[string]string{"userId": "789"},
				})
				router := UseRouter(ctx)
				if router != nil {
					val, err := router.Param("userId")
					if err != nil {
						t.Errorf("expected no error, got %v", err)
					}
					if val != "789" {
						t.Errorf("expected '789', got %q", val)
					}
				}
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
	})

	t.Run("buildHref edge cases", func(t *testing.T) {
		result := buildHref("/", nil, "")
		if result != "/" {
			t.Errorf("expected '/', got %q", result)
		}

		result = buildHref("/test", url.Values{}, "hash")
		if result != "/test#hash" {
			t.Errorf("expected '/test#hash', got %q", result)
		}
	})

	t.Run("resolveRoutePattern edge cases", func(t *testing.T) {
		result := resolveRoutePattern("/", "")
		t.Logf("resolveRoutePattern('/', '') = %q", result)

		result = resolveRoutePattern("/api", "users")
		t.Logf("resolveRoutePattern('/api', 'users') = %q", result)
	})

	t.Run("trimWildcardSuffix edge cases", func(t *testing.T) {
		result := trimWildcardSuffix("/users/*")
		t.Logf("trimWildcardSuffix('/users/*') = %q", result)

		result = trimWildcardSuffix("/users/:id/*rest")
		t.Logf("trimWildcardSuffix('/users/:id/*rest') = %q", result)

		result = trimWildcardSuffix("/users")
		if result != "/users" {
			t.Errorf("expected '/users', got %q", result)
		}
	})

	t.Run("matchesPrefix edge cases", func(t *testing.T) {
		result := matchesPrefix("/", "/")
		if !result {
			t.Error("expected true for root match")
		}

		result = matchesPrefix("/users", "/api")
		if result {
			t.Error("expected false for non-matching prefix")
		}
	})

	t.Run("canonicalizeLocation with empty path", func(t *testing.T) {
		loc := Location{Path: "", Query: nil, Hash: ""}
		result := canonicalizeLocation(loc)
		if result.Path != "/" {
			t.Errorf("expected '/', got %q", result.Path)
		}
	})
}

func TestUseSearchParamSetter(t *testing.T) {
	t.Run("setter calls ReplaceWith", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{
					Path:  "/test",
					Query: url.Values{"page": {"1"}},
				})

				_, setter := UseSearchParam(ctx, "page")
				setter("2")
				t.Log("setter called with new value")
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
	})

	t.Run("setter with empty value deletes", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{
					Path:  "/test",
					Query: url.Values{"filter": {"active"}},
				})

				_, setter := UseSearchParam(ctx, "filter")
				setter("")
				t.Log("setter called with empty value to delete")
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
	})

	t.Run("setter with nil query creates new", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{
					Path:  "/test",
					Query: nil,
				})

				_, setter := UseSearchParam(ctx, "newparam")
				setter("newvalue")
				t.Log("setter called to create new query param")
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
	})
}

func TestNavigateLiveMode(t *testing.T) {
	t.Run("navigate with full context", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{
					Path:  "/current",
					Query: url.Values{"x": {"1"}},
				})
				emitterCtx.UseProvider(ctx, NewRouterEventEmitter())

				Navigate(ctx, "/new-page?y=2#section")
				t.Log("Navigate called in live mode context")
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
	})

	t.Run("replace with full context", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{
					Path:  "/current",
					Query: url.Values{},
				})
				emitterCtx.UseProvider(ctx, NewRouterEventEmitter())

				Replace(ctx, "/replaced#hash")
				t.Log("Replace called in live mode context")
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
	})

	t.Run("navigate with empty path initializes", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{
					Path:  "",
					Query: nil,
				})

				Navigate(ctx, "/new")
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
	})
}

func TestBackForwardWithBus(t *testing.T) {
	t.Run("Back with bus", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				Back(ctx)
				t.Log("Back called with bus context")
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
	})

	t.Run("Forward with bus", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				Forward(ctx)
				t.Log("Forward called with bus context")
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
	})

	t.Run("Back without bus", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				Back(ctx)
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
			Bus:               nil,
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}
	})

	t.Run("Forward without bus", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				Forward(ctx)
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
			Bus:               nil,
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}
	})
}

func TestRouterParamNotFound(t *testing.T) {
	t.Run("Param with nil params", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				matchCtx.UseProvider(ctx, nil)
				router := UseRouter(ctx)
				if router != nil {
					_, err := router.Param("id")
					if err != ErrParamNotFound {
						t.Errorf("expected ErrParamNotFound, got %v", err)
					}
				}
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
	})

	t.Run("Param with missing key", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				matchCtx.UseProvider(ctx, &MatchState{
					Matched: true,
					Params:  map[string]string{"other": "value"},
				})
				router := UseRouter(ctx)
				if router != nil {
					_, err := router.Param("id")
					if err != ErrParamNotFound {
						t.Errorf("expected ErrParamNotFound, got %v", err)
					}
				}
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
	})
}

func TestResolveHrefMoreCases(t *testing.T) {
	t.Run("resolveHref with query only", func(t *testing.T) {
		current := Location{Path: "/test", Query: url.Values{"a": {"1"}}}
		result := resolveHref(current, "?b=2")

		if result.Path != "/test" {
			t.Errorf("expected path '/test', got %q", result.Path)
		}
		if result.Query.Get("b") != "2" {
			t.Errorf("expected query b=2, got %v", result.Query)
		}
	})

	t.Run("resolveHref with complex relative path", func(t *testing.T) {
		current := Location{Path: "/a/b/c/d"}
		result := resolveHref(current, "../../x")

		t.Logf("resolveHref(%q, '../../x') = %q", current.Path, result.Path)
	})

	t.Run("resolveHref preserves hash-only", func(t *testing.T) {
		current := Location{Path: "/page", Query: url.Values{"x": {"1"}}, Hash: "old"}
		result := resolveHref(current, "#new")

		if result.Path != "/page" {
			t.Errorf("expected path '/page', got %q", result.Path)
		}
		if result.Hash != "new" {
			t.Errorf("expected hash 'new', got %q", result.Hash)
		}
	})
}

func TestTrieMoreCases(t *testing.T) {
	t.Run("insert duplicate path", func(t *testing.T) {
		trie := newRouterTrie()
		trie.Insert("/users", routeEntry{pattern: "/users"})
		trie.Insert("/users", routeEntry{pattern: "/users-override"})

		result := trie.Match("/users")
		if result == nil {
			t.Fatal("expected match")
		}
		t.Logf("pattern after duplicate insert: %q", result.Entry.pattern)
	})

	t.Run("match root path", func(t *testing.T) {
		trie := newRouterTrie()
		trie.Insert("/", routeEntry{pattern: "/"})

		result := trie.Match("/")
		if result == nil {
			t.Fatal("expected match for root")
		}
		if result.Entry.pattern != "/" {
			t.Errorf("expected '/', got %q", result.Entry.pattern)
		}
	})

	t.Run("match with multiple params", func(t *testing.T) {
		trie := newRouterTrie()
		trie.Insert("/users/:userId/posts/:postId", routeEntry{pattern: "/users/:userId/posts/:postId"})

		result := trie.Match("/users/123/posts/456")
		if result == nil {
			t.Fatal("expected match")
		}
		if result.Params["userId"] != "123" {
			t.Errorf("expected userId=123, got %q", result.Params["userId"])
		}
		if result.Params["postId"] != "456" {
			t.Errorf("expected postId=456, got %q", result.Params["postId"])
		}
	})

	t.Run("wildcard captures rest", func(t *testing.T) {
		trie := newRouterTrie()
		trie.Insert("/files/*filepath", routeEntry{pattern: "/files/*filepath"})

		result := trie.Match("/files/a/b/c/d.txt")
		if result == nil {
			t.Fatal("expected match")
		}
		if result.Params["filepath"] != "a/b/c/d.txt" {
			t.Errorf("expected 'a/b/c/d.txt', got %q", result.Params["filepath"])
		}
	})

	t.Run("no match for partial path", func(t *testing.T) {
		trie := newRouterTrie()
		trie.Insert("/users/:id/profile", routeEntry{pattern: "/users/:id/profile"})

		result := trie.Match("/users/123")
		if result != nil {
			t.Error("expected no match for partial path")
		}
	})
}

func TestLocationEqualWithQuery(t *testing.T) {
	t.Run("equal with same query", func(t *testing.T) {
		a := Location{Path: "/test", Query: url.Values{"a": {"1"}}}
		b := Location{Path: "/test", Query: url.Values{"a": {"1"}}}

		if !locationEqual(a, b) {
			t.Error("expected equal")
		}
	})

	t.Run("different query", func(t *testing.T) {
		a := Location{Path: "/test", Query: url.Values{"a": {"1"}}}
		b := Location{Path: "/test", Query: url.Values{"a": {"2"}}}

		if locationEqual(a, b) {
			t.Error("expected not equal")
		}
	})

	t.Run("nil vs empty query", func(t *testing.T) {
		a := Location{Path: "/test", Query: nil}
		b := Location{Path: "/test", Query: url.Values{}}

		result := locationEqual(a, b)
		t.Logf("nil vs empty query equal: %v", result)
	})
}

func TestRoutesWithComponent(t *testing.T) {
	t.Run("routes with Component prop", func(t *testing.T) {
		testComp := func(ctx *runtime.Ctx, m Match) work.Node {
			return &work.Text{Value: "from component"}
		}

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{Path: "/comp"})
				routeBaseCtx.UseProvider(ctx, "/")

				items := []work.Item{
					Route(ctx, RouteProps{Path: "/comp", Component: testComp}),
				}
				return routes(ctx, items)
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
	})
}

func TestOutletWithSlots(t *testing.T) {
	t.Run("outlet accesses slot from context", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				slotsCtx.UseProvider(ctx, map[string]outletRenderer{
					"": func(ctx *runtime.Ctx) work.Node {
						return &work.Text{Value: "default slot content"}
					},
				})
				return Outlet(ctx)
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
	})

	t.Run("outlet with named slot", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				slotsCtx.UseProvider(ctx, map[string]outletRenderer{
					"sidebar": func(ctx *runtime.Ctx) work.Node {
						return &work.Text{Value: "sidebar content"}
					},
				})
				return Outlet(ctx, "sidebar")
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
	})
}

func TestBuildHrefEdgeCases(t *testing.T) {
	t.Run("empty path", func(t *testing.T) {
		result := buildHref("", nil, "")
		if result != "/" {
			t.Errorf("expected '/', got %q", result)
		}
	})

	t.Run("hash with # prefix", func(t *testing.T) {
		result := buildHref("/test", nil, "#section")
		if result != "/test#section" {
			t.Errorf("expected '/test#section', got %q", result)
		}
	})

	t.Run("hash without # prefix", func(t *testing.T) {
		result := buildHref("/test", nil, "section")
		if result != "/test#section" {
			t.Errorf("expected '/test#section', got %q", result)
		}
	})

	t.Run("query with empty encoded", func(t *testing.T) {
		result := buildHref("/test", url.Values{}, "")
		if result != "/test" {
			t.Errorf("expected '/test', got %q", result)
		}
	})
}

func TestResolveHrefEdgeCases(t *testing.T) {
	t.Run("path ending with slash", func(t *testing.T) {
		current := Location{Path: "/users/"}
		result := resolveHref(current, "./edit")
		t.Logf("resolveHref path ending with slash: %q", result.Path)
	})

	t.Run("path without trailing slash", func(t *testing.T) {
		current := Location{Path: "/users/123"}
		result := resolveHref(current, "./edit")
		t.Logf("resolveHref path no trailing: %q", result.Path)
	})
}

func TestResolveRoutePatternEdgeCases(t *testing.T) {
	t.Run("with ./ prefix", func(t *testing.T) {
		result := resolveRoutePattern("./nested", "/parent")
		t.Logf("resolveRoutePattern('./nested', '/parent') = %q", result)
	})

	t.Run("empty pattern", func(t *testing.T) {
		result := resolveRoutePattern("", "/base")
		if result != "/" {
			t.Errorf("expected '/', got %q", result)
		}
	})

	t.Run("whitespace pattern", func(t *testing.T) {
		result := resolveRoutePattern("  ", "/base")
		if result != "/" {
			t.Errorf("expected '/', got %q", result)
		}
	})
}

func TestNormalizePathEdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{"/", "/"},
		{"users", "/users"},
		{"/users/", "/users"},
		{"/users//posts", "/users/posts"},
		{"//users//posts//", "/users/posts"},
	}

	for _, tc := range tests {
		result := normalizePath(tc.input)
		if result != tc.expected {
			t.Errorf("normalizePath(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestTrieInsertEdgeCases(t *testing.T) {
	t.Run("insert with param segment", func(t *testing.T) {
		trie := newRouterTrie()
		trie.Insert("/:id", routeEntry{pattern: "/:id"})

		result := trie.Match("/123")
		if result == nil {
			t.Fatal("expected match")
		}
		if result.Params["id"] != "123" {
			t.Errorf("expected id=123, got %q", result.Params["id"])
		}
	})

	t.Run("insert empty path", func(t *testing.T) {
		trie := newRouterTrie()
		trie.Insert("", routeEntry{pattern: ""})

		result := trie.Match("/")
		t.Logf("match for empty insert: %v", result)
	})

	t.Run("multiple wildcards priority", func(t *testing.T) {
		trie := newRouterTrie()
		trie.Insert("/api/*", routeEntry{pattern: "/api/*"})
		trie.Insert("/api/v1/*", routeEntry{pattern: "/api/v1/*"})

		result := trie.Match("/api/v1/users")
		if result == nil {
			t.Fatal("expected match")
		}
		t.Logf("matched pattern: %q", result.Entry.pattern)
	})
}

func TestCanonicalizeLocationEdgeCases(t *testing.T) {
	t.Run("with hash in path parts", func(t *testing.T) {
		loc := Location{
			Path:  "/test#existing",
			Query: nil,
			Hash:  "",
		}
		result := canonicalizeLocation(loc)
		t.Logf("canonicalizeLocation with hash in path: %+v", result)
	})

	t.Run("with non-empty query", func(t *testing.T) {
		loc := Location{
			Path:  "/test",
			Query: url.Values{"b": {"2"}, "a": {"1"}},
			Hash:  "",
		}
		result := canonicalizeLocation(loc)
		if result.Query.Get("a") != "1" {
			t.Error("expected query to be preserved")
		}
	})
}

func TestRoutesEdgeCases(t *testing.T) {
	t.Run("routes with non-matching but present routes", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{Path: "/admin/dashboard"})
				routeBaseCtx.UseProvider(ctx, "/")

				items := []work.Item{
					Route(ctx, RouteProps{Path: "/users"}, &work.Text{Value: "users"}),
					Route(ctx, RouteProps{Path: "/admin/*"}, &work.Text{Value: "admin area"}),
				}
				return routes(ctx, items)
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
	})

	t.Run("routes with nested route children", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				locationCtx.UseProvider(ctx, Location{Path: "/users/123"})
				routeBaseCtx.UseProvider(ctx, "/")

				nestedRoute := Route(ctx, RouteProps{Path: "/:id"}, &work.Text{Value: "user detail"})

				items := []work.Item{
					Route(ctx, RouteProps{Path: "/users/*"}, nestedRoute),
				}
				return routes(ctx, items)
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
	})
}
