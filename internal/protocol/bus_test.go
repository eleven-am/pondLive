package protocol

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func waitForCallback(t *testing.T, done chan struct{}) {
	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for callback")
	}
}

func TestBusSubscribeAndPublish(t *testing.T) {
	bus := NewBus()

	received := false
	var receivedEvent string
	var receivedData interface{}
	done := make(chan struct{})

	sub := bus.Subscribe("test-id", func(event string, data interface{}) {
		received = true
		receivedEvent = event
		receivedData = data
		close(done)
	})

	if sub == nil {
		t.Fatal("expected subscription to be non-nil")
	}

	bus.Publish("test-id", "click", "payload")

	waitForCallback(t, done)

	if !received {
		t.Error("expected callback to be called")
	}
	if receivedEvent != "click" {
		t.Errorf("expected event 'click', got %s", receivedEvent)
	}
	if receivedData != "payload" {
		t.Errorf("expected data 'payload', got %v", receivedData)
	}
}

func TestBusMultipleSubscribers(t *testing.T) {
	bus := NewBus()

	var count1, count2 int32
	var wg sync.WaitGroup
	wg.Add(2)

	bus.Subscribe("test-id", func(event string, data interface{}) {
		atomic.AddInt32(&count1, 1)
		wg.Done()
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		atomic.AddInt32(&count2, 1)
		wg.Done()
	})

	bus.Publish("test-id", "event", nil)

	wg.Wait()

	if atomic.LoadInt32(&count1) != 1 {
		t.Errorf("expected subscriber 1 called once, got %d", count1)
	}
	if atomic.LoadInt32(&count2) != 1 {
		t.Errorf("expected subscriber 2 called once, got %d", count2)
	}

	wg.Add(2)
	bus.Publish("test-id", "event2", nil)

	wg.Wait()

	if atomic.LoadInt32(&count1) != 2 {
		t.Errorf("expected subscriber 1 called twice, got %d", count1)
	}
	if atomic.LoadInt32(&count2) != 2 {
		t.Errorf("expected subscriber 2 called twice, got %d", count2)
	}
}

func TestBusUnsubscribe(t *testing.T) {
	bus := NewBus()

	var count int32
	done := make(chan struct{}, 1)
	sub := bus.Subscribe("test-id", func(event string, data interface{}) {
		atomic.AddInt32(&count, 1)
		select {
		case done <- struct{}{}:
		default:
		}
	})

	bus.Publish("test-id", "event", nil)
	<-done

	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("expected callback called once, got %d", count)
	}

	sub.Unsubscribe()

	bus.Publish("test-id", "event2", nil)
	time.Sleep(20 * time.Millisecond)

	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("expected callback still called once (unsubscribed), got %d", count)
	}
}

func TestBusUnsubscribeOneOfMany(t *testing.T) {
	bus := NewBus()

	var count1, count2 int32
	var wg sync.WaitGroup
	wg.Add(2)

	sub1 := bus.Subscribe("test-id", func(event string, data interface{}) {
		atomic.AddInt32(&count1, 1)
		wg.Done()
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		atomic.AddInt32(&count2, 1)
		wg.Done()
	})

	bus.Publish("test-id", "event", nil)
	wg.Wait()

	if atomic.LoadInt32(&count1) != 1 || atomic.LoadInt32(&count2) != 1 {
		t.Errorf("expected both called once, got %d and %d", count1, count2)
	}

	sub1.Unsubscribe()

	wg.Add(1)
	bus.Publish("test-id", "event2", nil)
	wg.Wait()

	if atomic.LoadInt32(&count1) != 1 {
		t.Errorf("expected sub1 still 1 (unsubscribed), got %d", count1)
	}
	if atomic.LoadInt32(&count2) != 2 {
		t.Errorf("expected sub2 called twice, got %d", count2)
	}
}

