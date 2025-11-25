package runtime

import (
	"sync"
	"testing"
)

func TestBusSubscribeAndPublish(t *testing.T) {
	bus := NewBus()

	received := false
	var receivedEvent string
	var receivedData interface{}

	sub := bus.Subscribe("test-id", func(event string, data interface{}) {
		received = true
		receivedEvent = event
		receivedData = data
	})

	if sub == nil {
		t.Fatal("expected subscription to be non-nil")
	}

	bus.Publish("test-id", "click", "payload")

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

	count1 := 0
	count2 := 0

	bus.Subscribe("test-id", func(event string, data interface{}) {
		count1++
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		count2++
	})

	bus.Publish("test-id", "event", nil)

	if count1 != 1 {
		t.Errorf("expected subscriber 1 called once, got %d", count1)
	}
	if count2 != 1 {
		t.Errorf("expected subscriber 2 called once, got %d", count2)
	}

	bus.Publish("test-id", "event2", nil)

	if count1 != 2 {
		t.Errorf("expected subscriber 1 called twice, got %d", count1)
	}
	if count2 != 2 {
		t.Errorf("expected subscriber 2 called twice, got %d", count2)
	}
}

func TestBusUnsubscribe(t *testing.T) {
	bus := NewBus()

	count := 0
	sub := bus.Subscribe("test-id", func(event string, data interface{}) {
		count++
	})

	bus.Publish("test-id", "event", nil)
	if count != 1 {
		t.Errorf("expected callback called once, got %d", count)
	}

	sub.Unsubscribe()

	bus.Publish("test-id", "event2", nil)
	if count != 1 {
		t.Errorf("expected callback still called once (unsubscribed), got %d", count)
	}
}

func TestBusUnsubscribeOneOfMany(t *testing.T) {
	bus := NewBus()

	count1 := 0
	count2 := 0

	sub1 := bus.Subscribe("test-id", func(event string, data interface{}) {
		count1++
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		count2++
	})

	bus.Publish("test-id", "event", nil)

	if count1 != 1 || count2 != 1 {
		t.Errorf("expected both called once, got %d and %d", count1, count2)
	}

	sub1.Unsubscribe()

	bus.Publish("test-id", "event2", nil)

	if count1 != 1 {
		t.Errorf("expected sub1 still 1 (unsubscribed), got %d", count1)
	}
	if count2 != 2 {
		t.Errorf("expected sub2 called twice, got %d", count2)
	}
}

func TestBusDifferentIDs(t *testing.T) {
	bus := NewBus()

	count1 := 0
	count2 := 0

	bus.Subscribe("id1", func(event string, data interface{}) {
		count1++
	})

	bus.Subscribe("id2", func(event string, data interface{}) {
		count2++
	})

	bus.Publish("id1", "event", nil)

	if count1 != 1 {
		t.Errorf("expected id1 subscriber called, got %d", count1)
	}
	if count2 != 0 {
		t.Errorf("expected id2 subscriber not called, got %d", count2)
	}

	bus.Publish("id2", "event", nil)

	if count1 != 1 {
		t.Errorf("expected id1 subscriber still 1, got %d", count1)
	}
	if count2 != 1 {
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

	// Concurrent unsubscribes
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
	sub := bus.Upsert("test-id", func(event string, data interface{}) {
		called = true
	})

	if sub == nil {
		t.Fatal("expected non-nil subscription")
	}

	bus.Publish("test-id", "event", nil)

	if !called {
		t.Error("expected callback to be called")
	}

	if bus.SubscriberCount("test-id") != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount("test-id"))
	}
}

