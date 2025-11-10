package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
	"unsafe"

	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

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

	session := runtime.NewLiveSession(runtime.SessionID("sess"), 1, func(runtime.Ctx, struct{}) h.Node { return h.Div() }, struct{}{}, nil)
	registry.Put(session)

	payload = fmt.Sprintf(`{"sid":"%s","token":"abc"}`, session.ID())
	req = httptest.NewRequest(http.MethodPost, "http://example.com"+CookiePath, strings.NewReader(payload))
	rec = httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if res := rec.Result(); res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 when cookie batch token missing, got %d", res.StatusCode)
	}
}

func TestCookieHandlerAppliesMutations(t *testing.T) {
	registry := server.NewSessionRegistry()
	handler := NewCookieHandler(registry)

	session := runtime.NewLiveSession(runtime.SessionID("sess"), 1, func(runtime.Ctx, struct{}) h.Node { return h.Div() }, struct{}{}, nil)
	registry.Put(session)

	token := "pending-token"
	queueCookieBatch(t, session, token, runtime.CookieBatch{
		Set: []*http.Cookie{
			{Name: "auth", Value: "token"},
			nil,
			{Name: "prefs", Value: "1", Path: "   "},
		},
		Delete: []string{" stale ", "   "},
	})

	payload := fmt.Sprintf(`{"sid":"%s","token":"%s"}`, session.ID(), token)
	req := httptest.NewRequest(http.MethodPost, "http://example.com"+CookiePath, strings.NewReader(payload))
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", res.StatusCode)
	}

	cookies := res.Cookies()
	if len(cookies) != 3 {
		t.Fatalf("expected 3 cookies to be written, got %d", len(cookies))
	}
	found := map[string]*http.Cookie{}
	for _, ck := range cookies {
		found[ck.Name] = ck
	}

	if ck := found["auth"]; ck == nil {
		t.Fatalf("expected auth cookie to be set, got %+v", cookies)
	} else {
		if !ck.HttpOnly {
			t.Fatal("expected auth cookie to be HttpOnly")
		}
		if ck.Path != "/" {
			t.Fatalf("expected auth cookie path to default to /, got %q", ck.Path)
		}
	}

	if ck := found["prefs"]; ck == nil {
		t.Fatalf("expected prefs cookie to be set, got %+v", cookies)
	} else {
		if ck.Path != "/" {
			t.Fatalf("expected prefs cookie path to be normalized, got %q", ck.Path)
		}
	}

	if ck := found["stale"]; ck == nil {
		t.Fatalf("expected stale cookie deletion marker, got %+v", cookies)
	} else {
		if !ck.HttpOnly {
			t.Fatal("expected stale deletion to be HttpOnly")
		}
		if ck.Path != "/" {
			t.Fatalf("expected stale deletion path to default to /, got %q", ck.Path)
		}
		if !ck.Expires.Equal(time.Unix(0, 0)) {
			t.Fatalf("expected stale deletion expiry to be unix epoch, got %s", ck.Expires)
		}
		if ck.MaxAge != -1 {
			t.Fatalf("expected stale deletion max age -1, got %d", ck.MaxAge)
		}
	}

	if _, ok := session.ConsumeCookieBatch(token); ok {
		t.Fatal("expected cookie batch to be cleared after handshake")
	}
}

func queueCookieBatch(t *testing.T, session *runtime.LiveSession, token string, batch runtime.CookieBatch) {
	t.Helper()
	if session == nil {
		t.Fatalf("session is nil")
	}
	val := reflect.ValueOf(session).Elem()
	field := val.FieldByName("cookieBatches")
	if !field.IsValid() {
		t.Fatalf("cookieBatches field missing")
	}
	mapVal := field
	if mapVal.IsNil() {
		mapVal = reflect.MakeMap(field.Type())
	}
	entryType := field.Type().Elem()
	entry := reflect.New(entryType).Elem()
	mutations := entry.FieldByName("Mutations")
	mutations.Set(reflect.ValueOf(batch))
	mapVal.SetMapIndex(reflect.ValueOf(token), entry)
	reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Set(mapVal)
}
