package router

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

	history := store.History()
	if len(history) != 3 {
		t.Fatalf("expected 3 history events, got %d", len(history))
	}
	if history[0].Kind != NavKindPush || history[0].Target.Path != "/first" {
		t.Fatalf("first event mismatch: %#v", history[0])
	}
	if history[1].Kind != NavKindReplace || history[1].Target.Path != "/second" {
		t.Fatalf("second event mismatch: %#v", history[1])
	}
	if history[2].Kind != NavKindBack {
		t.Fatalf("expected back event last, got %#v", history[2])
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

type recordingNavHandler struct {
	events []NavEvent
}

func (h *recordingNavHandler) DrainNav(events []NavEvent) {
	h.events = append(h.events, events...)
}

func TestNavHandlerReceivesEvents(t *testing.T) {
	store := NewStore(Location{Path: "/"})
	handler := &recordingNavHandler{}
	cancel := store.RegisterNavHandler(handler)
	defer cancel()

	store.RecordNavigation(NavKindPush, Location{Path: "/alpha"})
	store.DrainAndDispatch()
	if len(handler.events) != 1 || handler.events[0].Target.Path != "/alpha" {
		t.Fatalf("expected drained event, got %#v", handler.events)
	}
}

func TestDrainNavEventsPreservesFIFO(t *testing.T) {
	store := NewStore(Location{Path: "/"})
	store.RecordNavigation(NavKindPush, Location{Path: "/first"})
	store.RecordNavigation(NavKindReplace, Location{Path: "/second"})
	store.RecordBack()

	events := store.DrainNavEvents()
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Kind != NavKindPush || events[0].Target.Path != "/first" {
		t.Fatalf("expected push first, got %#v", events[0])
	}
	if events[1].Kind != NavKindReplace || events[1].Target.Path != "/second" {
		t.Fatalf("expected replace second, got %#v", events[1])
	}
	if events[2].Kind != NavKindBack {
		t.Fatalf("expected back third, got %#v", events[2])
	}
	if again := store.DrainNavEvents(); len(again) != 0 {
		t.Fatalf("expected queue to be empty after drain, got %#v", again)
	}
}

func TestMultipleBackEventsAreDistinct(t *testing.T) {
	store := NewStore(Location{Path: "/"})
	store.RecordBackWithSource("first")
	store.RecordBackWithSource("second")
	events := store.DrainNavEvents()
	if len(events) != 2 {
		t.Fatalf("expected two back events, got %d", len(events))
	}
	if events[0].Source != "first" || events[1].Source != "second" {
		t.Fatalf("expected sources to persist, got %#v", events)
	}
}

func TestRouterStoreSeed(t *testing.T) {
	store := NewStore(Location{Path: "/"})
	history := []NavEvent{
		{Seq: 1, Kind: NavKindPush, Target: Location{Path: "/seed"}},
	}
	store.Seed(Location{Path: "/seeded"}, map[string]string{"id": "7"}, history)

	if loc := store.Location(); loc.Path != "/seeded" {
		t.Fatalf("expected seeded location /seeded, got %#v", loc)
	}
	if store.Params()["id"] != "7" {
		t.Fatalf("expected params to seed, got %#v", store.Params())
	}
	if hist := store.History(); len(hist) != 1 || hist[0].Target.Path != "/seed" {
		t.Fatalf("expected history clone, got %#v", hist)
	}
	history[0].Target.Path = "/mutated"
	if store.History()[0].Target.Path != "/seed" {
		t.Fatalf("store history should be isolated from seed input")
	}
}

func TestRouterStoreSeedCanonicalizes(t *testing.T) {
	store := NewStore(Location{})
	store.Seed(Location{Path: "users"}, map[string]string{" user ": " 1 "}, nil)
	if loc := store.Location(); loc.Path != "/users" {
		t.Fatalf("expected canonicalized path, got %#v", loc.Path)
	}
	if _, ok := store.Params()[" user "]; !ok {
		t.Fatalf("expected params key preserved, got %#v", store.Params())
	}
}
