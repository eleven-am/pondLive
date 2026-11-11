package runtime

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/upload"
	"github.com/eleven-am/pondlive/internal/work"
)

type uploadFixture struct {
	t      *testing.T
	sess   *Session
	handle *UploadHandle
	starts chan map[string]interface{}
}

func newUploadFixture(t *testing.T) *uploadFixture {
	t.Helper()
	f := &uploadFixture{t: t, starts: make(chan map[string]interface{}, 16)}
	f.sess = NewSession(Config{
		SessionID:      "sess-up",
		UploadRegistry: upload.NewRegistry(),
		Root: func(ctx *Ctx) work.Node {
			h := UseUpload(ctx)
			if f.handle == nil {
				f.handle = h
			}
			return work.BuildElement("input", AttachItem(h))
		},
	})
	f.sess.Start()
	t.Cleanup(f.sess.Close)
	onActor(t, f.sess, func() {
		if _, err := f.sess.RenderSnapshot(); err != nil {
			t.Fatal(err)
		}
	})
	f.sess.Bus.Subscribe(protocol.Topic("script:"+f.handle.script.slot.id), func(action string, data interface{}) {
		payload, ok := protocol.DecodePayload[protocol.ScriptPayload](data)
		if !ok || action != string(protocol.ScriptSendAction) || payload.Event != "start" {
			return
		}
		if m, ok := payload.Data.(map[string]interface{}); ok {
			f.starts <- m
		}
	})
	return f
}

func (f *uploadFixture) nextStart() map[string]interface{} {
	f.t.Helper()
	select {
	case m := <-f.starts:
		return m
	case <-time.After(time.Second):
		f.t.Fatal("no start message sent to the client")
		return nil
	}
}

func (f *uploadFixture) fromClient(event string, data map[string]interface{}) {
	f.t.Helper()
	id := f.handle.script.slot.id
	onActor(f.t, f.sess, func() {
		f.sess.Bus.Publish(protocol.Topic("script:"+id+":"+event), string(protocol.ScriptMessageAction), protocol.ScriptPayload{
			ScriptID: id,
			Event:    event,
			Data:     data,
		})
	})
}

func TestUseUpload(t *testing.T) {
	f := newUploadFixture(t)
	if f.handle.script.slot == nil {
		t.Fatal("expected script to be non-nil")
	}
	if f.handle.cell.grant != "" {
		t.Error("no grant may exist before the client selects files")
	}
	if f.sess.UploadRegistry.Len() != 0 {
		t.Error("expected no grants before the client selects files")
	}
}

func TestUseUploadAccept(t *testing.T) {
	f := newUploadFixture(t)
	f.handle.Accept(UploadConfig{MaxSize: 1024 * 1024, Accept: []string{"image/png", "image/jpeg"}, Multiple: true})

	f.fromClient("ready", map[string]interface{}{"name": "a.png", "size": float64(10), "type": "image/png", "count": float64(2)})
	start := f.nextStart()

	if start["maxSize"] != int64(1024*1024) {
		t.Errorf("expected maxSize 1048576, got %v", start["maxSize"])
	}
	accept, _ := start["accept"].([]string)
	if len(accept) != 2 {
		t.Errorf("expected 2 accept entries, got %v", start["accept"])
	}
	if start["multiple"] != true {
		t.Errorf("expected multiple true, got %v", start["multiple"])
	}
}

func TestUseUploadAttachTo(t *testing.T) {
	f := newUploadFixture(t)
	elem := &work.Element{Tag: "input"}
	work.Apply(AttachItem(f.handle), elem)
	if len(elem.Attachments) == 0 {
		t.Fatal("expected Script to be attached")
	}
}

func TestUseUploadOnProgress(t *testing.T) {
	f := newUploadFixture(t)

	var loaded, total int64
	f.handle.OnProgress(func(l, tt int64) {
		loaded = l
		total = tt
	})
	f.fromClient("progress", map[string]interface{}{"loaded": float64(500), "total": float64(1000)})

	onActor(t, f.sess, func() {})
	if loaded != 500 || total != 1000 {
		t.Errorf("expected 500/1000, got %d/%d", loaded, total)
	}
}

