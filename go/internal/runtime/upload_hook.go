package runtime

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sync"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

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

var ErrUploadTooLarge = errors.New("runtime: upload exceeds limit")

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

// UploadedFile provides access to the staged upload on the server.
type UploadedFile struct {
	FileMeta
	TempPath string
	Reader   io.ReadSeekCloser
}

// UploadHandle exposes lifecycle controls for the upload hook.
type UploadHandle struct {
	slot *uploadSlot
}

// BindInput returns props that attach the upload slot metadata to an <input type="file"> element.
func (handle UploadHandle) BindInput(props ...h.Prop) []h.Prop {
	if handle.slot == nil {
		if len(props) == 0 {
			return nil
		}
		out := make([]h.Prop, len(props))
		copy(out, props)
		return out
	}
	out := make([]h.Prop, 0, len(props)+1)
	out = append(out, h.Attach(handle))
	out = append(out, props...)
	return out
}

func (handle UploadHandle) AttachTo(el *h.Element) {
	if handle.slot == nil || el == nil {
		return
	}
	handle.slot.registerBinding(el)
}

// OnChange registers a callback invoked when the client picks a file.
func (handle UploadHandle) OnChange(fn func(FileMeta)) {
	if handle.slot == nil {
		return
	}
	handle.slot.setOnChange(fn)
}

// OnComplete registers the callback fired after the file is uploaded and staged on the server.
func (handle UploadHandle) OnComplete(fn func(UploadedFile) h.Updates) {
	if handle.slot == nil {
		return
	}
	handle.slot.setOnComplete(fn)
}

// OnError registers a callback for terminal upload failures.
func (handle UploadHandle) OnError(fn func(error) h.Updates) {
	if handle.slot == nil {
		return
	}
	handle.slot.setOnError(fn)
}

// Progress returns the most recent upload progress snapshot.
func (handle UploadHandle) Progress() UploadProgress {
	if handle.slot == nil {
		return UploadProgress{Status: UploadStatusIdle}
	}
	return handle.slot.progressSnapshot()
}

// Cancel requests cancellation of the in-flight upload.
func (handle UploadHandle) Cancel() {
	if handle.slot == nil {
		return
	}
	handle.slot.requestCancel()
}

// Accept overrides the accepted MIME types communicated to the browser input.
func (handle UploadHandle) Accept(types ...string) {
	if handle.slot == nil {
		return
	}
	handle.slot.setAccept(types)
}

// AllowMultiple toggles multiple file selection in the browser input.
func (handle UploadHandle) AllowMultiple(enabled bool) {
	if handle.slot == nil {
		return
	}
	handle.slot.setMultiple(enabled)
}

// MaxSize sets the maximum payload size (in bytes) enforced on the server.
func (handle UploadHandle) MaxSize(limit int64) {
	if handle.slot == nil {
		return
	}
	handle.slot.setMaxSize(limit)
}

// UseUpload registers an upload slot for the current component.
func UseUpload(ctx Ctx) UploadHandle {
	if ctx.frame == nil {
		panic("runtime: UseUpload called outside render")
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
		if cell.slot != nil && cell.slot.sess == nil {
			cell.slot = ctx.sess.registerUploadSlot(ctx.comp, idx)
		}
		if cell.slot == nil {
			cell.slot = ctx.sess.registerUploadSlot(ctx.comp, idx)
		}
	}
	return UploadHandle{slot: cell.slot}
}

type uploadCell struct {
	slot *uploadSlot
}

type uploadSlot struct {
	id        string
	sess      *ComponentSession
	component *component
	hookIndex int

	accept   []string
	multiple bool
	maxSize  int64

	onChange   func(FileMeta)
	onComplete func(UploadedFile) h.Updates
	onError    func(error) h.Updates

	progress   UploadProgress
	progressMu sync.RWMutex

	cancelled bool
}

func (slot *uploadSlot) registerBinding(el *h.Element) {
	if slot == nil || el == nil || slot.id == "" {
		return
	}
	binding := h.UploadBinding{
		UploadID: slot.id,
		Multiple: slot.multiple,
		MaxSize:  slot.maxSize,
	}
	if len(slot.accept) > 0 {
		binding.Accept = append([]string(nil), slot.accept...)
	}
	el.UploadBindings = append(el.UploadBindings, binding)
	if len(binding.Accept) > 0 {
		if el.Attrs == nil {
			el.Attrs = map[string]string{}
		}
		el.Attrs["accept"] = joinStrings(binding.Accept, ",")
	} else if el.Attrs != nil {
		delete(el.Attrs, "accept")
	}
	if slot.multiple {
		if el.Attrs == nil {
			el.Attrs = map[string]string{}
		}
		el.Attrs["multiple"] = "multiple"
	} else if el.Attrs != nil {
		delete(el.Attrs, "multiple")
	}
}

