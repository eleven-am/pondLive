package router

import (
	"testing"
)

func TestRouterTrie_Insert_AndMatch(t *testing.T) {
	trie := NewRouterTrie()

	homeEntry := routeEntry{pattern: "/"}
	usersEntry := routeEntry{pattern: "/users"}
	userDetailEntry := routeEntry{pattern: "/users/:id"}
	wildcardEntry := routeEntry{pattern: "/files/*path"}

	trie.Insert("/", homeEntry)
	trie.Insert("/users", usersEntry)
	trie.Insert("/users/:id", userDetailEntry)
	trie.Insert("/files/*path", wildcardEntry)

	tests := []struct {
		name        string
		path        string
		shouldMatch bool
		wantPattern string
		wantParams  map[string]string
		wantRest    string
	}{
		{
			name:        "root path",
			path:        "/",
			shouldMatch: true,
			wantPattern: "/",
			wantParams:  map[string]string{},
		},
		{
			name:        "static path",
			path:        "/users",
			shouldMatch: true,
			wantPattern: "/users",
			wantParams:  map[string]string{},
		},
		{
			name:        "parameterized path",
			path:        "/users/123",
			shouldMatch: true,
			wantPattern: "/users/:id",
			wantParams:  map[string]string{"id": "123"},
		},
		{
			name:        "wildcard path",
			path:        "/files/docs/readme.md",
			shouldMatch: true,
			wantPattern: "/files/*path",
			wantParams:  map[string]string{"path": "docs/readme.md"},
			wantRest:    "/docs/readme.md",
		},
		{
			name:        "no match",
			path:        "/products",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trie.Match(tt.path)

			if tt.shouldMatch {
				if result == nil {
					t.Fatalf("expected match for path %q, got nil", tt.path)
				}

				if result.Entry.pattern != tt.wantPattern {
					t.Errorf("pattern mismatch: got %q, want %q", result.Entry.pattern, tt.wantPattern)
				}

				if len(result.Params) != len(tt.wantParams) {
					t.Errorf("params count mismatch: got %d, want %d", len(result.Params), len(tt.wantParams))
				}

				for k, v := range tt.wantParams {
					if result.Params[k] != v {
						t.Errorf("param %q mismatch: got %q, want %q", k, result.Params[k], v)
					}
				}

				if result.Rest != tt.wantRest {
					t.Errorf("rest mismatch: got %q, want %q", result.Rest, tt.wantRest)
				}
			} else {
				if result != nil {
					t.Errorf("expected no match for path %q, got pattern %q", tt.path, result.Entry.pattern)
				}
			}
		})
	}
}

func TestRouterTrie_Priority(t *testing.T) {
	trie := NewRouterTrie()

	staticEntry := routeEntry{pattern: "/users/new"}
	paramEntry := routeEntry{pattern: "/users/:id"}

	trie.Insert("/users/new", staticEntry)
	trie.Insert("/users/:id", paramEntry)

	result := trie.Match("/users/new")
	if result == nil {
		t.Fatal("expected match")
	}
	if result.Entry.pattern != "/users/new" {
		t.Errorf("expected static match, got %q", result.Entry.pattern)
	}

	result = trie.Match("/users/123")
	if result == nil {
		t.Fatal("expected match")
	}
	if result.Entry.pattern != "/users/:id" {
		t.Errorf("expected param match, got %q", result.Entry.pattern)
	}
	if result.Params["id"] != "123" {
		t.Errorf("expected id=123, got id=%s", result.Params["id"])
	}
}

func TestRouterTrie_EmptyPath(t *testing.T) {
	trie := NewRouterTrie()
	entry := routeEntry{pattern: "/"}

	trie.Insert("", entry)

	result := trie.Match("")
	if result == nil {
		t.Fatal("expected match for empty path")
	}
	if result.Entry.pattern != "/" {
		t.Errorf("expected pattern /, got %q", result.Entry.pattern)
	}
}

func TestRouterTrie_MultipleParams(t *testing.T) {
	trie := NewRouterTrie()
	entry := routeEntry{pattern: "/users/:userId/posts/:postId"}

	trie.Insert("/users/:userId/posts/:postId", entry)

	result := trie.Match("/users/123/posts/456")
	if result == nil {
		t.Fatal("expected match")
	}

	if result.Params["userId"] != "123" {
		t.Errorf("expected userId=123, got %s", result.Params["userId"])
	}
	if result.Params["postId"] != "456" {
		t.Errorf("expected postId=456, got %s", result.Params["postId"])
	}
}