func TestBusDifferentIDs(t *testing.T) {
	bus := NewBus()

	var count1, count2 int32
	done1 := make(chan struct{})

	bus.Subscribe("id1", func(event string, data interface{}) {
		atomic.AddInt32(&count1, 1)
		close(done1)
	})

	done2 := make(chan struct{})
	bus.Subscribe("id2", func(event string, data interface{}) {
		atomic.AddInt32(&count2, 1)
		close(done2)
	})

	bus.Publish("id1", "event", nil)
	<-done1

	if atomic.LoadInt32(&count1) != 1 {
		t.Errorf("expected id1 subscriber called, got %d", count1)
	}
	if atomic.LoadInt32(&count2) != 0 {
		t.Errorf("expected id2 subscriber not called, got %d", count2)
	}

	bus.Publish("id2", "event", nil)
	<-done2

	if atomic.LoadInt32(&count1) != 1 {
		t.Errorf("expected id1 subscriber still 1, got %d", count1)
	}
	if atomic.LoadInt32(&count2) != 1 {
		t.Errorf("expected id2 subscriber called, got %d", count2)
	}
}

func TestBusPublishToNonexistentID(t *testing.T) {
	bus := NewBus()

	bus.Publish("nonexistent", "event", nil)
}

func TestBusNilSafety(t *testing.T) {
	var bus *Bus

	sub := bus.Subscribe("id", func(event string, data interface{}) {})
	if sub == nil {
		t.Error("expected non-nil subscription even from nil bus")
	}

	bus.Publish("id", "event", nil)
	sub.Unsubscribe()
}

func TestBusSubscriberCount(t *testing.T) {
	bus := NewBus()

	if bus.SubscriberCount("test-id") != 0 {
		t.Error("expected 0 subscribers initially")
	}

	sub1 := bus.Subscribe("test-id", func(event string, data interface{}) {})
	if bus.SubscriberCount("test-id") != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount("test-id"))
	}

	sub2 := bus.Subscribe("test-id", func(event string, data interface{}) {})
	if bus.SubscriberCount("test-id") != 2 {
		t.Errorf("expected 2 subscribers, got %d", bus.SubscriberCount("test-id"))
	}

	sub1.Unsubscribe()
	if bus.SubscriberCount("test-id") != 1 {
		t.Errorf("expected 1 subscriber after unsubscribe, got %d", bus.SubscriberCount("test-id"))
	}

	sub2.Unsubscribe()
	if bus.SubscriberCount("test-id") != 0 {
		t.Errorf("expected 0 subscribers after all unsubscribe, got %d", bus.SubscriberCount("test-id"))
	}
}

func TestBusConcurrentSubscribePublish(t *testing.T) {
	bus := NewBus()

	var wg sync.WaitGroup
	const numGoroutines = 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			bus.Subscribe("test-id", func(event string, data interface{}) {

			})
		}(i)
	}

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			bus.Publish("test-id", "event", id)
		}(i)
	}

	wg.Wait()

	if bus.SubscriberCount("test-id") != numGoroutines {
		t.Errorf("expected %d subscribers, got %d", numGoroutines, bus.SubscriberCount("test-id"))
	}
}

func TestBusConcurrentUnsubscribe(t *testing.T) {
	bus := NewBus()

	const numSubscribers = 100
	subs := make([]*Subscription, numSubscribers)

	for i := 0; i < numSubscribers; i++ {
		subs[i] = bus.Subscribe("test-id", func(event string, data interface{}) {})
	}

	var wg sync.WaitGroup
	for i := 0; i < numSubscribers; i++ {
		wg.Add(1)
		go func(sub *Subscription) {
			defer wg.Done()
			sub.Unsubscribe()
		}(subs[i])
	}

	wg.Wait()

	if bus.SubscriberCount("test-id") != 0 {
		t.Errorf("expected 0 subscribers after all unsubscribe, got %d", bus.SubscriberCount("test-id"))
	}
}

func TestSubscribeToHandlerInvokeUsesUpsert(t *testing.T) {
	bus := NewBus()

	var count int32
	done := make(chan struct{})

	bus.SubscribeToHandlerInvoke("handler", func(event interface{}) {
		atomic.AddInt32(&count, 1)
	})
	bus.SubscribeToHandlerInvoke("handler", func(event interface{}) {
		atomic.AddInt32(&count, 10)
		close(done)
	})

	bus.PublishHandlerInvoke("handler", nil)
	<-done

	if atomic.LoadInt32(&count) != 10 {
		t.Fatalf("expected only latest handler to fire once, got %d", count)
	}

	if bus.SubscriberCount("handler") != 1 {
		t.Fatalf("expected 1 subscriber after upsert, got %d", bus.SubscriberCount("handler"))
	}
}

