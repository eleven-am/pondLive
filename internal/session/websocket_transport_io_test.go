package session

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/eleven-am/pondlive/internal/protocol"
	"go.uber.org/goleak"
)

type wsPair struct {
	tr        *WebSocketTransport
	client    *websocket.Conn
	srv       *httptest.Server
	teardowns atomic.Int32
	finished  chan struct{}
}

func newPair(t *testing.T, opts WebSocketOptions) *wsPair {
	t.Helper()
	p := &wsPair{finished: make(chan struct{})}
	ready := make(chan struct{})
	p.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
		if err != nil {
			return
		}
		ws.SetReadLimit(1 << 16)
		p.tr = NewWebSocketTransport(context.Background(), ws, opts)
		p.tr.Start([]byte(`{"t":"joined","gen":1}`), TokenRefresh{})
		close(ready)
		defer func() {
			p.tr.Abort()
			p.teardowns.Add(1)
			p.tr.Wait()
			close(p.finished)
		}()
		for {
			if _, err := p.tr.Receive(); err != nil {
				return
			}
		}
	}))
	url := "ws" + strings.TrimPrefix(p.srv.URL, "http")
	client, _, err := websocket.Dial(context.Background(), url, nil)
	if err != nil {
		p.srv.Close()
		t.Fatalf("dial: %v", err)
	}
	p.client = client
	<-ready
	readText(t, client)
	return p
}

func (p *wsPair) close() {
	p.client.CloseNow()
	<-p.finished
	p.srv.Close()
}

func readText(t *testing.T, c *websocket.Conn) []byte {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("client read: %v", err)
	}
	return data
}

func sendMsg(tr *WebSocketTransport, seq uint64, data any) error {
	frame, err := json.Marshal(protocol.Message{Seq: seq, Topic: "frame", Event: "patch", Data: data})
	if err != nil {
		return err
	}
	return tr.SendFrame(frame)
}

func TestTransportDeliversSendsInOrder(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	p := newPair(t, WebSocketOptions{OutboundBudget: 64 << 20})
	defer p.close()

	const n = 10000
	done := make(chan error, 1)
	go func() {
		for i := 1; i <= n; i++ {
			if err := sendMsg(p.tr, uint64(i), i); err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	p.client.SetReadLimit(1 << 20)
	for i := 1; i <= n; i++ {
		var msg protocol.Message
		if err := json.Unmarshal(readText(t, p.client), &msg); err != nil {
			t.Fatal(err)
		}
		if msg.Seq != uint64(i) || msg.Data.(float64) != float64(i) {
			t.Fatalf("message %d arrived as seq=%d data=%v", i, msg.Seq, msg.Data)
		}
	}
	if err := <-done; err != nil {
		t.Fatalf("send: %v", err)
	}
}

func TestTransportConcurrentSendsDeliverEveryFrameOnce(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	p := newPair(t, WebSocketOptions{OutboundBudget: 64 << 20})
	defer p.close()

	const workers, each = 8, 500
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < each; i++ {
				_ = sendMsg(p.tr, uint64(w*each+i+1), nil)
			}
		}()
	}

	seen := make(map[uint64]bool, workers*each)
	last := make([]uint64, workers)
	for i := 1; i <= workers*each; i++ {
		var msg protocol.Message
		if err := json.Unmarshal(readText(t, p.client), &msg); err != nil {
			t.Fatal(err)
		}
		if seen[msg.Seq] {
			t.Fatalf("frame %d delivered twice", msg.Seq)
		}
		seen[msg.Seq] = true
		w := int(msg.Seq-1) / each
		if msg.Seq <= last[w] {
			t.Fatalf("worker %d frames reordered: %d after %d", w, msg.Seq, last[w])
		}
		last[w] = msg.Seq
	}
	wg.Wait()
}

func TestTransportOverBudgetClosesWithTryAgainLater(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	p := newPair(t, WebSocketOptions{OutboundBudget: 256 << 10})
	defer p.close()

	payload := strings.Repeat("x", 32<<10)
	var sendErr error
	started := time.Now()
	for i := 0; i < 8000 && sendErr == nil; i++ {
		sendErr = sendMsg(p.tr, 0, payload)
	}
	if sendErr != ErrOutboundBudgetExceeded {
		t.Fatalf("expected ErrOutboundBudgetExceeded from a client that never reads, got %v", sendErr)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("enqueue blocked for %v", elapsed)
	}

	p.client.SetReadLimit(1 << 20)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	for {
		_, _, err := p.client.Read(ctx)
		if err == nil {
			continue
		}
		if status := websocket.CloseStatus(err); status != websocket.StatusTryAgainLater {
			t.Fatalf("expected close 1013, got %v (%v)", status, err)
		}
		break
	}
	<-p.finished
	if got := p.teardowns.Load(); got != 1 {
		t.Fatalf("expected one teardown, got %d", got)
	}
}

