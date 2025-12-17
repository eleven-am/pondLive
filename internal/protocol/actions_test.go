package protocol

import (
	"sync"
	"testing"
	"time"
)

func TestPublishDOMCall(t *testing.T) {
	bus := NewBus()
	var received DOMCallPayload
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(DOMHandler, func(event string, data interface{}) {
		if event == string(DOMCallAction) {
			if payload, ok := data.(DOMCallPayload); ok {
				received = payload
			}
		}
		wg.Done()
	})

	bus.PublishDOMCall(DOMCallPayload{
		Ref:    "elem-1",
		Method: "focus",
		Args:   []any{"arg1"},
	})

	waitWithTimeout(t, &wg)

	if received.Ref != "elem-1" {
		t.Errorf("Ref = %q, want %q", received.Ref, "elem-1")
	}
	if received.Method != "focus" {
		t.Errorf("Method = %q, want %q", received.Method, "focus")
	}
}

func TestPublishDOMSet(t *testing.T) {
	bus := NewBus()
	var received DOMSetPayload
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(DOMHandler, func(event string, data interface{}) {
		if event == string(DOMSetAction) {
			if payload, ok := data.(DOMSetPayload); ok {
				received = payload
			}
		}
		wg.Done()
	})

	bus.PublishDOMSet(DOMSetPayload{
		Ref:   "elem-1",
		Prop:  "value",
		Value: "test",
	})

	waitWithTimeout(t, &wg)

	if received.Prop != "value" {
		t.Errorf("Prop = %q, want %q", received.Prop, "value")
	}
}

func TestPublishDOMQuery(t *testing.T) {
	bus := NewBus()
	var received DOMQueryPayload
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(DOMHandler, func(event string, data interface{}) {
		if event == string(DOMQueryAction) {
			if payload, ok := data.(DOMQueryPayload); ok {
				received = payload
			}
		}
		wg.Done()
	})

	bus.PublishDOMQuery(DOMQueryPayload{
		RequestID: "req-1",
		Ref:       "elem-1",
		Selectors: []string{"value", "checked"},
	})

	waitWithTimeout(t, &wg)

	if received.RequestID != "req-1" {
		t.Errorf("RequestID = %q, want %q", received.RequestID, "req-1")
	}
}

func TestPublishDOMAsync(t *testing.T) {
	bus := NewBus()
	var received DOMAsyncPayload
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(DOMHandler, func(event string, data interface{}) {
		if event == string(DOMAsyncAction) {
			if payload, ok := data.(DOMAsyncPayload); ok {
				received = payload
			}
		}
		wg.Done()
	})

	bus.PublishDOMAsync(DOMAsyncPayload{
		RequestID: "async-1",
		Ref:       "elem-1",
		Method:    "getBoundingClientRect",
	})

	waitWithTimeout(t, &wg)

	if received.Method != "getBoundingClientRect" {
		t.Errorf("Method = %q, want %q", received.Method, "getBoundingClientRect")
	}
}

func TestSubscribeToDOMActions(t *testing.T) {
	bus := NewBus()
	var mu sync.Mutex
	var receivedActions []DOMServerAction
	var wg sync.WaitGroup

	sub := bus.SubscribeToDOMActions(func(action DOMServerAction, data interface{}) {
		mu.Lock()
		receivedActions = append(receivedActions, action)
		mu.Unlock()
		wg.Done()
	})
	defer sub.Unsubscribe()

	wg.Add(4)
	bus.PublishDOMCall(DOMCallPayload{})
	bus.PublishDOMSet(DOMSetPayload{})
	bus.PublishDOMQuery(DOMQueryPayload{})
	bus.PublishDOMAsync(DOMAsyncPayload{})

	waitWithTimeout(t, &wg)

	mu.Lock()
	if len(receivedActions) != 4 {
		t.Errorf("received %d actions, want 4", len(receivedActions))
	}
	mu.Unlock()
}

func TestSubscribeToDOMResponses(t *testing.T) {
	bus := NewBus()
	var received DOMResponsePayload
	var wg sync.WaitGroup
	wg.Add(1)

	sub := bus.SubscribeToDOMResponses(func(response DOMResponsePayload) {
		received = response
		wg.Done()
	})
	defer sub.Unsubscribe()

	bus.Publish(DOMHandler, string(DOMResponseAction), DOMResponsePayload{
		RequestID: "req-1",
		Result:    "test-result",
	})

	waitWithTimeout(t, &wg)

	if received.RequestID != "req-1" {
		t.Errorf("RequestID = %q, want %q", received.RequestID, "req-1")
	}
}

func TestPublishRouterPush(t *testing.T) {
	bus := NewBus()
	var received RouterNavPayload
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(RouteHandler, func(event string, data interface{}) {
		if event == string(RouterPushAction) {
			if payload, ok := data.(RouterNavPayload); ok {
				received = payload
			}
		}
		wg.Done()
	})

	bus.PublishRouterPush(RouterNavPayload{
		Path:  "/users",
		Query: "page=1",
		Hash:  "section",
	})

	waitWithTimeout(t, &wg)

	if received.Path != "/users" {
		t.Errorf("Path = %q, want %q", received.Path, "/users")
	}
}

func TestPublishRouterReplace(t *testing.T) {
	bus := NewBus()
	var received RouterNavPayload
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(RouteHandler, func(event string, data interface{}) {
		if event == string(RouterReplaceAction) {
			if payload, ok := data.(RouterNavPayload); ok {
				received = payload
			}
		}
		wg.Done()
	})

	bus.PublishRouterReplace(RouterNavPayload{
		Path:    "/login",
		Replace: true,
	})

	waitWithTimeout(t, &wg)

	if received.Path != "/login" {
		t.Errorf("Path = %q, want %q", received.Path, "/login")
	}
}

