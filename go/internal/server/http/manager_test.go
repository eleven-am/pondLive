package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/session"
)

func TestManagerServeHTTPFallbackDocument(t *testing.T) {
	component := func(runtime.Ctx, struct{}) *dom2.StructuredNode {
		return dom2.ElementNode("div").WithChildren(dom2.TextNode("fallback"))
	}

	mgr := NewManager(&Config[struct{}]{
		Component: component,
		IDGenerator: func(*http.Request) (session.SessionID, error) {
			return session.SessionID("sess-fallback"), nil
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/refs?q=1", nil)

	mgr.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<div>fallback</div>") {
		t.Fatalf("expected body content to render, got %q", body)
	}

	payload := extractBootPayload(t, body)
	if payload.SID != "sess-fallback" {
		t.Fatalf("expected sid sess-fallback, got %q", payload.SID)
	}
	if payload.Location.Path != "/refs" {
		t.Fatalf("expected location path /refs, got %q", payload.Location.Path)
	}
	if payload.Location.Query != "q=1" {
		t.Fatalf("expected location query q=1, got %q", payload.Location.Query)
	}
	if payload.HTML != "<div>fallback</div>" {
		t.Fatalf("expected boot html to match body, got %q", payload.HTML)
	}

	if _, ok := mgr.Registry().Lookup(session.SessionID("sess-fallback")); !ok {
		t.Fatalf("expected session registered in registry")
	}
}

func TestManagerServeHTTPFullDocument(t *testing.T) {
	component := func(runtime.Ctx, struct{}) *dom2.StructuredNode {
		return dom2.ElementNode("html").WithChildren(
			dom2.ElementNode("head").WithChildren(
				dom2.ElementNode("title").WithChildren(dom2.TextNode("example")),
			),
			dom2.ElementNode("body").WithChildren(
				dom2.ElementNode("div").WithChildren(dom2.TextNode("full-doc")),
			),
		)
	}

	mgr := NewManager(&Config[struct{}]{
		Component: component,
		IDGenerator: func(*http.Request) (session.SessionID, error) {
			return session.SessionID("sess-full"), nil
		},
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "http://example.com/full", nil)

	mgr.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "<html>") {
		t.Fatalf("expected html root to be preserved, got %q", body)
	}

	scriptIdx := strings.Index(body, `<script id="live-boot"`)
	if scriptIdx == -1 {
		t.Fatalf("expected boot script to be injected, got %q", body)
	}

	payload := extractBootPayload(t, body)
	if payload.SID != "sess-full" {
		t.Fatalf("expected sid sess-full, got %q", payload.SID)
	}
	if payload.HTML != "<div>full-doc</div>" {
		t.Fatalf("expected boot html to match body content, got %q", payload.HTML)
	}
	if !strings.HasSuffix(strings.TrimSpace(payload.Location.Path), "/full") {
		t.Fatalf("expected location path /full, got %q", payload.Location.Path)
	}
}

func extractBootPayload(t *testing.T, document string) protocol.Boot {
	t.Helper()

	start := strings.Index(document, `<script id="live-boot"`)
	if start == -1 {
		t.Fatalf("boot script not found in document: %q", document)
	}
	open := strings.Index(document[start:], ">")
	if open == -1 {
		t.Fatalf("boot script missing closing bracket")
	}
	closeIdx := strings.Index(document[start+open:], "</script>")
	if closeIdx == -1 {
		t.Fatalf("boot script missing closing tag")
	}
	payload := document[start+open+1 : start+open+closeIdx]

	var boot protocol.Boot
	if err := json.Unmarshal([]byte(payload), &boot); err != nil {
		t.Fatalf("failed to parse boot payload: %v", err)
	}
	return boot
}
