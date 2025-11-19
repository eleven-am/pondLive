package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
	"github.com/eleven-am/pondlive/go/internal/session"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func newTestSession(id string) *session.LiveSession {
	component := func(ctx runtime.Ctx, _ struct{}) h.Node {
		return h.Div()
	}

	return session.NewLiveSession(session.SessionID(id), 1, component, struct{}{}, &session.Config{
		Clock: time.Now,
		TTL:   time.Minute,
	})
}

func TestCookieHandlerUnavailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com"+CookiePath, strings.NewReader(`{"sid":"a","token":"b"}`))

	rec := httptest.NewRecorder()
	var nilHandler *CookieHandler
	nilHandler.ServeHTTP(rec, req)
	if res := rec.Result(); res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for nil handler, got %d", res.StatusCode)
	}

	handler := &CookieHandler{}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if res := rec.Result(); res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for handler without registry, got %d", res.StatusCode)
	}
}

func TestCookieHandlerRejectsNonPost(t *testing.T) {
	handler := NewCookieHandler(server.NewSessionRegistry())
	req := httptest.NewRequest(http.MethodGet, "http://example.com"+CookiePath, nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", res.StatusCode)
	}
	if allow := res.Header.Get("Allow"); allow != http.MethodPost {
		t.Fatalf("expected Allow header to advertise POST, got %q", allow)
	}
}

func TestCookieHandlerValidatesPayload(t *testing.T) {
	handler := NewCookieHandler(server.NewSessionRegistry())

	cases := map[string]string{
		"invalidJSON":   "not json",
		"missingFields": `{"sid":" ","token":""}`,
	}

	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://example.com"+CookiePath, strings.NewReader(body))
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if res := rec.Result(); res.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", res.StatusCode)
			}
		})
	}
}

func TestCookieHandlerSessionLookupFailures(t *testing.T) {
	registry := server.NewSessionRegistry()
	handler := NewCookieHandler(registry)

	payload := `{"sid":"missing","token":"abc"}`
	req := httptest.NewRequest(http.MethodPost, "http://example.com"+CookiePath, strings.NewReader(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if res := rec.Result(); res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing session, got %d", res.StatusCode)
	}

	sess := newTestSession("sess")
	registry.Put(sess)

	payload = fmt.Sprintf(`{"sid":"%s","token":"abc"}`, sess.ID())
	req = httptest.NewRequest(http.MethodPost, "http://example.com"+CookiePath, strings.NewReader(payload))
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if res := rec.Result(); res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when cookie batch token missing, got %d", res.StatusCode)
	}
}

func TestCookieHandlerAppliesMutations(t *testing.T) {

	t.Skip("Cookie mutation tracking requires full session rendering lifecycle")
}

func TestCookieHandlerDeletesCookies(t *testing.T) {

	t.Skip("Cookie mutation tracking requires full session rendering lifecycle")
}