func TestOnProgressAndProgressBothSeeTheMessage(t *testing.T) {
	f := newUploadFixture(t)

	var called int
	f.handle.OnProgress(func(l, tt int64) { called++ })
	f.fromClient("progress", map[string]interface{}{"loaded": float64(250), "total": float64(500)})

	p := f.handle.Progress().Peek()
	if p.Loaded != 250 || p.Total != 500 {
		t.Errorf("Progress() lost the update when OnProgress is registered: %+v", p)
	}
	if called != 1 {
		t.Errorf("expected the user OnProgress callback once, got %d", called)
	}
}

func TestOnReadyStillSendsStart(t *testing.T) {
	f := newUploadFixture(t)

	var gotName string
	f.handle.OnReady(func(name string, size int64, fileType string) { gotName = name })
	f.fromClient("ready", map[string]interface{}{"name": "doc.pdf", "size": float64(2048), "type": "application/pdf"})

	start := f.nextStart()
	if tok, _ := start["token"].(string); tok == "" {
		t.Fatalf("start carried no token: %v", start)
	}
	if gotName != "doc.pdf" {
		t.Errorf("user OnReady not called, got %q", gotName)
	}
}

func TestStartCarriesEndpointAndScriptURLFromSession(t *testing.T) {
	f := newUploadFixture(t)
	f.sess.UploadEndpoint = "/mounted/uploads/"
	f.sess.UploadScriptURL = "/assets/tus.min.js"

	f.fromClient("ready", map[string]interface{}{"name": "a", "size": float64(1)})
	start := f.nextStart()
	if start["endpoint"] != "/mounted/uploads/" || start["tusScript"] != "/assets/tus.min.js" {
		t.Fatalf("start does not use the configured mount: %v", start)
	}
}

func reserveAll(t *testing.T, reg *upload.Registry, token string) int {
	t.Helper()
	n := 0
	for {
		_, _, err := reg.Reserve(token, 0, nil)
		if err != nil {
			if !errors.Is(err, upload.ErrGrantExhausted) {
				t.Fatalf("unexpected reserve error: %v", err)
			}
			return n
		}
		n++
		if n > upload.MaxGrantFiles+1 {
			t.Fatal("grant never exhausted")
		}
	}
}

func TestGrantAllowsOneFileUnlessMultiple(t *testing.T) {
	f := newUploadFixture(t)
	f.handle.Accept(UploadConfig{Multiple: false})
	f.fromClient("ready", map[string]interface{}{"name": "a", "size": float64(1), "count": float64(5)})
	token := f.nextStart()["token"].(string)

	if got := reserveAll(t, f.sess.UploadRegistry, token); got != 1 {
		t.Fatalf("a single-file config granted %d files", got)
	}
}

func TestGrantAllowsRequestedCountWhenMultiple(t *testing.T) {
	f := newUploadFixture(t)
	f.handle.Accept(UploadConfig{Multiple: true})
	f.fromClient("ready", map[string]interface{}{"name": "a", "size": float64(1), "count": float64(3)})
	token := f.nextStart()["token"].(string)

	if got := reserveAll(t, f.sess.UploadRegistry, token); got != 3 {
		t.Fatalf("expected 3 files granted, got %d", got)
	}
}

func TestNewSelectionRevokesThePreviousGrant(t *testing.T) {
	f := newUploadFixture(t)
	f.fromClient("ready", map[string]interface{}{"name": "a", "size": float64(1)})
	first := f.nextStart()["token"].(string)
	f.fromClient("ready", map[string]interface{}{"name": "b", "size": float64(1)})
	second := f.nextStart()["token"].(string)

	if first == second {
		t.Fatal("a new selection must issue a new token")
	}
	if _, _, err := f.sess.UploadRegistry.Reserve(first, 0, nil); !errors.Is(err, upload.ErrGrantUnknown) {
		t.Fatalf("expected the old grant to be revoked, got %v", err)
	}
	if _, _, err := f.sess.UploadRegistry.Reserve(second, 0, nil); err != nil {
		t.Fatalf("expected the new grant to work, got %v", err)
	}
}