func (s *ComponentSession) registerUploadSlot(comp *component, index int) *uploadSlot {
	if s == nil || comp == nil {
		return nil
	}
	s.uploadMu.Lock()
	defer s.uploadMu.Unlock()
	if s.uploads == nil {
		s.uploads = make(map[string]*uploadSlot)
	}
	s.uploadSeq++
	id := fmt.Sprintf("u%d", s.uploadSeq)
	slot := &uploadSlot{
		id:        id,
		sess:      s,
		component: comp,
		hookIndex: index,
		progress:  UploadProgress{Status: UploadStatusIdle},
	}
	if s.uploadByComponent == nil {
		s.uploadByComponent = make(map[*component]map[int]*uploadSlot)
	}
	compSlots := s.uploadByComponent[comp]
	if compSlots == nil {
		compSlots = make(map[int]*uploadSlot)
		s.uploadByComponent[comp] = compSlots
	}
	compSlots[index] = slot
	s.uploads[id] = slot
	return slot
}

func (s *ComponentSession) releaseUploadSlots(comp *component) {
	if s == nil || comp == nil {
		return
	}
	s.uploadMu.Lock()
	slots := s.uploadByComponent[comp]
	delete(s.uploadByComponent, comp)
	s.uploadMu.Unlock()
	for _, slot := range slots {
		if slot != nil {
			slot.dispose()
		}
	}
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

func (slot *uploadSlot) dispose() {
	if slot == nil || slot.sess == nil {
		return
	}
	slot.sess.uploadMu.Lock()
	delete(slot.sess.uploads, slot.id)
	slot.sess.uploadMu.Unlock()
	slot.sess = nil
}

func (slot *uploadSlot) setOnChange(fn func(FileMeta)) {
	slot.progressMu.Lock()
	slot.onChange = fn
	slot.progressMu.Unlock()
}

func (slot *uploadSlot) setOnComplete(fn func(UploadedFile) h.Updates) {
	slot.progressMu.Lock()
	slot.onComplete = fn
	slot.progressMu.Unlock()
}

func (slot *uploadSlot) setOnError(fn func(error) h.Updates) {
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
	if slot.sess != nil {
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
	if sess := slot.sess.owner; sess != nil {
		_ = sess.CancelUpload(slot.id)
	}
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

func (slot *uploadSlot) handleError(err error) h.Updates {
	if slot == nil {
		return nil
	}
	slot.fail(err)
	slot.progressMu.RLock()
	handler := slot.onError
	slot.progressMu.RUnlock()
	if handler != nil {
		return handler(err)
	}
	return nil
}

func (slot *uploadSlot) handleComplete(file UploadedFile) h.Updates {
	if slot == nil {
		return nil
	}
	slot.processing()
	slot.progressMu.RLock()
	handler := slot.onComplete
	slot.progressMu.RUnlock()
	if handler == nil {
		slot.complete()
		return nil
	}
	updates := handler(file)
	slot.complete()
	return updates
}

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
	if updates := slot.handleError(err); updates != nil {
		s.markDirty(slot.component)
	}
}

func (s *ComponentSession) HandleUploadCancelled(id string) {
	slot := s.findUploadSlot(id)
	if slot == nil {
		return
	}
	slot.cancel()
}

func (s *ComponentSession) CompleteUpload(id string, file UploadedFile) (h.Updates, error) {
	slot := s.findUploadSlot(id)
	if slot == nil {
		if file.Reader != nil {
			_ = file.Reader.Close()
		}
		if file.TempPath != "" {
			_ = os.Remove(file.TempPath)
		}
		return nil, errors.New("runtime: upload slot not found")
	}
	var updates h.Updates
	err := s.withRecovery("upload:complete", func() error {
		updates = slot.handleComplete(file)
		return nil
	})
	if updates != nil {
		s.markDirty(slot.component)
	}
	return updates, err
}

// UploadMaxSize reports the configured server-side size limit for the slot, if any.
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

func joinStrings(values []string, sep string) string {
	if len(values) == 0 {
		return ""
	}
	out := ""
	for i, v := range values {
		if i > 0 {
			out += sep
		}
		out += v
	}
	return out
}

func StageUploadedFile(part io.Reader, filename, contentType string, sizeLimit int64) (UploadedFile, error) {
	if part == nil {
		return UploadedFile{}, errors.New("runtime: missing upload payload")
	}
	dir := os.TempDir()
	file, err := os.CreateTemp(dir, "pond-upload-*")
	if err != nil {
		return UploadedFile{}, fmt.Errorf("runtime: create temp file: %w", err)
	}
	var written int64
	if sizeLimit > 0 {
		written, err = io.Copy(file, io.LimitReader(part, sizeLimit+1))
		if err == nil && written > sizeLimit {
			err = ErrUploadTooLarge
		}
	} else {
		written, err = io.Copy(file, part)
	}
	if err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return UploadedFile{}, err
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		_ = file.Close()
		_ = os.Remove(file.Name())
		return UploadedFile{}, err
	}
	meta := FileMeta{Name: filepath.Base(filename), Size: written, Type: contentType}
	return UploadedFile{FileMeta: meta, TempPath: file.Name(), Reader: file}, nil
}
