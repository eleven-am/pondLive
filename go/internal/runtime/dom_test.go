package runtime

import (
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/go/internal/protocol"
)

func makeTestRef(id string) *ElementRef {
	return &ElementRef{id: id}
}

func TestCtxCall(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	var received protocol.DOMCallPayload
	var receivedTopic protocol.Topic
	var receivedEvent string
	done := make(chan struct{})

	bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		receivedTopic = protocol.DOMHandler
		receivedEvent = event
		if payload, ok := data.(protocol.DOMCallPayload); ok {
			received = payload
		}
		close(done)
	})

	ref := makeTestRef("ref-1")
	ctx.Call(ref, "focus")

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for callback")
	}

	if receivedTopic != protocol.DOMHandler {
		t.Errorf("expected topic %q, got %q", protocol.DOMHandler, receivedTopic)
	}
	if receivedEvent != string(protocol.DOMCallAction) {
		t.Errorf("expected event %q, got %q", string(protocol.DOMCallAction), receivedEvent)
	}
	if received.Ref != "ref-1" {
		t.Errorf("expected ref 'ref-1', got %q", received.Ref)
	}
	if received.Method != "focus" {
		t.Errorf("expected method 'focus', got %q", received.Method)
	}
}

func TestCtxCallWithArgs(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	var received protocol.DOMCallPayload
	done := make(chan struct{})

	bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		if payload, ok := data.(protocol.DOMCallPayload); ok {
			received = payload
		}
		close(done)
	})

	ref := makeTestRef("ref-2")
	ctx.Call(ref, "scrollTo", 0, 100, map[string]any{"behavior": "smooth"})

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for callback")
	}

	if received.Method != "scrollTo" {
		t.Errorf("expected method 'scrollTo', got %q", received.Method)
	}
	if len(received.Args) != 3 {
		t.Fatalf("expected 3 args, got %d", len(received.Args))
	}
	if received.Args[0] != 0 {
		t.Errorf("expected args[0] = 0, got %v", received.Args[0])
	}
	if received.Args[1] != 100 {
		t.Errorf("expected args[1] = 100, got %v", received.Args[1])
	}
}

func TestCtxCallNilRef(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	called := false
	bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		called = true
	})

	ctx.Call(nil, "focus")

	if called {
		t.Error("expected no publish for nil ref")
	}
}

func TestCtxCallEmptyRef(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	called := false
	bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		called = true
	})

	ref := makeTestRef("")
	ctx.Call(ref, "focus")

	if called {
		t.Error("expected no publish for empty ref ID")
	}
}

func TestCtxCallNilSession(t *testing.T) {
	ctx := &Ctx{session: nil}

	ctx.Call(makeTestRef("ref-1"), "focus")
}

func TestCtxSet(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	var received protocol.DOMSetPayload
	var receivedEvent string
	done := make(chan struct{})

	bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		receivedEvent = event
		if payload, ok := data.(protocol.DOMSetPayload); ok {
			received = payload
		}
		close(done)
	})

	ref := makeTestRef("ref-3")
	ctx.Set(ref, "value", "hello world")

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for callback")
	}

	if receivedEvent != string(protocol.DOMSetAction) {
		t.Errorf("expected event %q, got %q", string(protocol.DOMSetAction), receivedEvent)
	}
	if received.Ref != "ref-3" {
		t.Errorf("expected ref 'ref-3', got %q", received.Ref)
	}
	if received.Prop != "value" {
		t.Errorf("expected prop 'value', got %q", received.Prop)
	}
	if received.Value != "hello world" {
		t.Errorf("expected value 'hello world', got %v", received.Value)
	}
}

func TestCtxSetNilRef(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	called := false
	bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		called = true
	})

	ctx.Set(nil, "value", "test")

	if called {
		t.Error("expected no publish for nil ref")
	}
}

func TestCtxQuery(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		if event != string(protocol.DOMQueryAction) {
			return
		}
		payload, ok := data.(protocol.DOMQueryPayload)
		if !ok {
			return
		}

		go func() {
			time.Sleep(10 * time.Millisecond)
			bus.Publish(protocol.DOMHandler, string(protocol.DOMResponseAction), protocol.DOMResponsePayload{
				RequestID: payload.RequestID,
				Values: map[string]any{
					"value":     "test value",
					"scrollTop": 42,
				},
			})
		}()
	})

	ref := makeTestRef("ref-4")
	values, err := ctx.Query(ref, "value", "scrollTop")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if values["value"] != "test value" {
		t.Errorf("expected value 'test value', got %v", values["value"])
	}
	if values["scrollTop"] != 42 {
		t.Errorf("expected scrollTop 42, got %v", values["scrollTop"])
	}
}

func TestCtxQueryTimeout(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}

	session.domReqMgr = newDOMRequestManager(bus, 50*time.Millisecond)

	ctx := &Ctx{session: session}

	ref := makeTestRef("ref-5")
	_, err := ctx.Query(ref, "value")

	if err != ErrQueryTimeout {
		t.Errorf("expected ErrQueryTimeout, got %v", err)
	}
}

func TestCtxQueryNilRef(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	_, err := ctx.Query(nil, "value")

	if err != ErrNilRef {
		t.Errorf("expected ErrNilRef, got %v", err)
	}
}

func TestCtxQueryEmptySelectors(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	ref := makeTestRef("ref-6")
	values, err := ctx.Query(ref)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(values) != 0 {
		t.Errorf("expected empty map for no selectors, got %v", values)
	}
}

