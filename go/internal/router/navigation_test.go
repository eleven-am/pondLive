package router

import (
	"net/url"
	"testing"
)

func TestResolveHref(t *testing.T) {
	tests := []struct {
		name     string
		base     Location
		href     string
		wantPath string
		wantHash string
	}{
		{
			name:     "absolute path",
			base:     Location{Path: "/current"},
			href:     "/new",
			wantPath: "/new",
		},
		{
			name:     "relative path",
			base:     Location{Path: "/users/123"},
			href:     "./edit",
			wantPath: "/users/edit",
		},
		{
			name:     "parent relative path",
			base:     Location{Path: "/users/123/posts"},
			href:     "../settings",
			wantPath: "/users/settings",
		},
		{
			name:     "hash only",
			base:     Location{Path: "/page"},
			href:     "#section",
			wantPath: "/page",
			wantHash: "section",
		},
		{
			name:     "query params in href",
			base:     Location{Path: "/search"},
			href:     "/results?q=test",
			wantPath: "/results",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := resolveHref(tt.base, tt.href)

			if result.Path != tt.wantPath {
				t.Errorf("path mismatch: got %q, want %q", result.Path, tt.wantPath)
			}

			if tt.wantHash != "" && result.Hash != tt.wantHash {
				t.Errorf("hash mismatch: got %q, want %q", result.Hash, tt.wantHash)
			}
		})
	}
}

func TestResolveHref_WithQuery(t *testing.T) {
	base := Location{
		Path:  "/search",
		Query: url.Values{"existing": []string{"value"}},
	}

	result := resolveHref(base, "/results?new=param")

	if result.Path != "/results" {
		t.Errorf("path mismatch: got %q, want %q", result.Path, "/results")
	}

	if result.Query.Get("new") != "param" {
		t.Errorf("expected new query param")
	}
	if result.Query.Get("existing") != "" {
		t.Errorf("base query should not persist")
	}
}

func TestResolveHref_RelativePaths(t *testing.T) {
	tests := []struct {
		basePath string
		href     string
		want     string
	}{
		{"/users", "./profile", "/profile"},
		{"/users/", "./profile", "/users/profile"},
		{"/users/123", "./edit", "/users/edit"},
		{"/users/123/", "./edit", "/users/123/edit"},
		{"/a/b/c", "../d", "/a/d"},
		{"/a/b/c", "../../d", "/d"},
		{"/a", "../b", "/b"},
		{"/", "./about", "/about"},
	}

	for _, tt := range tests {
		t.Run(tt.basePath+" + "+tt.href, func(t *testing.T) {
			base := Location{Path: tt.basePath}
			result := resolveHref(base, tt.href)

			if result.Path != tt.want {
				t.Errorf("resolveHref(%q, %q) path = %q, want %q",
					tt.basePath, tt.href, result.Path, tt.want)
			}
		})
	}
}
