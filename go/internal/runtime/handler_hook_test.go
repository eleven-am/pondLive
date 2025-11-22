package runtime

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

func TestUseHandlerBasic(t *testing.T) {
	var handlerCalled bool
	var receivedMethod string

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		h := UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {
			handlerCalled = true
			receivedMethod = r.Method
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			return json.NewEncoder(w).Encode(map[string]string{"status": "success"})
		})

		if h.URL() == "" {
			t.Error("expected non-empty URL from handler")
		}

		return dom.ElementNode("div").WithChildren(dom.TextNode(h.URL()))
	}

	sess := NewSession(comp, struct{}{})
	sess.SetSessionID("test-session")
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	handlerID := fmt.Sprintf("%s:h0", sess.root.id)
	entry := sess.FindHandler(handlerID)
	if entry == nil {
		t.Fatalf("expected handler to be registered with ID %s", handlerID)
	}

	if entry.Method != "POST" {
		t.Errorf("expected method POST, got %s", entry.Method)
	}

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()

	if err := entry.Chain[0](w, req); err != nil {
		t.Fatalf("handler execution failed: %v", err)
	}

	if !handlerCalled {
		t.Error("expected handler to be called")
	}

	if receivedMethod != "POST" {
		t.Errorf("expected method POST, got %s", receivedMethod)
	}

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestUseHandlerURL(t *testing.T) {
	var capturedURL string

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		h := UseHandler(ctx, "GET", func(w http.ResponseWriter, r *http.Request) error {
			return nil
		})
		capturedURL = h.URL()
		return dom.ElementNode("div")
	}

	sess := NewSession(comp, struct{}{})
	sess.SetSessionID("my-session-123")
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	expectedURL := fmt.Sprintf("/_handlers/my-session-123/%s:h0", sess.root.id)
	if capturedURL != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, capturedURL)
	}
}

func TestUseHandlerMultipleHandlers(t *testing.T) {
	var handler1Called, handler2Called bool

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		h1 := UseHandler(ctx, "GET", func(w http.ResponseWriter, r *http.Request) error {
			handler1Called = true
			return nil
		})

		h2 := UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {
			handler2Called = true
			return nil
		})

		if h1.URL() == h2.URL() {
			t.Error("expected different URLs for different handlers")
		}

		return dom.ElementNode("div")
	}

	sess := NewSession(comp, struct{}{})
	sess.SetSessionID("test")
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	handlerID1 := fmt.Sprintf("%s:h0", sess.root.id)
	entry1 := sess.FindHandler(handlerID1)
	if entry1 == nil {
		t.Fatalf("expected first handler to be registered with ID %s", handlerID1)
	}

	req1 := httptest.NewRequest("GET", "/test", nil)
	w1 := httptest.NewRecorder()
	if err := entry1.Chain[0](w1, req1); err != nil {
		t.Fatalf("handler1 execution failed: %v", err)
	}

	if !handler1Called {
		t.Error("expected handler1 to be called")
	}

	handlerID2 := fmt.Sprintf("%s:h1", sess.root.id)
	entry2 := sess.FindHandler(handlerID2)
	if entry2 == nil {
		t.Fatalf("expected second handler to be registered with ID %s", handlerID2)
	}

	req2 := httptest.NewRequest("POST", "/test", nil)
	w2 := httptest.NewRecorder()
	if err := entry2.Chain[0](w2, req2); err != nil {
		t.Fatalf("handler2 execution failed: %v", err)
	}

	if !handler2Called {
		t.Error("expected handler2 to be called")
	}
}

func TestUseHandlerMiddleware(t *testing.T) {
	var middlewareCalled bool
	var handlerCalled bool
	var executionOrder []string

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		middleware := func(w http.ResponseWriter, r *http.Request) error {
			middlewareCalled = true
			executionOrder = append(executionOrder, "middleware")
			return nil
		}

		handler := func(w http.ResponseWriter, r *http.Request) error {
			handlerCalled = true
			executionOrder = append(executionOrder, "handler")
			return nil
		}

		UseHandler(ctx, "GET", middleware, handler)
		return dom.ElementNode("div")
	}

	sess := NewSession(comp, struct{}{})
	sess.SetSessionID("test")
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	handlerID := fmt.Sprintf("%s:h0", sess.root.id)
	entry := sess.FindHandler(handlerID)
	if entry == nil {
		t.Fatalf("expected handler to be registered with ID %s", handlerID)
	}

	if len(entry.Chain) != 2 {
		t.Fatalf("expected 2 functions in chain, got %d", len(entry.Chain))
	}

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	for _, fn := range entry.Chain {
		if err := fn(w, req); err != nil {
			t.Fatalf("chain execution failed: %v", err)
		}
	}

	if !middlewareCalled {
		t.Error("expected middleware to be called")
	}

	if !handlerCalled {
		t.Error("expected handler to be called")
	}

	if len(executionOrder) != 2 || executionOrder[0] != "middleware" || executionOrder[1] != "handler" {
		t.Errorf("expected execution order [middleware, handler], got %v", executionOrder)
	}
}

