package session

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSSRTransportSend(t *testing.T) {
	transport := NewSSRTransport(nil)

	err := transport.Send("frame", "patch", map[string]string{"foo": "bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgs := transport.Messages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Seq != 1 {
		t.Errorf("expected seq 1, got %d", msgs[0].Seq)
	}
	if msgs[0].Topic != "frame" {
		t.Errorf("expected topic 'frame', got '%s'", msgs[0].Topic)
	}
	if msgs[0].Event != "patch" {
		t.Errorf("expected event 'patch', got '%s'", msgs[0].Event)
	}
}

func TestSSRTransportSequenceIncrement(t *testing.T) {
	transport := NewSSRTransport(nil)

	_ = transport.Send("topic1", "event1", nil)
	_ = transport.Send("topic2", "event2", nil)
	_ = transport.Send("topic3", "event3", nil)

	if transport.LastSeq() != 3 {
		t.Errorf("expected last seq 3, got %d", transport.LastSeq())
	}

	msgs := transport.Messages()
	if len(msgs) != 3 {
		t.Errorf("expected 3 messages, got %d", len(msgs))
	}
}

func TestSSRTransportIsLive(t *testing.T) {
	transport := NewSSRTransport(nil)

	if transport.IsLive() {
		t.Error("SSRTransport should not be live")
	}
}

func TestSSRTransportClose(t *testing.T) {
	transport := NewSSRTransport(nil)

	_ = transport.Send("topic", "event", nil)

	err := transport.Close()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = transport.Send("topic", "event", nil)
	if err != nil {
		t.Errorf("send after close should not error, got: %v", err)
	}

	msgs := transport.Messages()
	if len(msgs) != 1 {
		t.Errorf("expected 1 message (before close), got %d", len(msgs))
	}
}

func TestSSRTransportClear(t *testing.T) {
	transport := NewSSRTransport(nil)

	_ = transport.Send("topic", "event", nil)
	_ = transport.Send("topic", "event", nil)

	transport.Clear()

	msgs := transport.Messages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(msgs))
	}

	if transport.LastSeq() != 2 {
		t.Errorf("expected last seq to remain 2, got %d", transport.LastSeq())
	}
}

func TestSSRTransportDrain(t *testing.T) {
	transport := NewSSRTransport(nil)

	_ = transport.Send("topic", "event", "data1")
	_ = transport.Send("topic", "event", "data2")

	drained := transport.Drain()

	if len(drained) != 2 {
		t.Errorf("expected 2 drained messages, got %d", len(drained))
	}

	msgs := transport.Messages()
	if len(msgs) != 0 {
		t.Errorf("expected 0 messages after drain, got %d", len(msgs))
	}
}

func TestSSRTransportFilterByTopic(t *testing.T) {
	transport := NewSSRTransport(nil)

	_ = transport.Send("frame", "patch", nil)
	_ = transport.Send("nav", "redirect", nil)
	_ = transport.Send("frame", "replace", nil)

	filtered := transport.FilterByTopic("frame")

	if len(filtered) != 2 {
		t.Errorf("expected 2 frame messages, got %d", len(filtered))
	}

	for _, msg := range filtered {
		if msg.Topic != "frame" {
			t.Errorf("expected topic 'frame', got '%s'", msg.Topic)
		}
	}
}

func TestSSRTransportFilterByEvent(t *testing.T) {
	transport := NewSSRTransport(nil)

	_ = transport.Send("frame", "patch", nil)
	_ = transport.Send("frame", "replace", nil)
	_ = transport.Send("nav", "patch", nil)

	filtered := transport.FilterByEvent("patch")

	if len(filtered) != 2 {
		t.Errorf("expected 2 patch messages, got %d", len(filtered))
	}

	for _, msg := range filtered {
		if msg.Event != "patch" {
			t.Errorf("expected event 'patch', got '%s'", msg.Event)
		}
	}
}

func TestSSRTransportMessagesReturnsCopy(t *testing.T) {
	transport := NewSSRTransport(nil)

	_ = transport.Send("topic", "event", nil)

	msgs1 := transport.Messages()
	msgs2 := transport.Messages()

	msgs1[0].Topic = "modified"

	if msgs2[0].Topic == "modified" {
		t.Error("Messages() should return a copy, not the internal slice")
	}
}

func TestSSRTransportNilSafety(t *testing.T) {
	var transport *SSRTransport

	_ = transport.Send("topic", "event", nil)
	_ = transport.IsLive()
	_ = transport.Close()
	_ = transport.Messages()
	_ = transport.LastSeq()
	transport.Clear()
	_ = transport.Drain()
	_ = transport.FilterByTopic("topic")
	_ = transport.FilterByEvent("event")
	_ = transport.RequestInfo()
}

func TestSSRTransportRequestInfo(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/path?foo=bar&baz=qux", nil)
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Cookie", "session=abc123")

	transport := NewSSRTransport(req)

	info := transport.RequestInfo()
	if info == nil {
		t.Fatal("expected RequestInfo to be non-nil")
	}

	if info.Method != http.MethodGet {
		t.Errorf("expected method GET, got %s", info.Method)
	}

	if info.Path != "/path" {
		t.Errorf("expected path /path, got %s", info.Path)
	}

	if info.Query.Get("foo") != "bar" {
		t.Errorf("expected query foo=bar, got foo=%s", info.Query.Get("foo"))
	}

	if ua, ok := info.Get("User-Agent"); !ok || ua != "test-agent" {
		t.Errorf("expected User-Agent header test-agent, got %s", ua)
	}

	if cookie, ok := info.GetCookie("session"); !ok || cookie != "abc123" {
		t.Errorf("expected cookie session=abc123, got %s", cookie)
	}
}

func TestSSRTransportRequestInfoNilRequest(t *testing.T) {
	transport := NewSSRTransport(nil)

	info := transport.RequestInfo()
	if info == nil {
		t.Fatal("expected RequestInfo to be non-nil even with nil request")
	}

	if info.Method != "" {
		t.Errorf("expected empty method, got %s", info.Method)
	}

	if info.Query == nil {
		t.Error("expected Query to be initialized")
	}
}
