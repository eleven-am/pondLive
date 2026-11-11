package session

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func dummyComponent(_ *runtime.Ctx) work.Node {
	return &work.Element{Tag: "div"}
}

func TestLiveSessionDoubleClose(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)

	err := sess.Close()
	if err != nil {
		t.Fatalf("first close error: %v", err)
	}

	err = sess.Close()
	if err != nil {
		t.Fatalf("second close should not error: %v", err)
	}
}

func TestLiveSessionCloseIdempotent(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			_ = sess.Close()
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestLiveSessionMethodsAfterClose(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	_ = sess.Close()

	if err := deliver(sess, protocol.ClientEvt{}); err == nil {
		t.Error("Deliver after close must fail")
	}
	if err := sess.Flush(); err == nil {
		t.Error("Flush after close must fail")
	}
	if sess.State() != StateClosed {
		t.Errorf("state after close = %v", sess.State())
	}
}

func TestLiveSessionNilSafety(t *testing.T) {
	var sess *LiveSession

	_ = sess.ID()
	_ = sess.Version()
	_ = sess.Session()
	_ = deliver(sess, protocol.ClientEvt{})
	_ = sess.Flush()
	_ = sess.Close()
	_ = sess.ClientAsset()
	_ = sess.UploadRegistry()
	_ = sess.State()
	sess.RequestClose(ReasonRemoved)
	sess.Detach(nil)
}

func TestLiveSessionUploadRegistry(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	ur := sess.UploadRegistry()
	if ur == nil {
		t.Error("expected UploadRegistry to be non-nil")
	}
}

func TestLiveSessionID(t *testing.T) {
	sess := NewLiveSession("test-id-session", 1, dummyComponent, nil)
	defer sess.Close()

	if sess.ID() != "test-id-session" {
		t.Errorf("expected ID 'test-id-session', got %q", sess.ID())
	}
}

func TestLiveSessionVersion(t *testing.T) {
	sess := NewLiveSession("test-session", 42, dummyComponent, nil)
	defer sess.Close()

	if sess.Version() != 42 {
		t.Errorf("expected version 42, got %d", sess.Version())
	}
}

func TestLiveSessionFlush(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	err := sess.Flush()
	if err != nil {
		t.Errorf("expected no error from flush, got %v", err)
	}
}

func TestLiveSessionDeliver(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	err := deliver(sess, protocol.ClientEvt{
		Event:   protocol.Event{Type: "frame", SID: "test-session"},
		Action:  "test-event",
		Payload: map[string]string{"key": "value"},
	})
	if err != nil {
		t.Errorf("expected no error from deliver, got %v", err)
	}
}

func TestLiveSessionClientAsset(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, &Config{ClientAsset: "/js/app.js"})
	defer sess.Close()

	if sess.ClientAsset() != "/js/app.js" {
		t.Errorf("expected client asset '/js/app.js', got %q", sess.ClientAsset())
	}
}

func TestIsClientTopic(t *testing.T) {
	tests := []struct {
		topic    protocol.Topic
		event    string
		expected bool
	}{
		{protocol.TopicFrame, "any", true},
		{protocol.RouteHandler, "any", true},
		{protocol.DOMHandler, "any", true},
		{protocol.Topic("ack"), "any", false},
		{protocol.Topic("script:test"), string(protocol.ScriptSendAction), true},
		{protocol.Topic("script:test"), "other", false},
		{protocol.Topic("other"), "any", false},
	}

	for _, tt := range tests {
		result := isClientTopic(tt.topic, tt.event)
		if result != tt.expected {
			t.Errorf("isClientTopic(%q, %q) = %v, expected %v", tt.topic, tt.event, result, tt.expected)
		}
	}
}