func TestUploadCompletionRunsOnActorAndSurfacesItsError(t *testing.T) {
	f := newUploadFixture(t)

	diags := make(chan protocol.Diagnostic, 4)
	f.sess.Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) { diags <- d })

	var ranOnActor bool
	var mu sync.Mutex
	f.handle.OnComplete(func(info upload.FileInfo) error {
		mu.Lock()
		ranOnActor = f.sess.onActor()
		mu.Unlock()
		return errors.New("store failed")
	})
	f.fromClient("ready", map[string]interface{}{"name": "a", "size": float64(1)})
	token := f.nextStart()["token"].(string)

	_, spec, err := f.sess.UploadRegistry.Reserve(token, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := spec.OnComplete(upload.FileInfo{ID: "u1"}); err != nil {
		t.Fatalf("posting the completion failed: %v", err)
	}

	select {
	case d := <-diags:
		if d.Phase != "upload:complete" || d.Message != "store failed" {
			t.Fatalf("unexpected diagnostic: %+v", d)
		}
	case <-time.After(time.Second):
		t.Fatal("OnComplete error was not surfaced as a diagnostic")
	}
	mu.Lock()
	defer mu.Unlock()
	if !ranOnActor {
		t.Error("OnComplete did not run on the session actor")
	}
}

func TestUploadCompletionReportsAClosedSession(t *testing.T) {
	f := newUploadFixture(t)
	f.handle.OnComplete(func(info upload.FileInfo) error { return nil })
	f.fromClient("ready", map[string]interface{}{"name": "a", "size": float64(1)})
	token := f.nextStart()["token"].(string)
	_, spec, err := f.sess.UploadRegistry.Reserve(token, 0, nil)
	if err != nil {
		t.Fatal(err)
	}

	f.sess.Close()

	if err := spec.OnComplete(upload.FileInfo{ID: "u1"}); err == nil {
		t.Fatal("completion on a closed session must return an error, not drop the file silently")
	}
}

func TestUseUploadOnError(t *testing.T) {
	f := newUploadFixture(t)
	var errMsg string
	f.handle.OnError(func(msg string) { errMsg = msg })
	f.fromClient("error", map[string]interface{}{"error": "upload failed"})
	if errMsg != "upload failed" {
		t.Errorf("expected error 'upload failed', got '%s'", errMsg)
	}
}

func TestUseUploadCancel(t *testing.T) {
	f := newUploadFixture(t)

	var receivedAction string
	done := make(chan struct{})
	f.sess.Bus.Subscribe(protocol.Topic("script:"+f.handle.script.slot.id), func(action string, data interface{}) {
		payload, _ := protocol.DecodePayload[protocol.ScriptPayload](data)
		if payload.Event != "cancel" {
			return
		}
		receivedAction = action
		close(done)
	})

	f.handle.Cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for cancel message")
	}
	if receivedAction != "send" {
		t.Errorf("expected action 'send', got '%s'", receivedAction)
	}
}

func TestUseUploadOnCancelled(t *testing.T) {
	f := newUploadFixture(t)
	called := false
	f.handle.OnCancelled(func() { called = true })
	f.fromClient("cancelled", map[string]interface{}{})
	if !called {
		t.Error("expected cancelled handler to be called")
	}
}

func TestUseUploadProgress(t *testing.T) {
	f := newUploadFixture(t)
	if p := f.handle.Progress().Peek(); p.Loaded != 0 || p.Total != 0 {
		t.Error("expected initial progress to be 0")
	}
	f.fromClient("progress", map[string]interface{}{"loaded": float64(250), "total": float64(500)})
	if p := f.handle.Progress().Peek(); p.Loaded != 250 || p.Total != 500 {
		t.Errorf("expected 250/500, got %+v", p)
	}
}

func TestUseUploadWithoutRegistryReportsInsteadOfStarting(t *testing.T) {
	var handle *UploadHandle
	sess := NewSession(Config{SessionID: "s", Root: func(ctx *Ctx) work.Node {
		h := UseUpload(ctx)
		if handle == nil {
			handle = h
		}
		return work.BuildElement("input", AttachItem(h))
	}})
	sess.Start()
	t.Cleanup(sess.Close)
	onActor(t, sess, func() { _, _ = sess.RenderSnapshot() })
	diags := make(chan protocol.Diagnostic, 2)
	sess.Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) { diags <- d })

	onActor(t, sess, func() {
		id := handle.script.slot.id
		sess.Bus.Publish(protocol.Topic("script:"+id+":ready"), string(protocol.ScriptMessageAction), protocol.ScriptPayload{
			ScriptID: id, Event: "ready", Data: map[string]interface{}{"name": "a"},
		})
	})

	select {
	case d := <-diags:
		if d.Phase != "upload:grant" {
			t.Fatalf("unexpected diagnostic %+v", d)
		}
	case <-time.After(time.Second):
		t.Fatal("expected an upload:grant diagnostic")
	}
}

