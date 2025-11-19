package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom2/diff"
)

func TestDOMActionSenderReceivesEffects(t *testing.T) {
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		UseEffect(ctx, func() Cleanup {
			ctx.EnqueueDOMAction(dom2.DOMActionEffect{
				Kind:   "dom.call",
				Ref:    "ref:1",
				Method: "focus",
			})
			return nil
		})
		return dom2.ElementNode("div")
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	var batches [][]dom2.DOMActionEffect
	sess.SetDOMActionSender(func(effects []dom2.DOMActionEffect) error {
		copyBatch := append([]dom2.DOMActionEffect(nil), effects...)
		batches = append(batches, copyBatch)
		return nil
	})

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if len(batches) != 1 {
		t.Fatalf("expected 1 DOM action batch, got %d", len(batches))
	}
	if len(batches[0]) != 1 {
		t.Fatalf("expected 1 effect in batch, got %d", len(batches[0]))
	}
	if batches[0][0].Method != "focus" {
		t.Fatalf("expected focus action, got %+v", batches[0][0])
	}
}

func TestDOMGetHandler(t *testing.T) {
	sess := NewSession(func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		return dom2.ElementNode("div")
	}, struct{}{})

	sess.SetDOMRequestHandlers(func(ref string, selectors ...string) (map[string]any, error) {
		return map[string]any{
			"ref":       ref,
			"selectors": selectors,
		}, nil
	}, nil)

	result, err := sess.domGet("ref:9", "value", "checked")
	if err != nil {
		t.Fatalf("domGet failed: %v", err)
	}
	if result["ref"].(string) != "ref:9" {
		t.Fatalf("expected ref:9, got %v", result["ref"])
	}
}

func TestDOMGetHandlerMissing(t *testing.T) {
	sess := NewSession(func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		return dom2.ElementNode("div")
	}, struct{}{})

	if _, err := sess.domGet("ref:1"); err == nil {
		t.Fatal("expected error when DOMGet handler missing")
	}
}

func TestDOMAsyncCallHandler(t *testing.T) {
	sess := NewSession(func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		return dom2.ElementNode("div")
	}, struct{}{})

	sess.SetDOMRequestHandlers(nil, func(ref string, method string, args ...any) (any, error) {
		return map[string]any{
			"ref":    ref,
			"method": method,
			"args":   args,
		}, nil
	})

	r, err := sess.domAsyncCall("ref:2", "getBoundingClientRect", 1, 2)
	if err != nil {
		t.Fatalf("DOMAsyncCall failed: %v", err)
	}
	resp := r.(map[string]any)
	if resp["method"].(string) != "getBoundingClientRect" {
		t.Fatalf("unexpected method in response: %v", resp)
	}
}

func TestDOMAsyncCallHandlerMissing(t *testing.T) {
	sess := NewSession(func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		return dom2.ElementNode("div")
	}, struct{}{})

	if _, err := sess.domAsyncCall("ref:1", "focus"); err == nil {
		t.Fatal("expected error when DOMAsyncCall handler missing")
	}
}

func TestEffectBatchingLimitsExecution(t *testing.T) {
	sess := NewSession(func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		return dom2.ElementNode("div")
	}, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	totalTasks := maxEffectsPerFlush + 10
	done := 0

	for i := 0; i < totalTasks; i++ {
		sess.enqueueEffect(sess.root, 0, func() Cleanup {
			done++
			return nil
		})
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("first flush failed: %v", err)
	}
	if done != maxEffectsPerFlush {
		t.Fatalf("expected %d effects to run, got %d", maxEffectsPerFlush, done)
	}
	remaining := totalTasks - maxEffectsPerFlush
	if len(sess.pendingEffects) != remaining {
		t.Fatalf("expected %d pending effects, got %d", remaining, len(sess.pendingEffects))
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}
	if done != totalTasks {
		t.Fatalf("expected all effects to run after second flush, got %d", done)
	}
}
