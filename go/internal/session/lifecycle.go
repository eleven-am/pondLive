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

	lastTouch time.Time
	observers []chan struct{}
}

// NewLifecycle creates a lifecycle manager with the given clock and TTL.
func NewLifecycle(clock func() time.Time, ttl time.Duration) *Lifecycle {
	return &Lifecycle{
		clock:     clock,
		ttl:       ttl,
		lastTouch: clock(),
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
func (l *Lifecycle) IsExpired() bool {
	if l == nil {
		return true
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
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
func (l *Lifecycle) OnTouch(ch chan struct{}) {
	if l == nil {
		return
	}
	l.mu.Lock()
	l.observers = append(l.observers, ch)
	l.mu.Unlock()
}

func (l *Lifecycle) notifyObservers() {
	if l == nil {
		return
	}
	l.mu.RLock()
	observers := append([]chan struct{}(nil), l.observers...)
	l.mu.RUnlock()

	for _, ch := range observers {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
