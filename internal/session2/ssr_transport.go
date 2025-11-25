package session2

import (
	"net/http"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/headers2"
)

// SSRTransport buffers messages during server-side rendering.
// It collects all messages sent during the render cycle for embedding in HTML.
type SSRTransport struct {
	mu          sync.Mutex
	messages    []Message
	seq         uint64
	closed      bool
	requestInfo *headers2.RequestInfo
}

// NewSSRTransport creates a transport for server-side rendering.
// The request is captured at creation time and made available via RequestInfo().
func NewSSRTransport(r *http.Request) *SSRTransport {
	return &SSRTransport{
		messages:    make([]Message, 0),
		requestInfo: headers2.NewRequestInfo(r),
	}
}

// RequestInfo returns the HTTP request information captured at transport creation.
func (t *SSRTransport) RequestInfo() *headers2.RequestInfo {
	if t == nil {
		return nil
	}
	return t.requestInfo
}

// Send buffers a message for later retrieval.
func (t *SSRTransport) Send(topic, event string, data any) error {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}

	t.seq++
	t.messages = append(t.messages, Message{
		Seq:   t.seq,
		Topic: topic,
		Event: event,
		Data:  data,
	})

	return nil
}

// IsLive returns false since SSR is not a live connection.
func (t *SSRTransport) IsLive() bool {
	return false
}

// Close marks the transport as closed.
func (t *SSRTransport) Close() error {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()

	return nil
}

// Messages returns all buffered messages.
func (t *SSRTransport) Messages() []Message {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	result := make([]Message, len(t.messages))
	copy(result, t.messages)
	return result
}

// LastSeq returns the last sequence number used.
func (t *SSRTransport) LastSeq() uint64 {
	if t == nil {
		return 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	return t.seq
}

// Clear removes all buffered messages.
func (t *SSRTransport) Clear() {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.messages = t.messages[:0]
	t.mu.Unlock()
}

// Drain returns all buffered messages and clears the buffer.
func (t *SSRTransport) Drain() []Message {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	result := make([]Message, len(t.messages))
	copy(result, t.messages)
	t.messages = t.messages[:0]
	return result
}

// FilterByTopic returns messages matching the given topic.
func (t *SSRTransport) FilterByTopic(topic string) []Message {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	var result []Message
	for _, msg := range t.messages {
		if msg.Topic == topic {
			result = append(result, msg)
		}
	}
	return result
}

// FilterByEvent returns messages matching the given event.
func (t *SSRTransport) FilterByEvent(event string) []Message {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	var result []Message
	for _, msg := range t.messages {
		if msg.Event == event {
			result = append(result, msg)
		}
	}
	return result
}
