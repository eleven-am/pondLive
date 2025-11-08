package runtime

import (
	"net/url"
	"testing"

	h "github.com/eleven-am/pondlive/go/internal/html"
)

func noopComponent(ctx Ctx, _ struct{}) h.Node { return h.Fragment() }

func TestStoreSessionLocationUpdatesLiveSession(t *testing.T) {
	sess := NewLiveSession("sid-loc", 1, noopComponent, struct{}{}, nil)
	if sess == nil {
		t.Fatal("expected live session")
	}
	comp := sess.ComponentSession()
	if comp == nil {
		t.Fatal("expected component session")
	}

	target := Location{Path: "/users/1", Query: url.Values{"tab": {"details"}}}
	storeSessionLocation(comp, target)

	loc := sess.Location()
	if loc.Path != "/users/1" {
		t.Fatalf("expected path to update, got %q", loc.Path)
	}
	if loc.Query != "tab=details" {
		t.Fatalf("expected query to update, got %q", loc.Query)
	}
}