func TestCtxQueryError(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		if event != string(protocol.DOMQueryAction) {
			return
		}
		payload, ok := data.(protocol.DOMQueryPayload)
		if !ok {
			return
		}

		go func() {
			time.Sleep(10 * time.Millisecond)
			bus.Publish(protocol.DOMHandler, string(protocol.DOMResponseAction), protocol.DOMResponsePayload{
				RequestID: payload.RequestID,
				Error:     "element not found",
			})
		}()
	})

	ref := makeTestRef("ref-7")
	_, err := ctx.Query(ref, "value")

	if err == nil || err.Error() != "element not found" {
		t.Errorf("expected error 'element not found', got %v", err)
	}
}

func TestCtxAsyncCall(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		if event != string(protocol.DOMAsyncAction) {
			return
		}
		payload, ok := data.(protocol.DOMAsyncPayload)
		if !ok {
			return
		}

		go func() {
			time.Sleep(10 * time.Millisecond)
			bus.Publish(protocol.DOMHandler, string(protocol.DOMResponseAction), protocol.DOMResponsePayload{
				RequestID: payload.RequestID,
				Result:    "data:image/png;base64,abc123",
			})
		}()
	})

	ref := makeTestRef("ref-8")
	result, err := ctx.AsyncCall(ref, "toDataURL", "image/png")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "data:image/png;base64,abc123" {
		t.Errorf("expected data URL, got %v", result)
	}
}

func TestCtxAsyncCallTimeout(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}

	session.domReqMgr = newDOMRequestManager(bus, 50*time.Millisecond)

	ctx := &Ctx{session: session}

	ref := makeTestRef("ref-9")
	_, err := ctx.AsyncCall(ref, "someMethod")

	if err != ErrQueryTimeout {
		t.Errorf("expected ErrQueryTimeout, got %v", err)
	}
}

func TestCtxAsyncCallError(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		if event != string(protocol.DOMAsyncAction) {
			return
		}
		payload, ok := data.(protocol.DOMAsyncPayload)
		if !ok {
			return
		}

		go func() {
			time.Sleep(10 * time.Millisecond)
			bus.Publish(protocol.DOMHandler, string(protocol.DOMResponseAction), protocol.DOMResponsePayload{
				RequestID: payload.RequestID,
				Error:     "method not supported",
			})
		}()
	})

	ref := makeTestRef("ref-10")
	_, err := ctx.AsyncCall(ref, "unsupportedMethod")

	if err == nil || err.Error() != "method not supported" {
		t.Errorf("expected error 'method not supported', got %v", err)
	}
}

func TestDOMRequestManagerConcurrent(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}

	bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		if event != string(protocol.DOMQueryAction) {
			return
		}
		payload, ok := data.(protocol.DOMQueryPayload)
		if !ok {
			return
		}

		go func() {
			time.Sleep(5 * time.Millisecond)
			bus.Publish(protocol.DOMHandler, string(protocol.DOMResponseAction), protocol.DOMResponsePayload{
				RequestID: payload.RequestID,
				Values:    map[string]any{"value": payload.RequestID},
			})
		}()
	})

	var wg sync.WaitGroup
	const numGoroutines = 20

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			ref := makeTestRef("ref-concurrent")
			values, err := ctx.Query(ref, "value")

			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", idx, err)
				return
			}
			if values["value"] == nil {
				t.Errorf("goroutine %d: expected value, got nil", idx)
			}
		}(i)
	}

	wg.Wait()
}

func TestSessionGetDOMRequestManagerLazy(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}

	if session.domReqMgr != nil {
		t.Error("expected domReqMgr to be nil initially")
	}

	mgr1 := session.getDOMRequestManager()
	if mgr1 == nil {
		t.Fatal("expected manager to be created")
	}

	mgr2 := session.getDOMRequestManager()
	if mgr1 != mgr2 {
		t.Error("expected same manager instance")
	}
}

func TestSetDOMTimeoutUpdatesExistingManager(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{Bus: bus}
	ctx := &Ctx{session: session}
	ref := makeTestRef("ref-timeout")

	firstSub := bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {
		if event != string(protocol.DOMQueryAction) {
			return
		}
		payload := data.(protocol.DOMQueryPayload)
		bus.Publish(protocol.DOMHandler, string(protocol.DOMResponseAction), protocol.DOMResponsePayload{RequestID: payload.RequestID})
	})
	if _, err := ctx.Query(ref, "value"); err != nil {
		t.Fatalf("unexpected error priming query: %v", err)
	}
	firstSub.Unsubscribe()

	session.SetDOMTimeout(20 * time.Millisecond)

	start := time.Now()
	secondSub := bus.Subscribe(protocol.DOMHandler, func(event string, data interface{}) {})
	defer secondSub.Unsubscribe()

	if _, err := ctx.Query(ref, "value"); err != ErrQueryTimeout {
		t.Fatalf("expected timeout error, got %v", err)
	}

	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("timeout did not honor updated deadline; elapsed %v", elapsed)
	}
}

func TestSessionCloseCleansDOMManager(t *testing.T) {
	bus := protocol.NewBus()
	session := &Session{
		Bus:               bus,
		MountedComponents: make(map[*Instance]struct{}),
		DirtySet:          make(map[*Instance]struct{}),
	}

	mgr := session.getDOMRequestManager()
	if mgr == nil {
		t.Fatal("expected manager to be created")
	}

	session.Close()

	if session.domReqMgr != nil {
		t.Error("expected domReqMgr to be nil after Close")
	}
}
