package router

import "sync"

type Subscription struct {
	event   string
	fn      func(NavigationEvent)
	emitter *RouterEventEmitter
	active  bool
}

func (s *Subscription) Unsubscribe() {
	if s == nil {
		return
	}
	s.active = false
}

type RouterEventEmitter struct {
	mu          sync.RWMutex
	subscribers map[string][]*Subscription
}

func NewRouterEventEmitter() *RouterEventEmitter {
	return &RouterEventEmitter{
		subscribers: make(map[string][]*Subscription),
	}
}

func (e *RouterEventEmitter) Subscribe(event string, fn func(NavigationEvent)) *Subscription {
	e.mu.Lock()
	defer e.mu.Unlock()

	sub := &Subscription{
		event:   event,
		fn:      fn,
		emitter: e,
		active:  true,
	}
	e.subscribers[event] = append(e.subscribers[event], sub)
	return sub
}

func (e *RouterEventEmitter) Emit(event string, evt NavigationEvent) {
	e.mu.RLock()
	subs := e.subscribers[event]
	e.mu.RUnlock()

	for _, sub := range subs {
		if sub.active {
			sub.fn(evt)
		}
	}
}

func (e *RouterEventEmitter) cleanup() {
	e.mu.Lock()
	defer e.mu.Unlock()

	for event, subs := range e.subscribers {
		active := make([]*Subscription, 0, len(subs))
		for _, sub := range subs {
			if sub.active {
				active = append(active, sub)
			}
		}
		e.subscribers[event] = active
	}
}
