package runtime

import (
	"testing"

	handlers "github.com/eleven-am/pondlive/go/internal/handlers"
)

func TestWireEventToEvent(t *testing.T) {
	payload := WireEvent{
		Name:  "click",
		Value: "42",
		Payload: map[string]any{
			"id": 42,
		},
		Form: map[string]string{
			"field": "value",
		},
		Mods: WireModifiers{Ctrl: true, Button: 2},
	}
	ev := payload.ToEvent()
	if ev.Name != "click" || ev.Value != "42" {
		t.Fatalf("unexpected event: %+v", ev)
	}
	if ev.Payload["id"].(int) != 42 {
		t.Fatalf("expected payload to contain id, got %+v", ev.Payload)
	}
	if ev.Form["field"] != "value" {
		t.Fatalf("expected form field to round-trip, got %+v", ev.Form)
	}
	if !ev.Mods.Ctrl || ev.Mods.Button != 2 {
		t.Fatalf("unexpected modifiers: %+v", ev.Mods)
	}
	// ensure copies are returned
	ev.Payload["id"] = 99
	if payload.Payload["id"].(int) != 42 {
		t.Fatalf("expected payload map to be cloned")
	}
}

func TestWireEventToEventEmpty(t *testing.T) {
	ev := WireEvent{Name: "submit"}.ToEvent()
	if ev.Name != "submit" {
		t.Fatalf("unexpected event name: %s", ev.Name)
	}
	if ev.Payload == nil || len(ev.Payload) != 0 {
		t.Fatalf("expected empty payload map, got %+v", ev.Payload)
	}
	if ev.Form == nil || len(ev.Form) != 0 {
		t.Fatalf("expected empty form map, got %+v", ev.Form)
	}
	if ev.Mods != (handlers.Modifiers{}) {
		t.Fatalf("expected zero modifiers, got %+v", ev.Mods)
	}
}
