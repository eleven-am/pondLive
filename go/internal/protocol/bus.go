package protocol

import (
	"sync"
	"sync/atomic"
)

type Bus struct {
	subscribers         map[Topic][]*subscriber
	wildcardSubscribers []*wildcardSubscriber
	mu                  sync.RWMutex
	nextSubID           atomic.Uint64
}

type wildcardSubscriber struct {
	id       uint64
	callback func(topic Topic, event string, data interface{})
}

type subscriber struct {
	id       uint64
	callback func(event string, data interface{})
}

type Subscription struct {
	unsubscribe func()
}

func (s *Subscription) Unsubscribe() {
	if s.unsubscribe != nil {
		s.unsubscribe()
	}
}

func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[Topic][]*subscriber),
	}
}

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

func (b *Bus) SubscriberCount(id Topic) int {
	if b == nil {
		return 0
	}

	b.mu.RLock()
	count := len(b.subscribers[id])
	b.mu.RUnlock()
	return count
}

func (b *Bus) unsubscribeWildcard(subID uint64) {
	if b == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	for i, sub := range b.wildcardSubscribers {
		if sub.id == subID {
			b.wildcardSubscribers[i] = b.wildcardSubscribers[len(b.wildcardSubscribers)-1]
			b.wildcardSubscribers[len(b.wildcardSubscribers)-1] = nil
			b.wildcardSubscribers = b.wildcardSubscribers[:len(b.wildcardSubscribers)-1]
			return
		}
	}
}

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
			subs[len(subs)-1] = nil
			b.subscribers[id] = subs[:len(subs)-1]

			if len(b.subscribers[id]) == 0 {
				delete(b.subscribers, id)
			}
			return
		}
	}
}

func (b *Bus) Close() {
	if b == nil {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.subscribers = make(map[Topic][]*subscriber)
	b.wildcardSubscribers = nil
}
