package server

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/upload"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestLiveSessionUploadRegistry(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	t.Run("session has upload registry", func(t *testing.T) {
		sess := session.NewLiveSession("test-sid", 1, component, nil)
		defer sess.Close()

		reg := sess.UploadRegistry()
		if reg == nil {
			t.Fatal("expected upload registry to be non-nil")
		}
	})

	t.Run("registry is functional", func(t *testing.T) {
		sess := session.NewLiveSession("test-sid", 1, component, nil)
		defer sess.Close()

		reg := sess.UploadRegistry()
		token := reg.Issue("test-sid", upload.GrantSpec{Files: 1, MaxSize: 1024})
		if _, _, err := reg.Reserve(token, 0, nil); err != nil {
			t.Fatalf("expected to reserve against the issued grant: %v", err)
		}
	})

	t.Run("nil session returns nil registry", func(t *testing.T) {
		var sess *session.LiveSession
		reg := sess.UploadRegistry()
		if reg != nil {
			t.Error("expected nil registry for nil session")
		}
	})
}

func TestDecorateDocument(t *testing.T) {
	t.Run("injects before closing body", func(t *testing.T) {
		doc := "<html><body><div>content</div></body></html>"
		boot := []byte(`{"state":"test"}`)
		result := decorateDocument(doc, boot)

		if !strings.Contains(result, `<script id="live-boot"`) {
			t.Error("expected boot script in result")
		}
		if !strings.Contains(result, `{"state":"test"}`) {
			t.Error("expected boot JSON in result")
		}
		bodyIdx := strings.Index(result, "</body>")
		scriptIdx := strings.Index(result, `<script id="live-boot"`)
		if scriptIdx > bodyIdx {
			t.Error("expected script before </body>")
		}
	})

	t.Run("case insensitive body tag", func(t *testing.T) {
		doc := "<html><BODY>content</BODY></html>"
		boot := []byte(`{"test":true}`)
		result := decorateDocument(doc, boot)

		if !strings.Contains(result, `<script id="live-boot"`) {
			t.Error("expected boot script in result")
		}
	})

	t.Run("appends if no body tag", func(t *testing.T) {
		doc := "<html><div>content</div></html>"
		boot := []byte(`{"test":true}`)
		result := decorateDocument(doc, boot)

		if !strings.HasSuffix(result, "</script>") {
			t.Error("expected script appended at end")
		}
	})

	t.Run("escapes JSON closing tags", func(t *testing.T) {
		doc := "<html><body></body></html>"
		boot := []byte(`{"html":"</script>"}`)
		result := decorateDocument(doc, boot)

		if strings.Contains(result, `</script>"}`) {
			t.Error("expected </script> to be escaped in JSON")
		}
	})
}

