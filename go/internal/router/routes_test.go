package router

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/work"
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

	entries = collectRouteEntries([]work.Node{}, "/")
	if entries != nil {
		t.Errorf("expected nil for empty input, got %v", entries)
	}
}

func TestCollectSlotEntriesEmpty(t *testing.T) {
	entries := collectSlotEntries(nil, "/")
	if entries != nil {
		t.Errorf("expected nil for nil input, got %v", entries)
	}

	entries = collectSlotEntries([]work.Node{}, "/")
	if entries != nil {
		t.Errorf("expected nil for empty input, got %v", entries)
	}
}

func TestTrimWildcardSuffix(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "/"},
		{"/", "/"},
		{"/app", "/app"},
		{"/app/*", "/app"},
		{"/app/*rest", "/app"},
		{"/users/:id/*", "/users/:id"},
		{"/*", "/"},
		{"/a/b/c/*path", "/a/b/c"},
	}

	for _, tt := range tests {
		result := trimWildcardSuffix(tt.input)
		if result != tt.expected {
			t.Errorf("trimWildcardSuffix(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestJoinRelativePath(t *testing.T) {
	tests := []struct {
		base     string
		rel      string
		expected string
	}{
		{"/", "/", "/"},
		{"/", "/about", "/about"},
		{"/app", "/", "/app"},
		{"/app", "/dashboard", "/app/dashboard"},
		{"/app/*", "/settings", "/app/settings"},
		{"/users/:id/*rest", "/profile", "/users/:id/profile"},
	}

	for _, tt := range tests {
		result := joinRelativePath(tt.base, tt.rel)
		if result != tt.expected {
			t.Errorf("joinRelativePath(%q, %q) = %q, expected %q", tt.base, tt.rel, result, tt.expected)
		}
	}
}

func TestCollectRouteEntriesWithMetadata(t *testing.T) {
	children := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/about",
					component: nil,
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/contact",
					component: nil,
				},
			},
		},
	}

	entries := collectRouteEntries(children, "/")
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].pattern != "/about" {
		t.Errorf("expected first pattern /about, got %s", entries[0].pattern)
	}
	if entries[1].pattern != "/contact" {
		t.Errorf("expected second pattern /contact, got %s", entries[1].pattern)
	}
}

func TestCollectRouteEntriesSkipsSlots(t *testing.T) {
	children := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/home",
					component: nil,
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				slotMetadataKey: slotEntry{
					name: "sidebar",
					routes: []routeEntry{
						{pattern: "/sidebar-item", component: nil},
					},
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/about",
					component: nil,
				},
			},
		},
	}

	entries := collectRouteEntries(children, "/")
	if len(entries) != 2 {
		t.Errorf("expected 2 entries (slots skipped), got %d", len(entries))
	}
	if entries[0].pattern != "/home" {
		t.Errorf("expected first pattern /home, got %s", entries[0].pattern)
	}
	if entries[1].pattern != "/about" {
		t.Errorf("expected second pattern /about, got %s", entries[1].pattern)
	}
}

func TestCollectSlotEntriesWithMetadata(t *testing.T) {
	children := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				slotMetadataKey: slotEntry{
					name: "sidebar",
					routes: []routeEntry{
						{pattern: "/sidebar-item", component: nil},
					},
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				slotMetadataKey: slotEntry{
					name: "modal",
					routes: []routeEntry{
						{pattern: "/modal-content", component: nil},
					},
				},
			},
		},
	}

	entries := collectSlotEntries(children, "/")
	if len(entries) != 2 {
		t.Errorf("expected 2 slot entries, got %d", len(entries))
	}
	if entries[0].name != "sidebar" {
		t.Errorf("expected first slot name 'sidebar', got %s", entries[0].name)
	}
	if entries[1].name != "modal" {
		t.Errorf("expected second slot name 'modal', got %s", entries[1].name)
	}
}

func TestCollectRouteAndSlotSiblings(t *testing.T) {
	children := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/home",
					component: nil,
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/about",
					component: nil,
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				slotMetadataKey: slotEntry{
					name: "sidebar",
					routes: []routeEntry{
						{pattern: "/sidebar-home", component: nil},
						{pattern: "/sidebar-about", component: nil},
					},
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				slotMetadataKey: slotEntry{
					name: "modal",
					routes: []routeEntry{
						{pattern: "/modal-login", component: nil},
					},
				},
			},
		},
	}

	routeEntries := collectRouteEntries(children, "/")
	if len(routeEntries) != 2 {
		t.Errorf("expected 2 route entries, got %d", len(routeEntries))
	}

	slotEntries := collectSlotEntries(children, "/")
	if len(slotEntries) != 2 {
		t.Errorf("expected 2 slot entries, got %d", len(slotEntries))
	}
}

func TestCollectNestedRouteEntries(t *testing.T) {
	children := []work.Node{
		&work.Fragment{
			Children: []work.Node{
				&work.Fragment{
					Metadata: map[string]any{
						routeMetadataKey: routeEntry{
							pattern:   "/nested/route",
							component: nil,
						},
					},
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/top-level",
					component: nil,
				},
			},
		},
	}

	entries := collectRouteEntries(children, "/")
	if len(entries) != 2 {
		t.Errorf("expected 2 entries (including nested), got %d", len(entries))
	}
}