func TestUseUploadCleanupRevokesGrant(t *testing.T) {
	f := newUploadFixture(t)
	f.fromClient("ready", map[string]interface{}{"name": "a", "size": float64(1)})
	token := f.nextStart()["token"].(string)

	f.sess.Close()

	if _, _, err := f.sess.UploadRegistry.Reserve(token, 0, nil); !errors.Is(err, upload.ErrGrantUnknown) {
		t.Fatalf("expected the grant to be revoked on cleanup, got %v", err)
	}
}

func TestUseUploadOnReady(t *testing.T) {
	f := newUploadFixture(t)

	var name string
	var size int64
	var fileType string
	f.handle.OnReady(func(n string, s int64, ft string) { name, size, fileType = n, s, ft })
	f.fromClient("ready", map[string]interface{}{"name": "test.pdf", "size": float64(2048), "type": "application/pdf"})

	if name != "test.pdf" || size != 2048 || fileType != "application/pdf" {
		t.Errorf("unexpected ready values: %q %d %q", name, size, fileType)
	}
}

func TestUseUploadOnChange(t *testing.T) {
	f := newUploadFixture(t)
	var event UploadEvent
	f.handle.OnChange(func(e UploadEvent) { event = e })
	f.fromClient("change", map[string]interface{}{"name": "document.pdf", "size": float64(4096), "type": "application/pdf", "path": "/uploads/document.pdf"})

	if event.Name != "document.pdf" || event.Size != 4096 {
		t.Errorf("unexpected change event %+v", event)
	}
}

func TestUploadHandleProgressNilHandle(t *testing.T) {
	var handle *UploadHandle
	progress := handle.Progress().Peek()
	if progress.Loaded != 0 || progress.Total != 0 {
		t.Errorf("expected zero progress for nil handle, got %v", progress)
	}
}

func TestUploadHandleAcceptNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.Accept(UploadConfig{MaxSize: 1024})
}

func TestUploadHandleAttachToNilHandle(t *testing.T) {
	var handle *UploadHandle
	work.Apply(AttachItem(handle), &work.Element{Tag: "input"})
}

func TestUploadHandleOnReadyNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnReady(func(name string, size int64, fileType string) {})
}

func TestUploadHandleOnChangeNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnChange(func(e UploadEvent) {})
}

func TestUploadHandleOnProgressNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnProgress(func(loaded, total int64) {})
}

func TestUploadHandleOnErrorNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnError(func(err string) {})
}

func TestUploadHandleOnCancelledNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnCancelled(func() {})
}

func TestUploadHandleCancelNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.Cancel()
}

func TestUploadHandleCancelNilScript(t *testing.T) {
	handle := &UploadHandle{}
	handle.Cancel()
}

func TestUploadHandleOnCompleteNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnComplete(func(info upload.FileInfo) error { return nil })
}

func TestUploadHandleOnCompleteNilCell(t *testing.T) {
	handle := &UploadHandle{}
	handle.OnComplete(func(info upload.FileInfo) error { return nil })
}

func TestRunningTheBodyAgainDoesNotReplaceUploadConfigOrCallbacks(t *testing.T) {
	f := newUploadFixture(t)
	var calls []string
	f.handle.Accept(UploadConfig{MaxSize: 4096})
	f.handle.OnReady(func(string, int64, string) { calls = append(calls, "live") })

	onActor(t, f.sess, func() {
		f.sess.graph.Serialize(func() {
			h := &UploadHandle{script: f.handle.script, cell: f.handle.cell}
			h.Accept(UploadConfig{MaxSize: 1})
			h.OnReady(func(string, int64, string) { calls = append(calls, "serialize") })
		})
	})

	f.fromClient("ready", map[string]interface{}{"name": "a", "size": float64(1), "type": "image/png"})
	start := f.nextStart()
	if start["maxSize"] != int64(4096) || len(calls) != 1 || calls[0] != "live" {
		t.Fatalf("a serialize pass replaced live state: start=%v calls=%v", start, calls)
	}
}
