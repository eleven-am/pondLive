package session

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/router"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type recordingTransport struct {
	sent   atomic.Int64
	fmu    sync.Mutex
	frames [][]byte
	closed atomic.Bool
	code   atomic.Int64
}

func newRecordingTransport() *recordingTransport {
	return &recordingTransport{}
}

func (t *recordingTransport) SendFrame(frame []byte) error {
	t.sent.Add(1)
	t.fmu.Lock()
	t.frames = append(t.frames, append([]byte(nil), frame...))
	t.fmu.Unlock()
	return nil
}

func (t *recordingTransport) messages() []protocol.Message {
	t.fmu.Lock()
	defer t.fmu.Unlock()
	out := make([]protocol.Message, 0, len(t.frames))
	for _, f := range t.frames {
		var m protocol.Message
		if err := json.Unmarshal(f, &m); err != nil {
			panic(err)
		}
		out = append(out, m)
	}
	return out
}

func (t *recordingTransport) seqs() []uint64 {
	var out []uint64
	for _, m := range t.messages() {
		out = append(out, m.Seq)
	}
	return out
}

func (t *recordingTransport) Closed() bool { return t.closed.Load() }
func (t *recordingTransport) Terminate(code websocket.StatusCode, _ string) {
	t.code.Store(int64(code))
	t.closed.Store(true)
}

func startLive(t *testing.T, comp Component) *LiveSession {
	t.Helper()
	ls := NewLiveSession("regression", 1, comp, nil, WithRequest(httptest.NewRequest(http.MethodGet, "/", nil)))
	t.Cleanup(func() { _ = ls.Close() })
	if _, err := ls.Render(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, false); err != nil {
		t.Fatal(err)
	}
	return ls
}

func onSession(t *testing.T, ls *LiveSession, fn func()) {
	t.Helper()
	if err := ls.Session().Ask(5*time.Second, func() error { fn(); return nil }); err != nil {
		t.Fatalf("ask failed: %v", err)
	}
}

func clickEvt(refID uint32, i int) protocol.ClientEvt {
	return protocol.ClientEvt{
		Event:   protocol.Event{Type: protocol.HandlerTopic, SID: "regression"},
		Action:  "click",
		I:       refID,
		Payload: map[string]any{"i": float64(i)},
	}
}

type clickProbe struct {
	ref  *runtime.Ref
	log  []int
	bump func(int)
}

func (p *clickProbe) refID() uint32 { return p.ref.ID() }

func (p *clickProbe) component(ctx *runtime.Ctx) work.Node {
	sig := runtime.UseSignal(ctx, 0)
	p.bump = sig.Set
	ref := runtime.UseCell(ctx, func() *runtime.Ref { return runtime.NewRef(ctx, "InputRef", nil) })
	p.ref = ref
	return work.BuildElement("button", runtime.AttachItem(ref), work.On("click", func(e work.Event) {
		p.log = append(p.log, int(e.Payload["i"].(float64)))
	}))
}

func TestInboundEventsRunInArrivalOrder(t *testing.T) {
	probe := &clickProbe{}
	ls := startLive(t, probe.component)
	var refID uint32
	onSession(t, ls, func() { refID = probe.refID() })

	const n, batch = 2000, 500
	for i := 0; i < n; i++ {
		if err := deliver(ls, clickEvt(refID, i)); err != nil {
			t.Fatal(err)
		}
		if (i+1)%batch == 0 {
			if err := ls.Flush(); err != nil {
				t.Fatal(err)
			}
		}
	}

	deadline := time.Now().Add(5 * time.Second)
	for {
		var got int
		onSession(t, ls, func() { got = len(probe.log) })
		if got == n {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d of %d events were handled", got, n)
		}
		time.Sleep(2 * time.Millisecond)
	}
	onSession(t, ls, func() {
		for i, v := range probe.log {
			if v != i {
				t.Fatalf("event %d ran at position %d", v, i)
			}
		}
	})
}

