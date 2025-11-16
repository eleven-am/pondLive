package runtime

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/go/internal/diff"
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/render"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// TestRapidEventsDuringRerenders tests the real-world scenario:
// user rapidly clicking while component is constantly re-rendering.
func TestRapidEventsDuringRerenders(t *testing.T) {
	const numEvents = 10
	const renderDelay = 10 * time.Millisecond

	eventsProcessed := int32(0)
	renderCount := int32(0)

	comp := func(ctx Ctx, _ struct{}) h.Node {
		count := atomic.AddInt32(&renderCount, 1)
		get, set := UseState(ctx, 0)

		handler := func(evt h.Event) h.Updates {
			atomic.AddInt32(&eventsProcessed, 1)

			set(get() + 1)
			return nil
		}

		time.Sleep(renderDelay)

		t.Logf("Render #%d: state=%d", count, get())
		return h.Div(h.Button(h.On("click", handler), h.Textf("Count: %d", get())))
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(ops []diff.Op) error {
		t.Logf("Patch sent with %d ops", len(ops))
		return nil
	})
	structured := sess.InitialStructured()

	handlerID := findHandlerID(structured, "click")
	if handlerID == "" {
		t.Fatal("expected click handler id")
	}
	t.Logf("Handler ID: %s", handlerID)

	errorCount := int32(0)
	errors := make([]error, numEvents)

	for i := 0; i < numEvents; i++ {
		idx := i
		go func() {
			time.Sleep(time.Duration(idx) * 5 * time.Millisecond)
			t.Logf("Dispatching event #%d", idx)
			err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"})
			if err != nil {
				atomic.AddInt32(&errorCount, 1)
				errors[idx] = err
				t.Logf("Event #%d FAILED: %v", idx, err)
			} else {
				t.Logf("Event #%d succeeded", idx)
			}
		}()
	}

	time.Sleep(400 * time.Millisecond)

	processed := atomic.LoadInt32(&eventsProcessed)
	errors_count := atomic.LoadInt32(&errorCount)

	t.Logf("Dispatched %d events, processed %d, errors %d, renders %d", numEvents, processed, errors_count, atomic.LoadInt32(&renderCount))

	for i, err := range errors {
		if err != nil {
			t.Logf("  Event %d error: %v", i, err)
		}
	}

	if errors_count > 0 {
		t.Errorf("BUG: %d events failed during rapid re-renders (%.1f%% failure rate)",
			errors_count, float64(errors_count)/float64(numEvents)*100)
	}

	if processed < numEvents {
		t.Errorf("Expected %d events processed, got %d - %d events were dropped",
			numEvents, processed, numEvents-int(processed))
	}
}

// Helper to find handler ID from bindings
func findHandlerID(s render.Structured, eventName string) string {
	for _, binding := range s.Bindings {
		if binding.Event == eventName && binding.Handler != "" {
			fmt.Printf("Found binding: event=%s handler=%s\n", binding.Event, binding.Handler)
			return binding.Handler
		}
	}
	return ""
}
