package router

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/work"
	"net/url"
	"sort"
)

func TestTrieSimpleMatch(t *testing.T) {
	trie := newRouterTrie()

	entry := routeEntry{pattern: "/about", component: nil}
	trie.Insert("/about", entry)

	result := trie.Match("/about")
	if result == nil {
		t.Fatal("expected match for /about")
	}
	if result.Entry.pattern != "/about" {
		t.Errorf("expected pattern /about, got %s", result.Entry.pattern)
	}
}

func TestTrieRootMatch(t *testing.T) {
	trie := newRouterTrie()

	entry := routeEntry{pattern: "/", component: nil}
	trie.Insert("/", entry)

	result := trie.Match("/")
	if result == nil {
		t.Fatal("expected match for /")
	}
	if result.Entry.pattern != "/" {
		t.Errorf("expected pattern /, got %s", result.Entry.pattern)
	}
}

func TestTrieParamExtraction(t *testing.T) {
	trie := newRouterTrie()

	entry := routeEntry{pattern: "/users/:id", component: nil}
	trie.Insert("/users/:id", entry)

	result := trie.Match("/users/123")
	if result == nil {
		t.Fatal("expected match for /users/123")
	}
	if result.Entry.pattern != "/users/:id" {
		t.Errorf("expected pattern /users/:id, got %s", result.Entry.pattern)
	}
	if result.Params["id"] != "123" {
		t.Errorf("expected param id=123, got %s", result.Params["id"])
	}
}

func TestTrieMultipleParams(t *testing.T) {
	trie := newRouterTrie()

	entry := routeEntry{pattern: "/users/:userId/posts/:postId", component: nil}
	trie.Insert("/users/:userId/posts/:postId", entry)

	result := trie.Match("/users/42/posts/99")
	if result == nil {
		t.Fatal("expected match for /users/42/posts/99")
	}
	if result.Params["userId"] != "42" {
		t.Errorf("expected param userId=42, got %s", result.Params["userId"])
	}
	if result.Params["postId"] != "99" {
		t.Errorf("expected param postId=99, got %s", result.Params["postId"])
	}
}

func TestTrieWildcard(t *testing.T) {
	trie := newRouterTrie()

	entry := routeEntry{pattern: "/files/*path", component: nil}
	trie.Insert("/files/*path", entry)

	result := trie.Match("/files/documents/report.pdf")
	if result == nil {
		t.Fatal("expected match for /files/documents/report.pdf")
	}
	if result.Params["path"] != "documents/report.pdf" {
		t.Errorf("expected param path=documents/report.pdf, got %s", result.Params["path"])
	}
	if result.Rest != "/documents/report.pdf" {
		t.Errorf("expected rest=/documents/report.pdf, got %s", result.Rest)
	}
}

func TestTrieStaticBeforeParam(t *testing.T) {
	trie := newRouterTrie()

	staticEntry := routeEntry{pattern: "/users/new", component: nil}
	paramEntry := routeEntry{pattern: "/users/:id", component: nil}
	trie.Insert("/users/new", staticEntry)
	trie.Insert("/users/:id", paramEntry)

	result := trie.Match("/users/new")
	if result == nil {
		t.Fatal("expected match for /users/new")
	}
	if result.Entry.pattern != "/users/new" {
		t.Errorf("expected static pattern /users/new, got %s", result.Entry.pattern)
	}

	result = trie.Match("/users/123")
	if result == nil {
		t.Fatal("expected match for /users/123")
	}
	if result.Entry.pattern != "/users/:id" {
		t.Errorf("expected param pattern /users/:id, got %s", result.Entry.pattern)
	}
}

func TestTrieNoMatch(t *testing.T) {
	trie := newRouterTrie()

	entry := routeEntry{pattern: "/about", component: nil}
	trie.Insert("/about", entry)

	result := trie.Match("/contact")
	if result != nil {
		t.Error("expected no match for /contact")
	}
}