func TestBusEmptySubscription(t *testing.T) {
	bus := NewBus()

	sub := bus.Subscribe("", func(event string, data interface{}) {})
	if sub == nil {
		t.Error("expected non-nil subscription")
	}

	sub = bus.Subscribe("test-id", nil)
	if sub == nil {
		t.Error("expected non-nil subscription")
	}
}

func TestBusUpsertCreatesNew(t *testing.T) {
	bus := NewBus()

	called := false
	done := make(chan struct{})
	sub := bus.Upsert("test-id", func(event string, data interface{}) {
		called = true
		close(done)
	})

	if sub == nil {
		t.Fatal("expected non-nil subscription")
	}

	bus.Publish("test-id", "event", nil)
	<-done

	if !called {
		t.Error("expected callback to be called")
	}

	if bus.SubscriberCount("test-id") != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount("test-id"))
	}
}

func TestBusUpsertUpdatesExisting(t *testing.T) {
	bus := NewBus()

	var count1 int32
	done1 := make(chan struct{})
	bus.Upsert("test-id", func(event string, data interface{}) {
		atomic.AddInt32(&count1, 1)
		close(done1)
	})

	bus.Publish("test-id", "event", nil)
	<-done1

	if atomic.LoadInt32(&count1) != 1 {
		t.Errorf("expected count1=1, got %d", count1)
	}

	var count2 int32
	done2 := make(chan struct{})
	bus.Upsert("test-id", func(event string, data interface{}) {
		atomic.AddInt32(&count2, 1)
		close(done2)
	})

	if bus.SubscriberCount("test-id") != 1 {
		t.Errorf("expected 1 subscriber after upsert, got %d", bus.SubscriberCount("test-id"))
	}

	bus.Publish("test-id", "event", nil)
	<-done2

	if atomic.LoadInt32(&count1) != 1 {
		t.Errorf("expected count1 still 1 (old callback replaced), got %d", count1)
	}
	if atomic.LoadInt32(&count2) != 1 {
		t.Errorf("expected count2=1 (new callback), got %d", count2)
	}
}

func TestBusUpsertReturnsValidSubscription(t *testing.T) {
	bus := NewBus()

	sub := bus.Upsert("test-id", func(event string, data interface{}) {})

	sub2 := bus.Upsert("test-id", func(event string, data interface{}) {})

	if bus.SubscriberCount("test-id") != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount("test-id"))
	}

	sub.Unsubscribe()

	if bus.SubscriberCount("test-id") != 0 {
		t.Errorf("expected 0 subscribers after unsubscribe, got %d", bus.SubscriberCount("test-id"))
	}

	sub2.Unsubscribe()
}

func TestBusUpsertWithHandlerPattern(t *testing.T) {
	bus := NewBus()

	handlerID := Topic("h-123")

	var value1 int32
	done1 := make(chan struct{})
	bus.Upsert(handlerID, func(event string, data interface{}) {
		if event == string(HandlerInvokeAction) {
			atomic.StoreInt32(&value1, 1)
		}
		close(done1)
	})

	bus.PublishHandlerInvoke(string(handlerID), nil)
	<-done1

	if atomic.LoadInt32(&value1) != 1 {
		t.Errorf("expected value1=1, got %d", value1)
	}

	var value2 int32
	done2 := make(chan struct{})
	bus.Upsert(handlerID, func(event string, data interface{}) {
		if event == string(HandlerInvokeAction) {
			atomic.StoreInt32(&value2, 2)
		}
		close(done2)
	})

	if bus.SubscriberCount(handlerID) != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount(handlerID))
	}

	bus.PublishHandlerInvoke(string(handlerID), nil)
	<-done2

	if atomic.LoadInt32(&value1) != 1 {
		t.Errorf("expected value1 still 1, got %d", value1)
	}
	if atomic.LoadInt32(&value2) != 2 {
		t.Errorf("expected value2=2, got %d", value2)
	}
}

