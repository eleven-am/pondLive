package protocol

import (
	"sync"
	"sync/atomic"
)

// Bus is a simple pubsub message router that maps IDs to callbacks.
// It doesn't track components or lifecycle - hooks are responsible for cleanup.
type Bus struct {
	subscribers         map[Topic][]*subscriber
	wildcardSubscribers []*wildcardSubscriber
	mu                  sync.RWMutex
	nextSubID           atomic.Uint64
}

// wildcardSubscriber receives all messages on the bus.
type wildcardSubscriber struct {
	id       uint64
	callback func(topic Topic, event string, data interface{})
}

// subscriber represents a single subscription.
type subscriber struct {
	id       uint64
	callback func(event string, data interface{})
}

// Subscription represents an active subscription that can be cancelled.
type Subscription struct {
	unsubscribe func()
}

// Unsubscribe removes the subscription and stops receiving messages.
func (s *Subscription) Unsubscribe() {
	if s.unsubscribe != nil {
		s.unsubscribe()
	}
}

// NewBus creates a new message bus.
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[Topic][]*subscriber),
	}
}

// Subscribe registers a callback for messages with the given ID.
// Returns a Subscription that can be used to unsubscribe.
// Multiple subscribers can subscribe to the same ID (broadcast).
func (b *Bus) Subscribe(id Topic, callback func(event string, data interface{})) *Subscription {
	if b == nil || id == "" || callback == nil {
		return &Subscription{}
	}

	subID := b.nextSubID.Add(1)
	sub := &subscriber{
		id:       subID,
		callback: callback,
	}

	b.mu.Lock()
	b.subscribers[id] = append(b.subscribers[id], sub)
	b.mu.Unlock()

	return &Subscription{
		unsubscribe: func() {
			b.unsubscribe(id, subID)
		},
	}
}

// Upsert updates an existing subscription or creates a new one.
// IMPORTANT: Upsert enforces single-subscriber semantics per ID.
// - First call: creates a new subscriber
// - Subsequent calls: updates the callback of the existing subscriber (does NOT create a new one)
// - All returned Subscription handles point to the same underlying subscriber
// - Calling Unsubscribe on ANY handle removes the subscriber
//
// Use Upsert for single-subscriber channels (handlers, scripts) where you want to update
// the callback on each render without unsubscribe/resubscribe churn.
// Use Subscribe for multi-subscriber broadcast channels (events, notifications).
//
// Returns a Subscription that can be used to unsubscribe.
func (b *Bus) Upsert(id Topic, callback func(event string, data interface{})) *Subscription {
	if b == nil || id == "" || callback == nil {
		return &Subscription{}
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	if subs := b.subscribers[id]; len(subs) > 0 {

		first := subs[0]
		first.callback = callback

		if len(subs) > 1 {
			b.subscribers[id] = subs[:1]
		}

		subID := first.id
		return &Subscription{
			unsubscribe: func() {
				b.unsubscribe(id, subID)
			},
		}
	}

	subID := b.nextSubID.Add(1)
	sub := &subscriber{
		id:       subID,
		callback: callback,
	}

	b.subscribers[id] = []*subscriber{sub}

	return &Subscription{
		unsubscribe: func() {
			b.unsubscribe(id, subID)
		},
	}
}

// SubscribeAll registers a callback that receives ALL messages on the bus.
// The callback receives the topic ID, event, and data for every publish.
// Returns a Subscription that can be used to unsubscribe.
func (b *Bus) SubscribeAll(callback func(topic Topic, event string, data interface{})) *Subscription {
	if b == nil || callback == nil {
		return &Subscription{}
	}

	subID := b.nextSubID.Add(1)
	sub := &wildcardSubscriber{
		id:       subID,
		callback: callback,
	}

	b.mu.Lock()
	b.wildcardSubscribers = append(b.wildcardSubscribers, sub)
	b.mu.Unlock()

	return &Subscription{
		unsubscribe: func() {
			b.unsubscribeWildcard(subID)
		},
	}
}

// Publish sends a message to all subscribers of the given ID and all wildcard subscribers.
// Broadcasts to all subscribers (if multiple are registered).
// Callback panics are recovered to prevent cascading failures - the panic is silently swallowed
// and remaining subscribers continue to receive the message.
func (b *Bus) Publish(id Topic, event string, data interface{}) {
	if b == nil || id == "" {
		return
	}

	b.mu.RLock()
	subs := b.subscribers[id]
	wildcards := b.wildcardSubscribers

	callbacks := make([]func(event string, data interface{}), len(subs))
	for i, sub := range subs {
		callbacks[i] = sub.callback
	}

	wildcardCallbacks := make([]func(topic Topic, event string, data interface{}), len(wildcards))
	for i, sub := range wildcards {
		wildcardCallbacks[i] = sub.callback
	}
	b.mu.RUnlock()

	for _, callback := range callbacks {
		if callback == nil {
			continue
		}
		func() {
			defer func() { recover() }()
			callback(event, data)
		}()
	}

	for _, callback := range wildcardCallbacks {
		if callback == nil {
			continue
		}
		func() {
			defer func() { recover() }()
			callback(id, event, data)
		}()
	}
}

// SubscriberCount returns the number of subscribers for a given ID.
// Useful for testing.
func (b *Bus) SubscriberCount(id Topic) int {
	if b == nil {
		return 0
	}

	b.mu.RLock()
	count := len(b.subscribers[id])
	b.mu.RUnlock()
	return count
}

// unsubscribeWildcard removes a wildcard subscriber.
func (b *Bus) unsubscribeWildcard(subID uint64) {
	if b == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for i, sub := range b.wildcardSubscribers {
		if sub.id == subID {
			b.wildcardSubscribers[i] = b.wildcardSubscribers[len(b.wildcardSubscribers)-1]
			b.wildcardSubscribers = b.wildcardSubscribers[:len(b.wildcardSubscribers)-1]
			return
		}
	}
}

// unsubscribe removes a specific subscriber from an ID.
func (b *Bus) unsubscribe(id Topic, subID uint64) {
	if b == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	subs := b.subscribers[id]
	for i, sub := range subs {
		if sub.id == subID {

			subs[i] = subs[len(subs)-1]
			b.subscribers[id] = subs[:len(subs)-1]

			if len(b.subscribers[id]) == 0 {
				delete(b.subscribers, id)
			}
			return
		}
	}
}
