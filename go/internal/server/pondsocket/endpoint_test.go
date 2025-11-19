package pondsocket

import (
	"net/http"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	"github.com/eleven-am/pondlive/go/internal/protocol"
)

func TestGetSessionID(t *testing.T) {

	t.Skip("getSessionID testing requires PondSocket context integration")
}

func TestPayloadToDOMEventEmptyPayload(t *testing.T) {
	event := payloadToDOMEvent(map[string]interface{}{})

	if event.Name != "" {
		t.Fatalf("expected empty Name, got %q", event.Name)
	}
	if event.Value != "" {
		t.Fatalf("expected empty Value, got %q", event.Value)
	}
	if event.Form != nil {
		t.Fatalf("expected nil Form, got %v", event.Form)
	}
	if event.Payload == nil {
		t.Fatal("expected non-nil Payload map")
	}
	if len(event.Payload) != 0 {
		t.Fatalf("expected empty Payload, got %d entries", len(event.Payload))
	}
}

func TestPayloadToDOMEventFormNonStringValues(t *testing.T) {
	payload := map[string]interface{}{
		"name": "submit",
		"form": map[string]interface{}{
			"username": "test",
			"count":    123,
			"enabled":  true,
			"price":    45.67,
			"email":    "valid@email",
		},
	}

	event := payloadToDOMEvent(payload)

	if event.Form["username"] != "test" {
		t.Fatalf("expected Form[username]=test, got %q", event.Form["username"])
	}
	if event.Form["email"] != "valid@email" {
		t.Fatalf("expected Form[email]=valid@email, got %q", event.Form["email"])
	}

	if _, exists := event.Form["count"]; exists {
		t.Fatal("expected non-string count to be excluded from Form")
	}
	if _, exists := event.Form["enabled"]; exists {
		t.Fatal("expected non-string enabled to be excluded from Form")
	}
	if _, exists := event.Form["price"]; exists {
		t.Fatal("expected non-string price to be excluded from Form")
	}
}

func TestPayloadToDOMEventModsPartial(t *testing.T) {

	payload := map[string]interface{}{
		"name": "keydown",
		"mods": map[string]interface{}{
			"ctrl": true,
		},
	}

	event := payloadToDOMEvent(payload)

	if !event.Mods.Ctrl {
		t.Fatal("expected Ctrl=true")
	}
	if event.Mods.Shift {
		t.Fatal("expected Shift=false (default)")
	}
	if event.Mods.Alt {
		t.Fatal("expected Alt=false (default)")
	}
	if event.Mods.Meta {
		t.Fatal("expected Meta=false (default)")
	}
	if event.Mods.Button != 0 {
		t.Fatalf("expected Button=0 (default), got %d", event.Mods.Button)
	}
}

func TestPayloadToDOMEventNoMods(t *testing.T) {
	payload := map[string]interface{}{
		"name": "click",
	}

	event := payloadToDOMEvent(payload)

	if event.Mods.Ctrl || event.Mods.Shift || event.Mods.Alt || event.Mods.Meta {
		t.Fatal("expected all modifiers to be false")
	}
	if event.Mods.Button != 0 {
		t.Fatalf("expected Button=0, got %d", event.Mods.Button)
	}
}

func TestPayloadToDOMEventPayloadPreservation(t *testing.T) {
	payload := map[string]interface{}{
		"name":   "custom",
		"value":  "test",
		"custom": "field",
		"nested": map[string]interface{}{
			"key": "value",
		},
		"array": []interface{}{1, 2, 3},
	}

	event := payloadToDOMEvent(payload)

	if event.Payload["custom"] != "field" {
		t.Fatalf("expected Payload[custom]=field, got %v", event.Payload["custom"])
	}

	nested, ok := event.Payload["nested"].(map[string]interface{})
	if !ok {
		t.Fatal("expected nested to be preserved as map")
	}
	if nested["key"] != "value" {
		t.Fatalf("expected nested[key]=value, got %v", nested["key"])
	}

	array, ok := event.Payload["array"].([]interface{})
	if !ok {
		t.Fatal("expected array to be preserved")
	}
	if len(array) != 3 {
		t.Fatalf("expected array length 3, got %d", len(array))
	}
}

