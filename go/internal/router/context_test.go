package router

import (
	"net/url"
	"testing"
)

func TestMatchStateEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        *MatchState
		b        *MatchState
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "a nil b not nil",
			a:        nil,
			b:        &MatchState{Matched: true},
			expected: false,
		},
		{
			name:     "a not nil b nil",
			a:        &MatchState{Matched: true},
			b:        nil,
			expected: false,
		},
		{
			name:     "equal simple",
			a:        &MatchState{Matched: true, Pattern: "/users", Path: "/users", Rest: ""},
			b:        &MatchState{Matched: true, Pattern: "/users", Path: "/users", Rest: ""},
			expected: true,
		},
		{
			name:     "different matched",
			a:        &MatchState{Matched: true, Pattern: "/users", Path: "/users"},
			b:        &MatchState{Matched: false, Pattern: "/users", Path: "/users"},
			expected: false,
		},
		{
			name:     "different pattern",
			a:        &MatchState{Matched: true, Pattern: "/users", Path: "/users"},
			b:        &MatchState{Matched: true, Pattern: "/posts", Path: "/users"},
			expected: false,
		},
		{
			name:     "different path",
			a:        &MatchState{Matched: true, Pattern: "/users", Path: "/users"},
			b:        &MatchState{Matched: true, Pattern: "/users", Path: "/posts"},
			expected: false,
		},
		{
			name:     "different rest",
			a:        &MatchState{Matched: true, Pattern: "/users", Path: "/users", Rest: "/123"},
			b:        &MatchState{Matched: true, Pattern: "/users", Path: "/users", Rest: "/456"},
			expected: false,
		},
		{
			name: "equal with params",
			a: &MatchState{
				Matched: true,
				Pattern: "/users/:id",
				Path:    "/users/123",
				Params:  map[string]string{"id": "123"},
			},
			b: &MatchState{
				Matched: true,
				Pattern: "/users/:id",
				Path:    "/users/123",
				Params:  map[string]string{"id": "123"},
			},
			expected: true,
		},
		{
			name: "different params length",
			a: &MatchState{
				Matched: true,
				Pattern: "/users/:id",
				Path:    "/users/123",
				Params:  map[string]string{"id": "123"},
			},
			b: &MatchState{
				Matched: true,
				Pattern: "/users/:id",
				Path:    "/users/123",
				Params:  map[string]string{"id": "123", "extra": "value"},
			},
			expected: false,
		},
		{
			name: "different param value",
			a: &MatchState{
				Matched: true,
				Pattern: "/users/:id",
				Path:    "/users/123",
				Params:  map[string]string{"id": "123"},
			},
			b: &MatchState{
				Matched: true,
				Pattern: "/users/:id",
				Path:    "/users/456",
				Params:  map[string]string{"id": "456"},
			},
			expected: false,
		},
		{
			name: "empty params vs nil",
			a: &MatchState{
				Matched: true,
				Pattern: "/users",
				Path:    "/users",
				Params:  map[string]string{},
			},
			b: &MatchState{
				Matched: true,
				Pattern: "/users",
				Path:    "/users",
				Params:  nil,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchStateEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("matchStateEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestLocationEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        Location
		b        Location
		expected bool
	}{
		{
			name:     "equal empty",
			a:        Location{},
			b:        Location{},
			expected: true,
		},
		{
			name:     "equal simple path",
			a:        Location{Path: "/users"},
			b:        Location{Path: "/users"},
			expected: true,
		},
		{
			name:     "different path",
			a:        Location{Path: "/users"},
			b:        Location{Path: "/posts"},
			expected: false,
		},
		{
			name:     "equal with query",
			a:        Location{Path: "/search", Query: url.Values{"q": {"test"}}},
			b:        Location{Path: "/search", Query: url.Values{"q": {"test"}}},
			expected: true,
		},
		{
			name:     "different query value",
			a:        Location{Path: "/search", Query: url.Values{"q": {"test1"}}},
			b:        Location{Path: "/search", Query: url.Values{"q": {"test2"}}},
			expected: false,
		},
		{
			name:     "equal with hash",
			a:        Location{Path: "/page", Hash: "section"},
			b:        Location{Path: "/page", Hash: "section"},
			expected: true,
		},
		{
			name:     "different hash",
			a:        Location{Path: "/page", Hash: "section1"},
			b:        Location{Path: "/page", Hash: "section2"},
			expected: false,
		},
		{
			name:     "full location equal",
			a:        Location{Path: "/page", Query: url.Values{"a": {"1"}}, Hash: "top"},
			b:        Location{Path: "/page", Query: url.Values{"a": {"1"}}, Hash: "top"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := locationEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("locationEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCanonicalizeLocation(t *testing.T) {
	tests := []struct {
		name     string
		input    Location
		expected Location
	}{
		{
			name:     "empty path becomes root",
			input:    Location{},
			expected: Location{Path: "/", Query: url.Values{}},
		},
		{
			name:     "simple path unchanged",
			input:    Location{Path: "/users"},
			expected: Location{Path: "/users", Query: url.Values{}},
		},
		{
			name:  "trailing slash removed",
			input: Location{Path: "/users/"},
			expected: Location{
				Path:  "/users",
				Query: url.Values{},
			},
		},
		{
			name:  "hash normalized",
			input: Location{Path: "/page", Hash: "#section"},
			expected: Location{
				Path:  "/page",
				Query: url.Values{},
				Hash:  "section",
			},
		},
		{
			name:  "query preserved",
			input: Location{Path: "/search", Query: url.Values{"q": {"test"}}},
			expected: Location{
				Path:  "/search",
				Query: url.Values{"q": {"test"}},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canonicalizeLocation(tt.input)
			if result.Path != tt.expected.Path {
				t.Errorf("path = %q, expected %q", result.Path, tt.expected.Path)
			}
			if result.Hash != tt.expected.Hash {
				t.Errorf("hash = %q, expected %q", result.Hash, tt.expected.Hash)
			}
		})
	}
}

func TestNormalizeHash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"section", "section"},
		{"#section", "section"},
		{"##section", "#section"},
		{"#", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeHash(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeHash(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}
