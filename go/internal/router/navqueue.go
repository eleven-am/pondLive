package router

import "sync"

// NavQueue drains navigation events in FIFO order before dispatch.
type NavQueue struct {
	mu      sync.Mutex
	pending []NavEvent
}

// Enqueue appends events to the queue.
func (q *NavQueue) Enqueue(events []NavEvent) {
	if len(events) == 0 {
		return
	}
	q.mu.Lock()
	q.pending = append(q.pending, events...)
	q.mu.Unlock()
}

// Drain swaps pending events with the provided store drain.
func (q *NavQueue) Drain() []NavEvent {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.pending) == 0 {
		return nil
	}
	events := q.pending
	q.pending = nil
	return events
}