func TestBusUpsertCollapsesToSingleSubscriber(t *testing.T) {
	bus := NewBus()

	var legacyCount int32
	bus.Subscribe("id", func(event string, data interface{}) {
		atomic.AddInt32(&legacyCount, 1)
	})
	bus.Subscribe("id", func(event string, data interface{}) {
		atomic.AddInt32(&legacyCount, 1)
	})

	var newCount int32
	done := make(chan struct{})
	bus.Upsert("id", func(event string, data interface{}) {
		atomic.AddInt32(&newCount, 1)
		close(done)
	})

	bus.Publish("id", "event", nil)
	<-done

	if bus.SubscriberCount("id") != 1 {
		t.Fatalf("expected subscribers collapsed to 1, got %d", bus.SubscriberCount("id"))
	}
	if atomic.LoadInt32(&legacyCount) != 0 {
		t.Fatalf("expected legacy subscribers removed, got legacyCount=%d", legacyCount)
	}
	if atomic.LoadInt32(&newCount) != 1 {
		t.Fatalf("expected new upsert callback called once, got %d", newCount)
	}
}

func TestBusPublishRecoversFromPanic(t *testing.T) {
	bus := NewBus()

	var called int32
	var wg sync.WaitGroup
	wg.Add(2)

	bus.Subscribe("id", func(event string, data interface{}) {
		defer wg.Done()
		panic("boom")
	})
	bus.Subscribe("id", func(event string, data interface{}) {
		defer wg.Done()
		atomic.StoreInt32(&called, 1)
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Publish should recover panics, but panic propagated: %v", r)
		}
	}()

	bus.Publish("id", "event", nil)

	wg.Wait()

	if atomic.LoadInt32(&called) != 1 {
		t.Fatalf("expected non-panicking subscriber to be invoked despite panic in another")
	}
}

func TestBusPublishPanicRecovery(t *testing.T) {
	bus := NewBus()

	var called1, called3 int32
	var wg sync.WaitGroup
	wg.Add(3)

	bus.Subscribe("test-id", func(event string, data interface{}) {
		defer wg.Done()
		atomic.StoreInt32(&called1, 1)
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		defer wg.Done()
		panic("intentional panic")
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		defer wg.Done()
		atomic.StoreInt32(&called3, 1)
	})

	defer func() {
		if r := recover(); r != nil {
			t.Error("expected Publish to recover from panic, but it propagated")
		}
	}()

	bus.Publish("test-id", "event", nil)

	wg.Wait()

	if atomic.LoadInt32(&called1) != 1 {
		t.Error("expected first subscriber to be called")
	}
	if atomic.LoadInt32(&called3) != 1 {
		t.Error("expected third subscriber to be called despite second panicking")
	}
}

func TestBusPublishPanicInUpsert(t *testing.T) {
	bus := NewBus()

	done1 := make(chan struct{})
	bus.Upsert("test-id", func(event string, data interface{}) {
		close(done1)
		panic("upsert panic")
	})

	defer func() {
		if r := recover(); r != nil {
			t.Error("expected Publish to recover from panic in Upsert callback")
		}
	}()

	bus.Publish("test-id", "event", nil)
	<-done1

	time.Sleep(10 * time.Millisecond)

	if bus.SubscriberCount("test-id") != 1 {
		t.Error("expected subscriber to still exist after panic")
	}

	var called int32
	done2 := make(chan struct{})
	bus.Upsert("test-id", func(event string, data interface{}) {
		atomic.StoreInt32(&called, 1)
		close(done2)
	})

	bus.Publish("test-id", "event", nil)
	<-done2

	if atomic.LoadInt32(&called) != 1 {
		t.Error("expected updated callback to be called")
	}
}

func TestBusPublishMultiplePanics(t *testing.T) {
	bus := NewBus()

	var count int32
	var wg sync.WaitGroup
	wg.Add(3)

	bus.Subscribe("test-id", func(event string, data interface{}) {
		defer wg.Done()
		atomic.AddInt32(&count, 1)
		panic("panic 1")
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		defer wg.Done()
		atomic.AddInt32(&count, 1)
		panic("panic 2")
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		defer wg.Done()
		atomic.AddInt32(&count, 1)
		panic("panic 3")
	})

	defer func() {
		if r := recover(); r != nil {
			t.Error("expected Publish to handle multiple panics")
		}
	}()

	bus.Publish("test-id", "event", nil)

	wg.Wait()

	if atomic.LoadInt32(&count) != 3 {
		t.Errorf("expected all 3 subscribers to be called, got %d", count)
	}
}

