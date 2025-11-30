package session

import (
	"sync"
	"time"
)

// Lifecycle manages session TTL and touch tracking.
type Lifecycle struct {
	mu    sync.RWMutex
	clock func() time.Time
	ttl   time.Duration

	lastTouch      time.Time
	observers      map[int]chan struct{}
	nextObserverID int
}

// NewLifecycle creates a lifecycle manager with the given clock and TTL.
func NewLifecycle(clock func() time.Time, ttl time.Duration) *Lifecycle {
	return &Lifecycle{
		clock:     clock,
		ttl:       ttl,
		lastTouch: clock(),
		observers: make(map[int]chan struct{}),
	}
}

// Touch updates the last activity timestamp.
func (l *Lifecycle) Touch() {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.lastTouch = l.clock()
	l.mu.Unlock()
	l.notifyObservers()
}

// IsExpired returns true if the session has exceeded its TTL.
// A zero or negative TTL means the session never expires.
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

// TTL returns the configured time-to-live.
func (l *Lifecycle) TTL() time.Duration {
	if l == nil {
		return 0
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.ttl
}

// SetTTL updates the time-to-live.
func (l *Lifecycle) SetTTL(ttl time.Duration) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.ttl = ttl
	l.mu.Unlock()
}

// SetClock updates the clock function.
func (l *Lifecycle) SetClock(clock func() time.Time) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.clock = clock
	l.mu.Unlock()
}

// LastTouch returns the timestamp of the last activity.
func (l *Lifecycle) LastTouch() time.Time {
	if l == nil {
		return time.Time{}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.lastTouch
}

// OnTouch registers a channel that receives a signal on each touch.
// Returns an unsubscribe function to remove the observer.
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
