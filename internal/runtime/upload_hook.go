package runtime

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/upload"
)

//go:embed upload.js
var uploadScript string

type UploadProgress struct {
	Loaded int64
	Total  int64
}

type UploadConfig struct {
	MaxSize  int64
	Accept   []string
	Multiple bool
}

type UploadEvent struct {
	Name string
	Size int64
	Type string
	Path string
}

type uploadCell struct {
	sess     *Session
	o        *reactive.Owner
	slot     *scriptSlot
	progress *reactive.Signal[UploadProgress]

	config UploadConfig
	grant  string

	onReady     func(name string, size int64, fileType string)
	onChange    func(UploadEvent)
	onProgress  func(loaded, total int64)
	onError     func(error string)
	onCancelled func()
	onComplete  func(upload.FileInfo) error

	warnedAccept string
}

type UploadHandle struct {
	script ScriptHandle
	cell   *uploadCell
}

func (h *UploadHandle) mutable() bool {
	return h != nil && h.cell != nil && h.cell.sess.graph.Mode() != reactive.ModeSerialize
}

func (h *UploadHandle) Accept(cfg UploadConfig) {
	if !h.mutable() {
		return
	}
	h.cell.config = cfg
	h.cell.warnUnverifiableAccept(cfg.Accept)
}

func (h *UploadHandle) OnReady(fn func(name string, size int64, fileType string)) {
	if !h.mutable() {
		return
	}
	h.cell.onReady = fn
}

func (h *UploadHandle) OnChange(fn func(UploadEvent)) {
	if !h.mutable() {
		return
	}
	h.cell.onChange = fn
}

func (h *UploadHandle) OnProgress(fn func(loaded, total int64)) {
	if !h.mutable() {
		return
	}
	h.cell.onProgress = fn
}

func (h *UploadHandle) Progress() ReadSignal[UploadProgress] {
	if h == nil || h.cell == nil {
		return Const(UploadProgress{})
	}
	return h.cell.progress
}

func (h *UploadHandle) OnError(fn func(error string)) {
	if !h.mutable() {
		return
	}
	h.cell.onError = fn
}

func (h *UploadHandle) OnCancelled(fn func()) {
	if !h.mutable() {
		return
	}
	h.cell.onCancelled = fn
}

func (h *UploadHandle) Cancel() {
	if h == nil {
		return
	}
	h.script.Send("cancel", struct{}{})
}

func (h *UploadHandle) OnComplete(fn func(info upload.FileInfo) error) {
	if !h.mutable() {
		return
	}
	h.cell.onComplete = fn
}

func UseUpload(ctx *Ctx) *UploadHandle {
	cell := reactive.Cell(ctx.o, func() *uploadCell {
		c := &uploadCell{sess: ctx.s, o: ctx.o, progress: reactive.NewSignal(ctx.s.graph, UploadProgress{})}
		c.slot = ctx.s.newScriptSlot(ctx.o, uploadScript)
		ctx.o.OnCleanup(c.revokeGrant)
		c.slot.setEventHandler("ready", c.handleReady)
		c.slot.setEventHandler("change", c.handleChange)
		c.slot.setEventHandler("progress", c.handleProgress)
		c.slot.setEventHandler("error", c.handleError)
		c.slot.setEventHandler("cancelled", c.handleCancelled)
		return c
	})
	return &UploadHandle{script: ScriptHandle{slot: cell.slot}, cell: cell}
}

func (c *uploadCell) warnUnverifiableAccept(accept []string) {
	unknown := upload.UnknownExtensions(accept)
	key := strings.Join(unknown, ",")
	if key == c.warnedAccept {
		return
	}
	c.warnedAccept = key
	if len(unknown) == 0 || c.sess == nil {
		return
	}
	c.sess.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "upload:accept",
		Message: "accept extensions without a known content type cannot be verified and every upload will be refused: " + key,
		Meta: map[string]any{
			"extensions": unknown,
		},
	})
}

func (c *uploadCell) revokeGrant() {
	if c.sess != nil && c.sess.UploadRegistry != nil {
		c.sess.UploadRegistry.Revoke(c.grant)
	}
	c.grant = ""
}

func (c *uploadCell) handleReady(data interface{}) {
	m, _ := data.(map[string]interface{})
	name, _ := m["name"].(string)
	size, _ := m["size"].(float64)
	fileType, _ := m["type"].(string)
	count, _ := m["count"].(float64)

	c.sendStart(int(count))

	if c.onReady != nil {
		c.onReady(name, int64(size), fileType)
	}
}

func (c *uploadCell) handleChange(data interface{}) {
	if c.onChange == nil {
		return
	}
	m, ok := data.(map[string]interface{})
	if !ok {
		return
	}
	name, _ := m["name"].(string)
	size, _ := m["size"].(float64)
	fileType, _ := m["type"].(string)
	path, _ := m["path"].(string)
	c.onChange(UploadEvent{Name: name, Size: int64(size), Type: fileType, Path: path})
}

