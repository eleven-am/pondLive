package runtime

import (
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/work"
)

func TestRegisterHandlerUnsubscribesPrevious(t *testing.T) {
	sess := &Session{
		Bus:               protocol.NewBus(),
		currentHandlerIDs: make(map[string]bool),
		allHandlerSubs:    make(map[string]*protocol.Subscription),
	}

	inst := &Instance{
		ID:               "test-inst",
		HookFrame:        []HookSlot{},
		NextHandlerIndex: 0,
	}

	elem := &work.Element{
		Tag:      "button",
		Handlers: make(map[string]work.Handler),
	}

	var mu sync.Mutex
	handler1Called := false
	handler2Called := false
	done := make(chan struct{})

	handler1 := work.Handler{
		Fn: func(e work.Event) work.Updates {
			mu.Lock()
			handler1Called = true
			mu.Unlock()
			return nil
		},
	}

	handler2 := work.Handler{
		Fn: func(e work.Event) work.Updates {
			mu.Lock()
			handler2Called = true
			mu.Unlock()
			close(done)
			return nil
		},
	}

	handlerID1 := sess.registerHandler(inst, elem, "click", handler1)
	inst.NextHandlerIndex = 0

	handlerID2 := sess.registerHandler(inst, elem, "click", handler2)

	if handlerID1 != handlerID2 {
		t.Errorf("expected same handler ID, got %q and %q", handlerID1, handlerID2)
	}

	sess.Bus.PublishHandlerInvoke(handlerID1, map[string]any{"type": "click"})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for handler to be called")
	}

	mu.Lock()
	h1Called := handler1Called
	h2Called := handler2Called
	mu.Unlock()

	if h1Called {
		t.Error("expected old handler to NOT be called after re-registration")
	}

	if !h2Called {
		t.Error("expected new handler to be called")
	}
}

func TestRegisterHandlerSubscriptionCountStable(t *testing.T) {
	sess := &Session{
		Bus:               protocol.NewBus(),
		currentHandlerIDs: make(map[string]bool),
		allHandlerSubs:    make(map[string]*protocol.Subscription),
	}

	inst := &Instance{
		ID:               "test-inst",
		HookFrame:        []HookSlot{},
		NextHandlerIndex: 0,
	}

	elem := &work.Element{
		Tag:      "button",
		RefID:    "btn-ref",
		Handlers: make(map[string]work.Handler),
	}

	handler := work.Handler{
		Fn: func(e work.Event) work.Updates { return nil },
	}

	for i := 0; i < 100; i++ {
		sess.registerHandler(inst, elem, "click", handler)
	}

	sess.handlerIDsMu.Lock()
	subCount := len(sess.allHandlerSubs)
	sess.handlerIDsMu.Unlock()

	if subCount != 1 {
		t.Errorf("expected 1 subscription after 100 re-registrations, got %d", subCount)
	}

	handlerID := "btn-ref:click"
	busSubCount := sess.Bus.SubscriberCount(protocol.Topic(handlerID))

	if busSubCount != 1 {
		t.Errorf("expected 1 bus subscriber, got %d", busSubCount)
	}
}

func TestRegisterHandlerWithRefID(t *testing.T) {
	sess := &Session{
		Bus:               protocol.NewBus(),
		currentHandlerIDs: make(map[string]bool),
		allHandlerSubs:    make(map[string]*protocol.Subscription),
	}

	inst := &Instance{
		ID:               "test-inst",
		HookFrame:        []HookSlot{},
		NextHandlerIndex: 0,
	}

	elem := &work.Element{
		Tag:      "button",
		RefID:    "my-button",
		Handlers: make(map[string]work.Handler),
	}

	handler := work.Handler{
		Fn: func(e work.Event) work.Updates { return nil },
	}

	handlerID := sess.registerHandler(inst, elem, "click", handler)

	expected := "my-button:click"
	if handlerID != expected {
		t.Errorf("expected handler ID %q, got %q", expected, handlerID)
	}
}

func TestRegisterHandlerWithoutRefID(t *testing.T) {
	sess := &Session{
		Bus:               protocol.NewBus(),
		currentHandlerIDs: make(map[string]bool),
		allHandlerSubs:    make(map[string]*protocol.Subscription),
	}

	inst := &Instance{
		ID:               "test-inst",
		HookFrame:        []HookSlot{},
		NextHandlerIndex: 0,
	}

	elem := &work.Element{
		Tag:      "button",
		Handlers: make(map[string]work.Handler),
	}

	handler := work.Handler{
		Fn: func(e work.Event) work.Updates { return nil },
	}

	handlerID := sess.registerHandler(inst, elem, "click", handler)

	expected := "test-inst:h0"
	if handlerID != expected {
		t.Errorf("expected handler ID %q, got %q", expected, handlerID)
	}

	if inst.NextHandlerIndex != 1 {
		t.Errorf("expected NextHandlerIndex to be 1, got %d", inst.NextHandlerIndex)
	}
}
