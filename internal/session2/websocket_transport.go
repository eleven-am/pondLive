package session2

import (
	"encoding/json"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/eleven-am/pondlive/go/internal/headers2"
)

// Message represents a transport message with sequence tracking.
type Message struct {
	Seq   uint64 `json:"seq"`
	Topic string `json:"topic"`
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// ChannelSender abstracts the pondsocket channel for sending messages to clients.
type ChannelSender interface {
	BroadcastTo(event string, payload any, userIDs ...string) error
}

// WebSocketTransport provides reliable message delivery over WebSocket.
// It tracks sequence numbers and pending messages for acknowledgement.
type WebSocketTransport struct {
	sender  ChannelSender
	userID  string
	nextSeq uint64

	mu          sync.Mutex
	pending     map[uint64]Message
	closed      bool
	requestInfo *headers2.RequestInfo
}

// NewWebSocketTransport creates a transport with reliable delivery.
// The sender is the pondsocket channel, userID identifies this session's client.
// Headers should include the Cookie header if cookies are needed.
func NewWebSocketTransport(sender ChannelSender, userID string, headers http.Header) *WebSocketTransport {
	return &WebSocketTransport{
		sender:      sender,
		userID:      userID,
		pending:     make(map[uint64]Message),
		requestInfo: headers2.NewRequestInfoFromHeaders(headers),
	}
}

// RequestInfo returns the HTTP request information captured at transport creation.
// For WebSocket, this is from the handshake request (may be updated on reconnect).
func (t *WebSocketTransport) RequestInfo() *headers2.RequestInfo {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.requestInfo
}

// UpdateRequestInfo updates the request info from headers (used on reconnection).
func (t *WebSocketTransport) UpdateRequestInfo(headers http.Header) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.requestInfo = headers2.NewRequestInfoFromHeaders(headers)
	t.mu.Unlock()
}

// Send transmits a message with sequence tracking.
func (t *WebSocketTransport) Send(topic, event string, data any) error {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.mu.Unlock()

	seq := atomic.AddUint64(&t.nextSeq, 1)
	msg := Message{
		Seq:   seq,
		Topic: topic,
		Event: event,
		Data:  data,
	}

	t.mu.Lock()
	t.pending[seq] = msg
	t.mu.Unlock()

	if err := t.sender.BroadcastTo(event, msg, t.userID); err != nil {
		return err
	}

	return nil
}

// Ack acknowledges receipt of a message, removing it from pending.
func (t *WebSocketTransport) Ack(seq uint64) {
	if t == nil {
		return
	}

	t.mu.Lock()
	delete(t.pending, seq)
	t.mu.Unlock()
}

// AckThrough acknowledges all messages up to and including the given sequence.
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

// Pending returns the number of unacknowledged messages.
func (t *WebSocketTransport) Pending() int {
	if t == nil {
		return 0
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.pending)
}

// PendingMessages returns a copy of all unacknowledged messages.
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

// Resend retransmits all pending messages.
// Useful after reconnection.
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
	t.mu.Unlock()

	for _, msg := range msgs {
		if err := t.sender.BroadcastTo(msg.Event, msg, t.userID); err != nil {
			return err
		}
	}

	return nil
}

// SetSender replaces the underlying channel sender.
// Useful for reconnection scenarios.
func (t *WebSocketTransport) SetSender(sender ChannelSender) {
	if t == nil {
		return
	}

	t.mu.Lock()
	t.sender = sender
	t.closed = false
	t.mu.Unlock()
}

// LastSeq returns the last sequence number used.
func (t *WebSocketTransport) LastSeq() uint64 {
	if t == nil {
		return 0
	}
	return atomic.LoadUint64(&t.nextSeq)
}

// IsLive returns true since WebSocket is a live connection.
func (t *WebSocketTransport) IsLive() bool {
	return true
}

// Close marks the transport as closed.
// Note: The sender is not closed as it's typically a shared resource.
func (t *WebSocketTransport) Close() error {
	if t == nil {
		return nil
	}

	t.mu.Lock()
	t.closed = true
	t.mu.Unlock()

	return nil
}

// MarshalMessage serializes a message to JSON bytes.
func MarshalMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

// UnmarshalMessage deserializes JSON bytes to a message.
func UnmarshalMessage(data []byte) (Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return msg, err
}
