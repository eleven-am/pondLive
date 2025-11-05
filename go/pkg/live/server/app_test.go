package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	livehttp "github.com/eleven-am/pondlive/go/internal/server/http"
	ui "github.com/eleven-am/pondlive/go/pkg/live"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestNewAppBuildsHandlers(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	app, err := NewApp(ctx, func(ctx ui.Ctx) h.Node {
		ui.UseEffect(ctx, func() ui.Cleanup { return nil })
		return h.Div(h.Text("ok"))
	})
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()
	handler := app.Handler()
	if handler == nil {
		t.Fatal("expected handler to be initialised")
	}
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", res.StatusCode)
	}

	if body := rec.Body.String(); body == "" {
		t.Fatalf("expected body to be populated")
	}
	if !strings.Contains(rec.Body.String(), "\"endpoint\":\"/live\"") {
		t.Fatalf("expected boot payload to include pondsocket endpoint, body=%s", rec.Body.String())
	}
}

func TestNewAppRequiresComponent(t *testing.T) {
	if _, err := NewApp(context.Background(), nil); err == nil {
		t.Fatal("expected error when component missing")
	}
}

func TestTrimPatternPrefix(t *testing.T) {
	cases := map[string]string{
		"/live/:sid":   "/live/",
		"/live/*rest":  "/live/",
		"/ws":          "/ws/",
		"":             "/",
		"/socket":      "/socket/",
		"/prefix/:sid": "/prefix/",
	}
	for input, want := range cases {
		if got := trimPatternPrefix(input); got != want {
			t.Fatalf("trimPatternPrefix(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestEndpointFromPrefix(t *testing.T) {
	cases := map[string]string{
		"/live/":   "/live",
		"/ws/":     "/ws",
		"/":        "/",
		"":         "/",
		"/nested/": "/nested",
	}
	for input, want := range cases {
		if got := endpointFromPrefix(input); got != want {
			t.Fatalf("endpointFromPrefix(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestAppServesClientScript(t *testing.T) {
	app, err := NewApp(context.Background(), func(ctx ui.Ctx) h.Node {
		return h.Div(h.Text("asset"))
	})
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/pondlive.js", nil)
	rec := httptest.NewRecorder()

	handler := app.Handler()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "window.LiveUI") {
		snippet := body
		if len(snippet) > 64 {
			snippet = snippet[:64]
		}
		t.Fatalf("expected bundled client script to be returned, got %q", snippet)
	}
}

func TestAppRegistersCookieHandler(t *testing.T) {
	app, err := NewApp(context.Background(), func(ctx ui.Ctx) h.Node {
		return h.Div(h.Text("cookie"))
	})
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com"+livehttp.CookiePath, nil)
	rec := httptest.NewRecorder()

	handler := app.Handler()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", res.StatusCode)
	}
	if allow := res.Header.Get("Allow"); allow != http.MethodPost {
		t.Fatalf("expected Allow header to advertise POST, got %q", allow)
	}
}
