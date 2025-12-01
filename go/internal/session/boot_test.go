package session

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

func TestRequestInfoFetchedDynamicallyFromTransport(t *testing.T) {
	component := func(_ *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	sess := NewLiveSession("test-session", 1, component, nil)
	defer sess.Close()

	sess.transportMu.RLock()
	initialTransport := sess.transport
	sess.transportMu.RUnlock()

	if initialTransport != nil {
		t.Error("expected no transport initially")
	}

	req := httptest.NewRequest(http.MethodGet, "/test?foo=bar", nil)
	req.Header.Set("User-Agent", "test-agent")
	transport := NewSSRTransport(req)
	sess.SetTransport(transport)

	sess.transportMu.RLock()
	currentTransport := sess.transport
	sess.transportMu.RUnlock()

	if currentTransport == nil {
		t.Fatal("expected transport after SetTransport")
	}

	info := currentTransport.RequestInfo()
	if info == nil {
		t.Fatal("expected RequestInfo to be set")
	}
	if info.Path != "/test" {
		t.Errorf("expected path /test, got %s", info.Path)
	}
	if info.Query.Get("foo") != "bar" {
		t.Errorf("expected query foo=bar, got %s", info.Query.Get("foo"))
	}
}

func TestTransportUpdateReflectedInSession(t *testing.T) {
	component := func(_ *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	sess := NewLiveSession("test-session", 1, component, nil)
	defer sess.Close()

	req1 := httptest.NewRequest(http.MethodGet, "/path1", nil)
	transport1 := NewSSRTransport(req1)
	sess.SetTransport(transport1)

	sess.transportMu.RLock()
	t1 := sess.transport
	sess.transportMu.RUnlock()

	if t1.RequestInfo().Path != "/path1" {
		t.Errorf("expected path /path1, got %s", t1.RequestInfo().Path)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/path2", nil)
	transport2 := NewSSRTransport(req2)
	sess.SetTransport(transport2)

	sess.transportMu.RLock()
	t2 := sess.transport
	sess.transportMu.RUnlock()

	if t2.RequestInfo().Path != "/path2" {
		t.Errorf("expected path /path2, got %s", t2.RequestInfo().Path)
	}
}