func (c *uploadCell) handleProgress(data interface{}) {
	m, ok := data.(map[string]interface{})
	if !ok {
		return
	}
	loaded, _ := m["loaded"].(float64)
	total, _ := m["total"].(float64)
	c.progress.Set(UploadProgress{Loaded: int64(loaded), Total: int64(total)})
	if c.onProgress != nil {
		c.onProgress(int64(loaded), int64(total))
	}
}

func (c *uploadCell) handleError(data interface{}) {
	if c.onError == nil {
		return
	}
	m, ok := data.(map[string]interface{})
	if !ok {
		return
	}
	message, _ := m["error"].(string)
	c.onError(message)
}

func (c *uploadCell) handleCancelled(interface{}) {
	if c.onCancelled != nil {
		c.onCancelled()
	}
}

func (c *uploadCell) sendStart(requested int) {
	if c == nil || c.slot == nil {
		return
	}
	sess := c.sess
	if sess == nil || sess.UploadRegistry == nil {
		if sess != nil {
			sess.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "upload:grant",
				Message: "upload registry is not configured; configure the app with an upload config",
			})
		}
		return
	}

	files := requested
	if !c.config.Multiple {
		files = 1
	}
	files = upload.ClampFiles(files)

	sess.UploadRegistry.Revoke(c.grant)
	c.grant = sess.UploadRegistry.Issue(sess.SessionID, upload.GrantSpec{
		Files:      files,
		MaxSize:    c.config.MaxSize,
		Accept:     c.config.Accept,
		OnComplete: c.completion(),
	})
	if c.grant == "" {
		sess.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "upload:grant",
			Message: "failed to issue an upload grant",
		})
		return
	}

	payload := map[string]interface{}{
		"token":    c.grant,
		"multiple": c.config.Multiple,
		"files":    files,
	}
	if sess.UploadEndpoint != "" {
		payload["endpoint"] = sess.UploadEndpoint
	}
	if sess.UploadScriptURL != "" {
		payload["tusScript"] = sess.UploadScriptURL
	}
	if c.config.MaxSize > 0 {
		payload["maxSize"] = c.config.MaxSize
	}
	if len(c.config.Accept) > 0 {
		payload["accept"] = c.config.Accept
		if unknown := upload.UnknownExtensions(c.config.Accept); len(unknown) > 0 {
			payload["unverifiable"] = unknown
		}
	}

	c.slot.send("start", payload)
}

func (c *uploadCell) completion() func(upload.FileInfo) error {
	sess := c.sess
	owner := c.o
	return func(info upload.FileInfo) error {
		id := sess.completions.begin(info)
		err := sess.Post(func() {
			sess.completions.finish(id)
			fn := c.onComplete
			if fn == nil {
				return
			}
			var cbErr error
			sess.invokeHandler(owner, func() { cbErr = fn(info) })
			if cbErr != nil {
				sess.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "upload:complete",
					Message: cbErr.Error(),
					Meta: map[string]any{
						"upload_id": info.ID,
					},
				})
			}
		})
		if err != nil && sess.completions.abort(id) {
			return nil
		}
		return err
	}
}

type pendingCompletion struct {
	info     upload.FileInfo
	reported bool
}

type completionTracker struct {
	mu      sync.Mutex
	next    uint64
	pending map[uint64]*pendingCompletion
}

func (t *completionTracker) begin(info upload.FileInfo) uint64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.next++
	if t.pending == nil {
		t.pending = make(map[uint64]*pendingCompletion)
	}
	t.pending[t.next] = &pendingCompletion{info: info}
	return t.next
}

func (t *completionTracker) finish(id uint64) {
	t.mu.Lock()
	delete(t.pending, id)
	t.mu.Unlock()
}

func (t *completionTracker) abort(id uint64) (keep bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	entry := t.pending[id]
	delete(t.pending, id)
	return entry != nil && entry.reported
}

func (t *completionTracker) takeUnfinished() []upload.FileInfo {
	t.mu.Lock()
	defer t.mu.Unlock()
	infos := make([]upload.FileInfo, 0, len(t.pending))
	for _, entry := range t.pending {
		entry.reported = true
		infos = append(infos, entry.info)
	}
	return infos
}

func (s *Session) reportOrphanedCompletions() {
	for _, info := range s.completions.takeUnfinished() {
		s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "upload:orphaned",
			Message: "upload completed but the session closed before OnComplete ran; the file was kept",
			Meta: map[string]any{
				"upload_id": info.ID,
				"path":      info.Storage["Path"],
			},
		})
	}
}