func TestUseHandlerErrorHandling(t *testing.T) {
	testErr := fmt.Errorf("handler error")

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {
			return testErr
		})
		return dom.ElementNode("div")
	}

	sess := NewSession(comp, struct{}{})
	sess.SetSessionID("test")
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	handlerID := fmt.Sprintf("%s:h0", sess.root.id)
	entry := sess.FindHandler(handlerID)
	if entry == nil {
		t.Fatalf("expected handler to be registered with ID %s", handlerID)
	}

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()

	err := entry.Chain[0](w, req)
	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}
}

func TestUseHandlerRequestBody(t *testing.T) {
	var receivedBody string

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				return err
			}
			receivedBody = string(body)
			w.WriteHeader(http.StatusOK)
			return nil
		})
		return dom.ElementNode("div")
	}

	sess := NewSession(comp, struct{}{})
	sess.SetSessionID("test")
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	handlerID := fmt.Sprintf("%s:h0", sess.root.id)
	entry := sess.FindHandler(handlerID)
	if entry == nil {
		t.Fatalf("expected handler to be registered with ID %s", handlerID)
	}

	testBody := `{"test": "data"}`
	req := httptest.NewRequest("POST", "/test", strings.NewReader(testBody))
	w := httptest.NewRecorder()

	if err := entry.Chain[0](w, req); err != nil {
		t.Fatalf("handler execution failed: %v", err)
	}

	if receivedBody != testBody {
		t.Errorf("expected body %s, got %s", testBody, receivedBody)
	}
}

func TestUseHandlerPersistence(t *testing.T) {
	callCount := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		UseHandler(ctx, "GET", func(w http.ResponseWriter, r *http.Request) error {
			callCount++
			return nil
		})
		return dom.ElementNode("div")
	}

	sess := NewSession(comp, struct{}{})
	sess.SetSessionID("test")
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("first flush failed: %v", err)
	}

	handlerID := fmt.Sprintf("%s:h0", sess.root.id)
	entry1 := sess.FindHandler(handlerID)
	if entry1 == nil {
		t.Fatalf("expected handler to be registered after first render with ID %s", handlerID)
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	entry2 := sess.FindHandler(handlerID)
	if entry2 == nil {
		t.Fatalf("expected handler to persist after second render with ID %s", handlerID)
	}

	if entry1.ID != entry2.ID {
		t.Errorf("expected same handler ID, got %s and %s", entry1.ID, entry2.ID)
	}

	if entry1.Method != entry2.Method {
		t.Errorf("expected same method, got %s and %s", entry1.Method, entry2.Method)
	}
}

func TestUseHandlerEmptyURL(t *testing.T) {
	var handle HandlerHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		handle = UseHandler(ctx, "GET", func(w http.ResponseWriter, r *http.Request) error {
			return nil
		})
		return dom.ElementNode("div")
	}

	sess := NewSession(comp, struct{}{})

	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	expectedURL := fmt.Sprintf("/_handlers//%s:h0", sess.root.id)
	if handle.URL() != expectedURL {
		t.Errorf("expected URL %s, got %s", expectedURL, handle.URL())
	}
}

func TestUseHandlerNotFoundAfterRemoval(t *testing.T) {
	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		UseHandler(ctx, "GET", func(w http.ResponseWriter, r *http.Request) error {
			return nil
		})
		return dom.ElementNode("div")
	}

	sess := NewSession(comp, struct{}{})
	sess.SetSessionID("test")
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	handlerID := fmt.Sprintf("%s:h0", sess.root.id)
	entry := sess.FindHandler(handlerID)
	if entry == nil {
		t.Fatalf("expected handler to be registered with ID %s", handlerID)
	}

	sess.removeHandler(handlerID)

	entry = sess.FindHandler(handlerID)
	if entry != nil {
		t.Error("expected handler to be removed")
	}
}