func TestCollectNestedSlotEntries(t *testing.T) {
	children := []work.Node{
		&work.Fragment{
			Children: []work.Node{
				&work.Fragment{
					Metadata: map[string]any{
						slotMetadataKey: slotEntry{
							name: "nested-slot",
							routes: []routeEntry{
								{pattern: "/nested-item", component: nil},
							},
						},
					},
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				slotMetadataKey: slotEntry{
					name: "top-slot",
					routes: []routeEntry{
						{pattern: "/top-item", component: nil},
					},
				},
			},
		},
	}

	entries := collectSlotEntries(children, "/")
	if len(entries) != 2 {
		t.Errorf("expected 2 slot entries (including nested), got %d", len(entries))
	}
}

func TestRoutePatternResolutionWithBase(t *testing.T) {
	children := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "./dashboard",
					component: nil,
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/absolute",
					component: nil,
				},
			},
		},
	}

	entries := collectRouteEntries(children, "/app")
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].pattern != "/app/dashboard" {
		t.Errorf("expected relative pattern resolved to /app/dashboard, got %s", entries[0].pattern)
	}
	if entries[1].pattern != "/absolute" {
		t.Errorf("expected absolute pattern unchanged, got %s", entries[1].pattern)
	}
}

func TestFingerprintSlotsWithEntries(t *testing.T) {
	children := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				slotMetadataKey: slotEntry{
					name: "sidebar",
					routes: []routeEntry{
						{pattern: "/a", component: nil},
						{pattern: "/b", component: nil},
					},
				},
			},
		},
	}

	result := fingerprintSlots(children)
	if result == "" {
		t.Error("expected non-empty fingerprint for slots")
	}
	if result != "sidebar:/a,/b," {
		t.Errorf("unexpected fingerprint: %q", result)
	}
}

func TestFingerprintChildrenWithEntries(t *testing.T) {
	children := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/home",
					component: nil,
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/about",
					component: nil,
				},
			},
		},
	}

	result := fingerprintChildren(children)
	if result != "/home|/about" {
		t.Errorf("expected '/home|/about', got %q", result)
	}
}

func TestTrieStaticBeforeWildcard(t *testing.T) {
	trie := newRouterTrie()

	loginEntry := routeEntry{pattern: "/login", component: nil}
	layoutEntry := routeEntry{pattern: "/*", component: nil}

	trie.Insert("/login", loginEntry)
	trie.Insert("/*", layoutEntry)

	result := trie.Match("/login")
	if result == nil {
		t.Fatal("expected match for /login")
	}
	if result.Entry.pattern != "/login" {
		t.Errorf("expected /login to match static route, got %s", result.Entry.pattern)
	}

	result = trie.Match("/dashboard")
	if result == nil {
		t.Fatal("expected match for /dashboard")
	}
	if result.Entry.pattern != "/*" {
		t.Errorf("expected /dashboard to match wildcard, got %s", result.Entry.pattern)
	}

	result = trie.Match("/")
	if result == nil {
		t.Fatal("expected match for /")
	}
	if result.Entry.pattern != "/*" {
		t.Errorf("expected / to match wildcard, got %s", result.Entry.pattern)
	}
}

func TestTrieWildcardMatchesRoot(t *testing.T) {
	trie := newRouterTrie()

	entry := routeEntry{pattern: "/*", component: nil}
	trie.Insert("/*", entry)

	result := trie.Match("/")
	if result == nil {
		t.Fatal("expected /* to match /")
	}
	if result.Entry.pattern != "/*" {
		t.Errorf("expected pattern /*, got %s", result.Entry.pattern)
	}
	if result.Rest != "/" {
		t.Errorf("expected rest /, got %s", result.Rest)
	}
}

func TestTrieAuthExampleRouting(t *testing.T) {
	trie := newRouterTrie()

	loginEntry := routeEntry{pattern: "/login", component: nil}
	layoutEntry := routeEntry{pattern: "/*", component: nil}

	trie.Insert("/login", loginEntry)
	trie.Insert("/*", layoutEntry)

	tests := []struct {
		path            string
		expectedPattern string
		description     string
	}{
		{"/", "/*", "root should match layout wildcard"},
		{"/login", "/login", "login should match static route, not wildcard"},
		{"/dashboard", "/*", "dashboard should match layout wildcard"},
		{"/settings", "/*", "settings should match layout wildcard"},
	}

	for _, tt := range tests {
		result := trie.Match(tt.path)
		if result == nil {
			t.Errorf("%s: expected match for %s, got nil", tt.description, tt.path)
			continue
		}
		if result.Entry.pattern != tt.expectedPattern {
			t.Errorf("%s: expected %s to match %s, got %s",
				tt.description, tt.path, tt.expectedPattern, result.Entry.pattern)
		}
	}
}

func TestTrieNestedWildcardWithStaticSibling(t *testing.T) {
	trie := newRouterTrie()

	trie.Insert("/auth/login", routeEntry{pattern: "/auth/login", component: nil})
	trie.Insert("/auth/*", routeEntry{pattern: "/auth/*", component: nil})
	trie.Insert("/api/users", routeEntry{pattern: "/api/users", component: nil})

	result := trie.Match("/auth/login")
	if result == nil || result.Entry.pattern != "/auth/login" {
		t.Errorf("/auth/login should match static, got %v", result)
	}

	result = trie.Match("/auth/register")
	if result == nil || result.Entry.pattern != "/auth/*" {
		t.Errorf("/auth/register should match wildcard, got %v", result)
	}

	result = trie.Match("/auth")
	if result != nil && result.Entry.pattern == "/auth/*" {
		t.Logf("/auth matches wildcard as expected (rest=%s)", result.Rest)
	}
}