func TestBusUpsertUpdatesExisting(t *testing.T) {
	bus := NewBus()

	count1 := 0
	bus.Upsert("test-id", func(event string, data interface{}) {
		count1++
	})

	bus.Publish("test-id", "event", nil)
	if count1 != 1 {
		t.Errorf("expected count1=1, got %d", count1)
	}

	count2 := 0
	bus.Upsert("test-id", func(event string, data interface{}) {
		count2++
	})

	if bus.SubscriberCount("test-id") != 1 {
		t.Errorf("expected 1 subscriber after upsert, got %d", bus.SubscriberCount("test-id"))
	}

	bus.Publish("test-id", "event", nil)

	if count1 != 1 {
		t.Errorf("expected count1 still 1 (old callback replaced), got %d", count1)
	}
	if count2 != 1 {
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

	handlerID := "h-123"

	value1 := 0
	bus.Upsert(handlerID, func(event string, data interface{}) {
		if event == "invoke" {
			value1 = 1
		}
	})

	bus.Publish(handlerID, "invoke", nil)
	if value1 != 1 {
		t.Errorf("expected value1=1, got %d", value1)
	}

	value2 := 0
	bus.Upsert(handlerID, func(event string, data interface{}) {
		if event == "invoke" {
			value2 = 2
		}
	})

	if bus.SubscriberCount(handlerID) != 1 {
		t.Errorf("expected 1 subscriber, got %d", bus.SubscriberCount(handlerID))
	}

	bus.Publish(handlerID, "invoke", nil)

	if value1 != 1 {
		t.Errorf("expected value1 still 1, got %d", value1)
	}
	if value2 != 2 {
		t.Errorf("expected value2=2, got %d", value2)
	}
}

func TestBusUpsertCollapsesToSingleSubscriber(t *testing.T) {
	bus := NewBus()

	legacyCount := 0
	bus.Subscribe("id", func(event string, data interface{}) {
		legacyCount++
	})
	bus.Subscribe("id", func(event string, data interface{}) {
		legacyCount++
	})

	newCount := 0
	bus.Upsert("id", func(event string, data interface{}) {
		newCount++
	})

	bus.Publish("id", "event", nil)

	if bus.SubscriberCount("id") != 1 {
		t.Fatalf("expected subscribers collapsed to 1, got %d", bus.SubscriberCount("id"))
	}
	if legacyCount != 0 {
		t.Fatalf("expected legacy subscribers removed, got legacyCount=%d", legacyCount)
	}
	if newCount != 1 {
		t.Fatalf("expected new upsert callback called once, got %d", newCount)
	}
}

func TestBusPublishRecoversFromPanic(t *testing.T) {
	bus := NewBus()

	called := false

	bus.Subscribe("id", func(event string, data interface{}) {
		panic("boom")
	})
	bus.Subscribe("id", func(event string, data interface{}) {
		called = true
	})

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Publish should recover panics, but panic propagated: %v", r)
		}
	}()

	bus.Publish("id", "event", nil)

	if !called {
		t.Fatalf("expected non-panicking subscriber to be invoked despite panic in another")
	}
}

func TestBusPublishPanicRecovery(t *testing.T) {
	bus := NewBus()

	called1 := false
	called3 := false

	bus.Subscribe("test-id", func(event string, data interface{}) {
		called1 = true
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		panic("intentional panic")
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		called3 = true
	})

	defer func() {
		if r := recover(); r != nil {
			t.Error("expected Publish to recover from panic, but it propagated")
		}
	}()

	bus.Publish("test-id", "event", nil)

	if !called1 {
		t.Error("expected first subscriber to be called")
	}
	if !called3 {
		t.Error("expected third subscriber to be called despite second panicking")
	}
}

func TestBusPublishPanicInUpsert(t *testing.T) {
	bus := NewBus()

	called := false

	bus.Upsert("test-id", func(event string, data interface{}) {
		panic("upsert panic")
	})

	defer func() {
		if r := recover(); r != nil {
			t.Error("expected Publish to recover from panic in Upsert callback")
		}
	}()

	bus.Publish("test-id", "event", nil)

	if bus.SubscriberCount("test-id") != 1 {
		t.Error("expected subscriber to still exist after panic")
	}

	bus.Upsert("test-id", func(event string, data interface{}) {
		called = true
	})

	bus.Publish("test-id", "event", nil)

	if !called {
		t.Error("expected updated callback to be called")
	}
}

