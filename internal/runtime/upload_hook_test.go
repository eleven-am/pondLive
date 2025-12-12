package runtime

import (
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/upload"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestUseUpload(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)
	if handle.script == nil {
		t.Fatal("expected script to be non-nil")
	}

	if handle.token == "" {
		t.Error("expected token to be generated")
	}

	cb, ok := sess.UploadRegistry.Lookup(handle.token)
	if !ok {
		t.Fatal("expected callback to be registered in upload registry")
	}
	if cb.Token != handle.token {
		t.Errorf("expected token %s, got %s", handle.token, cb.Token)
	}
}

func TestUseUploadAccept(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)
	handle.Accept(UploadConfig{
		MaxSize:  1024 * 1024,
		Accept:   []string{"image/png", "image/jpeg"},
		Multiple: true,
	})

	cb, ok := sess.UploadRegistry.Lookup(handle.token)
	if !ok {
		t.Fatal("expected callback in registry")
	}
	if cb.MaxSize != 1024*1024 {
		t.Errorf("expected MaxSize 1048576, got %d", cb.MaxSize)
	}
	if len(cb.AllowedTypes) != 2 {
		t.Errorf("expected 2 allowed types, got %d", len(cb.AllowedTypes))
	}
}

func TestUseUploadAttachTo(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)
	elem := &work.Element{Tag: "input"}
	handle.AttachTo(elem)

	if elem.Script == nil {
		t.Fatal("expected Script to be attached")
	}
}

func TestUseUploadOnProgress(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)

	var loaded, total int64
	done := make(chan struct{})
	handle.OnProgress(func(l, t int64) {
		loaded = l
		total = t
		close(done)
	})

	handle.script.slot.handleMessage("progress", map[string]interface{}{
		"loaded": float64(500),
		"total":  float64(1000),
	})

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for progress handler")
	}

	if loaded != 500 {
		t.Errorf("expected loaded 500, got %d", loaded)
	}
	if total != 1000 {
		t.Errorf("expected total 1000, got %d", total)
	}
}

func TestUseUploadOnComplete(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)

	var completedInfo upload.FileInfo
	done := make(chan struct{})
	handle.OnComplete(func(info upload.FileInfo) error {
		completedInfo = info
		close(done)
		return nil
	})

	cb, _ := sess.UploadRegistry.Lookup(handle.token)
	if cb.OnComplete == nil {
		t.Fatal("expected OnComplete to be set")
	}

	err := cb.OnComplete(upload.FileInfo{
		ID:   "test-id",
		Size: 1024,
		MetaData: map[string]string{
			"filename": "test.png",
			"filetype": "image/png",
		},
		Storage: map[string]string{
			"Path": "/path/to/file",
		},
	})
	if err != nil {
		t.Fatalf("OnComplete returned error: %v", err)
	}

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for complete handler")
	}

	if completedInfo.MetaData["filename"] != "test.png" {
		t.Errorf("expected filename test.png, got %s", completedInfo.MetaData["filename"])
	}
}

func TestUseUploadOnError(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)

	var errMsg string
	done := make(chan struct{})
	handle.OnError(func(msg string) {
		errMsg = msg
		close(done)
	})

	handle.script.slot.handleMessage("error", map[string]interface{}{
		"error": "upload failed",
	})

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for error handler")
	}

	if errMsg != "upload failed" {
		t.Errorf("expected error 'upload failed', got '%s'", errMsg)
	}
}

func TestUseUploadCancel(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)

	var receivedAction string
	done := make(chan struct{})
	scriptTopic := protocol.Topic("script:" + handle.script.slot.id)
	sess.Bus.Subscribe(scriptTopic, func(action string, data interface{}) {
		receivedAction = action
		close(done)
	})

	handle.Cancel()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for cancel message")
	}

	if receivedAction != "send" {
		t.Errorf("expected action 'send', got '%s'", receivedAction)
	}
}

