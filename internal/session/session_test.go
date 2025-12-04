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

func TestLiveSessionTouchObserverPanicReported(t *testing.T) {
	sess := NewLiveSession("test-session", 1, dummyComponent, nil)
	defer sess.Close()

	var receivedPanic bool
	var mu sync.Mutex

	sess.Session().Bus.Subscribe(protocol.Topic("session:error"), func(event string, data interface{}) {
		mu.Lock()
		if event == "observer_panic" {
			receivedPanic = true
		}
		mu.Unlock()
	})

	sess.OnTouch(func(t time.Time) {
		panic("test panic")
	})

	sess.Touch()

	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	gotPanic := receivedPanic
	mu.Unlock()

	if !gotPanic {
		t.Error("expected observer panic to be reported via bus")
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
	sess.Touch()
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
	sess.Touch()
	_ = sess.IsExpired()
	_ = sess.TTL()
	_ = sess.Close()
	sess.SetDevMode(true)
	cancel := sess.OnTouch(func(t time.Time) {})
	cancel()
	_ = sess.ClientAsset()
	sess.SetClientAsset("test")
	_ = sess.Bus()
	sess.SetAutoFlush(nil)
	sess.SetDOMTimeout(time.Second)
}