func TestElementRefEventDuringRerenderDoesNotRace(t *testing.T) {
	probe := &clickProbe{}
	ls := startLive(t, probe.component)
	var refID uint32
	var bump func(int)
	onSession(t, ls, func() { refID, bump = probe.refID(), probe.bump })

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 1500; i++ {
			_ = deliver(ls, clickEvt(refID, i))
		}
	}()
	go func() {
		defer wg.Done()
		for i := 1; i <= 1500; i++ {
			bump(i)
		}
	}()
	for i := 0; i < 30; i++ {
		if err := ls.Flush(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
	}
	wg.Wait()
	if err := ls.Flush(); err != nil {
		t.Fatal(err)
	}
}

func TestPopstateDuringResubscribeIsNotDropped(t *testing.T) {
	var rtr *router.Router
	ls := startLive(t, func(ctx *runtime.Ctx) work.Node {
		rtr = router.UseRouter(ctx)
		return work.BuildElement("p", work.TextOf(rtr.Path))
	})

	const n = 300
	for i := 0; i < n; i++ {
		err := deliver(ls, protocol.ClientEvt{
			Event:   protocol.Event{Type: protocol.RouteHandler, SID: "regression"},
			Action:  string(protocol.RouterPopstateAction),
			Payload: map[string]any{"path": fmt.Sprintf("/p%d", i), "query": "", "hash": ""},
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	want := fmt.Sprintf("/p%d", n-1)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := ls.Flush(); err != nil {
			t.Fatal(err)
		}
		var got string
		onSession(t, ls, func() { got = rtr.Path.Peek() })
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("location is %q after %d popstates, want %q", got, n, want)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestNavigateFromUserGoroutineDoesNotRace(t *testing.T) {
	var rtr *router.Router
	var bump func(int)
	ls := startLive(t, func(ctx *runtime.Ctx) work.Node {
		sig := runtime.UseSignal(ctx, 0)
		bump = sig.Set
		rtr = router.UseRouter(ctx)
		return work.BuildElement("p", work.TextOf(rtr.Path))
	})
	var nav *router.Router
	var local func(int)
	onSession(t, ls, func() { nav, local = rtr, bump })

	const n = 200
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < n; i++ {
			nav.Navigate(fmt.Sprintf("/x%d", i))
		}
	}()
	go func() {
		defer wg.Done()
		for i := 1; i <= n; i++ {
			local(i)
		}
	}()
	for i := 0; i < 20; i++ {
		if err := ls.Flush(); err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Millisecond)
	}
	wg.Wait()

	want := fmt.Sprintf("/x%d", n-1)
	deadline := time.Now().Add(5 * time.Second)
	for {
		if err := ls.Flush(); err != nil {
			t.Fatal(err)
		}
		var got string
		onSession(t, ls, func() { got = rtr.Path.Peek() })
		if got == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("location is %q, want %q", got, want)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestLiveSessionFlushBlocksUntilRun(t *testing.T) {
	var bump func(int)
	var shownSig *runtime.Signal[int]
	ls := startLive(t, func(ctx *runtime.Ctx) work.Node {
		sig := runtime.UseSignal(ctx, 0)
		bump = sig.Set
		shownSig = sig
		return work.BuildElement("p", work.TextOf(sig))
	})
	var local func(int)
	onSession(t, ls, func() { local = bump })

	rt := ls.Session()
	for i := 1; i <= 50; i++ {
		before := rt.FlushCount()
		local(i)
		if err := ls.Flush(); err != nil {
			t.Fatal(err)
		}
		if rt.FlushCount() == before {
			t.Fatalf("Flush #%d returned before any flush had run", i)
		}
		var got int
		onSession(t, ls, func() { got = shownSig.Peek() })
		if got != i {
			t.Fatalf("after Flush #%d the render shows %d", i, got)
		}
	}
}

type domProbeTransport struct {
	recordingTransport
	queries chan protocol.DOMQueryPayload
}

func (t *domProbeTransport) SendFrame(frame []byte) error {
	var m struct {
		Topic string                   `json:"topic"`
		Event string                   `json:"event"`
		Data  protocol.DOMQueryPayload `json:"data"`
	}
	if err := json.Unmarshal(frame, &m); err == nil && m.Topic == string(protocol.DOMHandler) && m.Event == string(protocol.DOMQueryAction) {
		t.queries <- m.Data
	}
	return t.recordingTransport.SendFrame(frame)
}

func TestASlowDOMReplyDoesNotDelayAnotherEvent(t *testing.T) {
	var ref *runtime.Ref
	var counter *runtime.Signal[int]
	var got atomic.Value
	comp := func(ctx *runtime.Ctx) work.Node {
		ref = runtime.UseCell(ctx, func() *runtime.Ref { return runtime.NewRef(ctx, "InputRef", nil) })
		counter = runtime.UseSignal(ctx, 0)
		return work.BuildElement("div",
			work.BuildElement("input", runtime.AttachItem(ref), work.On("click", func(work.Event) {
				ref.Query(func(values map[string]any, err error) {
					if err != nil {
						got.Store(err.Error())
						return
					}
					got.Store(fmt.Sprint(values["selectionStart"]))
				}, "selectionStart")
			})),
			work.BuildElement("button", work.On("click", func(work.Event) { counter.Set(counter.Peek() + 1) })))
	}

	ls := NewLiveSession("regression", 1, comp, &Config{DOMTimeout: 3 * time.Second})
	t.Cleanup(func() { _ = ls.Close() })
	tr := &domProbeTransport{queries: make(chan protocol.DOMQueryPayload, 1)}
	if _, err := ls.Attach(context.Background(), tr, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	var inputID, buttonID uint32
	onSession(t, ls, func() { inputID = ref.ID(); buttonID = inputID + 1 })

	started := time.Now()
	if err := deliver(ls, clickEvt(inputID, 0)); err != nil {
		t.Fatal(err)
	}
	var q protocol.DOMQueryPayload
	select {
	case q = <-tr.queries:
	case <-time.After(2 * time.Second):
		t.Fatal("the handler never issued its DOM query")
	}

	if err := deliver(ls, clickEvt(buttonID, 0)); err != nil {
		t.Fatal(err)
	}
	if err := ls.Flush(); err != nil {
		t.Fatal(err)
	}
	var count int
	onSession(t, ls, func() { count = counter.Peek() })
	if count != 1 {
		t.Fatalf("the second event was not applied while the first query waits for the browser: counter = %d", count)
	}
	if waited := time.Since(started); waited > time.Second {
		t.Fatalf("the second event waited %v behind a query the browser had not answered", waited)
	}
	if got.Load() != nil {
		t.Fatalf("the callback ran before the browser answered: %v", got.Load())
	}

	err := deliver(ls, protocol.ClientEvt{
		Event:   protocol.Event{Type: protocol.DOMHandler, SID: "regression"},
		Action:  string(protocol.DOMResponseAction),
		Payload: map[string]any{"requestId": q.RequestID, "values": map[string]any{"selectionStart": "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for got.Load() == nil {
		if time.Now().After(deadline) {
			t.Fatal("the query callback never ran")
		}
		time.Sleep(time.Millisecond)
	}
	if got.Load() != "hello" {
		t.Fatalf("query result %v", got.Load())
	}
	if elapsed := time.Since(started); elapsed >= 3*time.Second {
		t.Fatalf("took %v, as long as the DOM timeout", elapsed)
	}
}

func TestProductionConstructorStartsTheActor(t *testing.T) {
	ls := NewLiveSession("regression", 1, func(*runtime.Ctx) work.Node { return &work.Element{Tag: "div"} }, nil)
	t.Cleanup(func() { _ = ls.Close() })
	rt := ls.Session()
	select {
	case <-rt.Done():
		t.Fatal("actor exited immediately")
	default:
	}
	if rt.Done() == nil {
		t.Fatal("NewLiveSession returned an unstarted session")
	}
}
