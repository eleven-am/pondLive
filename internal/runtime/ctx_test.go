package runtime

import (
	"context"
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
)

func TestCtxComponentID(t *testing.T) {
	t.Run("returns instance ID", func(t *testing.T) {
		inst := &Instance{ID: "test-component-123"}
		ctx := &Ctx{instance: inst}

		got := ctx.ComponentID()
		if got != "test-component-123" {
			t.Errorf("ComponentID() = %q, want %q", got, "test-component-123")
		}
	})

	t.Run("returns empty for nil ctx", func(t *testing.T) {
		var ctx *Ctx
		got := ctx.ComponentID()
		if got != "" {
			t.Errorf("ComponentID() on nil ctx = %q, want empty string", got)
		}
	})

	t.Run("returns empty for nil instance", func(t *testing.T) {
		ctx := &Ctx{instance: nil}
		got := ctx.ComponentID()
		if got != "" {
			t.Errorf("ComponentID() with nil instance = %q, want empty string", got)
		}
	})
}

func TestCtxComponentDepth(t *testing.T) {
	t.Run("root component has depth 0", func(t *testing.T) {
		root := &Instance{ID: "root", Parent: nil}
		ctx := &Ctx{instance: root}

		got := ctx.ComponentDepth()
		if got != 0 {
			t.Errorf("ComponentDepth() for root = %d, want 0", got)
		}
	})

	t.Run("child has depth 1", func(t *testing.T) {
		root := &Instance{ID: "root", Parent: nil}
		child := &Instance{ID: "child", Parent: root}
		ctx := &Ctx{instance: child}

		got := ctx.ComponentDepth()
		if got != 1 {
			t.Errorf("ComponentDepth() for child = %d, want 1", got)
		}
	})

	t.Run("grandchild has depth 2", func(t *testing.T) {
		root := &Instance{ID: "root", Parent: nil}
		child := &Instance{ID: "child", Parent: root}
		grandchild := &Instance{ID: "grandchild", Parent: child}
		ctx := &Ctx{instance: grandchild}

		got := ctx.ComponentDepth()
		if got != 2 {
			t.Errorf("ComponentDepth() for grandchild = %d, want 2", got)
		}
	})

	t.Run("deep nesting", func(t *testing.T) {
		var current *Instance
		for i := 0; i < 10; i++ {
			current = &Instance{ID: "level", Parent: current}
		}
		ctx := &Ctx{instance: current}

		got := ctx.ComponentDepth()
		if got != 9 {
			t.Errorf("ComponentDepth() for 10-level tree = %d, want 9", got)
		}
	})

	t.Run("returns 0 for nil ctx", func(t *testing.T) {
		var ctx *Ctx
		got := ctx.ComponentDepth()
		if got != 0 {
			t.Errorf("ComponentDepth() on nil ctx = %d, want 0", got)
		}
	})

	t.Run("returns 0 for nil instance", func(t *testing.T) {
		ctx := &Ctx{instance: nil}
		got := ctx.ComponentDepth()
		if got != 0 {
			t.Errorf("ComponentDepth() with nil instance = %d, want 0", got)
		}
	})
}

func TestCtxContext(t *testing.T) {
	t.Run("returns Background", func(t *testing.T) {
		ctx := &Ctx{}
		got := ctx.Context()
		if got != context.Background() {
			t.Error("Context() should return context.Background()")
		}
	})

	t.Run("returns Background for nil ctx", func(t *testing.T) {
		var ctx *Ctx
		got := ctx.Context()
		if got == nil {
			t.Error("Context() on nil ctx should not return nil")
		}
	})
}

func TestGetBus(t *testing.T) {
	t.Run("returns bus from session", func(t *testing.T) {
		bus := protocol.NewBus()
		sess := &Session{Bus: bus}
		ctx := &Ctx{session: sess}

		got := GetBus(ctx)
		if got != bus {
			t.Error("GetBus() should return the session bus")
		}
	})

	t.Run("returns nil for nil ctx", func(t *testing.T) {
		got := GetBus(nil)
		if got != nil {
			t.Error("GetBus(nil) should return nil")
		}
	})

	t.Run("returns nil for nil session", func(t *testing.T) {
		ctx := &Ctx{session: nil}
		got := GetBus(ctx)
		if got != nil {
			t.Error("GetBus() with nil session should return nil")
		}
	})
}

func TestNewCtxForTest(t *testing.T) {
	inst := &Instance{ID: "test-inst"}
	sess := &Session{SessionID: "test-sess"}

	ctx := NewCtxForTest(inst, sess)
	if ctx == nil {
		t.Fatal("NewCtxForTest should return non-nil ctx")
	}
	if ctx.instance != inst {
		t.Error("ctx.instance should be set")
	}
	if ctx.session != sess {
		t.Error("ctx.session should be set")
	}
}

func TestCtxSessionID(t *testing.T) {
	t.Run("returns session ID", func(t *testing.T) {
		sess := &Session{SessionID: "sess-123"}
		ctx := &Ctx{session: sess}

		got := ctx.SessionID()
		if got != "sess-123" {
			t.Errorf("SessionID() = %q, want %q", got, "sess-123")
		}
	})

	t.Run("returns empty for nil ctx", func(t *testing.T) {
		var ctx *Ctx
		got := ctx.SessionID()
		if got != "" {
			t.Errorf("SessionID() on nil ctx = %q, want empty string", got)
		}
	})

	t.Run("returns empty for nil session", func(t *testing.T) {
		ctx := &Ctx{session: nil}
		got := ctx.SessionID()
		if got != "" {
			t.Errorf("SessionID() with nil session = %q, want empty string", got)
		}
	})
}
