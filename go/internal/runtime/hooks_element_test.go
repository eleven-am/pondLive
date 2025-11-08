package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"
	h "github.com/eleven-am/pondlive/go/internal/html"
)

func TestUseElementSharesHookSlotAndResetsAttachment(t *testing.T) {
	var (
		counts   []int
		setCount func(int)
	)
	component := func(ctx Ctx, _ struct{}) h.Node {
		ref := UseElement[h.HTMLAudioElement](ctx)
		get, set := UseState(ctx, 0)
		counts = append(counts, get())
		if setCount == nil {
			setCount = set
		}
		return h.Audio(
			h.Attach(ref),
		)
	}

	sess := NewSession(component, struct{}{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })

	structured := sess.InitialStructured()
	if len(structured.S) == 0 && len(structured.D) == 0 {
		t.Fatal("expected initial structured render")
	}
	if setCount == nil {
		t.Fatal("expected setter to be captured during render")
	}
	setCount(1)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}

	if len(counts) != 2 {
		t.Fatalf("expected two renders, got %d", len(counts))
	}
	if counts[0] != 0 || counts[1] != 1 {
		t.Fatalf("expected state progression [0 1], got %v", counts)
	}
}
