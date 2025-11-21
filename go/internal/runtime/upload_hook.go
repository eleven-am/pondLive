package runtime

import (
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

var ErrUploadTooLarge = errors.New("runtime2: upload exceeds limit")

// UploadStatus enumerates the states an upload slot can be in.
type UploadStatus string

const (
	UploadStatusIdle       UploadStatus = "idle"
	UploadStatusSelecting  UploadStatus = "selecting"
	UploadStatusUploading  UploadStatus = "uploading"
	UploadStatusProcessing UploadStatus = "processing"
	UploadStatusComplete   UploadStatus = "complete"
	UploadStatusError      UploadStatus = "error"
	UploadStatusCancelled  UploadStatus = "cancelled"
)

// FileMeta describes the file selected in the browser before upload.
type FileMeta struct {
	Name string
	Size int64
	Type string
}

// UploadProgress exposes the current upload status and byte counts.
type UploadProgress struct {
	Loaded  int64
	Total   int64
	Percent float64
	Status  UploadStatus
	Error   error
}

// UploadHandle exposes lifecycle controls for the upload hook.
type UploadHandle struct {
	slot *uploadSlot
}

// AttachTo implements the Attachment interface, allowing UploadHandle to be used with h.Attach.
func (h UploadHandle) AttachTo(node *dom.StructuredNode) {
	if h.slot == nil || node == nil {
		return
	}
	h.slot.registerBinding(node)
}

// OnChange registers a callback invoked when the client picks a file.
func (h UploadHandle) OnChange(fn func(FileMeta)) {
	if h.slot != nil {
		h.slot.setOnChange(fn)
	}
}

// OnComplete registers the callback fired after the file is uploaded and staged.
func (h UploadHandle) OnComplete(fn func(FileMeta)) {
	if h.slot != nil {
		h.slot.setOnComplete(fn)
	}
}

// OnError registers a callback for terminal upload failures.
func (h UploadHandle) OnError(fn func(error)) {
	if h.slot != nil {
		h.slot.setOnError(fn)
	}
}

// Progress returns the most recent upload progress snapshot.
func (h UploadHandle) Progress() UploadProgress {
	if h.slot == nil {
		return UploadProgress{Status: UploadStatusIdle}
	}
	return h.slot.progressSnapshot()
}

// Cancel requests cancellation of the in-flight upload.
func (h UploadHandle) Cancel() {
	if h.slot != nil {
		h.slot.requestCancel()
	}
}

// Accept overrides the accepted MIME types communicated to the browser input.
func (h UploadHandle) Accept(types ...string) {
	if h.slot != nil {
		h.slot.setAccept(types)
	}
}

// AllowMultiple toggles multiple file selection in the browser input.
func (h UploadHandle) AllowMultiple(enabled bool) {
	if h.slot != nil {
		h.slot.setMultiple(enabled)
	}
}

// MaxSize sets the maximum payload size (in bytes) enforced on the server.
func (h UploadHandle) MaxSize(limit int64) {
	if h.slot != nil {
		h.slot.setMaxSize(limit)
	}
}

// UseUpload registers an upload slot for the current component.
func UseUpload(ctx Ctx) UploadHandle {
	if ctx.frame == nil {
		panic("runtime2: UseUpload called outside render")
	}

	idx := ctx.frame.idx
	ctx.frame.idx++

	if idx >= len(ctx.frame.cells) {
		cell := &uploadCell{}
		ctx.frame.cells = append(ctx.frame.cells, cell)
	}

	raw := ctx.frame.cells[idx]
	cell, ok := raw.(*uploadCell)
	if !ok {
		panicHookMismatch(ctx.comp, idx, "UseUpload", raw)
	}

	if ctx.sess != nil {
		if cell.slot == nil || cell.slot.sess == nil {
			cell.slot = ctx.sess.registerUploadSlot(ctx.comp, idx)
		}
	}

	return UploadHandle{slot: cell.slot}
}

type uploadCell struct {
	slot *uploadSlot
}

// uploadSlot tracks a single upload instance.
type uploadSlot struct {
	id        string
	sess      *ComponentSession
	component *component
	hookIndex int

	accept   []string
	multiple bool
	maxSize  int64

	onChange   func(FileMeta)
	onComplete func(FileMeta)
	onError    func(error)

	progress   UploadProgress
	progressMu sync.RWMutex

	cancelled bool
}

func (slot *uploadSlot) registerBinding(node *dom.StructuredNode) {
	if slot == nil || node == nil || slot.id == "" {
		return
	}

	binding := dom.UploadBinding{
		UploadID: slot.id,
		Multiple: slot.multiple,
		MaxSize:  slot.maxSize,
	}

	if len(slot.accept) > 0 {
		binding.Accept = append([]string(nil), slot.accept...)
	}

	node.UploadBindings = append(node.UploadBindings, binding)
}

func (s *ComponentSession) registerUploadSlot(comp *component, index int) *uploadSlot {
	if s == nil || comp == nil {
		return nil
	}

	s.uploadMu.Lock()
	defer s.uploadMu.Unlock()

	id := fmt.Sprintf("%s:u%d", comp.id, index)

	slot := &uploadSlot{
		id:        id,
		sess:      s,
		component: comp,
		hookIndex: index,
		progress:  UploadProgress{Status: UploadStatusIdle},
	}

	s.uploads[id] = slot
	return slot
}

func (s *ComponentSession) findUploadSlot(id string) *uploadSlot {
	if s == nil || id == "" {
		return nil
	}
	s.uploadMu.Lock()
	slot := s.uploads[id]
	s.uploadMu.Unlock()
	return slot
}

func (slot *uploadSlot) setOnChange(fn func(FileMeta)) {
	slot.progressMu.Lock()
	slot.onChange = fn
	slot.progressMu.Unlock()
}

func (slot *uploadSlot) setOnComplete(fn func(FileMeta)) {
	slot.progressMu.Lock()
	slot.onComplete = fn
	slot.progressMu.Unlock()
}

func (slot *uploadSlot) setOnError(fn func(error)) {
	slot.progressMu.Lock()
	slot.onError = fn
	slot.progressMu.Unlock()
}

func (slot *uploadSlot) setAccept(types []string) {
	slot.progressMu.Lock()
	slot.accept = append([]string(nil), types...)
	slot.progressMu.Unlock()
}

func (slot *uploadSlot) setMultiple(enabled bool) {
	slot.progressMu.Lock()
	slot.multiple = enabled
	slot.progressMu.Unlock()
}

func (slot *uploadSlot) setMaxSize(limit int64) {
	if limit < 0 {
		limit = 0
	}
	slot.progressMu.Lock()
	slot.maxSize = limit
	slot.progressMu.Unlock()
}

func (slot *uploadSlot) progressSnapshot() UploadProgress {
	slot.progressMu.RLock()
	progress := slot.progress
	slot.progressMu.RUnlock()
	return progress
}

func (slot *uploadSlot) updateProgress(update func(*UploadProgress)) {
	if slot == nil {
		return
	}

	slot.progressMu.Lock()
	update(&slot.progress)
	slot.progressMu.Unlock()

	if slot.sess != nil && slot.component != nil {
		slot.sess.markDirty(slot.component)
	}
}

func (slot *uploadSlot) beginUpload(meta FileMeta) {
	slot.progressMu.Lock()
	slot.cancelled = false
	slot.progressMu.Unlock()

	slot.updateProgress(func(p *UploadProgress) {
		p.Status = UploadStatusUploading
		p.Total = meta.Size
		p.Loaded = 0
		p.Percent = 0
		p.Error = nil
	})
}

func (slot *uploadSlot) updateBytes(loaded, total int64) {
	slot.updateProgress(func(p *UploadProgress) {
		p.Status = UploadStatusUploading
		p.Loaded = loaded
		if total >= 0 {
			p.Total = total
		}
		if p.Total > 0 {
			pct := (float64(p.Loaded) / float64(p.Total)) * 100
			p.Percent = math.Min(100, pct)
		} else {
			p.Percent = 0
		}
		p.Error = nil
	})
}

func (slot *uploadSlot) processing() {
	slot.updateProgress(func(p *UploadProgress) {
		p.Status = UploadStatusProcessing
		p.Percent = 100
		p.Error = nil
	})
}

func (slot *uploadSlot) complete() {
	slot.updateProgress(func(p *UploadProgress) {
		p.Status = UploadStatusComplete
		p.Percent = 100
		p.Error = nil
	})
}

func (slot *uploadSlot) fail(err error) {
	slot.updateProgress(func(p *UploadProgress) {
		p.Status = UploadStatusError
		p.Error = err
	})
}

func (slot *uploadSlot) cancel() {
	slot.updateProgress(func(p *UploadProgress) {
		p.Status = UploadStatusCancelled
	})
}

func (slot *uploadSlot) requestCancel() {
	if slot == nil || slot.sess == nil {
		return
	}

	slot.progressMu.Lock()
	if slot.cancelled {
		slot.progressMu.Unlock()
		return
	}
	slot.cancelled = true
	slot.progressMu.Unlock()

	slot.cancel()
}

func (slot *uploadSlot) handleChange(meta FileMeta) {
	if slot == nil {
		return
	}

	slot.beginUpload(meta)

	slot.progressMu.RLock()
	handler := slot.onChange
	slot.progressMu.RUnlock()

	if handler != nil {
		handler(meta)
	}
}

func (slot *uploadSlot) handleError(err error) {
	if slot == nil {
		return
	}

	slot.fail(err)

	slot.progressMu.RLock()
	handler := slot.onError
	slot.progressMu.RUnlock()

	if handler != nil {
		handler(err)
	}
}

func (slot *uploadSlot) handleComplete(file FileMeta) {
	if slot == nil {
		return
	}

	slot.processing()

	slot.progressMu.RLock()
	handler := slot.onComplete
	slot.progressMu.RUnlock()

	if handler != nil {
		handler(file)
	}

	slot.complete()
}

// ComponentSession upload event handlers

func (s *ComponentSession) HandleUploadChange(id string, meta FileMeta) {
	slot := s.findUploadSlot(id)
	if slot == nil {
		return
	}

	s.withRecovery("upload:change", func() error {
		slot.handleChange(meta)
		return nil
	})
}

func (s *ComponentSession) HandleUploadProgress(id string, loaded, total int64) {
	slot := s.findUploadSlot(id)
	if slot == nil {
		return
	}
	slot.updateBytes(loaded, total)
}

func (s *ComponentSession) HandleUploadError(id string, err error) {
	slot := s.findUploadSlot(id)
	if slot == nil {
		return
	}
	slot.handleError(err)
}

func (s *ComponentSession) HandleUploadComplete(id string, file FileMeta) {
	slot := s.findUploadSlot(id)
	if slot == nil {
		return
	}

	s.withRecovery("upload:complete", func() error {
		slot.handleComplete(file)
		return nil
	})
}

func (s *ComponentSession) UploadMaxSize(id string) (int64, bool) {
	slot := s.findUploadSlot(id)
	if slot == nil {
		return 0, false
	}

	slot.progressMu.RLock()
	limit := slot.maxSize
	slot.progressMu.RUnlock()

	return limit, true
}
