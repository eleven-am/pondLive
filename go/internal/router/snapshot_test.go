package router

import (
	"net/url"
	"testing"
	"time"
)

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

func TestSnapshotPayloadEncoding(t *testing.T) {
	snap := Snapshot{
		Location: Location{Path: "/foo", Query: url.Values{"q": {"1"}}},
		Params:   map[string]string{"id": "123"},
		History: []NavEvent{
			{Seq: 1, Kind: NavKindPush, Target: Location{Path: "/bar"}, Source: "test", Time: time.Unix(100, 0)},
		},
	}
	data, err := snap.ToPayload()
	if err != nil {
		t.Fatalf("encode error: %v", err)
	}
	decoded, err := SnapshotFromPayload(data)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if decoded.Location.Path != "/foo" || decoded.Location.Query.Get("q") != "1" {
		t.Fatalf("location mismatch: %#v", decoded.Location)
	}
	if decoded.Params["id"] != "123" {
		t.Fatalf("params mismatch: %#v", decoded.Params)
	}
	if len(decoded.History) != 1 || decoded.History[0].Target.Path != "/bar" || decoded.History[0].Source != "test" {
		t.Fatalf("history mismatch: %#v", decoded.History)
	}
}

func TestSnapshotPayloadHelpers(t *testing.T) {
	store := NewStore(Location{Path: "/payload"})
	store.RecordNavigation(NavKindPush, Location{Path: "/payload/next"})
	data, err := StoreSnapshotPayload(store)
	if err != nil {
		t.Fatalf("StoreSnapshotPayload error: %v", err)
	}
	clone := NewStore(Location{})
	if err := ApplySnapshotPayload(clone, data); err != nil {
		t.Fatalf("ApplySnapshotPayload error: %v", err)
	}
	if loc := clone.Location(); loc.Path != "/payload/next" {
		t.Fatalf("expected cloned location, got %#v", loc)
	}
}

func TestSnapshotPayloadHelpersNilStore(t *testing.T) {
	data, err := StoreSnapshotPayload(nil)
	if err != nil {
		t.Fatalf("expected nil error for nil store, got %v", err)
	}
	if len(data) != 0 {
		t.Fatalf("expected nil payload for nil store, got %v", data)
	}
	if err := ApplySnapshotPayload(nil, nil); err != nil {
		t.Fatalf("expected nil error when applying to nil store, got %v", err)
	}
}

func TestApplySnapshotPayloadInvalid(t *testing.T) {
	store := NewStore(Location{})
	if err := ApplySnapshotPayload(store, []byte("{invalid")); err == nil {
		t.Fatalf("expected error for invalid payload")
	}
	if loc := store.Location(); loc.Path != "/" {
		t.Fatalf("store should remain untouched on error, got %q", loc.Path)
	}
}
