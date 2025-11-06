package runtime

import (
	"sync/atomic"
	"time"
)

type sessionLifecycle struct {
	clock     func() time.Time
	ttl       atomic.Int64
	updatedAt time.Time
	observers []func(time.Time)
}

func newSessionLifecycle(clock func() time.Time, ttl time.Duration) *sessionLifecycle {
	if clock == nil {
		clock = time.Now
	}
	lifecycle := &sessionLifecycle{clock: clock}
	lifecycle.ttl.Store(int64(ttl))
	lifecycle.updatedAt = lifecycle.now()
	return lifecycle
}

func (l *sessionLifecycle) now() time.Time {
	if l == nil {
		return time.Now()
	}
	if l.clock == nil {
		l.clock = time.Now
	}
	return l.clock()
}

func (l *sessionLifecycle) touch() time.Time {
	if l == nil {
		return time.Time{}
	}
	ts := l.now()
	l.updatedAt = ts
	observers := append([]func(time.Time){}, l.observers...)
	for _, cb := range observers {
		if cb != nil {
			cb(ts)
		}
	}
	return ts
}

func (l *sessionLifecycle) setClock(clock func() time.Time) {
	if l == nil {
		return
	}
	if clock == nil {
		clock = time.Now
	}
	l.clock = clock
}

func (l *sessionLifecycle) setTTL(ttl time.Duration) {
	if l == nil {
		return
	}
	l.ttl.Store(int64(ttl))
}

func (l *sessionLifecycle) ttlDuration() time.Duration {
	if l == nil {
		return 0
	}
	return time.Duration(l.ttl.Load())
}

func (l *sessionLifecycle) expired() bool {
	if l == nil {
		return false
	}
	ttl := l.ttl.Load()
	if ttl <= 0 {
		return false
	}
	return l.now().Sub(l.updatedAt) > time.Duration(ttl)
}

func (l *sessionLifecycle) addObserver(cb func(time.Time)) int {
	if l == nil {
		return -1
	}
	l.observers = append(l.observers, cb)
	return len(l.observers) - 1
}

func (l *sessionLifecycle) removeObserver(idx int) {
	if l == nil {
		return
	}
	if idx < 0 || idx >= len(l.observers) {
		return
	}
	l.observers[idx] = nil
}
