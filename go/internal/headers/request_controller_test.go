package headers

import (
	"net/http"
	"net/url"
	"testing"
)

func TestNewRequestController(t *testing.T) {
	rc := NewRequestController()
	if rc == nil {
		t.Fatal("NewRequestController returned nil")
	}
	if rc.requestHeaders == nil {
		t.Error("requestHeaders not initialized")
	}
	if rc.responseHeaders == nil {
		t.Error("responseHeaders not initialized")
	}
}

func TestRequestController_Get(t *testing.T) {
	rc := NewRequestController()

	val, ok := rc.Get("X-Test")
	if ok {
		t.Error("expected ok=false for non-existent header")
	}
	if val != "" {
		t.Errorf("expected empty value, got %q", val)
	}

	headers := make(http.Header)
	headers.Set("X-Test", "test-value")
	headers.Set("Authorization", "Bearer token123")
	rc.SetInitialHeaders(headers)

	val, ok = rc.Get("X-Test")
	if !ok {
		t.Error("expected ok=true for existing header")
	}
	if val != "test-value" {
		t.Errorf("expected 'test-value', got %q", val)
	}

	val, ok = rc.Get("Authorization")
	if !ok {
		t.Error("expected ok=true for Authorization header")
	}
	if val != "Bearer token123" {
		t.Errorf("expected 'Bearer token123', got %q", val)
	}

	val, ok = rc.Get("authorization")
	if !ok {
		t.Error("expected ok=true for lowercase 'authorization'")
	}
	if val != "Bearer token123" {
		t.Errorf("expected 'Bearer token123', got %q", val)
	}
}

func TestRequestController_Get_NilController(t *testing.T) {
	var rc *RequestController
	val, ok := rc.Get("X-Test")
	if ok {
		t.Error("expected ok=false for nil controller")
	}
	if val != "" {
		t.Errorf("expected empty value for nil controller, got %q", val)
	}
}

func TestRequestController_Set(t *testing.T) {
	rc := NewRequestController()

	rc.Set("X-Custom", "custom-value")
	rc.Set("Content-Type", "application/json")

	headers := rc.GetResponseHeaders()
	if headers == nil {
		t.Fatal("GetResponseHeaders returned nil")
	}

	if headers.Get("X-Custom") != "custom-value" {
		t.Errorf("expected 'custom-value', got %q", headers.Get("X-Custom"))
	}
	if headers.Get("Content-Type") != "application/json" {
		t.Errorf("expected 'application/json', got %q", headers.Get("Content-Type"))
	}
}

func TestRequestController_Set_NilController(t *testing.T) {
	var rc *RequestController

	rc.Set("X-Test", "value")
}

func TestRequestController_GetResponseHeaders(t *testing.T) {
	rc := NewRequestController()

	rc.Set("X-Header-1", "value1")
	rc.Set("X-Header-2", "value2")
	rc.Set("Set-Cookie", "session=abc123")

	headers := rc.GetResponseHeaders()
	if headers == nil {
		t.Fatal("GetResponseHeaders returned nil")
	}

	if headers.Get("X-Header-1") != "value1" {
		t.Error("X-Header-1 not set correctly")
	}
	if headers.Get("X-Header-2") != "value2" {
		t.Error("X-Header-2 not set correctly")
	}
	if headers.Get("Set-Cookie") != "session=abc123" {
		t.Error("Set-Cookie not set correctly")
	}

	headers.Set("X-Header-1", "modified")
	newHeaders := rc.GetResponseHeaders()
	if newHeaders.Get("X-Header-1") != "value1" {
		t.Error("returned headers are not a copy")
	}
}

func TestRequestController_GetResponseHeaders_NilController(t *testing.T) {
	var rc *RequestController
	headers := rc.GetResponseHeaders()
	if headers != nil {
		t.Error("expected nil for nil controller")
	}
}

func TestRequestController_SetInitialHeaders(t *testing.T) {
	rc := NewRequestController()

	headers := make(http.Header)
	headers.Set("User-Agent", "TestAgent/1.0")
	headers.Set("Accept", "application/json")
	headers.Add("X-Multi", "value1")
	headers.Add("X-Multi", "value2")

	rc.SetInitialHeaders(headers)

	val, ok := rc.Get("User-Agent")
	if !ok || val != "TestAgent/1.0" {
		t.Error("User-Agent not set correctly")
	}

	val, ok = rc.Get("Accept")
	if !ok || val != "application/json" {
		t.Error("Accept not set correctly")
	}

	headers.Set("User-Agent", "Modified/2.0")
	val, ok = rc.Get("User-Agent")
	if !ok || val != "TestAgent/1.0" {
		t.Error("initial headers were not cloned")
	}
}

func TestRequestController_SetInitialHeaders_NilController(t *testing.T) {
	var rc *RequestController
	headers := make(http.Header)
	headers.Set("X-Test", "value")

	rc.SetInitialHeaders(headers)
}

