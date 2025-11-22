package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	internalserver "github.com/eleven-am/pondlive/go/internal/server"
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

	if len(body) == 0 || (!strings.Contains(body, "function") && !strings.Contains(body, "class") && !strings.Contains(body, "const") && !strings.Contains(body, "var")) {
		snippet := body
		if len(snippet) > 64 {
			snippet = snippet[:64]
		}
		t.Fatalf("expected bundled client script to be returned, got %q", snippet)
	}
}

func TestAppServesDevClientScriptAndSourceMap(t *testing.T) {
	app, err := NewApp(context.Background(), func(ctx ui.Ctx) h.Node {
		return h.Div(h.Text("dev"))
	}, WithDevMode(true))
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}

	handler := app.Handler()
	if handler == nil {
		t.Fatal("expected handler to be initialised")
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	body := rec.Body.String()

	if !strings.Contains(body, "pondlive-dev.js") {
		t.Fatalf("expected dev HTML to reference pondlive-dev.js, body=%s", body)
	}

	scriptReq := httptest.NewRequest(http.MethodGet, "http://example.com/pondlive-dev.js", nil)
	scriptRec := httptest.NewRecorder()
	handler.ServeHTTP(scriptRec, scriptReq)
	scriptRes := scriptRec.Result()
	t.Cleanup(func() { _ = scriptRes.Body.Close() })
	if scriptRes.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for dev script, got %d", scriptRes.StatusCode)
	}
	if ct := scriptRes.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/javascript") {
		t.Fatalf("expected javascript content type, got %q", ct)
	}

	mapReq := httptest.NewRequest(http.MethodGet, "http://example.com/pondlive-dev.js.map", nil)
	mapRec := httptest.NewRecorder()
	handler.ServeHTTP(mapRec, mapReq)
	mapRes := mapRec.Result()
	t.Cleanup(func() { _ = mapRes.Body.Close() })
	if mapRes.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200 for dev source map, got %d", mapRes.StatusCode)
	}
	if ct := mapRes.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("expected json content type for source map, got %q", ct)
	}
	buf := make([]byte, 1)
	n, err := mapRes.Body.Read(buf)
	if err != nil && err != io.EOF {
		t.Fatalf("expected to read from source map: %v", err)
	}
	if n == 0 {
		t.Fatal("expected source map to contain data")
	}
}

func TestAppRegistersCookieHandler(t *testing.T) {
	app, err := NewApp(context.Background(), func(ctx ui.Ctx) h.Node {
		return h.Div(h.Text("cookie"))
	})
	if err != nil {
		t.Fatalf("NewApp returned error: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com"+CookiePath, nil)
	rec := httptest.NewRecorder()

	handler := app.Handler()
	handler.ServeHTTP(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 200 or 405, got %d", res.StatusCode)
	}
}