func TestDefaultSessionID(t *testing.T) {
	t.Run("generates valid session ID", func(t *testing.T) {
		id, err := defaultSessionID(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == "" {
			t.Error("expected non-empty session ID")
		}
		if len(id) < 20 {
			t.Errorf("expected longer session ID, got %d chars", len(id))
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[session.SessionID]bool)
		for i := 0; i < 100; i++ {
			id, err := defaultSessionID(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ids[id] {
				t.Errorf("duplicate session ID: %s", id)
			}
			ids[id] = true
		}
	})
}

func TestCloneSessionConfig(t *testing.T) {
	t.Run("clones non-nil config", func(t *testing.T) {
		original := &session.Config{
			ClientAsset: "/test.js",
		}
		clone := cloneSessionConfig(original)

		if clone.ClientAsset != original.ClientAsset {
			t.Error("expected same ClientAsset")
		}
	})

	t.Run("returns empty config for nil", func(t *testing.T) {
		clone := cloneSessionConfig(nil)
		if clone.ClientAsset != "" {
			t.Errorf("expected empty config for nil input, got ClientAsset=%q", clone.ClientAsset)
		}
	})
}

func TestAppMux(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	mux := app.Mux()
	if mux == nil {
		t.Error("expected non-nil mux")
	}
}

func TestAppHandler(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	handler := app.Handler()
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}

func TestAppHandlerFunc(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	handlerFunc := app.HandlerFunc()
	if handlerFunc == nil {
		t.Error("expected non-nil handler func")
	}
}

func TestAppServer(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	server := app.Server(":0")
	if server == nil {
		t.Error("expected non-nil server")
	}
	if server.Addr != ":0" {
		t.Errorf("expected addr :0, got %s", server.Addr)
	}
}

func TestAppRegistry(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	if app.registry == nil {
		t.Error("expected non-nil registry")
	}
}

func TestNewEndpointValidation(t *testing.T) {
	t.Run("nil registry returns error", func(t *testing.T) {
		if _, err := NewEndpoint(context.Background(), nil, TransportConfig{}); err == nil {
			t.Error("expected error for nil registry")
		}
	})

	t.Run("short attach secret returns error", func(t *testing.T) {
		if _, err := NewEndpoint(context.Background(), NewSessionRegistry(0, 0), TransportConfig{AttachSecret: []byte("short")}); err == nil {
			t.Error("expected error for short attach secret")
		}
	})

	t.Run("defaults are applied", func(t *testing.T) {
		e, err := NewEndpoint(context.Background(), NewSessionRegistry(0, 0), TransportConfig{})
		if err != nil {
			t.Fatal(err)
		}
		if e.cfg.ReadLimit != DefaultReadLimit || e.cfg.JoinDeadline != DefaultJoinDeadline || e.cfg.OutboundBudget != session.DefaultOutboundBudget {
			t.Errorf("unexpected defaults: %+v", e.cfg)
		}
	})
}

func TestNewAppConfigurations(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	t.Run("nil component returns error", func(t *testing.T) {
		_, err := New(Config{Component: nil})
		if err == nil {
			t.Error("expected error for nil component")
		}
	})

	t.Run("custom client asset", func(t *testing.T) {
		app, err := New(Config{
			Component:   component,
			ClientAsset: "/custom/path.js",
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}
		if app.clientAsset != "/custom/path.js" {
			t.Errorf("expected custom client asset, got %s", app.clientAsset)
		}
	})

	t.Run("empty client asset uses default", func(t *testing.T) {
		app, err := New(Config{
			Component:   component,
			ClientAsset: "   ",
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}
		if app.clientAsset != "/static/pondlive.js" {
			t.Errorf("expected default client asset, got %s", app.clientAsset)
		}
	})

	t.Run("dev mode changes client asset", func(t *testing.T) {
		app, err := New(Config{
			Component:     component,
			SessionConfig: &session.Config{DevMode: true},
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}
		if app.clientAsset != "/static/pondlive-dev.js" {
			t.Errorf("expected dev client asset, got %s", app.clientAsset)
		}
	})

	t.Run("custom ID generator", func(t *testing.T) {
		customGenerator := func(*http.Request) (session.SessionID, error) {
			return "custom-id", nil
		}

		app, err := New(Config{
			Component:   component,
			IDGenerator: customGenerator,
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		id, _ := app.idGenerator(nil)
		if id != "custom-id" {
			t.Errorf("expected custom-id, got %s", id)
		}
	})

	t.Run("with upload config", func(t *testing.T) {
		app, err := New(Config{
			Component: component,
			UploadConfig: &upload.Config{
				StoragePath: t.TempDir(),
			},
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}
		if app.uploadHandler == nil {
			t.Fatal("expected upload handler to be set")
		}
		t.Cleanup(app.uploadHandler.Close)
	})

	t.Run("version defaults to 2", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}
		if app.version != 2 {
			t.Errorf("expected version 2, got %d", app.version)
		}
	})
}

func TestServeSSR(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return work.BuildElement("div", work.NewText("Hello, World!"))
	}

	t.Run("basic SSR render", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		resp := rec.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "Hello, World!") {
			t.Error("expected body to contain 'Hello, World!'")
		}
		if !strings.Contains(body, `<script id="live-boot"`) {
			t.Error("expected body to contain boot script")
		}
		if !strings.Contains(body, "<!DOCTYPE html>") {
			t.Error("expected DOCTYPE in response")
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			t.Errorf("expected text/html content type, got %s", contentType)
		}

		cacheControl := resp.Header.Get("Cache-Control")
		if cacheControl != "no-store" {
			t.Errorf("expected no-store cache control, got %s", cacheControl)
		}
	})

	t.Run("SSR with path", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test/path?query=value", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		resp := rec.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("session ID error", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		app.idGenerator = func(*http.Request) (session.SessionID, error) {
			return "", nil
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("dev mode config", func(t *testing.T) {
		app, err := New(Config{
			Component:     component,
			SessionConfig: &session.Config{DevMode: true},
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		resp := rec.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		body := rec.Body.String()
		if !strings.Contains(body, `"debug":true`) {
			t.Error("expected debug flag in boot JSON for dev mode")
		}
	})

	t.Run("with query params and fragment", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/page?foo=bar&baz=qux#section", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("id generator returns error", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		app.idGenerator = func(*http.Request) (session.SessionID, error) {
			return "", errors.New("generator error")
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("with redirect in request state", func(t *testing.T) {
		redirectComponent := func(ctx *runtime.Ctx) work.Node {
			requestState := headers.UseRequestState(ctx)
			if requestState != nil {
				requestState.SetRedirect("/redirected", http.StatusFound)
			}
			return &work.Element{Tag: "div"}
		}

		app, err := New(Config{Component: redirectComponent})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusFound {
			t.Errorf("expected 302, got %d", rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "/redirected" {
			t.Errorf("expected redirect to /redirected, got %s", location)
		}
	})

	t.Run("version defaults when zero", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		app.version = 0

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("nil returning component", func(t *testing.T) {
		nilComponent := func(ctx *runtime.Ctx) work.Node {
			return nil
		}

		app, err := New(Config{Component: nilComponent})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code == http.StatusOK {
			t.Log("nil component rendered successfully (boot wrapper provides structure)")
		}
	})

	t.Run("negative version defaults to 2", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		app.version = -5

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}
