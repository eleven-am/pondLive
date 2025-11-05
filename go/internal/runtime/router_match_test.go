package runtime

import "testing"

func TestMatchParams(t *testing.T) {
	mr, err := Parse("/users/:id", "/users/42", "")
	if err != nil {
		t.Fatalf("expected match to succeed: %v", err)
	}
	if mr.Params["id"] != "42" {
		t.Fatalf("expected id=42, got %q", mr.Params["id"])
	}
	if mr.Score <= 0 {
		t.Fatalf("expected positive score, got %d", mr.Score)
	}
}

func TestMatchWildcardRest(t *testing.T) {
	mr, err := Parse("/settings/*", "/settings/security", "")
	if err != nil {
		t.Fatalf("expected wildcard match to succeed: %v", err)
	}
	if mr.Rest != "/security" {
		t.Fatalf("expected rest=/security, got %q", mr.Rest)
	}
}

func TestMatchOptionalParams(t *testing.T) {
	present, err := Parse("/docs/:slug?", "/docs/intro", "")
	if err != nil {
		t.Fatalf("expected optional param present: %v", err)
	}
	if present.Params["slug"] != "intro" {
		t.Fatalf("expected slug=intro, got %q", present.Params["slug"])
	}

	absent, err := Parse("/docs/:slug?", "/docs", "")
	if err != nil {
		t.Fatalf("expected optional param to allow absence: %v", err)
	}
	if len(absent.Params) != 0 {
		t.Fatalf("expected no params when optional missing, got %#v", absent.Params)
	}
}

func TestMatchSpecificity(t *testing.T) {
	wildcard, err := Parse("/users/*", "/users/42", "")
	if err != nil {
		t.Fatalf("expected wildcard parse to succeed: %v", err)
	}
	specific, err := Parse("/users/:id", "/users/42", "")
	if err != nil {
		t.Fatalf("expected param parse to succeed: %v", err)
	}
	if !Prefer(specific, wildcard) {
		t.Fatalf("expected parameter route to outrank wildcard")
	}
}
