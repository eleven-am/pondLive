package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
	routerui "github.com/eleven-am/pondlive/go/pkg/live/router"
)

func appComponent(ctx runtime.Ctx, _ struct{}) h.Node {
	search := routerui.UseSearch(ctx)
	tab := search.Get("tab")
	routerui.UseMetadata(ctx, &runtime.Meta{
		Title:       "Profile",
		Description: "Viewing profile for user",
		Meta: []h.MetaTag{{
			Property: "og:title",
			Content:  "Profile",
		}},
		Links: []h.LinkTag{{
			Rel:  "canonical",
			Href: "/users",
		}},
	})
	return runtime.WithMetadata(
		h.Div(
			h.Data("id", ""),
			h.Data("tab", tab),
			h.Text("profile"),
		),
		&runtime.Meta{
			Scripts: []h.ScriptTag{{
				Src:   "https://cdn.example.com/analytics.js",
				Defer: true,
			}},
		},
	)
}

func TestManagerServeHTTP(t *testing.T) {
	mgr := NewManager(&ManagerConfig{
		IDGenerator: func(*http.Request) (runtime.SessionID, error) {
			return runtime.SessionID("fixed"), nil
		},
		Component: appComponent,
	})
	mgr.SetClientConfig(protocol.ClientConfig{Endpoint: "/ws"})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/users/42?tab=info", nil)
	rec := httptest.NewRecorder()

	mgr.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); ct != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", ct)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	content := string(body)
	checks := map[string]string{
		"doctype":         "<!DOCTYPE html>",
		"htmlLang":        "<html lang=\"en\">",
		"metaCharset":     "<meta charset=\"utf-8\">",
		"metaViewport":    "<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">",
		"title":           "<title>Profile</title>",
		"description":     "<meta name=\"description\" content=\"Viewing profile for user\" data-live-head=\"description\" data-live-key=\"description\">",
		"ogTitle":         "<meta content=\"Profile\" property=\"og:title\" data-live-head=\"meta\" data-live-key=\"meta:property:og:title\">",
		"canonicalLink":   "<link rel=\"canonical\" href=\"/users\" data-live-head=\"link\" data-live-key=\"link:rel:canonical|href:/users\">",
		"analyticsScript": "<script src=\"https://cdn.example.com/analytics.js\" defer data-live-head=\"script\" data-live-key=\"script:src:https://cdn.example.com/analytics.js\"></script>",
		"bundleScript":    "<script src=\"/pondlive.js\" defer></script>",
		"bootScript":      "<script id=\"live-boot\" type=\"application/json\">",
		"bootSession":     "\"sid\":\"fixed\"",
		"bootPath":        "\"path\":\"/users/42\"",
	}
	for name, needle := range checks {
		if !strings.Contains(content, needle) {
			t.Fatalf("expected SSR response to contain %s (%q), body=%s", name, needle, content)
		}
	}

	if bodyIdx := strings.Index(content, "<script id=\"live-boot\""); bodyIdx != -1 {
		if endIdx := strings.Index(content, "</body>"); endIdx != -1 && bodyIdx > endIdx {
			t.Fatalf("expected boot script to appear before </body>, body=%s", content)
		}
	}
	if !strings.Contains(content, "\"endpoint\":\"/ws\"") {
		t.Fatalf("expected boot payload to include client endpoint, body=%s", content)
	}

	sess, ok := mgr.Registry().Lookup(runtime.SessionID("fixed"))
	if !ok {
		t.Fatal("expected session to be registered")
	}
	loc := sess.Location()
	if loc.Path != "/users/42" {
		t.Fatalf("unexpected session path: %s", loc.Path)
	}
	if loc.Query != "tab=info" {
		t.Fatalf("expected query to be tracked, got %q", loc.Query)
	}
	if len(loc.Params) != 0 {
		t.Fatalf("expected route params to be empty, got %v", loc.Params)
	}
}

func TestManagerClientAssetOverride(t *testing.T) {
	mgr := NewManager(&ManagerConfig{ClientAssetURL: "https://cdn.example.com/liveui.js", Component: appComponent})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()

	mgr.ServeHTTP(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	if !strings.Contains(string(body), "<script src=\"https://cdn.example.com/live.js\"></script>") {
		t.Fatalf("expected custom asset script tag, body=%s", string(body))
	}
}

func TestManagerNoMatchReturns404(t *testing.T) {
	mgr := NewManager(nil)

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()

	mgr.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when component is missing, got %d", res.StatusCode)
	}
}