func TestUseUploadOnCancelled(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)

	called := false
	done := make(chan struct{})
	handle.OnCancelled(func() {
		called = true
		close(done)
	})

	handle.script.slot.handleMessage("cancelled", map[string]interface{}{})

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for cancelled handler")
	}

	if !called {
		t.Error("expected cancelled handler to be called")
	}
}

func TestUseUploadProgress(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)

	progress := handle.Progress()
	if progress.Loaded != 0 || progress.Total != 0 {
		t.Error("expected initial progress to be 0")
	}

	handle.script.slot.handleMessage("progress", map[string]interface{}{
		"loaded": float64(250),
		"total":  float64(500),
	})

	time.Sleep(50 * time.Millisecond)

	progress = handle.Progress()
	if progress.Loaded != 250 {
		t.Errorf("expected loaded 250, got %d", progress.Loaded)
	}
	if progress.Total != 500 {
		t.Errorf("expected total 500, got %d", progress.Total)
	}
}

func TestUseUploadStableAcrossRerenders(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle1 := UseUpload(ctx)
	token1 := handle1.token

	ctx.hookIndex = 0
	handle2 := UseUpload(ctx)
	token2 := handle2.token

	if token1 != token2 {
		t.Errorf("expected same token across renders, got %s and %s", token1, token2)
	}

	if handle1.script.slot != handle2.script.slot {
		t.Error("expected same script slot across renders")
	}
}

func TestUseUploadNilRegistry(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		Scripts: make(map[string]*scriptSlot),
		Bus:     protocol.NewBus(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)

	if handle.token == "" {
		t.Error("expected token to be generated even without UploadRegistry")
	}
}

func TestUseUploadCleanup(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)
	token := handle.token

	_, ok := sess.UploadRegistry.Lookup(token)
	if !ok {
		t.Fatal("expected callback in registry")
	}

	for _, cleanup := range inst.cleanups {
		cleanup()
	}

	_, ok = sess.UploadRegistry.Lookup(token)
	if ok {
		t.Error("expected callback to be removed after cleanup")
	}
}

func TestUseUploadOnReady(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)

	var name string
	var size int64
	var fileType string
	done := make(chan struct{})
	handle.OnReady(func(n string, s int64, ft string) {
		name = n
		size = s
		fileType = ft
		close(done)
	})

	handle.script.slot.handleMessage("ready", map[string]interface{}{
		"name": "test.pdf",
		"size": float64(2048),
		"type": "application/pdf",
	})

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for ready handler")
	}

	if name != "test.pdf" {
		t.Errorf("expected name 'test.pdf', got '%s'", name)
	}
	if size != 2048 {
		t.Errorf("expected size 2048, got %d", size)
	}
	if fileType != "application/pdf" {
		t.Errorf("expected type 'application/pdf', got '%s'", fileType)
	}
}

func TestUseUploadOnChange(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)

	var event UploadEvent
	done := make(chan struct{})
	handle.OnChange(func(e UploadEvent) {
		event = e
		close(done)
	})

	handle.script.slot.handleMessage("change", map[string]interface{}{
		"name": "document.pdf",
		"size": float64(4096),
		"type": "application/pdf",
		"path": "/uploads/document.pdf",
	})

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for change handler")
	}

	if event.Name != "document.pdf" {
		t.Errorf("expected name 'document.pdf', got '%s'", event.Name)
	}
	if event.Size != 4096 {
		t.Errorf("expected size 4096, got %d", event.Size)
	}
}

func TestUploadHandleProgressNilHandle(t *testing.T) {
	var handle *UploadHandle
	progress := handle.Progress()
	if progress.Loaded != 0 || progress.Total != 0 {
		t.Errorf("expected zero progress for nil handle, got %v", progress)
	}
}

func TestUploadHandleProgressNilProgressGetter(t *testing.T) {
	handle := &UploadHandle{
		progressGetter: nil,
	}
	progress := handle.Progress()
	if progress.Loaded != 0 || progress.Total != 0 {
		t.Errorf("expected zero progress for nil progressGetter, got %v", progress)
	}
}

func TestUploadHandleAcceptNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.Accept(UploadConfig{MaxSize: 1024})
}

func TestUploadHandleAcceptNilSession(t *testing.T) {
	handle := &UploadHandle{
		session: nil,
	}
	handle.Accept(UploadConfig{MaxSize: 1024})
}

func TestUploadHandleAcceptNilRegistry(t *testing.T) {
	handle := &UploadHandle{
		session: &Session{UploadRegistry: nil},
	}
	handle.Accept(UploadConfig{MaxSize: 1024})
}

func TestUploadHandleAttachToNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.AttachTo(&work.Element{Tag: "input"})
}

func TestUploadHandleAttachToNilScript(t *testing.T) {
	handle := &UploadHandle{
		script: nil,
	}
	handle.AttachTo(&work.Element{Tag: "input"})
}

func TestUploadHandleOnReadyNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnReady(func(name string, size int64, fileType string) {})
}

func TestUploadHandleOnReadyNilScript(t *testing.T) {
	handle := &UploadHandle{
		script: nil,
	}
	handle.OnReady(func(name string, size int64, fileType string) {})
}

func TestUploadHandleOnChangeNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnChange(func(e UploadEvent) {})
}

func TestUploadHandleOnChangeNilScript(t *testing.T) {
	handle := &UploadHandle{
		script: nil,
	}
	handle.OnChange(func(e UploadEvent) {})
}

func TestUploadHandleOnProgressNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnProgress(func(loaded, total int64) {})
}

func TestUploadHandleOnProgressNilScript(t *testing.T) {
	handle := &UploadHandle{
		script: nil,
	}
	handle.OnProgress(func(loaded, total int64) {})
}

func TestUploadHandleOnErrorNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnError(func(err string) {})
}

func TestUploadHandleOnErrorNilScript(t *testing.T) {
	handle := &UploadHandle{
		script: nil,
	}
	handle.OnError(func(err string) {})
}

func TestUploadHandleOnCancelledNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnCancelled(func() {})
}

func TestUploadHandleOnCancelledNilScript(t *testing.T) {
	handle := &UploadHandle{
		script: nil,
	}
	handle.OnCancelled(func() {})
}

func TestUploadHandleCancelNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.Cancel()
}

func TestUploadHandleCancelNilScript(t *testing.T) {
	handle := &UploadHandle{
		script: nil,
	}
	handle.Cancel()
}

func TestUploadHandleOnCompleteNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.OnComplete(func(info upload.FileInfo) error { return nil })
}

func TestUploadHandleOnCompleteNilSession(t *testing.T) {
	handle := &UploadHandle{
		session: nil,
	}
	handle.OnComplete(func(info upload.FileInfo) error { return nil })
}

func TestUploadHandleOnCompleteNilRegistry(t *testing.T) {
	handle := &UploadHandle{
		session: &Session{UploadRegistry: nil},
	}
	handle.OnComplete(func(info upload.FileInfo) error { return nil })
}

func TestUploadHandleSendStartNilHandle(t *testing.T) {
	var handle *UploadHandle
	handle.sendStart()
}

func TestUploadHandleSendStartNilScript(t *testing.T) {
	handle := &UploadHandle{
		script: nil,
	}
	handle.sendStart()
}

func TestUploadHandleSendStartWithConfig(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		Scripts:        make(map[string]*scriptSlot),
		Bus:            protocol.NewBus(),
		UploadRegistry: upload.NewRegistry(),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	handle := UseUpload(ctx)
	handle.Accept(UploadConfig{
		MaxSize:  2048,
		Accept:   []string{"image/*"},
		Multiple: true,
	})

	var receivedData interface{}
	done := make(chan struct{})
	scriptTopic := protocol.Topic("script:" + handle.script.slot.id)
	sess.Bus.Subscribe(scriptTopic, func(action string, data interface{}) {
		if action == "send" {
			receivedData = data
			close(done)
		}
	})

	handle.sendStart()

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for start message")
	}

	if receivedData == nil {
		t.Fatal("expected to receive data")
	}
}
