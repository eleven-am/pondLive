package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	livehttp "github.com/eleven-am/go/pondlive/internal/server/http"
	ui "github.com/eleven-am/go/pondlive/pkg/live"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
)

func TestTailwindCounterSSR(t *testing.T) {
	mgr := livehttp.NewManager(&livehttp.ManagerConfig{
		IDGenerator: func(*http.Request) (ui.SessionID, error) {
			return ui.SessionID("example-session"), nil
		},
		Component: func(ctx ui.Ctx, _ struct{}) h.Node {
			return counter(ctx)
		},
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com/", nil)
	rec := httptest.NewRecorder()
	mgr.ServeHTTP(rec, req)

	res := rec.Result()
	t.Cleanup(func() { _ = res.Body.Close() })

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 response, got %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	content := string(body)

	for _, want := range []string{
		"href=\"/static/tailwind.css\"",
		"LiveUI Tailwind Counter",
		"<button",
		"<script src=\"/pondlive.js\" defer></script>",
		"src=\"/pondlive.js\"",
		"LiveUI Tailwind Counter",
		"<button",
		"<script id=\"live-boot\"",
		"\"sid\":\"example-session\"",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("expected SSR response to contain %q, body=%s", want, content)
		}
	}
}
