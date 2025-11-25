package router

import (
	"net/url"
	"testing"
)

func TestCanonicalizeLocation(t *testing.T) {
	tests := []struct {
		name     string
		input    *Location
		wantPath string
		wantHash string
	}{
		{
			name:     "nil input",
			input:    nil,
			wantPath: "/",
			wantHash: "",
		},
		{
			name:     "empty path",
			input:    &Location{Path: ""},
			wantPath: "/",
			wantHash: "",
		},
		{
			name:     "simple path",
			input:    &Location{Path: "/about"},
			wantPath: "/about",
			wantHash: "",
		},
		{
			name:     "path with hash",
			input:    &Location{Path: "/about", Hash: "#section"},
			wantPath: "/about",
			wantHash: "section",
		},
		{
			name:     "hash without prefix",
			input:    &Location{Path: "/about", Hash: "section"},
			wantPath: "/about",
			wantHash: "section",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := canonicalizeLocation(tt.input)
			if result.Path != tt.wantPath {
				t.Errorf("canonicalizeLocation() path = %q, want %q", result.Path, tt.wantPath)
			}
			if result.Hash != tt.wantHash {
				t.Errorf("canonicalizeLocation() hash = %q, want %q", result.Hash, tt.wantHash)
			}
		})
	}
}

func TestCloneLocation(t *testing.T) {
	original := &Location{
		Path:  "/users",
		Query: url.Values{"id": []string{"123"}},
		Hash:  "top",
	}

	cloned := cloneLocation(original)

	if cloned.Path != original.Path {
		t.Errorf("cloned path = %q, want %q", cloned.Path, original.Path)
	}

	original.Query["id"] = []string{"456"}
	if cloned.Query.Get("id") != "123" {
		t.Error("clone query was mutated when original changed")
	}
}

func TestCloneLocationNil(t *testing.T) {
	cloned := cloneLocation(nil)
	if cloned.Path != "/" {
		t.Errorf("cloneLocation(nil) path = %q, want /", cloned.Path)
	}
}

