package runtime

import (
	"bytes"
	"errors"
	"testing"

	h "github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	render "github.com/eleven-am/pondlive/go/internal/render"
)

type readSeekCloser struct{ *bytes.Reader }

func (r readSeekCloser) Close() error { return nil }

func TestUseUploadLifecycle(t *testing.T) {
	var (
		lastError error
		completed bool
	)

	component := func(ctx Ctx, _ struct{}) h.Node {
		upload := UseUpload(ctx)
		upload.OnError(func(err error) h.Updates {
			lastError = err
			return nil
		})
		upload.OnComplete(func(file UploadedFile) h.Updates {
			completed = true
			if file.FileMeta.Name != "avatar.png" {
				t.Fatalf("unexpected file name %q", file.FileMeta.Name)
			}
			return h.Rerender()
		})
		progress := upload.Progress()
		props := upload.BindInput()
		items := make([]h.Item, 1+len(props))
		items[0] = h.Type("file")
		for i, prop := range props {
			items[i+1] = prop
		}
		return h.Div(
			h.Input(items...),
			h.Textf("%d/%d", progress.Loaded, progress.Total),
		)
	}

	sess := NewLiveSession("sid", 1, component, struct{}{}, nil)
	node := sess.RenderRoot()
	if node == nil {
		t.Fatal("expected initial render to produce a node")
	}

	comp := sess.ComponentSession()
	if comp == nil {
		t.Fatal("expected component session")
	}

	uploadID := findUploadID(comp.prev)
	if uploadID == "" {
		t.Fatal("expected upload id in rendered attributes")
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush: %v", err)
	}

	if refreshedID := findUploadID(comp.prev); refreshedID != "" {
		uploadID = refreshedID
	}

	change := protocol.UploadClient{SID: "sid", ID: uploadID, Op: "change", Meta: &protocol.UploadMeta{Name: "avatar.png", Size: 128, Type: "image/png"}}
	if err := sess.HandleUploadMessage(change); err != nil {
		t.Fatalf("handle change: %v", err)
	}

	progressMsg := protocol.UploadClient{SID: "sid", ID: uploadID, Op: "progress", Loaded: 64, Total: 128}
	if err := sess.HandleUploadMessage(progressMsg); err != nil {
		t.Fatalf("handle progress: %v", err)
	}

	comp.mu.Lock()
	if len(comp.uploads) == 0 {
		comp.mu.Unlock()
		t.Fatal("expected registered upload slots")
	}
	comp.mu.Unlock()
	slot := comp.findUploadSlot(uploadID)
	if slot == nil {
		t.Fatalf("expected upload slot %s", uploadID)
	}
	slot.progressMu.RLock()
	loaded := slot.progress.Loaded
	total := slot.progress.Total
	slot.progressMu.RUnlock()
	if loaded != 64 || total != 128 {
		t.Fatalf("expected slot progress 64/128, got %d/%d", loaded, total)
	}
	if got := findProgressText(comp.prev); got != "64/128" {
		t.Fatalf("expected progress text 64/128, got %q", got)
	}

	payload := UploadedFile{
		FileMeta: FileMeta{Name: "avatar.png", Size: 4, Type: "text/plain"},
		Reader:   readSeekCloser{bytes.NewReader([]byte("data"))},
	}
	updates, err := comp.CompleteUpload(uploadID, payload)
	if err != nil {
		t.Fatalf("complete upload: %v", err)
	}
	if updates == nil {
		t.Fatal("expected OnComplete to request rerender")
	}
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after completion: %v", err)
	}
	if !completed {
		t.Fatal("expected completion handler to run")
	}

	boom := errors.New("boom")
	errMsg := protocol.UploadClient{SID: "sid", ID: uploadID, Op: "error", Error: boom.Error()}
	if err := sess.HandleUploadMessage(errMsg); err != nil {
		t.Fatalf("handle error: %v", err)
	}
	if lastError == nil || lastError.Error() != boom.Error() {
		t.Fatalf("expected last error to be %q, got %v", boom.Error(), lastError)
	}
}

func findUploadID(structured render.Structured) string {
	for _, dyn := range structured.D {
		if dyn.Kind != render.DynAttrs {
			continue
		}
		if id := dyn.Attrs["data-pond-upload"]; id != "" {
			return id
		}
	}
	for _, attrs := range staticAttrMaps(structured.S) {
		if id := attrs["data-pond-upload"]; id != "" {
			return id
		}
	}
	return ""
}

func findProgressText(structured render.Structured) string {
	for _, dyn := range structured.D {
		if dyn.Kind == render.DynText && dyn.Text != "" {
			return dyn.Text
		}
	}
	return ""
}
