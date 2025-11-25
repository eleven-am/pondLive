package router

import (
	"testing"
)

func TestRouterTrie_StaticRoutes(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/", routeEntry{pattern: "/"})
	trie.Insert("/about", routeEntry{pattern: "/about"})
	trie.Insert("/users", routeEntry{pattern: "/users"})
	trie.Insert("/users/profile", routeEntry{pattern: "/users/profile"})

	tests := []struct {
		path        string
		wantPattern string
		wantMatch   bool
	}{
		{"/", "/", true},
		{"/about", "/about", true},
		{"/users", "/users", true},
		{"/users/profile", "/users/profile", true},
		{"/notfound", "", false},
		{"/users/other", "", false},
	}

	for _, tt := range tests {
		result := trie.Match(tt.path)
		if tt.wantMatch {
			if result == nil {
				t.Errorf("Match(%q) = nil, want pattern %q", tt.path, tt.wantPattern)
				continue
			}
			if result.Entry.pattern != tt.wantPattern {
				t.Errorf("Match(%q) pattern = %q, want %q", tt.path, result.Entry.pattern, tt.wantPattern)
			}
		} else {
			if result != nil {
				t.Errorf("Match(%q) = %v, want nil", tt.path, result.Entry.pattern)
			}
		}
	}
}

func TestRouterTrie_ParamRoutes(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/users/:id", routeEntry{pattern: "/users/:id"})
	trie.Insert("/users/:id/posts", routeEntry{pattern: "/users/:id/posts"})
	trie.Insert("/users/:id/posts/:postId", routeEntry{pattern: "/users/:id/posts/:postId"})

	tests := []struct {
		path        string
		wantPattern string
		wantParams  map[string]string
	}{
		{"/users/123", "/users/:id", map[string]string{"id": "123"}},
		{"/users/abc", "/users/:id", map[string]string{"id": "abc"}},
		{"/users/123/posts", "/users/:id/posts", map[string]string{"id": "123"}},
		{"/users/123/posts/456", "/users/:id/posts/:postId", map[string]string{"id": "123", "postId": "456"}},
	}

	for _, tt := range tests {
		result := trie.Match(tt.path)
		if result == nil {
			t.Errorf("Match(%q) = nil, want pattern %q", tt.path, tt.wantPattern)
			continue
		}
		if result.Entry.pattern != tt.wantPattern {
			t.Errorf("Match(%q) pattern = %q, want %q", tt.path, result.Entry.pattern, tt.wantPattern)
		}
		for k, v := range tt.wantParams {
			if result.Params[k] != v {
				t.Errorf("Match(%q) param[%q] = %q, want %q", tt.path, k, result.Params[k], v)
			}
		}
	}
}