func TestBusSubscribeAllReceivesAllMessages(t *testing.T) {
	bus := NewBus()

	var mu sync.Mutex
	var received []struct {
		topic string
		event string
		data  interface{}
	}

	var wg sync.WaitGroup
	wg.Add(3)

	bus.SubscribeAll(func(topic Topic, event string, data interface{}) {
		defer wg.Done()
		mu.Lock()
		received = append(received, struct {
			topic string
			event string
			data  interface{}
		}{string(topic), event, data})
		mu.Unlock()
	})

	bus.Publish("topic1", "event1", "data1")
	bus.Publish("topic2", "event2", "data2")
	bus.Publish("topic3", "event3", "data3")

	wg.Wait()

	mu.Lock()
	defer mu.Unlock()

	if len(received) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(received))
	}
}

func TestBusSubscribeAllUnsubscribe(t *testing.T) {
	bus := NewBus()

	var count int32
	done := make(chan struct{})
	sub := bus.SubscribeAll(func(topic Topic, event string, data interface{}) {
		atomic.AddInt32(&count, 1)
		select {
		case done <- struct{}{}:
		default:
		}
	})

	bus.Publish("topic1", "event", nil)
	<-done

	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("expected 1 call, got %d", count)
	}

	sub.Unsubscribe()

	bus.Publish("topic2", "event", nil)
	time.Sleep(20 * time.Millisecond)

	if atomic.LoadInt32(&count) != 1 {
		t.Errorf("expected still 1 call after unsubscribe, got %d", count)
	}
}

func TestBusSubscribeAllWithRegularSubscribers(t *testing.T) {
	bus := NewBus()

	var wildcardCount, regularCount int32
	var wg sync.WaitGroup

	bus.SubscribeAll(func(topic Topic, event string, data interface{}) {
		atomic.AddInt32(&wildcardCount, 1)
		wg.Done()
	})

	bus.Subscribe("specific-topic", func(event string, data interface{}) {
		atomic.AddInt32(&regularCount, 1)
		wg.Done()
	})

	wg.Add(2)
	bus.Publish("specific-topic", "event", nil)
	wg.Wait()

	if atomic.LoadInt32(&wildcardCount) != 1 {
		t.Errorf("expected wildcard called once, got %d", wildcardCount)
	}
	if atomic.LoadInt32(&regularCount) != 1 {
		t.Errorf("expected regular called once, got %d", regularCount)
	}

	wg.Add(1)
	bus.Publish("other-topic", "event", nil)
	wg.Wait()

	if atomic.LoadInt32(&wildcardCount) != 2 {
		t.Errorf("expected wildcard called twice, got %d", wildcardCount)
	}
	if atomic.LoadInt32(&regularCount) != 1 {
		t.Errorf("expected regular still 1, got %d", regularCount)
	}
}

func TestBusSubscribeAllMultipleWildcards(t *testing.T) {
	bus := NewBus()

	var count1, count2 int32
	var wg sync.WaitGroup
	wg.Add(2)

	bus.SubscribeAll(func(topic Topic, event string, data interface{}) {
		atomic.AddInt32(&count1, 1)
		wg.Done()
	})

	bus.SubscribeAll(func(topic Topic, event string, data interface{}) {
		atomic.AddInt32(&count2, 1)
		wg.Done()
	})

	bus.Publish("topic", "event", nil)

	wg.Wait()

	if atomic.LoadInt32(&count1) != 1 {
		t.Errorf("expected wildcard 1 called once, got %d", count1)
	}
	if atomic.LoadInt32(&count2) != 1 {
		t.Errorf("expected wildcard 2 called once, got %d", count2)
	}
}

func TestBusSubscribeAllPanicRecovery(t *testing.T) {
	bus := NewBus()

	var called int32
	var wg sync.WaitGroup
	wg.Add(2)

	bus.SubscribeAll(func(topic Topic, event string, data interface{}) {
		defer wg.Done()
		panic("wildcard panic")
	})

	bus.SubscribeAll(func(topic Topic, event string, data interface{}) {
		defer wg.Done()
		atomic.StoreInt32(&called, 1)
	})

	defer func() {
		if r := recover(); r != nil {
			t.Error("expected Publish to recover from wildcard panic")
		}
	}()

	bus.Publish("topic", "event", nil)

	wg.Wait()

	if atomic.LoadInt32(&called) != 1 {
		t.Error("expected second wildcard to be called despite first panicking")
	}
}

