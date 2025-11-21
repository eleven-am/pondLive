package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

func TestUseUploadBasic(t *testing.T) {
	var handle UploadHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		handle = UseUpload(ctx)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	progress := handle.Progress()
	if progress.Status != UploadStatusIdle {
		t.Errorf("expected idle status, got %v", progress.Status)
	}
}

func TestUseUploadBindTo(t *testing.T) {
	var handle UploadHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		handle = UseUpload(ctx)
		handle.Accept("image/*", "video/*")
		handle.AllowMultiple(true)
		handle.MaxSize(1024 * 1024 * 10)

		node := &dom.StructuredNode{Tag: "input"}
		handle.AttachTo(node)
		return node
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	node := sess.root.node
	if len(node.UploadBindings) != 1 {
		t.Fatalf("expected 1 upload binding, got %d", len(node.UploadBindings))
	}

	binding := node.UploadBindings[0]
	if !binding.Multiple {
		t.Error("expected multiple to be true")
	}
	if binding.MaxSize != 1024*1024*10 {
		t.Errorf("expected maxSize 10MB, got %d", binding.MaxSize)
	}
	if len(binding.Accept) != 2 {
		t.Errorf("expected 2 accept types, got %d", len(binding.Accept))
	}
}

func TestUseUploadCallbacks(t *testing.T) {
	var handle UploadHandle
	var changeCount int
	var completeCount int
	var errorCount int

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		handle = UseUpload(ctx)
		handle.OnChange(func(meta FileMeta) {
			changeCount++
		})
		handle.OnComplete(func(meta FileMeta) {
			completeCount++
		})
		handle.OnError(func(err error) {
			errorCount++
		})
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	uploadID := handle.slot.id
	sess.HandleUploadChange(uploadID, FileMeta{Name: "test.txt", Size: 100, Type: "text/plain"})

	if changeCount != 1 {
		t.Errorf("expected onChange to be called once, got %d", changeCount)
	}

	progress := handle.Progress()
	if progress.Status != UploadStatusUploading {
		t.Errorf("expected uploading status, got %v", progress.Status)
	}
}

func TestUseUploadProgress(t *testing.T) {
	var handle UploadHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		handle = UseUpload(ctx)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	uploadID := handle.slot.id

	sess.HandleUploadChange(uploadID, FileMeta{Name: "test.txt", Size: 1000, Type: "text/plain"})

	sess.HandleUploadProgress(uploadID, 500, 1000)

	progress := handle.Progress()
	if progress.Loaded != 500 {
		t.Errorf("expected loaded 500, got %d", progress.Loaded)
	}
	if progress.Total != 1000 {
		t.Errorf("expected total 1000, got %d", progress.Total)
	}
	if progress.Percent != 50.0 {
		t.Errorf("expected 50%% progress, got %.2f", progress.Percent)
	}
}

func TestUseUploadComplete(t *testing.T) {
	var handle UploadHandle
	var completed bool

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		handle = UseUpload(ctx)
		handle.OnComplete(func(meta FileMeta) {
			completed = true
		})
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	uploadID := handle.slot.id
	sess.HandleUploadComplete(uploadID, FileMeta{Name: "test.txt", Size: 100, Type: "text/plain"})

	if !completed {
		t.Error("expected onComplete to be called")
	}

	progress := handle.Progress()
	if progress.Status != UploadStatusComplete {
		t.Errorf("expected complete status, got %v", progress.Status)
	}
	if progress.Percent != 100 {
		t.Errorf("expected 100%% progress, got %.2f", progress.Percent)
	}
}

func TestUseUploadError(t *testing.T) {
	var handle UploadHandle
	var errorReceived error

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		handle = UseUpload(ctx)
		handle.OnError(func(err error) {
			errorReceived = err
		})
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	uploadID := handle.slot.id
	testErr := ErrUploadTooLarge
	sess.HandleUploadError(uploadID, testErr)

	if errorReceived != testErr {
		t.Errorf("expected error %v, got %v", testErr, errorReceived)
	}

	progress := handle.Progress()
	if progress.Status != UploadStatusError {
		t.Errorf("expected error status, got %v", progress.Status)
	}
	if progress.Error != testErr {
		t.Errorf("expected error in progress, got %v", progress.Error)
	}
}

func TestUseUploadCancel(t *testing.T) {
	var handle UploadHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		handle = UseUpload(ctx)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	uploadID := handle.slot.id
	sess.HandleUploadChange(uploadID, FileMeta{Name: "test.txt", Size: 1000, Type: "text/plain"})

	handle.Cancel()

	progress := handle.Progress()
	if progress.Status != UploadStatusCancelled {
		t.Errorf("expected cancelled status, got %v", progress.Status)
	}
}

func TestUseUploadMaxSize(t *testing.T) {
	var handle UploadHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		handle = UseUpload(ctx)
		handle.MaxSize(1024 * 1024)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	uploadID := handle.slot.id
	maxSize, ok := sess.UploadMaxSize(uploadID)
	if !ok {
		t.Fatal("expected to find upload slot")
	}
	if maxSize != 1024*1024 {
		t.Errorf("expected maxSize 1MB, got %d", maxSize)
	}
}

func TestUseUploadMultipleSlots(t *testing.T) {
	var handle1, handle2 UploadHandle

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		handle1 = UseUpload(ctx)
		handle2 = UseUpload(ctx)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	if handle1.slot.id == handle2.slot.id {
		t.Error("expected different upload IDs for different hooks")
	}

	id1 := handle1.slot.id
	id2 := handle2.slot.id

	if !endsWith(id1, ":u0") {
		t.Errorf("expected first upload ID to end with ':u0', got %q", id1)
	}
	if !endsWith(id2, ":u1") {
		t.Errorf("expected second upload ID to end with ':u1', got %q", id2)
	}
}

func endsWith(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