func TestRequestController_SetRedirect(t *testing.T) {
	rc := NewRequestController()

	rc.SetRedirect("/login", http.StatusFound)

	url, code, hasRedirect := rc.GetRedirect()
	if !hasRedirect {
		t.Error("expected hasRedirect=true")
	}
	if url != "/login" {
		t.Errorf("expected '/login', got %q", url)
	}
	if code != http.StatusFound {
		t.Errorf("expected %d, got %d", http.StatusFound, code)
	}
}

func TestRequestController_SetRedirect_NilController(t *testing.T) {
	var rc *RequestController

	rc.SetRedirect("/test", http.StatusFound)
}

func TestRequestController_GetRedirect(t *testing.T) {
	rc := NewRequestController()

	url, code, hasRedirect := rc.GetRedirect()
	if hasRedirect {
		t.Error("expected hasRedirect=false when no redirect set")
	}
	if url != "" {
		t.Errorf("expected empty url, got %q", url)
	}
	if code != 0 {
		t.Errorf("expected code=0, got %d", code)
	}

	rc.SetRedirect("/dashboard", http.StatusSeeOther)

	url, code, hasRedirect = rc.GetRedirect()
	if !hasRedirect {
		t.Error("expected hasRedirect=true after setting redirect")
	}
	if url != "/dashboard" {
		t.Errorf("expected '/dashboard', got %q", url)
	}
	if code != http.StatusSeeOther {
		t.Errorf("expected %d, got %d", http.StatusSeeOther, code)
	}
}

func TestRequestController_GetRedirect_NilController(t *testing.T) {
	var rc *RequestController
	url, code, hasRedirect := rc.GetRedirect()
	if hasRedirect {
		t.Error("expected hasRedirect=false for nil controller")
	}
	if url != "" {
		t.Error("expected empty url for nil controller")
	}
	if code != 0 {
		t.Error("expected code=0 for nil controller")
	}
}

func TestRequestController_IsLive(t *testing.T) {
	rc := NewRequestController()

	if rc.IsLive() {
		t.Error("expected IsLive=false by default")
	}

	rc.SetIsLive(true)
	if !rc.IsLive() {
		t.Error("expected IsLive=true after SetIsLive(true)")
	}

	rc.SetIsLive(false)
	if rc.IsLive() {
		t.Error("expected IsLive=false after SetIsLive(false)")
	}
}

func TestRequestController_IsLive_NilController(t *testing.T) {
	var rc *RequestController
	if rc.IsLive() {
		t.Error("expected IsLive=false for nil controller")
	}
}

func TestRequestController_SetIsLive_NilController(t *testing.T) {
	var rc *RequestController

	rc.SetIsLive(true)
}

func TestRequestController_SetInitialLocation(t *testing.T) {
	rc := NewRequestController()

	query := make(url.Values)
	query.Set("page", "1")
	query.Set("sort", "name")

	rc.SetInitialLocation("/products", query, "section-1")

	path, gotQuery, hash := rc.GetInitialLocation()
	if path != "/products" {
		t.Errorf("expected '/products', got %q", path)
	}
	if hash != "section-1" {
		t.Errorf("expected 'section-1', got %q", hash)
	}
	if gotQuery.Get("page") != "1" {
		t.Error("query parameter 'page' not set correctly")
	}
	if gotQuery.Get("sort") != "name" {
		t.Error("query parameter 'sort' not set correctly")
	}
}

func TestRequestController_SetInitialLocation_NilController(t *testing.T) {
	var rc *RequestController
	query := make(url.Values)

	rc.SetInitialLocation("/test", query, "hash")
}

func TestRequestController_GetInitialLocation(t *testing.T) {
	rc := NewRequestController()

	path, query, hash := rc.GetInitialLocation()
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
	if query == nil {
		t.Error("expected non-nil query")
	}
	if hash != "" {
		t.Errorf("expected empty hash, got %q", hash)
	}

	origQuery := make(url.Values)
	origQuery.Set("id", "123")
	rc.SetInitialLocation("/item", origQuery, "details")

	path, query, hash = rc.GetInitialLocation()
	if path != "/item" {
		t.Errorf("expected '/item', got %q", path)
	}
	if hash != "details" {
		t.Errorf("expected 'details', got %q", hash)
	}
	if query.Get("id") != "123" {
		t.Error("query not copied correctly")
	}

	query.Set("id", "456")
	_, newQuery, _ := rc.GetInitialLocation()
	if newQuery.Get("id") != "123" {
		t.Error("query was not copied")
	}
}

func TestRequestController_GetInitialLocation_NilController(t *testing.T) {
	var rc *RequestController
	path, query, hash := rc.GetInitialLocation()
	if path != "" {
		t.Error("expected empty path for nil controller")
	}
	if query != nil {
		t.Error("expected nil query for nil controller")
	}
	if hash != "" {
		t.Error("expected empty hash for nil controller")
	}
}