func TestBusSubscribeAllNilSafety(t *testing.T) {
	var bus *Bus

	sub := bus.SubscribeAll(func(topic Topic, event string, data interface{}) {})
	if sub == nil {
		t.Error("expected non-nil subscription from nil bus")
	}

	sub.Unsubscribe()

	bus2 := NewBus()
	sub2 := bus2.SubscribeAll(nil)
	if sub2 == nil {
		t.Error("expected non-nil subscription for nil callback")
	}
}

func TestBusClose(t *testing.T) {
	bus := NewBus()

	bus.Subscribe("topic1", func(event string, data interface{}) {})
	bus.Subscribe("topic2", func(event string, data interface{}) {})
	bus.SubscribeAll(func(topic Topic, event string, data interface{}) {})

	if bus.SubscriberCount("topic1") != 1 {
		t.Error("expected 1 subscriber for topic1")
	}

	bus.Close()

	if bus.SubscriberCount("topic1") != 0 {
		t.Error("expected 0 subscribers after Close")
	}
	if bus.SubscriberCount("topic2") != 0 {
		t.Error("expected 0 subscribers after Close")
	}
}

func TestBusCloseNil(t *testing.T) {
	var bus *Bus
	bus.Close()
}

func TestBusReportDiagnostic(t *testing.T) {
	bus := NewBus()

	var received Diagnostic
	var wg sync.WaitGroup
	wg.Add(1)

	bus.Subscribe(TopicDiagnostic, func(event string, data interface{}) {
		if event == "report" {
			if diag, ok := data.(Diagnostic); ok {
				received = diag
			}
		}
		wg.Done()
	})

	diag := Diagnostic{
		Phase:      "render",
		Message:    "test error",
		StackTrace: "stack...",
		Metadata:   map[string]any{"key": "value"},
	}

	bus.ReportDiagnostic(diag)

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for diagnostic")
	}

	if received.Phase != "render" {
		t.Errorf("Phase = %q, want %q", received.Phase, "render")
	}
	if received.Message != "test error" {
		t.Errorf("Message = %q, want %q", received.Message, "test error")
	}
}

func TestBusSubscribeToDiagnostics(t *testing.T) {
	bus := NewBus()

	var received Diagnostic
	var wg sync.WaitGroup
	wg.Add(1)

	sub := bus.SubscribeToDiagnostics(func(diag Diagnostic) {
		received = diag
		wg.Done()
	})

	if sub == nil {
		t.Fatal("expected non-nil subscription")
	}

	bus.ReportDiagnostic(Diagnostic{
		Phase:   "mount",
		Message: "mount error",
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for diagnostic")
	}

	if received.Phase != "mount" {
		t.Errorf("Phase = %q, want %q", received.Phase, "mount")
	}
}

func TestDecodePayload(t *testing.T) {
	t.Run("direct type match", func(t *testing.T) {
		data := "hello"
		result, ok := DecodePayload[string](data)
		if !ok {
			t.Error("expected ok to be true")
		}
		if result != "hello" {
			t.Errorf("result = %q, want %q", result, "hello")
		}
	})

	t.Run("json decode struct", func(t *testing.T) {
		data := map[string]interface{}{
			"channel": "test",
			"event":   "msg",
		}
		result, ok := DecodePayload[ChannelMessagePayload](data)
		if !ok {
			t.Error("expected ok to be true")
		}
		if result.Channel != "test" {
			t.Errorf("Channel = %q, want %q", result.Channel, "test")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		data := make(chan int)
		_, ok := DecodePayload[string](data)
		if ok {
			t.Error("expected ok to be false for unmarshalable type")
		}
	})
}

func TestServerErrorError(t *testing.T) {
	t.Run("non-nil error", func(t *testing.T) {
		err := &ServerError{
			Code:    "ERR001",
			Message: "test error message",
		}
		if err.Error() != "test error message" {
			t.Errorf("Error() = %q, want %q", err.Error(), "test error message")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		var err *ServerError
		if err.Error() != "" {
			t.Errorf("Error() = %q, want empty string", err.Error())
		}
	})
}