func TestTrieNestedPaths(t *testing.T) {
	trie := newRouterTrie()

	entry1 := routeEntry{pattern: "/app", component: nil}
	entry2 := routeEntry{pattern: "/app/dashboard", component: nil}
	entry3 := routeEntry{pattern: "/app/settings", component: nil}
	trie.Insert("/app", entry1)
	trie.Insert("/app/dashboard", entry2)
	trie.Insert("/app/settings", entry3)

	result := trie.Match("/app")
	if result == nil || result.Entry.pattern != "/app" {
		t.Error("expected match for /app")
	}

	result = trie.Match("/app/dashboard")
	if result == nil || result.Entry.pattern != "/app/dashboard" {
		t.Error("expected match for /app/dashboard")
	}

	result = trie.Match("/app/settings")
	if result == nil || result.Entry.pattern != "/app/settings" {
		t.Error("expected match for /app/settings")
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{"/", "/"},
		{"about", "/about"},
		{"/about", "/about"},
		{"/about/", "/about"},
		{"//about//", "/about"},
	}

	for _, tt := range tests {
		result := normalizePath(tt.input)
		if result != tt.expected {
			t.Errorf("normalizePath(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestResolveRoutePattern(t *testing.T) {
	tests := []struct {
		raw      string
		base     string
		expected string
	}{
		{"/about", "/", "/about"},
		{"/", "/app", "/"},
		{"./dashboard", "/app", "/app/dashboard"},
		{"./", "/app", "/app"},
		{"./settings", "/app/*", "/app/settings"},
	}

	for _, tt := range tests {
		result := resolveRoutePattern(tt.raw, tt.base)
		if result != tt.expected {
			t.Errorf("resolveRoutePattern(%q, %q) = %q, expected %q", tt.raw, tt.base, result, tt.expected)
		}
	}
}

func TestCopyParams(t *testing.T) {
	original := map[string]string{"id": "123", "name": "test"}
	copied := copyParams(original)

	if copied["id"] != "123" || copied["name"] != "test" {
		t.Error("copied params don't match original")
	}

	copied["id"] = "456"
	if original["id"] != "123" {
		t.Error("modifying copy affected original")
	}
}

func TestCopyParamsNil(t *testing.T) {
	copied := copyParams(nil)
	if copied == nil {
		t.Error("expected empty map, got nil")
	}
	if len(copied) != 0 {
		t.Error("expected empty map")
	}
}

func TestFingerprintChildren(t *testing.T) {
	result := fingerprintChildren(nil)
	if result != "" {
		t.Errorf("expected empty string for nil children, got %q", result)
	}
}

func TestFingerprintSlots(t *testing.T) {
	result := fingerprintSlots(nil)
	if result != "" {
		t.Errorf("expected empty string for nil children, got %q", result)
	}
}

func TestCollectRouteEntriesEmpty(t *testing.T) {
	entries := collectRouteEntries(nil, "/")
	if entries != nil {
		t.Errorf("expected nil for nil input, got %v", entries)
	}
}

func TestCollectSlotEntriesEmpty(t *testing.T) {
	entries := collectSlotEntries(nil, "/")
	if entries != nil {
		t.Errorf("expected nil for nil input, got %v", entries)
	}
}

func TestCollectRouteEntriesNested(t *testing.T) {
	child := Route(nil, RouteProps{Path: "/child"})
	parent := Route(nil, RouteProps{Path: "/parent"}, child)

	entries := collectRouteEntries([]work.Node{parent}, "/")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].fullPath == "" {
		t.Errorf("expected fullPath to be set")
	}
}

func TestCollectRouteEntriesAppendsWildcardForChildren(t *testing.T) {
	child := Route(nil, RouteProps{Path: "/child"})
	parent := Route(nil, RouteProps{Path: "/parent"}, child)

	entries := collectRouteEntries([]work.Node{parent}, "/")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].fullPath != "/parent/*" {
		t.Fatalf("expected fullPath /parent/*, got %s", entries[0].fullPath)
	}
	if len(entries[0].children) != 1 {
		t.Fatalf("expected 1 child route, got %d", len(entries[0].children))
	}
}

func TestCollectSlotEntriesSetsFullPath(t *testing.T) {
	r := Route(nil, RouteProps{Path: "/slot-route"})
	slot := Slot(nil, SlotProps{Name: "main"}, r)

	entries := collectSlotEntries([]work.Node{slot}, "/")
	if len(entries) != 1 {
		t.Fatalf("expected 1 slot entry, got %d", len(entries))
	}
	if len(entries[0].routes) != 1 {
		t.Fatalf("expected 1 route in slot, got %d", len(entries[0].routes))
	}
	if entries[0].routes[0].fullPath != "/slot-route" {
		t.Fatalf("expected fullPath /slot-route, got %s", entries[0].routes[0].fullPath)
	}
}

func TestResolveHrefVariants(t *testing.T) {
	base := Location{Path: "/base/path", Query: url.Values{"q": []string{"1"}}, Hash: "h"}

	abs := resolveHref(base, "/abs?x=1#z")
	if abs.Path != "/abs" || abs.Query.Get("x") != "1" || abs.Hash != "z" {
		t.Fatalf("abs resolve mismatch: %+v", abs)
	}

	hash := resolveHref(base, "#top")
	if hash.Path != "/base/path" || hash.Hash != "top" {
		t.Fatalf("hash resolve mismatch: %+v", hash)
	}

	rel := resolveHref(base, "./child")
	if rel.Path != "/base/child" {
		t.Fatalf("rel resolve mismatch: %+v", rel)
	}

	up := resolveHref(base, "../sibling")
	if up.Path != "/sibling" {
		t.Fatalf("up resolve mismatch: %+v", up)
	}
}

func TestCanonicalizeLocationSortsQuery(t *testing.T) {
	loc := Location{
		Path: "/p",
		Query: url.Values{
			"b": {"2"},
			"a": {"1"},
		},
		Hash: "h",
	}
	canon := canonicalizeLocation(loc)
	keys := []string{}
	for k := range canon.Query {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if keys[0] != "a" || keys[1] != "b" {
		t.Fatalf("expected sorted keys [a b], got %v", keys)
	}
}

func TestBuildHref(t *testing.T) {
	h := buildHref("/p", url.Values{"a": {"1"}}, "hash")
	if h != "/p?a=1#hash" {
		t.Fatalf("expected /p?a=1#hash, got %s", h)
	}
}

func TestMatchesPrefix(t *testing.T) {
	if !matchesPrefix("/users/123", "/users") {
		t.Fatalf("expected /users/123 to match prefix /users")
	}
	if matchesPrefix("/users", "/users/123") {
		t.Fatalf("did not expect /users to match prefix /users/123")
	}
	if matchesPrefix("/users2/123", "/users") {
		t.Fatalf("did not expect /users2/123 to match prefix /users")
	}
}
