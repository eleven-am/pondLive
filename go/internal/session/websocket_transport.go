package session

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/eleven-am/pondlive/go/internal/headers"
)

type Message struct {
	Seq   uint64 `json:"seq"`
	Topic string `json:"topic"`
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type ChannelSender interface {
	BroadcastTo(event string, payload any, userIDs ...string) error
}

type WebSocketTransport struct {
	sender  ChannelSender
	userID  string
	nextSeq uint64

	mu          sync.Mutex
	pending     map[uint64]Message
	closed      bool
	requestInfo *headers.RequestInfo
}

func NewWebSocketTransport(sender ChannelSender, userID string, h http.Header) *WebSocketTransport {
	return &WebSocketTransport{
		sender:      sender,
		userID:      userID,
		pending:     make(map[uint64]Message),
		requestInfo: headers.NewRequestInfoFromHeaders(h),
	}
}

func (t *WebSocketTransport) RequestInfo() *headers.RequestInfo {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.requestInfo
}

func (t *WebSocketTransport) RequestState() *headers.RequestState {
	if t == nil {
		return nil
	}

	return headers.NewRequestState(t.requestInfo)
}

func (t *WebSocketTransport) UpdateRequestInfo(h http.Header) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.requestInfo = headers.NewRequestInfoFromHeaders(h)
	t.mu.Unlock()
}

func (t *WebSocketTransport) Send(topic, event string, data any) error {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}

	seq := atomic.AddUint64(&t.nextSeq, 1)
	msg := Message{
		Seq:   seq,
		Topic: topic,
		Event: event,
		Data:  data,
	}
	t.pending[seq] = msg
	sender := t.sender
	userID := t.userID
	t.mu.Unlock()

	if err := sender.BroadcastTo(event, msg, userID); err != nil {
		t.mu.Lock()
		delete(t.pending, seq)
		t.mu.Unlock()
		return err
	}

	return nil
}

func (t *WebSocketTransport) AckThrough(seq uint64) {
	if t == nil {
		return
	}

	t.mu.Lock()
	for s := range t.pending {
		if s <= seq {
			delete(t.pending, s)
		}
	}
	t.mu.Unlock()
}

func (t *WebSocketTransport) Pending() int {
	if t == nil {
		return 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.pending)
}

func (t *WebSocketTransport) PendingMessages() []Message {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	msgs := make([]Message, 0, len(t.pending))
	for _, msg := range t.pending {
		msgs = append(msgs, msg)
	}
	return msgs
}

func (t *WebSocketTransport) Resend() error {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}

	msgs := make([]Message, 0, len(t.pending))
	for _, msg := range t.pending {
		msgs = append(msgs, msg)
	}
	sender := t.sender
	userID := t.userID
	t.mu.Unlock()

	var firstErr error
	for _, msg := range msgs {
		if err := sender.BroadcastTo(msg.Event, msg, userID); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (t *WebSocketTransport) SetSender(sender ChannelSender) {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.sender = sender
	t.closed = false
	t.mu.Unlock()
}

func (t *WebSocketTransport) LastSeq() uint64 {
	if t == nil {
		return 0
	}
	return atomic.LoadUint64(&t.nextSeq)
}

func (t *WebSocketTransport) SendAck(sid string) uint64 {
	if t == nil {
		return 0
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return 0
	}

	seq := atomic.AddUint64(&t.nextSeq, 1)

	ack := struct {
		T   string `json:"t"`
		SID string `json:"sid"`
		Seq uint64 `json:"seq"`
	}{
		T:   "ack",
		SID: sid,
		Seq: seq,
	}

	msg := Message{
		Seq:   seq,
		Topic: "ack",
		Event: "ack",
		Data:  ack,
	}

	sender := t.sender
	userID := t.userID
	t.mu.Unlock()

	_ = sender.BroadcastTo("ack", msg, userID)

	return seq
}

func (t *WebSocketTransport) IsLive() bool {
	return true
}

func (t *WebSocketTransport) Close() error {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()

	return nil
}

func MarshalMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

func UnmarshalMessage(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}