func TestCloneHeader(t *testing.T) {
	original := http.Header{
		"Content-Type":  []string{"application/json"},
		"Authorization": []string{"Bearer token"},
		"Custom":        []string{"value1", "value2"},
	}

	cloned := cloneHeader(original)

	if cloned.Get("Content-Type") != "application/json" {
		t.Fatal("expected Content-Type to be cloned")
	}
	if cloned.Get("Authorization") != "Bearer token" {
		t.Fatal("expected Authorization to be cloned")
	}
	if len(cloned["Custom"]) != 2 {
		t.Fatalf("expected 2 Custom values, got %d", len(cloned["Custom"]))
	}

	cloned.Set("Content-Type", "text/html")
	if original.Get("Content-Type") != "application/json" {
		t.Fatal("modification of clone affected original")
	}

	emptyClone := cloneHeader(nil)
	if emptyClone == nil {
		t.Fatal("expected non-nil header for nil input")
	}
	if len(emptyClone) != 0 {
		t.Fatalf("expected empty header, got %d entries", len(emptyClone))
	}
}

func TestCloneCookies(t *testing.T) {
	original := []*http.Cookie{
		{Name: "session", Value: "abc123"},
		{Name: "user", Value: "john"},
	}

	cloned := cloneCookies(original)

	if len(cloned) != len(original) {
		t.Fatalf("expected %d cookies, got %d", len(original), len(cloned))
	}

	if cloned[0].Name != "session" || cloned[0].Value != "abc123" {
		t.Fatal("expected first cookie to be cloned")
	}
	if cloned[1].Name != "user" || cloned[1].Value != "john" {
		t.Fatal("expected second cookie to be cloned")
	}

	if &original[0] == &cloned[0] {
		t.Fatal("expected cloned cookies to have different pointers")
	}

	cloned[0].Value = "modified"
	if original[0].Value == "modified" {
		t.Fatal("modification of clone affected original")
	}

	if cloneCookies(nil) != nil {
		t.Fatal("expected nil for nil input")
	}

	if cloneCookies([]*http.Cookie{}) != nil {
		t.Fatal("expected nil for empty slice")
	}

	withNil := []*http.Cookie{
		{Name: "valid", Value: "value"},
		nil,
		{Name: "also-valid", Value: "value2"},
	}
	clonedWithNil := cloneCookies(withNil)
	if len(clonedWithNil) != 3 {
		t.Fatalf("expected 3 cookies including nil, got %d", len(clonedWithNil))
	}
	if clonedWithNil[1] != nil {
		t.Fatal("expected nil cookie to remain nil")
	}
}

func TestDOMEventTypes(t *testing.T) {

	event := dom2.Event{
		Name:  "click",
		Value: "button",
		Payload: map[string]any{
			"clientX": 100,
			"clientY": 200,
		},
		Form: map[string]string{
			"username": "test",
		},
		Mods: dom2.Modifiers{
			Ctrl:   true,
			Shift:  false,
			Alt:    false,
			Meta:   false,
			Button: 0,
		},
	}

	if event.Name != "click" {
		t.Fatal("expected Name field to exist")
	}
	if event.Value != "button" {
		t.Fatal("expected Value field to exist")
	}
	if event.Payload == nil {
		t.Fatal("expected Payload field to exist")
	}
	if event.Form == nil {
		t.Fatal("expected Form field to exist")
	}
	if !event.Mods.Ctrl {
		t.Fatal("expected Mods.Ctrl to be accessible")
	}
}

func TestProtocolTypes(t *testing.T) {

	evt := protocol.ClientEvent{
		T:       "evt",
		SID:     "session-id",
		HID:     "handler-id",
		CSeq:    1,
		Payload: map[string]interface{}{"name": "click"},
	}

	if evt.T != "evt" {
		t.Fatal("expected T field")
	}
	if evt.SID != "session-id" {
		t.Fatal("expected SID field")
	}
	if evt.HID != "handler-id" {
		t.Fatal("expected HID field")
	}

	ack := protocol.EventAck{
		T:    "evt-ack",
		SID:  "session-id",
		CSeq: 1,
	}

	if ack.T != "evt-ack" {
		t.Fatal("expected T field")
	}

	nav := protocol.Location{
		Path:  "/home",
		Query: "q=search",
		Hash:  "section",
	}

	if nav.Path != "/home" {
		t.Fatal("expected Path field")
	}

	resp := protocol.DOMResponse{
		T:      "domres",
		ID:     "req-1",
		Values: map[string]any{"width": 100},
		Result: "success",
		Error:  "",
	}

	if resp.T != "domres" {
		t.Fatal("expected T field")
	}
	if resp.ID != "req-1" {
		t.Fatal("expected ID field")
	}
}