func TestBusPublishMultiplePanics(t *testing.T) {
	bus := NewBus()

	count := 0

	bus.Subscribe("test-id", func(event string, data interface{}) {
		count++
		panic("panic 1")
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		count++
		panic("panic 2")
	})

	bus.Subscribe("test-id", func(event string, data interface{}) {
		count++
		panic("panic 3")
	})

	defer func() {
		if r := recover(); r != nil {
			t.Error("expected Publish to handle multiple panics")
		}
	}()

	bus.Publish("test-id", "event", nil)

	if count != 3 {
		t.Errorf("expected all 3 subscribers to be called, got %d", count)
	}
}

func TestBusSubscribeAllReceivesAllMessages(t *testing.T) {
	bus := NewBus()

	var received []struct {
		topic string
		event string
		data  interface{}
	}

	bus.SubscribeAll(func(topic string, event string, data interface{}) {
		received = append(received, struct {
			topic string
			event string
			data  interface{}
		}{topic, event, data})
	})

	bus.Publish("topic1", "event1", "data1")
	bus.Publish("topic2", "event2", "data2")
	bus.Publish("topic3", "event3", "data3")

	if len(received) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(received))
	}

	if received[0].topic != "topic1" || received[0].event != "event1" || received[0].data != "data1" {
		t.Errorf("message 0 mismatch: %+v", received[0])
	}
	if received[1].topic != "topic2" || received[1].event != "event2" || received[1].data != "data2" {
		t.Errorf("message 1 mismatch: %+v", received[1])
	}
	if received[2].topic != "topic3" || received[2].event != "event3" || received[2].data != "data3" {
		t.Errorf("message 2 mismatch: %+v", received[2])
	}
}

func TestBusSubscribeAllUnsubscribe(t *testing.T) {
	bus := NewBus()

	count := 0
	sub := bus.SubscribeAll(func(topic string, event string, data interface{}) {
		count++
	})

	bus.Publish("topic1", "event", nil)
	if count != 1 {
		t.Errorf("expected 1 call, got %d", count)
	}

	sub.Unsubscribe()

	bus.Publish("topic2", "event", nil)
	if count != 1 {
		t.Errorf("expected still 1 call after unsubscribe, got %d", count)
	}
}

func TestBusSubscribeAllWithRegularSubscribers(t *testing.T) {
	bus := NewBus()

	wildcardCount := 0
	regularCount := 0

	bus.SubscribeAll(func(topic string, event string, data interface{}) {
		wildcardCount++
	})

	bus.Subscribe("specific-topic", func(event string, data interface{}) {
		regularCount++
	})

	bus.Publish("specific-topic", "event", nil)

	if wildcardCount != 1 {
		t.Errorf("expected wildcard called once, got %d", wildcardCount)
	}
	if regularCount != 1 {
		t.Errorf("expected regular called once, got %d", regularCount)
	}

	bus.Publish("other-topic", "event", nil)

	if wildcardCount != 2 {
		t.Errorf("expected wildcard called twice, got %d", wildcardCount)
	}
	if regularCount != 1 {
		t.Errorf("expected regular still 1, got %d", regularCount)
	}
}

func TestBusSubscribeAllMultipleWildcards(t *testing.T) {
	bus := NewBus()

	count1 := 0
	count2 := 0

	bus.SubscribeAll(func(topic string, event string, data interface{}) {
		count1++
	})

	bus.SubscribeAll(func(topic string, event string, data interface{}) {
		count2++
	})

	bus.Publish("topic", "event", nil)

	if count1 != 1 {
		t.Errorf("expected wildcard 1 called once, got %d", count1)
	}
	if count2 != 1 {
		t.Errorf("expected wildcard 2 called once, got %d", count2)
	}
}

func TestBusSubscribeAllPanicRecovery(t *testing.T) {
	bus := NewBus()

	called := false

	bus.SubscribeAll(func(topic string, event string, data interface{}) {
		panic("wildcard panic")
	})

	bus.SubscribeAll(func(topic string, event string, data interface{}) {
		called = true
	})

	defer func() {
		if r := recover(); r != nil {
			t.Error("expected Publish to recover from wildcard panic")
		}
	}()

	bus.Publish("topic", "event", nil)

	if !called {
		t.Error("expected second wildcard to be called despite first panicking")
	}
}

func TestBusSubscribeAllNilSafety(t *testing.T) {
	var bus *Bus

	sub := bus.SubscribeAll(func(topic string, event string, data interface{}) {})
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
