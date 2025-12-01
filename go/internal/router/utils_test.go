package router

import (
	"net/url"
	"testing"
)

func TestBuildHref(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		query    url.Values
		hash     string
		expected string
	}{
		{"empty path", "", nil, "", "/"},
		{"simple path", "/about", nil, "", "/about"},
		{"path with query", "/search", url.Values{"q": {"test"}}, "", "/search?q=test"},
		{"path with hash", "/page", nil, "section", "/page#section"},
		{"path with hash prefix", "/page", nil, "#section", "/page#section"},
		{"full url", "/page", url.Values{"a": {"1"}}, "top", "/page?a=1#top"},
		{"multiple query params", "/search", url.Values{"q": {"test"}, "page": {"1"}}, "", "/search?page=1&q=test"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildHref(tt.path, tt.query, tt.hash)
			if result != tt.expected {
				t.Errorf("buildHref(%q, %v, %q) = %q, expected %q", tt.path, tt.query, tt.hash, result, tt.expected)
			}
		})
	}
}

func TestResolveHref(t *testing.T) {
	tests := []struct {
		name     string
		current  Location
		href     string
		expected Location
	}{
		{
			name:     "empty href returns current",
			current:  Location{Path: "/current", Query: url.Values{"a": {"1"}}},
			href:     "",
			expected: Location{Path: "/current", Query: url.Values{"a": {"1"}}},
		},
		{
			name:     "absolute path",
			current:  Location{Path: "/current"},
			href:     "/new",
			expected: Location{Path: "/new", Query: url.Values{}},
		},
		{
			name:     "absolute path with query",
			current:  Location{Path: "/current"},
			href:     "/new?foo=bar",
			expected: Location{Path: "/new", Query: url.Values{"foo": {"bar"}}},
		},
		{
			name:     "hash only",
			current:  Location{Path: "/current", Query: url.Values{"a": {"1"}}},
			href:     "#section",
			expected: Location{Path: "/current", Query: url.Values{"a": {"1"}}, Hash: "section"},
		},
		{
			name:     "relative path ./",
			current:  Location{Path: "/app/dashboard"},
			href:     "./settings",
			expected: Location{Path: "/app/settings", Query: url.Values{}},
		},
		{
			name:     "relative path ../",
			current:  Location{Path: "/app/dashboard/view"},
			href:     "../settings",
			expected: Location{Path: "/app/settings", Query: url.Values{}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveHref(tt.current, tt.href)
			if result.Path != tt.expected.Path {
				t.Errorf("resolveHref path = %q, expected %q", result.Path, tt.expected.Path)
			}
			if result.Hash != tt.expected.Hash {
				t.Errorf("resolveHref hash = %q, expected %q", result.Hash, tt.expected.Hash)
			}
		})
	}
}

func TestCloneLocation(t *testing.T) {
	original := Location{
		Path:  "/test",
		Query: url.Values{"key": {"value"}},
		Hash:  "section",
	}

	cloned := cloneLocation(original)

	if cloned.Path != original.Path {
		t.Errorf("cloned path = %q, expected %q", cloned.Path, original.Path)
	}
	if cloned.Hash != original.Hash {
		t.Errorf("cloned hash = %q, expected %q", cloned.Hash, original.Hash)
	}

	cloned.Query["key"] = []string{"modified"}
	if original.Query["key"][0] == "modified" {
		t.Error("modifying cloned query affected original")
	}
}

func TestCloneValues(t *testing.T) {
	original := url.Values{"a": {"1", "2"}, "b": {"3"}}
	cloned := cloneValues(original)

	if len(cloned) != len(original) {
		t.Errorf("cloned length = %d, expected %d", len(cloned), len(original))
	}

	cloned["a"] = []string{"modified"}
	if original["a"][0] == "modified" {
		t.Error("modifying cloned values affected original")
	}
}

func TestCloneValuesNil(t *testing.T) {
	cloned := cloneValues(nil)
	if cloned == nil {
		t.Error("expected empty map, got nil")
	}
	if len(cloned) != 0 {
		t.Error("expected empty map")
	}
}

func TestMatchesPrefix(t *testing.T) {
	tests := []struct {
		current  string
		target   string
		expected bool
	}{
		{"/", "/", true},
		{"/about", "/", false},
		{"/app", "/app", true},
		{"/app/dashboard", "/app", true},
		{"/app-other", "/app", false},
		{"/application", "/app", false},
		{"/ap", "/app", false},
		{"/app/settings/profile", "/app/settings", true},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_"+tt.target, func(t *testing.T) {
			result := matchesPrefix(tt.current, tt.target)
			if result != tt.expected {
				t.Errorf("matchesPrefix(%q, %q) = %v, expected %v", tt.current, tt.target, result, tt.expected)
			}
		})
	}
}

func TestCanonicalizeValues(t *testing.T) {
	input := url.Values{"z": {"3"}, "a": {"1"}, "m": {"2"}}
	result := canonicalizeValues(input)

	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}

	if len(keys) != 3 {
		t.Errorf("expected 3 keys, got %d", len(keys))
	}
}

func TestCanonicalizeValuesEmpty(t *testing.T) {
	result := canonicalizeValues(nil)
	if result == nil {
		t.Error("expected empty map, got nil")
	}
	if len(result) != 0 {
		t.Error("expected empty map")
	}
}

func TestCanonicalizeList(t *testing.T) {
	input := []string{"  b  ", "a", "  c"}
	result := canonicalizeList(input)

	if len(result) != 3 {
		t.Errorf("expected 3 items, got %d", len(result))
	}
	if result[0] != "a" {
		t.Errorf("expected first item 'a', got %q", result[0])
	}
	if result[1] != "b" {
		t.Errorf("expected second item 'b', got %q", result[1])
	}
	if result[2] != "c" {
		t.Errorf("expected third item 'c', got %q", result[2])
	}
}

func TestCanonicalizeListEmpty(t *testing.T) {
	result := canonicalizeList(nil)
	if result == nil {
		t.Error("expected empty slice, got nil")
	}
	if len(result) != 0 {
		t.Error("expected empty slice")
	}
}