func TestLocationEqual(t *testing.T) {
	tests := []struct {
		name string
		a    *Location
		b    *Location
		want bool
	}{
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "a nil",
			a:    nil,
			b:    &Location{Path: "/"},
			want: false,
		},
		{
			name: "equal paths",
			a:    &Location{Path: "/about"},
			b:    &Location{Path: "/about"},
			want: true,
		},
		{
			name: "different paths",
			a:    &Location{Path: "/about"},
			b:    &Location{Path: "/home"},
			want: false,
		},
		{
			name: "same path different hash",
			a:    &Location{Path: "/about", Hash: "section1"},
			b:    &Location{Path: "/about", Hash: "section2"},
			want: false,
		},
		{
			name: "same path different query",
			a:    &Location{Path: "/search", Query: url.Values{"q": []string{"foo"}}},
			b:    &Location{Path: "/search", Query: url.Values{"q": []string{"bar"}}},
			want: false,
		},
		{
			name: "fully equal",
			a:    &Location{Path: "/search", Query: url.Values{"q": []string{"foo"}}, Hash: "results"},
			b:    &Location{Path: "/search", Query: url.Values{"q": []string{"foo"}}, Hash: "results"},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := locationEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("locationEqual() = %v, want %v", got, tt.want)
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
			path: "/about",
			want: "/about",
		},
		{
			name: "empty path",
			path: "",
			want: "/",
		},
		{
			name:  "path with query",
			path:  "/search",
			query: url.Values{"q": []string{"test"}},
			want:  "/search?q=test",
		},
		{
			name: "path with hash",
			path: "/about",
			hash: "section",
			want: "/about#section",
		},
		{
			name: "path with hash prefix",
			path: "/about",
			hash: "#section",
			want: "/about#section",
		},
		{
			name:  "path with query and hash",
			path:  "/search",
			query: url.Values{"q": []string{"test"}},
			hash:  "results",
			want:  "/search?q=test#results",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildHref(tt.path, tt.query, tt.hash)
			if got != tt.want {
				t.Errorf("buildHref() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveHref(t *testing.T) {
	current := &Location{
		Path:  "/users/123",
		Query: url.Values{"tab": []string{"posts"}},
		Hash:  "recent",
	}

	tests := []struct {
		name     string
		current  *Location
		href     string
		wantPath string
		wantHash string
	}{
		{
			name:     "empty href",
			current:  current,
			href:     "",
			wantPath: "/users/123",
		},
		{
			name:     "absolute path",
			current:  current,
			href:     "/about",
			wantPath: "/about",
		},
		{
			name:     "absolute path with query",
			current:  current,
			href:     "/search?q=test",
			wantPath: "/search",
		},
		{
			name:     "hash only",
			current:  current,
			href:     "#section",
			wantPath: "/users/123",
			wantHash: "section",
		},
		{
			name:     "relative ./",
			current:  current,
			href:     "./edit",
			wantPath: "/users/edit",
		},
		{
			name:     "relative ../",
			current:  &Location{Path: "/users/123/posts"},
			href:     "../settings",
			wantPath: "/users/settings",
		},
		{
			name:     "relative ../../",
			current:  &Location{Path: "/users/123/posts/456"},
			href:     "../../comments",
			wantPath: "/users/comments",
		},
		{
			name:     "relative ../../../",
			current:  &Location{Path: "/a/b/c/d/e"},
			href:     "../../../x",
			wantPath: "/a/x",
		},
		{
			name:     "relative ../ to root",
			current:  &Location{Path: "/users/123"},
			href:     "../",
			wantPath: "/",
		},
		{
			name:     "relative ../../ to root",
			current:  &Location{Path: "/users/123/posts"},
			href:     "../../",
			wantPath: "/",
		},
		{
			name:     "relative ./../ mixed",
			current:  &Location{Path: "/a/b/c/d"},
			href:     "./../x",
			wantPath: "/a/b/x",
		},
		{
			name:     "relative beyond root clamps to root",
			current:  &Location{Path: "/a/b"},
			href:     "../../../x",
			wantPath: "/x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveHref(tt.current, tt.href)
			if result.Path != tt.wantPath {
				t.Errorf("resolveHref() path = %q, want %q", result.Path, tt.wantPath)
			}
			if tt.wantHash != "" && result.Hash != tt.wantHash {
				t.Errorf("resolveHref() hash = %q, want %q", result.Hash, tt.wantHash)
			}
		})
	}
}

func TestCloneValues(t *testing.T) {
	original := url.Values{
		"a": []string{"1", "2"},
		"b": []string{"3"},
	}

	cloned := cloneValues(original)

	if cloned.Get("a") != "1" {
		t.Errorf("cloned[a] = %q, want 1", cloned.Get("a"))
	}

	original["a"] = []string{"changed"}
	if cloned.Get("a") != "1" {
		t.Error("clone was mutated when original changed")
	}
}

func TestCloneValuesEmpty(t *testing.T) {
	cloned := cloneValues(nil)
	if cloned == nil {
		t.Error("cloneValues(nil) should return empty Values, not nil")
	}
	if len(cloned) != 0 {
		t.Error("cloneValues(nil) should return empty Values")
	}
}

func TestValuesEqual(t *testing.T) {
	tests := []struct {
		name string
		a    url.Values
		b    url.Values
		want bool
	}{
		{
			name: "both empty",
			a:    url.Values{},
			b:    url.Values{},
			want: true,
		},
		{
			name: "equal single value",
			a:    url.Values{"q": []string{"test"}},
			b:    url.Values{"q": []string{"test"}},
			want: true,
		},
		{
			name: "different values",
			a:    url.Values{"q": []string{"foo"}},
			b:    url.Values{"q": []string{"bar"}},
			want: false,
		},
		{
			name: "different keys",
			a:    url.Values{"a": []string{"1"}},
			b:    url.Values{"b": []string{"1"}},
			want: false,
		},
		{
			name: "different lengths",
			a:    url.Values{"a": []string{"1"}},
			b:    url.Values{"a": []string{"1"}, "b": []string{"2"}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := valuesEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("valuesEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMatchesPrefix(t *testing.T) {
	tests := []struct {
		current string
		target  string
		want    bool
	}{
		{"/", "/", true},
		{"/about", "/", false},
		{"/users", "/users", true},
		{"/users/123", "/users", true},
		{"/users123", "/users", false},
		{"/user", "/users", false},
		{"/about", "/users", false},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_"+tt.target, func(t *testing.T) {
			if got := matchesPrefix(tt.current, tt.target); got != tt.want {
				t.Errorf("matchesPrefix(%q, %q) = %v, want %v", tt.current, tt.target, got, tt.want)
			}
		})
	}
}