func TestRequestController_UpdateCookie(t *testing.T) {
	rc := NewRequestController()

	headers := make(http.Header)
	headers.Set("Cookie", "session=old123; theme=dark")
	rc.SetInitialHeaders(headers)

	val, ok := rc.Get("Cookie")
	if !ok || val != "session=old123; theme=dark" {
		t.Errorf("initial cookies not set correctly: %q", val)
	}

	rc.UpdateCookie("session", "new456")

	val, ok = rc.Get("Cookie")
	if !ok {
		t.Error("Cookie header disappeared after UpdateCookie")
	}

	req := &http.Request{Header: make(http.Header)}
	req.Header.Set("Cookie", val)
	cookies := req.Cookies()

	foundSession := false
	foundTheme := false
	for _, cookie := range cookies {
		if cookie.Name == "session" {
			if cookie.Value != "new456" {
				t.Errorf("expected session=new456, got %s=%s", cookie.Name, cookie.Value)
			}
			foundSession = true
		}
		if cookie.Name == "theme" {
			if cookie.Value != "dark" {
				t.Errorf("expected theme=dark, got %s=%s", cookie.Name, cookie.Value)
			}
			foundTheme = true
		}
	}

	if !foundSession {
		t.Error("session cookie not found after update")
	}
	if !foundTheme {
		t.Error("theme cookie was lost after update")
	}
}

func TestRequestController_UpdateCookie_NewCookie(t *testing.T) {
	rc := NewRequestController()

	headers := make(http.Header)
	headers.Set("Cookie", "existing=value")
	rc.SetInitialHeaders(headers)

	rc.UpdateCookie("newcookie", "newvalue")

	val, ok := rc.Get("Cookie")
	if !ok {
		t.Fatal("Cookie header missing")
	}

	req := &http.Request{Header: make(http.Header)}
	req.Header.Set("Cookie", val)
	cookies := req.Cookies()

	if len(cookies) != 2 {
		t.Errorf("expected 2 cookies, got %d", len(cookies))
	}
}

func TestRequestController_UpdateCookie_NilController(t *testing.T) {
	var rc *RequestController

	rc.UpdateCookie("test", "value")
}

func TestRequestController_DeleteCookieFromRequest(t *testing.T) {
	rc := NewRequestController()

	headers := make(http.Header)
	headers.Set("Cookie", "session=abc123; theme=dark; lang=en")
	rc.SetInitialHeaders(headers)

	rc.DeleteCookieFromRequest("theme")

	val, ok := rc.Get("Cookie")
	if !ok {
		t.Error("Cookie header disappeared after delete")
	}

	req := &http.Request{Header: make(http.Header)}
	req.Header.Set("Cookie", val)
	cookies := req.Cookies()

	for _, cookie := range cookies {
		if cookie.Name == "theme" {
			t.Error("theme cookie should be deleted")
		}
	}

	foundSession := false
	foundLang := false
	for _, cookie := range cookies {
		if cookie.Name == "session" {
			foundSession = true
		}
		if cookie.Name == "lang" {
			foundLang = true
		}
	}

	if !foundSession {
		t.Error("session cookie was lost")
	}
	if !foundLang {
		t.Error("lang cookie was lost")
	}
}

func TestRequestController_DeleteCookieFromRequest_LastCookie(t *testing.T) {
	rc := NewRequestController()

	headers := make(http.Header)
	headers.Set("Cookie", "onlyone=value")
	rc.SetInitialHeaders(headers)

	rc.DeleteCookieFromRequest("onlyone")

	val, ok := rc.Get("Cookie")
	if ok && val != "" {
		t.Errorf("expected empty Cookie header after deleting last cookie, got %q", val)
	}
}

func TestRequestController_DeleteCookieFromRequest_NonExistent(t *testing.T) {
	rc := NewRequestController()

	headers := make(http.Header)
	headers.Set("Cookie", "existing=value")
	rc.SetInitialHeaders(headers)

	rc.DeleteCookieFromRequest("nonexistent")

	val, ok := rc.Get("Cookie")
	if !ok || val != "existing=value" {
		t.Error("existing cookie should remain unchanged")
	}
}

func TestRequestController_DeleteCookieFromRequest_NilController(t *testing.T) {
	var rc *RequestController

	rc.DeleteCookieFromRequest("test")
}

func TestRequestController_ConcurrentAccess(t *testing.T) {
	rc := NewRequestController()

	headers := make(http.Header)
	headers.Set("X-Test", "value")
	rc.SetInitialHeaders(headers)

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				rc.Get("X-Test")
				rc.GetResponseHeaders()
				rc.GetRedirect()
				rc.IsLive()
				rc.GetInitialLocation()
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				rc.Set("X-Header", "value")
				rc.SetRedirect("/test", http.StatusFound)
				rc.SetIsLive(true)
				rc.UpdateCookie("test", "value")
				rc.DeleteCookieFromRequest("test")
			}
			done <- true
		}(i)
	}

	for i := 0; i < 20; i++ {
		<-done
	}
}
