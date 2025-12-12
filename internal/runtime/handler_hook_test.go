package runtime

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUseHandlerRegistersAndServes(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root

	ctx := &Ctx{instance: root, session: sess}

	h := UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("ok"))
		return nil
	})

	if h.URL() != "/_handlers/sess1/root:h0" {
		t.Fatalf("unexpected URL: %s", h.URL())
	}

	req := httptest.NewRequest(http.MethodGet, h.URL(), nil)
	rr := httptest.NewRecorder()
	sess.ServeHTTP(rr, req)

	if rr.Code != http.StatusTeapot {
		t.Fatalf("expected 418, got %d", rr.Code)
	}
	if body := rr.Body.String(); body != "ok" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestUseHandlerStoresHandlerHookType(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root
	ctx := &Ctx{instance: root, session: sess}

	_ = UseHandler(ctx, http.MethodGet)

	if got := root.HookFrame[0].Type; got != HookTypeHandler {
		t.Fatalf("expected HookTypeHandler, got %v", got)
	}
}

func TestUseHandlerUpdatesOnRerender(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root
	ctx := &Ctx{instance: root, session: sess}

	h := UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		_, _ = w.Write([]byte("v1"))
		return nil
	})

	ctx.hookIndex = 0
	_ = UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		_, _ = w.Write([]byte("v2"))
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, h.URL(), nil)
	rr := httptest.NewRecorder()
	sess.ServeHTTP(rr, req)

	if body := rr.Body.String(); body != "v2" {
		t.Fatalf("expected updated handler body v2, got %q", body)
	}
}

func TestUseHandlerMethodGuard(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root
	ctx := &Ctx{instance: root, session: sess}

	h := UseHandler(ctx, http.MethodPost, func(w http.ResponseWriter, r *http.Request) error {
		_, _ = w.Write([]byte("ok"))
		return nil
	})

	req := httptest.NewRequest(http.MethodGet, h.URL(), nil)
	rr := httptest.NewRecorder()
	sess.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rr.Code)
	}
	if allow := rr.Header().Get("Allow"); allow != http.MethodPost {
		t.Fatalf("expected Allow header %s, got %s", http.MethodPost, allow)
	}
}

func TestUseHandlerPanicRecovery(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root
	ctx := &Ctx{instance: root, session: sess}

	h := UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, h.URL(), nil)
	rr := httptest.NewRecorder()
	sess.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestUseHandlerDestroy(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root
	ctx := &Ctx{instance: root, session: sess}

	h := UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		_, _ = w.Write([]byte("ok"))
		return nil
	})

	h.Destroy()

	req := httptest.NewRequest(http.MethodGet, h.URL(), nil)
	rr := httptest.NewRecorder()
	sess.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after destroy, got %d", rr.Code)
	}
}

func TestUseHandlerCleanupOnUnmount(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root
	ctx := &Ctx{instance: root, session: sess}

	h := UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		_, _ = w.Write([]byte("ok"))
		return nil
	})

	sess.cleanupInstance(root)

	req := httptest.NewRequest(http.MethodGet, h.URL(), nil)
	rr := httptest.NewRecorder()
	sess.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("expected 404 after cleanup, got %d", rr.Code)
	}
}

func TestUseHandler500OnErrorAndNoWrite(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root
	ctx := &Ctx{instance: root, session: sess}

	h := UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		return fmt.Errorf("fail")
	})

	req := httptest.NewRequest(http.MethodGet, h.URL(), nil)
	rr := httptest.NewRecorder()
	sess.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rr.Code)
	}
}

func TestUseHandlerSkipsNilHandlers(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root
	ctx := &Ctx{instance: root, session: sess}

	h := UseHandler(ctx, http.MethodGet,
		nil,
		func(w http.ResponseWriter, r *http.Request) error {
			_, _ = w.Write([]byte("ok"))
			return nil
		},
	)

	req := httptest.NewRequest(http.MethodGet, h.URL(), nil)
	rr := httptest.NewRecorder()
	sess.ServeHTTP(rr, req)

	if body := strings.TrimSpace(rr.Body.String()); body != "ok" {
		t.Fatalf("expected ok, got %q", body)
	}
}

func TestUseHandlerCleanupRegisteredOnceOnMount(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root

	for i := 0; i < 10; i++ {
		ctx := &Ctx{instance: root, session: sess, hookIndex: 0}
		_ = UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
			return nil
		})
	}

	root.mu.Lock()
	cleanupCount := len(root.cleanups)
	root.mu.Unlock()

	if cleanupCount != 1 {
		t.Errorf("expected 1 cleanup registered (on mount only), got %d", cleanupCount)
	}
}

func TestUseHandlerCleanupNotAccumulatingOnRerender(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root", HookFrame: []HookSlot{}}
	sess.Root = root

	ctx := &Ctx{instance: root, session: sess, hookIndex: 0}
	_ = UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	initialCleanups := len(root.cleanups)

	for i := 0; i < 50; i++ {
		ctx := &Ctx{instance: root, session: sess, hookIndex: 0}
		_ = UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
			return nil
		})
	}

	finalCleanups := len(root.cleanups)

	if finalCleanups != initialCleanups {
		t.Errorf("cleanups grew from %d to %d during re-renders", initialCleanups, finalCleanups)
	}
}