func TestPublishRouterBack(t *testing.T) {
	bus := NewBus()
	var receivedEvent string
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(RouteHandler, func(event string, data interface{}) {
		receivedEvent = event
		wg.Done()
	})

	bus.PublishRouterBack()

	waitWithTimeout(t, &wg)

	if receivedEvent != string(RouterBackAction) {
		t.Errorf("event = %q, want %q", receivedEvent, string(RouterBackAction))
	}
}

func TestPublishRouterForward(t *testing.T) {
	bus := NewBus()
	var receivedEvent string
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(RouteHandler, func(event string, data interface{}) {
		receivedEvent = event
		wg.Done()
	})

	bus.PublishRouterForward()

	waitWithTimeout(t, &wg)

	if receivedEvent != string(RouterForwardAction) {
		t.Errorf("event = %q, want %q", receivedEvent, string(RouterForwardAction))
	}
}

func TestSubscribeToRouterCommands(t *testing.T) {
	bus := NewBus()
	var mu sync.Mutex
	var receivedActions []RouterServerAction
	var wg sync.WaitGroup

	sub := bus.SubscribeToRouterCommands(func(action RouterServerAction, data interface{}) {
		mu.Lock()
		receivedActions = append(receivedActions, action)
		mu.Unlock()
		wg.Done()
	})
	defer sub.Unsubscribe()

	wg.Add(4)
	bus.PublishRouterPush(RouterNavPayload{})
	bus.PublishRouterReplace(RouterNavPayload{})
	bus.PublishRouterBack()
	bus.PublishRouterForward()

	waitWithTimeout(t, &wg)

	mu.Lock()
	if len(receivedActions) != 4 {
		t.Errorf("received %d actions, want 4", len(receivedActions))
	}
	mu.Unlock()
}

func TestSubscribeToRouterPopstate(t *testing.T) {
	bus := NewBus()
	var received RouterNavPayload
	var wg sync.WaitGroup
	wg.Add(1)

	sub := bus.SubscribeToRouterPopstate(func(payload RouterNavPayload) {
		received = payload
		wg.Done()
	})
	defer sub.Unsubscribe()

	bus.Publish(RouteHandler, string(RouterPopstateAction), RouterNavPayload{
		Path: "/back-page",
	})

	waitWithTimeout(t, &wg)

	if received.Path != "/back-page" {
		t.Errorf("Path = %q, want %q", received.Path, "/back-page")
	}
}

func TestPublishFramePatch(t *testing.T) {
	bus := NewBus()
	var received interface{}
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(TopicFrame, func(event string, data interface{}) {
		if event == string(FramePatchAction) {
			received = data
		}
		wg.Done()
	})

	patches := []map[string]string{{"op": "add", "path": "/test"}}
	bus.PublishFramePatch(patches)

	waitWithTimeout(t, &wg)

	if received == nil {
		t.Error("expected patches to be received")
	}
}

func TestSubscribeToFramePatches(t *testing.T) {
	bus := NewBus()
	var received interface{}
	var wg sync.WaitGroup
	wg.Add(1)

	sub := bus.SubscribeToFramePatches(func(patches interface{}) {
		received = patches
		wg.Done()
	})
	defer sub.Unsubscribe()

	bus.PublishFramePatch("test-patches")

	waitWithTimeout(t, &wg)

	if received != "test-patches" {
		t.Errorf("received = %v, want %v", received, "test-patches")
	}
}

func TestPublishScriptSend(t *testing.T) {
	bus := NewBus()
	var received ScriptPayload
	var wg sync.WaitGroup
	wg.Add(1)

	topic := Topic("script:my-script")
	bus.Subscribe(topic, func(event string, data interface{}) {
		if event == string(ScriptSendAction) {
			if payload, ok := data.(ScriptPayload); ok {
				received = payload
			}
		}
		wg.Done()
	})

	bus.PublishScriptSend("my-script", "custom-event", "data")

	waitWithTimeout(t, &wg)

	if received.ScriptID != "my-script" {
		t.Errorf("ScriptID = %q, want %q", received.ScriptID, "my-script")
	}
	if received.Event != "custom-event" {
		t.Errorf("Event = %q, want %q", received.Event, "custom-event")
	}
}

func TestSubscribeToScript(t *testing.T) {
	bus := NewBus()
	var received ScriptPayload
	var receivedAction string
	var wg sync.WaitGroup
	wg.Add(1)

	sub := bus.SubscribeToScript("test-script", func(action string, payload ScriptPayload) {
		receivedAction = action
		received = payload
		wg.Done()
	})
	defer sub.Unsubscribe()

	bus.PublishScriptSend("test-script", "test-event", nil)

	waitWithTimeout(t, &wg)

	if receivedAction != string(ScriptSendAction) {
		t.Errorf("action = %q, want %q", receivedAction, string(ScriptSendAction))
	}
	if received.Event != "test-event" {
		t.Errorf("Event = %q, want %q", received.Event, "test-event")
	}
}

func TestSubscribeToScriptMessages(t *testing.T) {
	bus := NewBus()
	var receivedData interface{}
	var wg sync.WaitGroup
	wg.Add(1)

	sub := bus.SubscribeToScriptMessages("msg-script", "hello", func(data interface{}) {
		receivedData = data
		wg.Done()
	})
	defer sub.Unsubscribe()

	topic := Topic("script:msg-script:hello")
	bus.Publish(topic, string(ScriptMessageAction), ScriptPayload{
		ScriptID: "msg-script",
		Event:    "hello",
		Data:     "world",
	})

	waitWithTimeout(t, &wg)

	if receivedData != "world" {
		t.Errorf("data = %v, want %v", receivedData, "world")
	}
}

func waitWithTimeout(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for callback")
	}
}
