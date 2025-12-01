package session

import (
	"errors"
	"net/http"
	"sync"
	"testing"
)

type mockSender struct {
	mu       sync.Mutex
	messages []any
	events   []string
	userIDs  [][]string
	failNext bool
}

func (m *mockSender) BroadcastTo(event string, payload any, userIDs ...string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.failNext {
		m.failNext = false
		return errors.New("broadcast failed")
	}

	m.events = append(m.events, event)
	m.messages = append(m.messages, payload)
	m.userIDs = append(m.userIDs, userIDs)
	return nil
}

func (m *mockSender) MessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func (m *mockSender) LastMessage() any {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return nil
	}
	return m.messages[len(m.messages)-1]
}

func (m *mockSender) LastEvent() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.events) == 0 {
		return ""
	}
	return m.events[len(m.events)-1]
}

func TestWebSocketTransportSend(t *testing.T) {
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", nil)

	err := transport.Send("frame", "patch", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sender.MessageCount() != 1 {
		t.Errorf("expected 1 message, got %d", sender.MessageCount())
	}

	msg, ok := sender.LastMessage().(Message)
	if !ok {
		t.Fatal("expected Message type")
	}

	if msg.Seq != 1 {
		t.Errorf("expected seq 1, got %d", msg.Seq)
	}
	if msg.Topic != "frame" {
		t.Errorf("expected topic 'frame', got '%s'", msg.Topic)
	}
	if msg.Event != "patch" {
		t.Errorf("expected event 'patch', got '%s'", msg.Event)
	}

	if sender.LastEvent() != "patch" {
		t.Errorf("expected BroadcastTo event 'patch', got '%s'", sender.LastEvent())
	}
}

func TestWebSocketTransportSequenceIncrement(t *testing.T) {
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", nil)

	_ = transport.Send("topic1", "event1", nil)
	_ = transport.Send("topic2", "event2", nil)
	_ = transport.Send("topic3", "event3", nil)

	if transport.LastSeq() != 3 {
		t.Errorf("expected last seq 3, got %d", transport.LastSeq())
	}

	if transport.Pending() != 3 {
		t.Errorf("expected 3 pending, got %d", transport.Pending())
	}
}

func TestWebSocketTransportAckThrough(t *testing.T) {
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", nil)

	_ = transport.Send("topic", "event", nil)
	_ = transport.Send("topic", "event", nil)
	_ = transport.Send("topic", "event", nil)
	_ = transport.Send("topic", "event", nil)

	transport.AckThrough(2)

	if transport.Pending() != 2 {
		t.Errorf("expected 2 pending after ack through 2, got %d", transport.Pending())
	}

	pending := transport.PendingMessages()
	for _, msg := range pending {
		if msg.Seq <= 2 {
			t.Errorf("message with seq %d should have been acked", msg.Seq)
		}
	}
}

func TestWebSocketTransportResend(t *testing.T) {
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", nil)

	_ = transport.Send("topic", "event", "data1")
	_ = transport.Send("topic", "event", "data2")

	initialCount := sender.MessageCount()

	err := transport.Resend()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if sender.MessageCount() != initialCount+2 {
		t.Errorf("expected %d messages after resend, got %d", initialCount+2, sender.MessageCount())
	}
}

func TestWebSocketTransportSetSender(t *testing.T) {
	sender1 := &mockSender{}
	transport := NewWebSocketTransport(sender1, "user123", nil)

	_ = transport.Send("topic", "event", nil)

	sender2 := &mockSender{}
	transport.SetSender(sender2)

	_ = transport.Send("topic", "event", nil)

	if sender1.MessageCount() != 1 {
		t.Errorf("expected 1 message on sender1, got %d", sender1.MessageCount())
	}
	if sender2.MessageCount() != 1 {
		t.Errorf("expected 1 message on sender2, got %d", sender2.MessageCount())
	}
}

func TestWebSocketTransportClose(t *testing.T) {
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", nil)

	err := transport.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = transport.Send("topic", "event", nil)
	if err != nil {
		t.Errorf("send after close should not error, got: %v", err)
	}

	if sender.MessageCount() != 0 {
		t.Error("expected no messages sent after close")
	}
}

func TestWebSocketTransportIsLive(t *testing.T) {
	transport := NewWebSocketTransport(nil, "", nil)

	if !transport.IsLive() {
		t.Error("WebSocketTransport should always be live")
	}
}

func TestWebSocketTransportNilSafety(t *testing.T) {
	var transport *WebSocketTransport

	_ = transport.Send("topic", "event", nil)
	transport.AckThrough(1)
	_ = transport.Pending()
	_ = transport.PendingMessages()
	_ = transport.Resend()
	transport.SetSender(nil)
	_ = transport.LastSeq()
	_ = transport.Close()
	_ = transport.RequestInfo()
	transport.UpdateRequestInfo(nil)
}

func TestWebSocketTransportSendError(t *testing.T) {
	sender := &mockSender{failNext: true}
	transport := NewWebSocketTransport(sender, "user123", nil)

	err := transport.Send("topic", "event", nil)
	if err == nil {
		t.Error("expected error on failed broadcast")
	}

	if transport.Pending() != 0 {
		t.Errorf("expected 0 pending messages after failed send, got %d", transport.Pending())
	}
}

func TestMarshalUnmarshalMessage(t *testing.T) {
	msg := Message{
		Seq:   42,
		Topic: "frame",
		Event: "patch",
		Data:  map[string]string{"key": "value"},
	}

	data, err := MarshalMessage(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	parsed, err := UnmarshalMessage(data)
	if err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed.Seq != msg.Seq {
		t.Errorf("seq mismatch: got %d, want %d", parsed.Seq, msg.Seq)
	}
	if parsed.Topic != msg.Topic {
		t.Errorf("topic mismatch: got %s, want %s", parsed.Topic, msg.Topic)
	}
	if parsed.Event != msg.Event {
		t.Errorf("event mismatch: got %s, want %s", parsed.Event, msg.Event)
	}
}

func TestWebSocketTransportRequestInfo(t *testing.T) {

	headers := http.Header{
		"User-Agent":      []string{"pondsocket-client"},
		"Cookie":          []string{"session=abc123; auth=token456"},
		"X-Forwarded-For": []string{"192.168.1.1"},
	}

	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", headers)

	info := transport.RequestInfo()
	if info == nil {
		t.Fatal("expected RequestInfo to be non-nil")
	}

	if ua, ok := info.Get("User-Agent"); !ok || ua != "pondsocket-client" {
		t.Errorf("expected User-Agent pondsocket-client, got %s", ua)
	}

	if cookie, ok := info.GetCookie("session"); !ok || cookie != "abc123" {
		t.Errorf("expected cookie session=abc123, got %s", cookie)
	}

	if cookie, ok := info.GetCookie("auth"); !ok || cookie != "token456" {
		t.Errorf("expected cookie auth=token456, got %s", cookie)
	}
}

func TestWebSocketTransportUpdateRequestInfo(t *testing.T) {
	headers1 := http.Header{"Cookie": []string{"version=1"}}
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", headers1)

	if cookie, ok := transport.RequestInfo().GetCookie("version"); !ok || cookie != "1" {
		t.Errorf("expected version=1, got %s", cookie)
	}

	headers2 := http.Header{"Cookie": []string{"version=2"}}
	transport.UpdateRequestInfo(headers2)

	if cookie, ok := transport.RequestInfo().GetCookie("version"); !ok || cookie != "2" {
		t.Errorf("expected version=2 after update, got %s", cookie)
	}
}

func TestWebSocketTransportRequestInfoNilHeaders(t *testing.T) {
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", nil)

	info := transport.RequestInfo()
	if info == nil {
		t.Fatal("expected RequestInfo to be non-nil even with nil headers")
	}

	if info.Query == nil {
		t.Error("expected Query to be initialized")
	}

	if info.Headers == nil {
		t.Error("expected Headers to be initialized")
	}
}

func TestWebSocketTransportSendAck(t *testing.T) {
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", nil)

	seq := transport.SendAck("session-1")
	if seq != 1 {
		t.Errorf("expected seq 1, got %d", seq)
	}

	if sender.MessageCount() != 1 {
		t.Errorf("expected 1 message, got %d", sender.MessageCount())
	}

	msg, ok := sender.LastMessage().(Message)
	if !ok {
		t.Fatal("expected Message type")
	}

	if msg.Seq != 1 {
		t.Errorf("expected msg seq 1, got %d", msg.Seq)
	}
	if msg.Topic != "ack" {
		t.Errorf("expected topic 'ack', got '%s'", msg.Topic)
	}
	if msg.Event != "ack" {
		t.Errorf("expected event 'ack', got '%s'", msg.Event)
	}

	if transport.Pending() != 0 {
		t.Errorf("expected 0 pending (SendAck is fire-and-forget), got %d", transport.Pending())
	}
}

func TestWebSocketTransportSendAckSequenceAtomic(t *testing.T) {
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", nil)

	_ = transport.Send("topic", "event", nil)
	seq := transport.SendAck("session-1")
	_ = transport.Send("topic", "event", nil)

	if seq != 2 {
		t.Errorf("expected ack seq 2, got %d", seq)
	}

	if transport.LastSeq() != 3 {
		t.Errorf("expected last seq 3, got %d", transport.LastSeq())
	}
}

func TestWebSocketTransportSendAckClosed(t *testing.T) {
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", nil)

	_ = transport.Close()

	seq := transport.SendAck("session-1")
	if seq != 0 {
		t.Errorf("expected seq 0 for closed transport, got %d", seq)
	}

	if sender.MessageCount() != 0 {
		t.Errorf("expected no messages for closed transport, got %d", sender.MessageCount())
	}
}

func TestWebSocketTransportSendFailureClearsPending(t *testing.T) {
	sender := &mockSender{failNext: true}
	transport := NewWebSocketTransport(sender, "user123", nil)

	err := transport.Send("topic", "event", nil)
	if err == nil {
		t.Fatalf("expected send error")
	}

	if pending := transport.Pending(); pending != 0 {
		t.Fatalf("expected pending cleared on send failure, got %d", pending)
	}
}

func TestWebSocketTransportSendAckFailureClearsPending(t *testing.T) {
	sender := &mockSender{failNext: true}
	transport := NewWebSocketTransport(sender, "user123", nil)

	seq := transport.SendAck("sid")
	if seq == 0 {
		t.Fatalf("expected seq allocated")
	}

	if pending := transport.Pending(); pending != 0 {
		t.Fatalf("expected ack pending cleared on failure, got %d", pending)
	}
}

func TestWebSocketTransportSendRaceWithClose(t *testing.T) {
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", nil)

	done := make(chan struct{})
	go func() {
		for i := 0; i < 1000; i++ {
			_ = transport.Send("topic", "event", i)
		}
		close(done)
	}()

	go func() {
		for i := 0; i < 100; i++ {
			_ = transport.Close()
			transport.SetSender(sender)
		}
	}()

	<-done
}

func TestWebSocketTransportConcurrentSend(t *testing.T) {
	sender := &mockSender{}
	transport := NewWebSocketTransport(sender, "user123", nil)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func(n int) {
			for j := 0; j < 100; j++ {
				_ = transport.Send("topic", "event", n*100+j)
			}
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	if transport.LastSeq() != 1000 {
		t.Errorf("expected 1000 messages, got %d", transport.LastSeq())
	}
}
