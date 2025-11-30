package router

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

func TestNavigate_NoRequestState_NoPanic(t *testing.T) {
	session := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
		Handlers:   make(map[string]work.Handler),
		Bus:        protocol.NewBus(),
	}

	inst := &runtime.Instance{
		ID:        "test",
		HookFrame: []runtime.HookSlot{},
	}

	ctx := runtime.NewCtxForTest(inst, session)

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Navigate panicked with nil request state: %v", r)
		}
	}()

	Navigate(ctx, "/foo")
}
