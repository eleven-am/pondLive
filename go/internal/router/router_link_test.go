package router

import (
	"net/url"
	"testing"

	h "github.com/eleven-am/pondlive/go/internal/html"
)

func TestExtractTargetFromEventPrefersCurrentTargetDataset(t *testing.T) {
	base := Location{Path: "/base"}
	ev := h.Event{
		Payload: map[string]any{
			"currentTarget.dataset.routerPath":  "/nested/path",
			"currentTarget.dataset.routerQuery": "foo=bar&baz=qux",
			"currentTarget.dataset.routerHash":  "section",
		},
	}

	next := extractTargetFromEvent(ev, base)

	if next.Path != "/nested/path" {
		t.Fatalf("expected dataset path to be used, got %q", next.Path)
	}

	expectedQuery := url.Values{
		"baz": {"qux"},
		"foo": {"bar"},
	}
	if !valuesEqual(next.Query, expectedQuery) {
		t.Fatalf("expected query %v, got %v", expectedQuery, next.Query)
	}

	if next.Hash != "section" {
		t.Fatalf("expected dataset hash to be used, got %q", next.Hash)
	}
}

func TestExtractTargetFromEventFallsBackToCurrentTargetHref(t *testing.T) {
	base := Location{Path: "/current", Query: url.Values{"foo": {"bar"}}, Hash: "frag"}
	ev := h.Event{
		Payload: map[string]any{
			"currentTarget.href": "../other?baz=qux#next",
		},
	}

	next := extractTargetFromEvent(ev, base)

	if next.Path != "/other" {
		t.Fatalf("expected href path to resolve to /other, got %q", next.Path)
	}

	expectedQuery := url.Values{"baz": {"qux"}}
	if !valuesEqual(next.Query, expectedQuery) {
		t.Fatalf("expected href query %v, got %v", expectedQuery, next.Query)
	}

	if next.Hash != "next" {
		t.Fatalf("expected href hash 'next', got %q", next.Hash)
	}
}