func TestUseHandlerMultipleHooksEachGetOneCleanup(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root", HookFrame: []HookSlot{}}
	sess.Root = root

	ctx := &Ctx{instance: root, session: sess, hookIndex: 0}
	_ = UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})
	_ = UseHandler(ctx, http.MethodPost, func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})
	_ = UseHandler(ctx, http.MethodPut, func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	if len(root.cleanups) != 3 {
		t.Errorf("expected 3 cleanups for 3 handlers, got %d", len(root.cleanups))
	}

	for i := 0; i < 10; i++ {
		ctx := &Ctx{instance: root, session: sess, hookIndex: 0}
		_ = UseHandler(ctx, http.MethodGet)
		_ = UseHandler(ctx, http.MethodPost)
		_ = UseHandler(ctx, http.MethodPut)
	}

	if len(root.cleanups) != 3 {
		t.Errorf("cleanups should remain at 3 after re-renders, got %d", len(root.cleanups))
	}
}

func TestHandlerHandleGenerateToken(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root
	ctx := &Ctx{instance: root, session: sess}

	h := UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		return nil
	})

	token1 := h.GenerateToken()
	if token1 == "" {
		t.Fatal("expected non-empty token")
	}
	if len(token1) != 32 {
		t.Errorf("expected token length 32, got %d", len(token1))
	}

	token2 := h.GenerateToken()
	if token2 == "" {
		t.Fatal("expected non-empty second token")
	}
	if token1 == token2 {
		t.Error("expected different tokens on each call")
	}
}

func TestHandlerHandleGenerateTokenNilEntry(t *testing.T) {
	h := HandlerHandle{entry: nil}
	token := h.GenerateToken()
	if token == "" {
		t.Error("GenerateToken should work even with nil entry")
	}
}

func TestHandlerHandleURLNilEntry(t *testing.T) {
	h := HandlerHandle{entry: nil}
	url := h.URL()
	if url != "" {
		t.Errorf("expected empty URL for nil entry, got %q", url)
	}
}

func TestHandlerHandleDestroyNilEntry(t *testing.T) {
	h := HandlerHandle{entry: nil}
	h.Destroy()
}

func TestServeHTTPNilSession(t *testing.T) {
	var sess *Session
	req := httptest.NewRequest(http.MethodGet, "/_handlers/sess/handler", nil)
	rr := httptest.NewRecorder()
	sess.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for nil session, got %d", rr.Code)
	}
}

func TestServeHTTPInvalidPath(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root

	tests := []struct {
		name string
		path string
	}{
		{"empty path", "/"},
		{"wrong prefix", "/other/sess1/handler"},
		{"too few parts", "/_handlers/sess1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rr := httptest.NewRecorder()
			sess.ServeHTTP(rr, req)
			if rr.Code != http.StatusNotFound {
				t.Errorf("expected 404, got %d", rr.Code)
			}
		})
	}
}

func TestServeHTTPWrongSessionID(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	root := &Instance{ID: "root"}
	sess.Root = root

	req := httptest.NewRequest(http.MethodGet, "/_handlers/wrong-sess/handler", nil)
	rr := httptest.NewRecorder()
	sess.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for wrong session ID, got %d", rr.Code)
	}
}

func TestRegisterHTTPHandlerNilSession(t *testing.T) {
	var sess *Session
	entry := sess.registerHTTPHandler(&Instance{ID: "test"}, 0, "GET", nil)
	if entry != nil {
		t.Error("expected nil entry for nil session")
	}
}

func TestRegisterHTTPHandlerNilInstance(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	entry := sess.registerHTTPHandler(nil, 0, "GET", nil)
	if entry != nil {
		t.Error("expected nil entry for nil instance")
	}
}

func TestUpdateHTTPHandlerNilSession(t *testing.T) {
	var sess *Session
	sess.updateHTTPHandler(&handlerEntry{}, "GET", nil)
}

func TestUpdateHTTPHandlerNilEntry(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	sess.updateHTTPHandler(nil, "GET", nil)
}

func TestRemoveHTTPHandlerNilSession(t *testing.T) {
	var sess *Session
	sess.removeHTTPHandler("handler-id")
}

func TestRemoveHTTPHandlerEmptyID(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	sess.removeHTTPHandler("")
}

func TestFindHTTPHandlerNilSession(t *testing.T) {
	var sess *Session
	entry := sess.findHTTPHandler("handler-id")
	if entry != nil {
		t.Error("expected nil for nil session")
	}
}

func TestFindHTTPHandlerEmptyID(t *testing.T) {
	sess := &Session{SessionID: "sess1"}
	entry := sess.findHTTPHandler("")
	if entry != nil {
		t.Error("expected nil for empty ID")
	}
}
