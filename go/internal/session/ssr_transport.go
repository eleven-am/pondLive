package session

import (
	"errors"
	"net/http"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/headers"
)

type SSRTransport struct {
	mu          sync.Mutex
	messages    []Message
	seq         uint64
	closed      bool
	requestInfo *headers.RequestInfo
	maxMessages int
}

func NewSSRTransport(r *http.Request) *SSRTransport {
	return &SSRTransport{
		messages:    make([]Message, 0),
		maxMessages: 10000,
		requestInfo: headers.NewRequestInfo(r),
	}
}

func (t *SSRTransport) RequestInfo() *headers.RequestInfo {
	if t == nil {
		return nil
	}
	return t.requestInfo
}

func (t *SSRTransport) Send(topic, event string, data any) error {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	if t.closed {
		return nil
	}
	if t.maxMessages > 0 && len(t.messages) >= t.maxMessages {
		return errors.New("ssr: buffer limit exceeded")
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

func (t *SSRTransport) IsLive() bool {
	return false
}

func (t *SSRTransport) Close() error {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()

	return nil
}

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

func (t *SSRTransport) LastSeq() uint64 {
	if t == nil {
		return 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	return t.seq
}

func (t *SSRTransport) Clear() {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.messages = t.messages[:0]
	t.mu.Unlock()
}

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

func (t *SSRTransport) SetMaxMessages(n int) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.maxMessages = n
	t.mu.Unlock()
}

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
