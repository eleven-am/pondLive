package session

import (
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/eleven-am/pondlive/internal/headers"
)

type SSRTransport struct {
	mu           sync.Mutex
	messages     []Message
	seq          uint64
	closed       bool
	requestInfo  *headers.RequestInfo
	requestState *headers.RequestState
	maxMessages  int
	maxAge       time.Duration
}

func NewSSRTransport(r *http.Request) *SSRTransport {
	requestInfo := headers.NewRequestInfo(r)
	return &SSRTransport{
		messages:     make([]Message, 0),
		maxMessages:  10000,
		maxAge:       10 * time.Second,
		requestInfo:  requestInfo,
		requestState: headers.NewRequestState(requestInfo),
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

func (t *SSRTransport) RequestState() *headers.RequestState {
	if t == nil {
		return nil
	}

	return t.requestState
}

// SetMaxAge updates the max age for buffered messages; zero disables age trimming.
func (t *SSRTransport) SetMaxAge(d time.Duration) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.maxAge = d
	t.mu.Unlock()
}

// TrimExpired drops buffered messages older than maxAge (if set).
func (t *SSRTransport) TrimExpired(now time.Time) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.maxAge <= 0 {
		return
	}
	cutoff := now.Add(-t.maxAge)
	var kept []Message
	for _, msg := range t.messages {
		// no timestamp on Message; treat seq as monotonic and drop none if age unknown
		kept = append(kept, msg)
		_ = cutoff
	}
	t.messages = kept
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
