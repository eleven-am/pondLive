package runtime

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	render "github.com/eleven-am/pondlive/go/internal/render"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
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

	binding, ok := findUploadBinding(comp.prev)
	if !ok {
		t.Fatal("expected upload binding in structured output")
	}
	uploadID := binding.UploadID
	if binding.ComponentID == "" {
		t.Fatal("expected upload binding to include component id")
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush: %v", err)
	}

	if refreshed, ok := findUploadBinding(comp.prev); ok {
		uploadID = refreshed.UploadID
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
		t.Fatal("expected OnComplete to request render")
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

func TestUseUploadAttachPropRegistersMetadata(t *testing.T) {
	component := func(ctx Ctx, _ struct{}) h.Node {
		upload := UseUpload(ctx)
		upload.Accept("image/png")
		upload.AllowMultiple(true)
		upload.MaxSize(2048)
		return h.Div(
			h.Input(
				h.Type("file"),
				h.Attach(upload),
			),
		)
	}

	sess := NewSession(component, struct{}{})
	structured := sess.InitialStructured()
	if len(structured.UploadBindings) != 1 {
		t.Fatalf("expected one upload binding, got %d", len(structured.UploadBindings))
	}
	binding := structured.UploadBindings[0]
	if binding.ComponentID == "" {
		t.Fatal("expected binding to include component id")
	}
	if len(binding.Path) == 0 {
		t.Fatal("expected binding path metadata to be recorded")
	}
	if binding.UploadID == "" {
		t.Fatal("expected upload id to be populated")
	}
	if !binding.Multiple {
		t.Fatal("expected multiple selection flag to propagate")
	}
	if binding.MaxSize != 2048 {
		t.Fatalf("expected max size 2048, got %d", binding.MaxSize)
	}
	if len(binding.Accept) != 1 || binding.Accept[0] != "image/png" {
		t.Fatalf("expected accept metadata [image/png], got %v", binding.Accept)
	}

	statics := strings.Join(structured.S, "")
	if !strings.Contains(statics, "accept=\"image/png\"") {
		t.Fatalf("expected accept attribute in statics, got %q", statics)
	}
	if !strings.Contains(statics, "multiple=\"multiple\"") {
		t.Fatalf("expected multiple attribute in statics, got %q", statics)
	}
}

func TestAttachAllowsRefAndUploadOnSameElement(t *testing.T) {
	component := func(ctx Ctx, _ struct{}) h.Node {
		ref := UseElement[h.HTMLInputElement](ctx)
		upload := UseUpload(ctx)
		upload.AllowMultiple(true)
		return h.Div(
			h.Input(
				h.Type("file"),
				h.Attach(ref),
				h.Attach(upload),
			),
		)
	}

	sess := NewSession(component, struct{}{})
	structured := sess.InitialStructured()
	if len(structured.UploadBindings) != 1 {
		t.Fatalf("expected one upload binding, got %d", len(structured.UploadBindings))
	}
	binding := structured.UploadBindings[0]
	if binding.UploadID == "" {
		t.Fatal("expected upload id to be populated")
	}
	if !binding.Multiple {
		t.Fatal("expected upload binding to honour AllowMultiple")
	}

	sess.mu.Lock()
	refs := cloneRefMetaMap(sess.lastRefs)
	sess.mu.Unlock()
	if len(refs) == 0 {
		t.Fatal("expected ref metadata to be recorded")
	}
	foundInput := false
	for _, meta := range refs {
		if meta.Tag == "input" {
			foundInput = true
			break
		}
	}
	if !foundInput {
		t.Fatalf("expected input ref metadata, got %v", refs)
	}
}

func findUploadID(structured render.Structured) string {
	binding, ok := findUploadBinding(structured)
	if !ok {
		return ""
	}
	return binding.UploadID
}

func findUploadBinding(structured render.Structured) (render.UploadBinding, bool) {
	for _, binding := range structured.UploadBindings {
		if binding.UploadID != "" {
			return binding, true
		}
	}
	return render.UploadBinding{}, false
}

func findProgressText(structured render.Structured) string {
	for _, dyn := range structured.D {
		if dyn.Kind == render.DynamicText && dyn.Text != "" {
			return dyn.Text
		}
	}
	return ""
}