func TestRouterTrie_WildcardRoutes(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/files/*path", routeEntry{pattern: "/files/*path"})
	trie.Insert("/api/*rest", routeEntry{pattern: "/api/*rest"})

	tests := []struct {
		path        string
		wantPattern string
		wantParam   string
		wantRest    string
	}{
		{"/files/documents/report.pdf", "/files/*path", "documents/report.pdf", "/documents/report.pdf"},
		{"/files/a/b/c", "/files/*path", "a/b/c", "/a/b/c"},
		{"/api/users/123", "/api/*rest", "users/123", "/users/123"},
	}

	for _, tt := range tests {
		result := trie.Match(tt.path)
		if result == nil {
			t.Errorf("Match(%q) = nil, want pattern %q", tt.path, tt.wantPattern)
			continue
		}
		if result.Entry.pattern != tt.wantPattern {
			t.Errorf("Match(%q) pattern = %q, want %q", tt.path, result.Entry.pattern, tt.wantPattern)
		}
		if result.Rest != tt.wantRest {
			t.Errorf("Match(%q) rest = %q, want %q", tt.path, result.Rest, tt.wantRest)
		}
	}
}

func TestRouterTrie_Priority(t *testing.T) {

	trie := NewRouterTrie()
	trie.Insert("/users/new", routeEntry{pattern: "/users/new"})
	trie.Insert("/users/:id", routeEntry{pattern: "/users/:id"})

	result := trie.Match("/users/new")
	if result == nil {
		t.Fatal("Match(/users/new) = nil, want /users/new")
	}
	if result.Entry.pattern != "/users/new" {
		t.Errorf("Match(/users/new) pattern = %q, want /users/new (static priority)", result.Entry.pattern)
	}

	result = trie.Match("/users/123")
	if result == nil {
		t.Fatal("Match(/users/123) = nil, want /users/:id")
	}
	if result.Entry.pattern != "/users/:id" {
		t.Errorf("Match(/users/123) pattern = %q, want /users/:id", result.Entry.pattern)
	}
}

func TestRouterTrie_EmptyPath(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/", routeEntry{pattern: "/"})

	result := trie.Match("")
	if result == nil {
		t.Fatal("Match('') = nil, want /")
	}
	if result.Entry.pattern != "/" {
		t.Errorf("Match('') pattern = %q, want /", result.Entry.pattern)
	}

	result = trie.Match("/")
	if result == nil {
		t.Fatal("Match('/') = nil, want /")
	}
}

func TestRouterTrie_TrailingSlash(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/about", routeEntry{pattern: "/about"})

	result := trie.Match("/about/")
	if result == nil {
		t.Fatal("Match('/about/') = nil, want /about")
	}
	if result.Entry.pattern != "/about" {
		t.Errorf("Match('/about/') pattern = %q, want /about", result.Entry.pattern)
	}
}

func TestRouterTrie_InsertNormalization(t *testing.T) {
	trie := NewRouterTrie()

	trie.Insert("users", routeEntry{pattern: "users"})
	trie.Insert("posts/", routeEntry{pattern: "posts/"})
	trie.Insert("/comments", routeEntry{pattern: "/comments"})

	tests := []string{"/users", "/posts", "/comments"}
	for _, path := range tests {
		result := trie.Match(path)
		if result == nil {
			t.Errorf("Match(%q) = nil, want match", path)
		}
	}
}

func TestRouterTrie_MultipleParams(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/orgs/:orgId/repos/:repoId/issues/:issueId", routeEntry{pattern: "/orgs/:orgId/repos/:repoId/issues/:issueId"})

	result := trie.Match("/orgs/github/repos/cli/issues/42")
	if result == nil {
		t.Fatal("Match() = nil, want match")
	}
	if result.Entry.pattern != "/orgs/:orgId/repos/:repoId/issues/:issueId" {
		t.Errorf("pattern = %q, want /orgs/:orgId/repos/:repoId/issues/:issueId", result.Entry.pattern)
	}

	expectedParams := map[string]string{
		"orgId":   "github",
		"repoId":  "cli",
		"issueId": "42",
	}
	for k, v := range expectedParams {
		if result.Params[k] != v {
			t.Errorf("param[%q] = %q, want %q", k, result.Params[k], v)
		}
	}
}

func TestRouterTrie_NoMatch(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/users/:id", routeEntry{pattern: "/users/:id"})
	trie.Insert("/posts", routeEntry{pattern: "/posts"})

	tests := []string{
		"/",
		"/users",
		"/users/123/extra",
		"/posts/new",
		"/unknown",
		"/Users/123",
	}

	for _, path := range tests {
		result := trie.Match(path)
		if result != nil {
			t.Errorf("Match(%q) = %v, want nil", path, result.Entry.pattern)
		}
	}
}

func TestRouterTrie_WildcardAtRoot(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/*path", routeEntry{pattern: "/*path"})

	tests := []struct {
		path     string
		wantRest string
	}{
		{"/anything", "/anything"},
		{"/a/b/c/d", "/a/b/c/d"},
	}

	for _, tt := range tests {
		result := trie.Match(tt.path)
		if result == nil {
			t.Errorf("Match(%q) = nil, want match", tt.path)
			continue
		}
		if result.Rest != tt.wantRest {
			t.Errorf("Match(%q) rest = %q, want %q", tt.path, result.Rest, tt.wantRest)
		}
	}

	result := trie.Match("/")
	if result != nil {
		t.Errorf("Match('/') = %v, want nil (wildcard needs content)", result.Entry.pattern)
	}
}

func TestRouterTrie_StaticOverParam(t *testing.T) {

	trie := NewRouterTrie()
	trie.Insert("/users/:id", routeEntry{pattern: "/users/:id"})
	trie.Insert("/users/new", routeEntry{pattern: "/users/new"})
	trie.Insert("/users/settings", routeEntry{pattern: "/users/settings"})

	tests := []struct {
		path        string
		wantPattern string
	}{
		{"/users/new", "/users/new"},
		{"/users/settings", "/users/settings"},
		{"/users/123", "/users/:id"},
		{"/users/abc", "/users/:id"},
		{"/users/NEW", "/users/:id"},
	}

	for _, tt := range tests {
		result := trie.Match(tt.path)
		if result == nil {
			t.Errorf("Match(%q) = nil, want %q", tt.path, tt.wantPattern)
			continue
		}
		if result.Entry.pattern != tt.wantPattern {
			t.Errorf("Match(%q) pattern = %q, want %q", tt.path, result.Entry.pattern, tt.wantPattern)
		}
	}
}

func TestRouterTrie_StaticOverWildcard(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/files/*path", routeEntry{pattern: "/files/*path"})
	trie.Insert("/files/public", routeEntry{pattern: "/files/public"})

	result := trie.Match("/files/public")
	if result == nil {
		t.Fatal("Match(/files/public) = nil")
	}
	if result.Entry.pattern != "/files/public" {
		t.Errorf("Match(/files/public) pattern = %q, want /files/public", result.Entry.pattern)
	}

	result = trie.Match("/files/public/nested")
	if result == nil {
		t.Fatal("Match(/files/public/nested) = nil")
	}
	if result.Entry.pattern != "/files/*path" {
		t.Errorf("Match(/files/public/nested) pattern = %q, want /files/*path", result.Entry.pattern)
	}
}

func TestRouterTrie_ParamOverWildcard(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/api/:version/*rest", routeEntry{pattern: "/api/:version/*rest"})
	trie.Insert("/api/:version", routeEntry{pattern: "/api/:version"})

	result := trie.Match("/api/v1")
	if result == nil {
		t.Fatal("Match(/api/v1) = nil")
	}
	if result.Entry.pattern != "/api/:version" {
		t.Errorf("Match(/api/v1) pattern = %q, want /api/:version", result.Entry.pattern)
	}

	result = trie.Match("/api/v1/users/123")
	if result == nil {
		t.Fatal("Match(/api/v1/users/123) = nil")
	}
	if result.Entry.pattern != "/api/:version/*rest" {
		t.Errorf("Match(/api/v1/users/123) pattern = %q, want /api/:version/*rest", result.Entry.pattern)
	}
}

func TestRouterTrie_SpecialCharactersInPath(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/search/:query", routeEntry{pattern: "/search/:query"})

	tests := []struct {
		path      string
		wantParam string
	}{
		{"/search/hello", "hello"},
		{"/search/hello%20world", "hello%20world"},
		{"/search/foo+bar", "foo+bar"},
		{"/search/test@example", "test@example"},
	}

	for _, tt := range tests {
		result := trie.Match(tt.path)
		if result == nil {
			t.Errorf("Match(%q) = nil", tt.path)
			continue
		}
		if result.Params["query"] != tt.wantParam {
			t.Errorf("Match(%q) param[query] = %q, want %q", tt.path, result.Params["query"], tt.wantParam)
		}
	}
}

func TestRouterTrie_DeepNesting(t *testing.T) {
	trie := NewRouterTrie()
	trie.Insert("/a/b/c/d/e/f/g", routeEntry{pattern: "/a/b/c/d/e/f/g"})
	trie.Insert("/a/b/c/:d/e/:f/g", routeEntry{pattern: "/a/b/c/:d/e/:f/g"})

	result := trie.Match("/a/b/c/d/e/f/g")
	if result == nil {
		t.Fatal("Match() = nil")
	}
	if result.Entry.pattern != "/a/b/c/d/e/f/g" {
		t.Errorf("pattern = %q, want static path", result.Entry.pattern)
	}

	result = trie.Match("/a/b/c/x/e/y/g")
	if result == nil {
		t.Fatal("Match() = nil for param route")
	}
	if result.Entry.pattern != "/a/b/c/:d/e/:f/g" {
		t.Errorf("pattern = %q, want param path", result.Entry.pattern)
	}
	if result.Params["d"] != "x" || result.Params["f"] != "y" {
		t.Errorf("params = %v, want d=x, f=y", result.Params)
	}
}

func TestRouterTrie_EmptyTrie(t *testing.T) {
	trie := NewRouterTrie()

	result := trie.Match("/anything")
	if result != nil {
		t.Errorf("Match on empty trie = %v, want nil", result)
	}

	result = trie.Match("/")
	if result != nil {
		t.Errorf("Match on empty trie = %v, want nil", result)
	}
}
