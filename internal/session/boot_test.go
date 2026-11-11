package session

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestRequestStateComesFromTheSSRRequest(t *testing.T) {
	var gotPath, gotQuery, gotAgent string
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}
	req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Authorization", "secret")

	sess := NewLiveSession("test-session", 1, component, nil, WithRequest(req))
	defer sess.Close()

	info := sess.requestState.Info()
	gotPath, gotQuery = info.Path, info.Query.Get("foo")
	gotAgent = info.Headers.Get("User-Agent")
	if gotPath != "/test" || gotQuery != "bar" || gotAgent != "test-agent" {
		t.Fatalf("request state = %q %q %q", gotPath, gotQuery, gotAgent)
	}
	if info.Headers.Get("Authorization") != "" {
		t.Fatal("Authorization must not be retained")
	}
}

func TestRetainedHeadersConfigReachesTheRequestState(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant", "acme")

	sess := NewLiveSession("test-session", 1, dummyComponent, &Config{RetainedHeaders: []string{"X-Tenant"}}, WithRequest(req))
	defer sess.Close()

	if got := sess.requestState.Info().Headers.Get("X-Tenant"); got != "acme" {
		t.Fatalf("configured header = %q, want acme", got)
	}

	plain := NewLiveSession("test-session-2", 1, dummyComponent, nil, WithRequest(req))
	defer plain.Close()
	if got := plain.requestState.Info().Headers.Get("X-Tenant"); got != "" {
		t.Fatalf("unconfigured header retained: %q", got)
	}
}

func TestAttachMergesUpgradeHeadersAndMarksTheStateLive(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil, WithRequest(httptest.NewRequest(http.MethodGet, "/page?x=1", nil)))
	defer sess.Close()

	if sess.requestState.IsLive() {
		t.Fatal("request state must not be live before attach")
	}
	upgrade := http.Header{"Cookie": {"sid=abc"}}
	if _, err := sess.Attach(context.Background(), newRecordingTransport(), upgrade, 0, false); err != nil {
		t.Fatal(err)
	}
	if !sess.requestState.IsLive() {
		t.Fatal("request state must be live when Attach returns")
	}
	if v, ok := sess.requestState.GetCookie("sid"); !ok || v != "abc" {
		t.Fatalf("upgrade cookie = %q %v", v, ok)
	}
	if sess.requestState.URL().Path != "/page" || sess.requestState.URL().Query.Get("x") != "1" {
		t.Fatal("page location must survive the merge")
	}
}
