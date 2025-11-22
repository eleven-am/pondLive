package headers

import (
	"net/http"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// mockCtx implements runtime.Ctx for testing
type mockCtx struct {
	live bool
}

func (m *mockCtx) IsLive() bool {
	return m.live
}

func (m *mockCtx) IsPaused() bool {
	return false
}

func (m *mockCtx) IsServer() bool {
	return !m.live
}

// mockScriptHandle implements basic ScriptHandle methods for testing
type mockScriptHandle struct {
	attachCalled bool
	attachedNode *dom.StructuredNode
}

func (m *mockScriptHandle) AttachTo(node *dom.StructuredNode) {
	m.attachCalled = true
	m.attachedNode = node
}

func TestManager_Get(t *testing.T) {

	controller := NewRequestController()
	headers := make(http.Header)
	headers.Set("Cookie", "session=abc123; theme=dark")
	controller.SetInitialHeaders(headers)

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

func TestManager_SetCookie_SSRMode(t *testing.T) {
	controller := NewRequestController()
	ctx := &mockCtx{live: false}

	manager := &Manager{
		controller: controller,
		ctx:        ctx,
	}

	manager.SetCookie("session", "test123")

	headers := controller.GetResponseHeaders()
	setCookie := headers.Get("Set-Cookie")
	if setCookie != "session=test123" {
		t.Errorf("expected 'session=test123', got %q", setCookie)
	}
}

func TestManager_SetCookieWithOptions_SSRMode(t *testing.T) {
	controller := NewRequestController()
	ctx := &mockCtx{live: false}

	manager := &Manager{
		controller: controller,
		ctx:        ctx,
	}

	opts := CookieOptions{
		Path:     "/admin",
		Domain:   "example.com",
		MaxAge:   3600,
		Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	}

	manager.SetCookieWithOptions("auth", "token456", opts)

	headers := controller.GetResponseHeaders()
	setCookie := headers.Get("Set-Cookie")
	if setCookie == "" {
		t.Fatal("Set-Cookie header not set")
	}

	cookie := &http.Cookie{}
	req := &http.Request{Header: make(http.Header)}
	req.Header.Set("Cookie", setCookie)

	if setCookie != "auth=token456; Path=/admin; Domain=example.com; Max-Age=3600; HttpOnly; Secure; SameSite=Strict" {
		t.Logf("Actual Set-Cookie: %q", setCookie)

		expected := []string{"auth=token456", "Path=/admin", "Domain=example.com", "Max-Age=3600", "HttpOnly", "Secure", "SameSite=Strict"}
		for _, exp := range expected {
			if !contains(setCookie, exp) {
				t.Errorf("Set-Cookie missing expected component: %q", exp)
			}
		}
	}
}

func TestManager_DeleteCookie_SSRMode(t *testing.T) {
	controller := NewRequestController()
	ctx := &mockCtx{live: false}

	manager := &Manager{
		controller: controller,
		ctx:        ctx,
	}

	manager.DeleteCookie("session")

	headers := controller.GetResponseHeaders()
	setCookie := headers.Get("Set-Cookie")
	if setCookie != "session=; Max-Age=-1" {
		t.Errorf("expected 'session=; Max-Age=-1', got %q", setCookie)
	}
}

func TestManager_SetCookie_NilController(t *testing.T) {
	manager := &Manager{
		controller: nil,
	}

	manager.SetCookie("test", "value")
}

func TestManager_DeleteCookie_NilController(t *testing.T) {
	manager := &Manager{
		controller: nil,
	}

	manager.DeleteCookie("test")
}

func TestManager_getAction(t *testing.T) {
	manager := &Manager{
		actions: []actionRequest{
			{Name: "session", Value: "abc123", Token: "token1"},
			{Name: "theme", Value: "dark", Token: "token2"},
		},
	}

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

	action, ok = manager.getAction("token2")
	if !ok {
		t.Error("expected ok=true for token2")
	}
	if action.Name != "theme" {
		t.Error("wrong action returned")
	}

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

	newAction := actionRequest{
		Name:  "session",
		Value: "new",
		Token: "token3",
	}
	manager.replaceAction(newAction)

	if len(manager.actions) != 2 {
		t.Errorf("expected 2 actions, got %d", len(manager.actions))
	}

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

func TestCookieOptions_Expires(t *testing.T) {
	controller := NewRequestController()
	ctx := &mockCtx{live: false}

	manager := &Manager{
		controller: controller,
		ctx:        ctx,
	}

	expires := time.Now().Add(24 * time.Hour)
	opts := CookieOptions{
		Expires: expires,
	}

	manager.SetCookieWithOptions("temp", "value", opts)

	headers := controller.GetResponseHeaders()
	setCookie := headers.Get("Set-Cookie")

	if !contains(setCookie, "Expires=") {
		t.Error("Set-Cookie should contain Expires")
	}
}

func TestCookieOptions_SameSiteVariants(t *testing.T) {
	tests := []struct {
		name     string
		sameSite http.SameSite
		expected string
	}{
		{"None", http.SameSiteNoneMode, "SameSite=None"},
		{"Lax", http.SameSiteLaxMode, "SameSite=Lax"},
		{"Strict", http.SameSiteStrictMode, "SameSite=Strict"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := NewRequestController()
			ctx := &mockCtx{live: false}

			manager := &Manager{
				controller: controller,
				ctx:        ctx,
			}

			opts := CookieOptions{
				SameSite: tt.sameSite,
			}

			manager.SetCookieWithOptions("test", "value", opts)

			headers := controller.GetResponseHeaders()
			setCookie := headers.Get("Set-Cookie")

			if !contains(setCookie, tt.expected) {
				t.Errorf("expected Set-Cookie to contain %q, got %q", tt.expected, setCookie)
			}
		})
	}
}

func TestManager_AttachTo(t *testing.T) {
	mockScript := &runtime.MockScriptHandle{
		AttachCalled: false,
	}

	manager := &Manager{
		script: mockScript,
	}

	node := &dom.StructuredNode{
		Type: "div",
	}

	manager.AttachTo(node)

	if !mockScript.AttachCalled {
		t.Error("expected script.AttachTo to be called")
	}
}

func TestManager_AttachTo_NilScript(t *testing.T) {
	manager := &Manager{
		script: nil,
	}

	node := &dom.StructuredNode{
		Type: "div",
	}

	manager.AttachTo(node)
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
