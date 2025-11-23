package router

import (
	"net/url"
	"testing"
)

func TestCanonicalizeLocation(t *testing.T) {
	tests := []struct {
		name     string
		input    Location
		wantPath string
		wantHash string
	}{
		{
			name:     "normalizes path",
			input:    Location{Path: "/users/"},
			wantPath: "/users",
		},
		{
			name:     "normalizes hash",
			input:    Location{Path: "/", Hash: "#section"},
			wantPath: "/",
			wantHash: "section",
		},
		{
			name: "sorts query params",
			input: Location{
				Path: "/search",
				Query: url.Values{
					"z": []string{"last"},
					"a": []string{"first"},
				},
			},
			wantPath: "/search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canonicalizeLocation(tt.input)

			if result.Path != tt.wantPath {
				t.Errorf("path mismatch: got %q, want %q", result.Path, tt.wantPath)
			}

			if tt.wantHash != "" && result.Hash != tt.wantHash {
				t.Errorf("hash mismatch: got %q, want %q", result.Hash, tt.wantHash)
			}
		})
	}
}

func TestCloneLocation(t *testing.T) {
	original := Location{
		Path: "/test",
		Query: url.Values{
			"key": []string{"value"},
		},
		Hash: "section",
	}

	cloned := cloneLocation(original)

	if cloned.Path != original.Path {
		t.Errorf("path mismatch: got %q, want %q", cloned.Path, original.Path)
	}
	if cloned.Hash != original.Hash {
		t.Errorf("hash mismatch: got %q, want %q", cloned.Hash, original.Hash)
	}
	if cloned.Query.Get("key") != "value" {
		t.Errorf("query mismatch: got %q, want %q", cloned.Query.Get("key"), "value")
	}

	cloned.Query.Set("key", "modified")
	if original.Query.Get("key") != "value" {
		t.Error("original was mutated when cloned was modified")
	}
}

func TestLocationEqual(t *testing.T) {
	tests := []struct {
		name  string
		a     Location
		b     Location
		equal bool
	}{
		{
			name:  "identical locations",
			a:     Location{Path: "/test", Hash: "section"},
			b:     Location{Path: "/test", Hash: "section"},
			equal: true,
		},
		{
			name:  "different paths",
			a:     Location{Path: "/test"},
			b:     Location{Path: "/other"},
			equal: false,
		},
		{
			name:  "different hashes",
			a:     Location{Path: "/test", Hash: "a"},
			b:     Location{Path: "/test", Hash: "b"},
			equal: false,
		},
		{
			name: "identical query params",
			a: Location{
				Path:  "/test",
				Query: url.Values{"key": []string{"value"}},
			},
			b: Location{
				Path:  "/test",
				Query: url.Values{"key": []string{"value"}},
			},
			equal: true,
		},
		{
			name: "different query params",
			a: Location{
				Path:  "/test",
				Query: url.Values{"key": []string{"a"}},
			},
			b: Location{
				Path:  "/test",
				Query: url.Values{"key": []string{"b"}},
			},
			equal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := locationEqual(tt.a, tt.b)
			if result != tt.equal {
				t.Errorf("locationEqual() = %v, want %v", result, tt.equal)
			}
		})
	}
}

func TestBuildHref(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		query url.Values
		hash  string
		want  string
	}{
		{
			name: "path only",
			path: "/test",
			want: "/test",
		},
		{
			name: "path with query",
			path: "/search",
			query: url.Values{
				"q": []string{"hello"},
			},
			want: "/search?q=hello",
		},
		{
			name: "path with hash",
			path: "/docs",
			hash: "section",
			want: "/docs#section",
		},
		{
			name: "path with query and hash",
			path: "/search",
			query: url.Values{
				"q": []string{"test"},
			},
			hash: "results",
			want: "/search?q=test#results",
		},
		{
			name: "hash with # prefix",
			path: "/page",
			hash: "#already",
			want: "/page#already",
		},
		{
			name: "empty path defaults to /",
			path: "",
			want: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildHref(tt.path, tt.query, tt.hash)
			if result != tt.want {
				t.Errorf("buildHref() = %q, want %q", result, tt.want)
			}
		})
	}
}

func TestNormalizeHash(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"#section", "section"},
		{"section", "section"},
		{"", ""},
		{"##double", "#double"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := normalizeHash(tt.input)
			if result != tt.want {
				t.Errorf("normalizeHash(%q) = %q, want %q", tt.input, result, tt.want)
			}
		})
	}
}

func TestCloneValues(t *testing.T) {
	original := url.Values{
		"key1": []string{"a", "b"},
		"key2": []string{"c"},
	}

	cloned := cloneValues(original)

	if cloned.Get("key1") != "a" {
		t.Error("cloned values don't match original")
	}

	cloned.Set("key1", "modified")
	if original.Get("key1") != "a" {
		t.Error("original was mutated when cloned was modified")
	}

	empty := cloneValues(url.Values{})
	if len(empty) != 0 {
		t.Error("cloned empty values should be empty")
	}
}

func TestValuesEqual(t *testing.T) {
	tests := []struct {
		name  string
		a     url.Values
		b     url.Values
		equal bool
	}{
		{
			name:  "both empty",
			a:     url.Values{},
			b:     url.Values{},
			equal: true,
		},
		{
			name:  "identical single values",
			a:     url.Values{"key": []string{"value"}},
			b:     url.Values{"key": []string{"value"}},
			equal: true,
		},
		{
			name:  "identical multiple values",
			a:     url.Values{"key": []string{"a", "b"}},
			b:     url.Values{"key": []string{"a", "b"}},
			equal: true,
		},
		{
			name:  "different values",
			a:     url.Values{"key": []string{"a"}},
			b:     url.Values{"key": []string{"b"}},
			equal: false,
		},
		{
			name:  "different keys",
			a:     url.Values{"key1": []string{"value"}},
			b:     url.Values{"key2": []string{"value"}},
			equal: false,
		},
		{
			name:  "different lengths",
			a:     url.Values{"key": []string{"a"}},
			b:     url.Values{"key": []string{"a", "b"}},
			equal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valuesEqual(tt.a, tt.b)
			if result != tt.equal {
				t.Errorf("valuesEqual() = %v, want %v", result, tt.equal)
			}
		})
	}
}
