package router2

import (
	"net/url"
	"testing"
)

func TestRouterStoreLocationLifecycle(t *testing.T) {
	store := NewStore(Location{Path: "users", Query: url.Values{"page": {"1"}}, Hash: "details"})
	if got := store.Location(); got.Path != "/users" || got.Hash != "details" || got.Query.Get("page") != "1" {
		t.Fatalf("unexpected initial location: %#v", got)
	}

	changes := make(chan Location, 2)
	cancel := store.Subscribe(func(loc Location) {
		changes <- loc
	})
	defer cancel()

	store.SetLocation(Location{Path: "/users/2", Query: url.Values{"page": {"2"}}})

	select {
	case loc := <-changes:
		if loc.Path != "/users/2" || loc.Query.Get("page") != "2" {
			t.Fatalf("unexpected callback value: %#v", loc)
		}
	default:
		t.Fatalf("expected listener to fire")
	}
}

func TestRouterStoreRecordNavigationFIFO(t *testing.T) {
	store := NewStore(Location{Path: "/"})
	store.RecordNavigation(NavKindPush, Location{Path: "/first"})
	store.RecordNavigation(NavKindReplace, Location{Path: "/second"})
	store.RecordNavigation(NavKindBack, Location{})

	pending := store.DrainPending()
	if len(pending) != 3 {
		t.Fatalf("expected 3 pending events, got %d", len(pending))
	}
	if pending[0].Kind != NavKindPush || pending[0].Target.Path != "/first" {
		t.Fatalf("first event mismatch: %#v", pending[0])
	}
	if pending[1].Kind != NavKindReplace || pending[1].Target.Path != "/second" {
		t.Fatalf("second event mismatch: %#v", pending[1])
	}
	if pending[2].Kind != NavKindBack {
		t.Fatalf("expected back event last, got %#v", pending[2])
	}

	if drained := store.DrainPending(); len(drained) != 0 {
		t.Fatalf("expected drain to clear pending events, got %d", len(drained))
	}
}

func TestRouterStoreNavEventMetadata(t *testing.T) {
	store := NewStore(Location{Path: "/"})
	event := store.RecordNavigationWithSource(NavKindPush, Location{Path: "/meta"}, "componentA")
	if event.Source != "componentA" || event.Time.IsZero() {
		t.Fatalf("expected nav event metadata, got %#v", event)
	}
	back := store.RecordBackWithSource("componentB")
	if back.Source != "componentB" || back.Kind != NavKindBack {
		t.Fatalf("expected back metadata, got %#v", back)
	}
}

func TestRouterStoreParamsIsolation(t *testing.T) {
	store := NewStore(Location{Path: "/"})
	store.SetParams(map[string]string{"id": "123"})

	params := store.Params()
	params["id"] = "mutated"

	if got := store.Params()["id"]; got != "123" {
		t.Fatalf("params should be isolated, got %q", got)
	}
}
