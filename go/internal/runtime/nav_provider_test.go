package runtime

import (
	"testing"

	"github.com/eleven-am/go/pondlive/internal/diff"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
)

func TestDrainNavUpdateUsesProvider(t *testing.T) {
	RegisterNavProvider(nil)
	defer RegisterNavProvider(nil)

	expected := NavUpdate{Push: "/next"}
	RegisterNavProvider(func(*ComponentSession) NavUpdate {
		return expected
	})

	sess := &ComponentSession{}
	got := drainNavUpdate(sess)
	if got != expected {
		t.Fatalf("unexpected nav update: %+v", got)
	}
}

func TestComponentSessionFlushSendsNavOnlyFrame(t *testing.T) {
	sess := NewSession(func(Ctx, struct{}) h.Node { return h.Fragment() }, struct{}{})
	if sess == nil {
		t.Fatal("expected session")
	}
	sess.InitialStructured()

	RegisterNavProvider(nil)
	defer RegisterNavProvider(nil)

	var called bool
	sess.SetPatchSender(func(ops []diff.Op) error {
		called = true
		if len(ops) != 0 {
			t.Fatalf("expected no diff ops, got %v", ops)
		}
		if sess.pendingNav == nil {
			t.Fatalf("expected pending nav to be set")
		}
		return nil
	})

	RegisterNavProvider(func(*ComponentSession) NavUpdate {
		return NavUpdate{Push: "/dest"}
	})

	sess.markDirty(sess.root)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if !called {
		t.Fatalf("expected patch sender to be invoked")
	}
}
