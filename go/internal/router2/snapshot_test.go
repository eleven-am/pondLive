package router2

import "testing"

func TestSnapshotRoundTrip(t *testing.T) {
	store := NewStore(Location{Path: "/start"})
	store.SetParams(map[string]string{"foo": "bar"})
	store.RecordNavigation(NavKindPush, Location{Path: "/next"})
	store.RecordBack()

	snap := store.Snapshot()
	clone := NewStore(Location{})
	clone.ApplySnapshot(snap)

	if loc := clone.Location(); loc.Path != "/next" {
		t.Fatalf("expected cloned location /next, got %#v", loc)
	}
	if params := clone.Params(); params["foo"] != "bar" {
		t.Fatalf("expected params to round-trip, got %#v", params)
	}
	if history := clone.History(); len(history) != 2 {
		t.Fatalf("expected 2 history events, got %d", len(history))
	}
}
