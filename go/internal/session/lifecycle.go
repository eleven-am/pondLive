package session

import (
	"sync"
	"time"
)

type Lifecycle struct {
	mu    sync.RWMutex
	clock func() time.Time
	ttl   time.Duration

	lastTouch      time.Time
	observers      map[int]chan struct{}
	nextObserverID int
}

func NewLifecycle(clock func() time.Time, ttl time.Duration) *Lifecycle {
	return &Lifecycle{
		clock:     clock,
		ttl:       ttl,
		lastTouch: clock(),
		observers: make(map[int]chan struct{}),
	}
}

func (l *Lifecycle) Touch() {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.lastTouch = l.clock()
	l.mu.Unlock()
	l.notifyObservers()
}

func (l *Lifecycle) IsExpired() bool {
	if l == nil {
		return true
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.ttl <= 0 {
		return false
	}
	return l.clock().Sub(l.lastTouch) > l.ttl
}

func (l *Lifecycle) TTL() time.Duration {
	if l == nil {
		return 0
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.ttl
}

func (l *Lifecycle) SetTTL(ttl time.Duration) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.ttl = ttl
	l.mu.Unlock()
}

func (l *Lifecycle) SetClock(clock func() time.Time) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.clock = clock
	l.mu.Unlock()
}

func (l *Lifecycle) LastTouch() time.Time {
	if l == nil {
		return time.Time{}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.lastTouch
}

func (l *Lifecycle) OnTouch(ch chan struct{}) func() {
	if l == nil || ch == nil {
		return func() {}
	}
	l.mu.Lock()
	id := l.nextObserverID
	l.nextObserverID++
	l.observers[id] = ch
	l.mu.Unlock()

	return func() {
		l.mu.Lock()
		delete(l.observers, id)
		l.mu.Unlock()
	}
}

func (l *Lifecycle) notifyObservers() {
	if l == nil {
		return
	}
	l.mu.RLock()
	observers := make([]chan struct{}, 0, len(l.observers))
	for _, ch := range l.observers {
		observers = append(observers, ch)
	}
	l.mu.RUnlock()

	for _, ch := range observers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
