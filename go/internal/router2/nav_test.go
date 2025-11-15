package router2

import "testing"

type stubNavHandler struct {
	events []NavEvent
}

func (h *stubNavHandler) DrainNav(events []NavEvent) { h.events = append(h.events, events...) }

func TestNavDispatcherDrainsPending(t *testing.T) {
	store := NewStore(Location{Path: "/"})
	handler := &stubNavHandler{}
	dispatcher := NewNavDispatcher(store, handler)
	cancel := dispatcher.Start()
	defer cancel()

	store.RecordNavigation(NavKindPush, Location{Path: "/a"})
	dispatcher.Drain()

	if len(handler.events) != 1 || handler.events[0].Target.Path != "/a" {
		t.Fatalf("expected handler to receive nav event, got %#v", handler.events)
	}
}
