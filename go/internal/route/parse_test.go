package route

import "testing"

func TestParseStatic(t *testing.T) {
	match, err := Parse("/about", "/about", "")
	if err != nil {
		t.Fatalf("expected static match: %v", err)
	}
	if match.Pattern != "/about" {
		t.Fatalf("unexpected pattern: %q", match.Pattern)
	}
	if len(match.Params) != 0 {
		t.Fatalf("expected no params, got %#v", match.Params)
	}
}

func TestParseDynamic(t *testing.T) {
	match, err := Parse("/users/:id", "/users/123", "")
	if err != nil {
		t.Fatalf("expected dynamic match: %v", err)
	}
	if match.Params["id"] != "123" {
		t.Fatalf("expected id=123, got %#v", match.Params)
	}
}

func TestParseWildcard(t *testing.T) {
	match, err := Parse("/files/*rest", "/files/a/b/c", "")
	if err != nil {
		t.Fatalf("expected wildcard match: %v", err)
	}
	if match.Params["rest"] != "a/b/c" {
		t.Fatalf("expected rest=a/b/c, got %#v", match.Params)
	}
}

func TestParseOptional(t *testing.T) {
	match, err := Parse("/blog/:slug?", "/blog", "")
	if err != nil {
		t.Fatalf("expected optional match: %v", err)
	}
	if len(match.Params) != 0 {
		t.Fatalf("expected empty params, got %#v", match.Params)
	}
}

func TestParseQueryValues(t *testing.T) {
	match, err := Parse("/search", "/search", "q=golang&tag=ui&tag=go")
	if err != nil {
		t.Fatalf("expected query parse: %v", err)
	}
	if match.Query.Get("q") != "golang" {
		t.Fatalf("expected q parameter, got %#v", match.Query)
	}
	values := match.Query["tag"]
	if len(values) != 2 {
		t.Fatalf("expected two tag values, got %#v", values)
	}
}

func TestBestMatchPrefersSpecific(t *testing.T) {
	patterns := []string{"/users/:id", "/users/:id/settings"}
	match, idx, ok := BestMatch("/users/1/settings", "", patterns)
	if !ok {
		t.Fatal("expected best match")
	}
	if match.Pattern != "/users/:id/settings" {
		t.Fatalf("expected specific pattern, got %q", match.Pattern)
	}
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
}

func TestBestMatchNoMatch(t *testing.T) {
	patterns := []string{"/users/:id"}
	_, _, ok := BestMatch("/about", "", patterns)
	if ok {
		t.Fatal("expected no match")
	}
}
