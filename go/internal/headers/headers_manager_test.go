package headers

import (
	"testing"
)

func TestManager_Get(t *testing.T) {
	controller := NewRequestController()
	headers := make(map[string][]string)
	headers["Cookie"] = []string{"session=abc123; theme=dark"}

	// Manually set headers for testing
	controller.requestHeaders = headers

	manager := &Manager{
		controller: controller,
	}

	val, ok := manager.Get("Cookie")
	if !ok {
		t.Error("expected ok=true for existing Cookie header")
	}
	if val != "session=abc123; theme=dark" {
		t.Errorf("expected 'session=abc123; theme=dark', got %q", val)
	}

	val, ok = manager.Get("Authorization")
	if ok {
		t.Error("expected ok=false for non-existent header")
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
}

func TestManager_Get_NilController(t *testing.T) {
	manager := &Manager{
		controller: nil,
	}

	val, ok := manager.Get("Cookie")
	if ok {
		t.Error("expected ok=false when controller is nil")
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
}

func TestManager_SetCookie_NilController(t *testing.T) {
	manager := &Manager{
		controller: nil,
	}

	// Should not panic
	manager.SetCookie("test", "value")
}

func TestManager_DeleteCookie_NilController(t *testing.T) {
	manager := &Manager{
		controller: nil,
	}

	// Should not panic
	manager.DeleteCookie("test")
}

func TestManager_getAction(t *testing.T) {
	manager := &Manager{
		actions: []actionRequest{
			{Name: "session", Value: "abc123", Token: "token1"},
			{Name: "theme", Value: "dark", Token: "token2"},
		},
	}

	// Test finding existing action
	action, ok := manager.getAction("token1")
	if !ok {
		t.Error("expected ok=true for existing token")
	}
	if action == nil {
		t.Fatal("expected non-nil action")
	}
	if action.Name != "session" || action.Value != "abc123" {
		t.Errorf("wrong action returned: %+v", action)
	}

	// Test finding another action
	action, ok = manager.getAction("token2")
	if !ok {
		t.Error("expected ok=true for token2")
	}
	if action.Name != "theme" {
		t.Error("wrong action returned")
	}

	// Test non-existent token
	action, ok = manager.getAction("nonexistent")
	if ok {
		t.Error("expected ok=false for non-existent token")
	}
	if action != nil {
		t.Error("expected nil action for non-existent token")
	}
}

func TestManager_replaceAction(t *testing.T) {
	manager := &Manager{
		actions: []actionRequest{
			{Name: "session", Value: "old", Token: "token1"},
			{Name: "theme", Value: "light", Token: "token2"},
		},
	}

	// Replace existing action
	newAction := actionRequest{
		Name:  "session",
		Value: "new",
		Token: "token3",
	}
	manager.replaceAction(newAction)

	if len(manager.actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(manager.actions))
	}

	// Verify session was replaced
	found := false
	for _, action := range manager.actions {
		if action.Name == "session" {
			if action.Value != "new" || action.Token != "token3" {
				t.Error("action not replaced correctly")
			}
			found = true
		}
	}
	if !found {
		t.Error("session action not found")
	}

	// Add new action
	newAction2 := actionRequest{
		Name:  "lang",
		Value: "en",
		Token: "token4",
	}
	manager.replaceAction(newAction2)

	if len(manager.actions) != 3 {
		t.Errorf("expected 3 actions after adding new one, got %d", len(manager.actions))
	}
}
