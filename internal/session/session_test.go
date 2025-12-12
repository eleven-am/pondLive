package session

import (
	"sync"
	"testing"
	"time"

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

func TestLiveSessionCloseNilsSession(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)

	if sess.Session() == nil {
		t.Fatal("expected session before close")
	}

	_ = sess.Close()

	if sess.Session() != nil {
		t.Error("expected session to be nil after close")
	}
}

func TestLiveSessionSSRSendErrorReported(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	transport := NewSSRTransport(nil)
	transport.SetMaxMessages(1)

	sess.SetTransport(transport)

	var receivedError bool
	var mu sync.Mutex

	sess.Session().Bus.Subscribe(protocol.Topic("session:error"), func(event string, data interface{}) {
		mu.Lock()
		if event == "send_error" {
			receivedError = true
		}
		mu.Unlock()
	})

	sess.Session().Bus.Publish(protocol.TopicFrame, "patch", map[string]string{"a": "1"})
	sess.Session().Bus.Publish(protocol.TopicFrame, "patch", map[string]string{"b": "2"})

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	gotError := receivedError
	mu.Unlock()

	if !gotError {
		t.Error("expected SSR send error to be reported via bus")
	}
}

func TestLiveSessionMethodsAfterClose(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	_ = sess.Close()

	sess.Receive("topic", "event", nil)
	_ = sess.Flush()
	sess.SetDevMode(true)
	sess.SetAutoFlush(nil)
	sess.SetDOMTimeout(time.Second)

	if sess.Bus() != nil {
		t.Error("expected Bus() to return nil after close")
	}
}

func TestLiveSessionNilSafety(t *testing.T) {
	var sess *LiveSession

	_ = sess.ID()
	_ = sess.Version()
	_ = sess.Session()
	sess.SetTransport(nil)
	sess.Receive("topic", "event", nil)
	_ = sess.Flush()
	_ = sess.Close()
	sess.SetDevMode(true)
	_ = sess.ClientAsset()
	sess.SetClientAsset("test")
	_ = sess.Bus()
	sess.SetAutoFlush(nil)
	sess.SetDOMTimeout(time.Second)
	_ = sess.ChannelManager()
	_ = sess.UploadRegistry()
}

func TestLiveSessionChannelManager(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	cm := sess.ChannelManager()
	if cm == nil {
		t.Error("expected ChannelManager to be non-nil")
	}
}

func TestLiveSessionUploadRegistry(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	ur := sess.UploadRegistry()
	if ur == nil {
		t.Error("expected UploadRegistry to be non-nil")
	}
}

func TestLiveSessionChannelManagerAfterClose(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	_ = sess.Close()

	if sess.ChannelManager() != nil {
		t.Error("expected nil ChannelManager after close")
	}
}

func TestLiveSessionUploadRegistryAfterClose(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	_ = sess.Close()

	if sess.UploadRegistry() != nil {
		t.Error("expected nil UploadRegistry after close")
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

func TestLiveSessionReceive(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	sess.Receive("frame", "test-event", map[string]string{"key": "value"})
}

func TestLiveSessionSetDevMode(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	sess.SetDevMode(true)
	sess.SetDevMode(false)
}

func TestLiveSessionClientAsset(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	sess.SetClientAsset("/js/app.js")
	if sess.ClientAsset() != "/js/app.js" {
		t.Errorf("expected client asset '/js/app.js', got %q", sess.ClientAsset())
	}
}

func TestLiveSessionBus(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	bus := sess.Bus()
	if bus == nil {
		t.Error("expected non-nil bus")
	}
}

func TestLiveSessionSetAutoFlush(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	called := false
	sess.SetAutoFlush(func() {
		called = true
	})
	if called {
		t.Error("auto flush should not be called immediately")
	}
}

func TestLiveSessionSetDOMTimeout(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	sess.SetDOMTimeout(5 * time.Second)
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
		{protocol.AckTopic, "any", true},
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