func TestTransportSlowConsumerDoesNotDelayOtherConnections(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	slow := newPair(t, WebSocketOptions{OutboundBudget: 256 << 10})
	defer slow.close()
	fast := newPair(t, WebSocketOptions{OutboundBudget: 64 << 20})
	defer fast.close()

	stop := make(chan struct{})
	flooded := make(chan struct{})
	go func() {
		defer close(flooded)
		payload := strings.Repeat("x", 32<<10)
		for {
			select {
			case <-stop:
				return
			default:
			}
			if err := sendMsg(slow.tr, 0, payload); err != nil {
				return
			}
		}
	}()

	var worst time.Duration
	for i := 0; i < 200; i++ {
		sentAt := time.Now()
		if err := sendMsg(fast.tr, 0, i); err != nil {
			t.Fatal(err)
		}
		readText(t, fast.client)
		if d := time.Since(sentAt); d > worst {
			worst = d
		}
	}
	close(stop)
	<-flooded
	if worst > time.Second {
		t.Fatalf("fast connection delayed by %v while another consumer stalled", worst)
	}
}

func TestTransportTeardownIsIdempotentUnderConcurrentCloses(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	p := newPair(t, WebSocketOptions{})

	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			switch i % 3 {
			case 0:
				p.tr.Terminate(CloseReplaced, "replaced")
			case 1:
				p.tr.Abort()
			default:
				_ = p.tr.Close()
			}
		}(i)
	}
	wg.Wait()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		if _, _, err := p.client.Read(ctx); err != nil {
			break
		}
	}
	<-p.finished
	p.srv.Close()
	if got := p.teardowns.Load(); got != 1 {
		t.Fatalf("expected one teardown, got %d", got)
	}
	if err := sendMsg(p.tr, 0, nil); err != nil {
		t.Fatalf("send after close must be dropped silently: %v", err)
	}
}

func TestTransportTerminateDeliversCloseCode(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	p := newPair(t, WebSocketOptions{})

	p.tr.Terminate(CloseReplaced, "replaced")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, err := p.client.Read(ctx)
	if status := websocket.CloseStatus(err); status != CloseReplaced {
		t.Fatalf("expected 4409, got %v (%v)", status, err)
	}
	<-p.finished
	p.srv.Close()
}

func TestTransportRejectsBinaryFrames(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	p := newPair(t, WebSocketOptions{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.client.Write(ctx, websocket.MessageBinary, []byte{1}); err != nil {
		t.Fatal(err)
	}
	_, _, err := p.client.Read(ctx)
	if status := websocket.CloseStatus(err); status != websocket.StatusUnsupportedData {
		t.Fatalf("expected 1003, got %v (%v)", status, err)
	}
	<-p.finished
	p.srv.Close()
}

func TestTransportClosesWhenPongIsMissing(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	p := newPair(t, WebSocketOptions{PingInterval: 50 * time.Millisecond, PongTimeout: 100 * time.Millisecond})

	select {
	case <-p.finished:
	case <-time.After(5 * time.Second):
		t.Fatal("transport stayed open although the client never answered pings")
	}
	p.client.CloseNow()
	p.srv.Close()
}

func TestTransportOverReadLimitClosesWith1009(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	p := newPair(t, WebSocketOptions{})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := p.client.Write(ctx, websocket.MessageText, []byte(strings.Repeat("a", (1<<16)+1))); err != nil {
		t.Fatal(err)
	}
	_, _, err := p.client.Read(ctx)
	if status := websocket.CloseStatus(err); status != websocket.StatusMessageTooBig {
		t.Fatalf("expected 1009, got %v (%v)", status, err)
	}
	<-p.finished
	p.srv.Close()
}

func TestTheHandBuiltEvackFrameIsExactlyWhatTheStructMarshals(t *testing.T) {
	for _, cs := range []uint64{0, 1, 42, 1 << 40} {
		want, err := json.Marshal(protocol.EvackFrame{T: protocol.FrameEvack, CS: cs})
		if err != nil {
			t.Fatal(err)
		}
		if got := evackFrame(cs); string(got) != string(want) {
			t.Fatalf("evackFrame(%d) = %s, struct marshals %s", cs, got, want)
		}
	}
}
