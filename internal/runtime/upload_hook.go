package runtime

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"sync"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/upload"
	"github.com/eleven-am/pondlive/internal/work"
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
	token   string
	script  *scriptCell
	progRef *Ref[UploadProgress]
}

type UploadHandle struct {
	script         *ScriptHandle
	config         *UploadConfig
	progressGetter func() UploadProgress
	onComplete     func(upload.FileInfo) error
	token          string
	session        *Session
	mu             sync.Mutex
}

func (h *UploadHandle) Accept(cfg UploadConfig) {
	if h == nil || h.session == nil || h.session.UploadRegistry == nil {
		return
	}

	h.mu.Lock()
	h.config = &cfg
	h.mu.Unlock()

	cb := upload.UploadCallback{
		Token:        h.token,
		MaxSize:      cfg.MaxSize,
		AllowedTypes: cfg.Accept,
		OnComplete:   h.onComplete,
	}
	h.session.UploadRegistry.Register(cb)
}

func (h *UploadHandle) AttachTo(elem *work.Element) {
	if h == nil || h.script == nil {
		return
	}
	h.script.AttachTo(elem)
}

func (h *UploadHandle) OnReady(fn func(name string, size int64, fileType string)) {
	if h == nil || h.script == nil {
		return
	}
	h.script.On("ready", func(data interface{}) {
		m, ok := data.(map[string]interface{})
		if !ok {
			return
		}
		name, _ := m["name"].(string)
		size, _ := m["size"].(float64)
		fileType, _ := m["type"].(string)
		fn(name, int64(size), fileType)
	})
}

func (h *UploadHandle) OnChange(fn func(UploadEvent)) {
	if h == nil || h.script == nil {
		return
	}
	h.script.On("change", func(data interface{}) {
		m, ok := data.(map[string]interface{})
		if !ok {
			return
		}
		name, _ := m["name"].(string)
		size, _ := m["size"].(float64)
		fileType, _ := m["type"].(string)
		path, _ := m["path"].(string)
		fn(UploadEvent{
			Name: name,
			Size: int64(size),
			Type: fileType,
			Path: path,
		})
	})
}

func (h *UploadHandle) OnProgress(fn func(loaded, total int64)) {
	if h == nil || h.script == nil {
		return
	}
	h.script.On("progress", func(data interface{}) {
		m, ok := data.(map[string]interface{})
		if !ok {
			return
		}
		loaded, _ := m["loaded"].(float64)
		total, _ := m["total"].(float64)
		fn(int64(loaded), int64(total))
	})
}

func (h *UploadHandle) Progress() UploadProgress {
	if h == nil || h.progressGetter == nil {
		return UploadProgress{Loaded: 0, Total: 0}
	}
	return h.progressGetter()
}

func (h *UploadHandle) OnError(fn func(error string)) {
	if h == nil || h.script == nil {
		return
	}
	h.script.On("error", func(data interface{}) {
		m, ok := data.(map[string]interface{})
		if !ok {
			return
		}
		err, _ := m["error"].(string)
		fn(err)
	})
}

func (h *UploadHandle) OnCancelled(fn func()) {
	if h == nil || h.script == nil {
		return
	}
	h.script.On("cancelled", func(data interface{}) {
		fn()
	})
}

func (h *UploadHandle) Cancel() {
	if h == nil || h.script == nil {
		return
	}
	h.script.Send("cancel", struct{}{})
}

func (h *UploadHandle) OnComplete(fn func(info upload.FileInfo) error) {
	if h == nil || h.session == nil || h.session.UploadRegistry == nil {
		return
	}

	h.mu.Lock()
	h.onComplete = fn
	h.mu.Unlock()

	h.mu.Lock()
	cfg := h.config
	h.mu.Unlock()

	cb := upload.UploadCallback{
		Token:      h.token,
		OnComplete: fn,
	}
	if cfg != nil {
		cb.MaxSize = cfg.MaxSize
		cb.AllowedTypes = cfg.Accept
	}
	h.session.UploadRegistry.Register(cb)
}

func UseUpload(ctx *Ctx) *UploadHandle {
	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		cell := &uploadCell{}
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeUpload,
			Value: cell,
		})
	}

	cell, ok := ctx.instance.HookFrame[idx].Value.(*uploadCell)
	if !ok {
		panic("runtime: UseUpload hook mismatch")
	}

	if cell.token == "" {
		cell.token = generateUploadToken()
	}

	if cell.script == nil {
		cell.script = &scriptCell{}
	}

	if cell.script.slot == nil {
		cell.script.slot = ctx.session.registerScriptSlot(ctx.instance, idx, uploadScript)

		slotID := cell.script.slot.id
		ctx.instance.RegisterCleanup(func() {
			ctx.session.unregisterScriptSlot(slotID)
		})
	}

	if cell.progRef == nil {
		cell.progRef = &Ref[UploadProgress]{Current: UploadProgress{}}
	}

	handle := &UploadHandle{
		script:         &ScriptHandle{slot: cell.script.slot},
		progressGetter: func() UploadProgress { return cell.progRef.Current },
		token:          cell.token,
		session:        ctx.session,
	}

	if ctx.session != nil && ctx.session.UploadRegistry != nil {
		if _, exists := ctx.session.UploadRegistry.Lookup(cell.token); !exists {
			cb := upload.UploadCallback{
				Token: cell.token,
			}
			ctx.session.UploadRegistry.Register(cb)

			token := cell.token
			ctx.instance.RegisterCleanup(func() {
				if ctx.session != nil && ctx.session.UploadRegistry != nil {
					ctx.session.UploadRegistry.Remove(token)
				}
			})
		}
	}

	cell.script.slot.sess.Bus.Upsert(protocol.Topic("script:"+cell.script.slot.id+":progress"), func(action string, data interface{}) {
		if action != string(protocol.ScriptMessageAction) {
			return
		}
		payload, ok := protocol.DecodePayload[protocol.ScriptPayload](data)
		if !ok {
			return
		}
		m, ok := payload.Data.(map[string]interface{})
		if !ok {
			return
		}
		loaded, _ := m["loaded"].(float64)
		total, _ := m["total"].(float64)
		cell.progRef.Current = UploadProgress{Loaded: int64(loaded), Total: int64(total)}
	})

	cell.script.slot.sess.Bus.Upsert(protocol.Topic("script:"+cell.script.slot.id+":ready"), func(action string, data interface{}) {
		if action != string(protocol.ScriptMessageAction) {
			return
		}
		handle.sendStart()
	})

	return handle
}

func (h *UploadHandle) sendStart() {
	if h == nil || h.script == nil {
		return
	}

	payload := map[string]interface{}{
		"token": h.token,
	}

	h.mu.Lock()
	cfg := h.config
	h.mu.Unlock()

	if cfg != nil {
		if cfg.MaxSize > 0 {
			payload["maxSize"] = cfg.MaxSize
		}
		if len(cfg.Accept) > 0 {
			payload["accept"] = cfg.Accept
		}
		if cfg.Multiple {
			payload["multiple"] = true
		}
	}

	h.script.Send("start", payload)
}

func generateUploadToken() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return ""
	}
	return base64.RawURLEncoding.EncodeToString(buf[:])
}
