package runtime

import (
	_ "embed"
	"mime/multipart"
	"net/http"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

//go:embed upload.js
var uploadScript string

type UploadProgress struct {
	Loaded int64
	Total  int64
}

type UploadHandle struct {
	script         *ScriptHandle
	handler        *HandlerHandle
	config         *UploadConfig
	progressGetter func() UploadProgress
	onComplete     func(file multipart.File, header *multipart.FileHeader) error
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

func (h *UploadHandle) Accept(cfg UploadConfig) {
	if h.script == nil {
		return
	}

	h.config = &cfg
}

func (h *UploadHandle) AttachTo(node *dom.StructuredNode) {
	if h.script != nil {
		h.script.AttachTo(node)
	}
}

func (h *UploadHandle) OnReady(fn func(name string, size int64, fileType string)) {
	if h.script != nil {
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
}

func (h *UploadHandle) OnChange(fn func(UploadEvent)) {
	if h.script != nil {
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
}

func (h *UploadHandle) OnProgress(fn func(loaded, total int64)) {
	if h.script != nil {
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
}

func (h *UploadHandle) Progress() UploadProgress {
	if h.progressGetter != nil {
		return h.progressGetter()
	}
	return UploadProgress{Loaded: 0, Total: 0}
}

func (h *UploadHandle) OnError(fn func(error string)) {
	if h.script != nil {
		h.script.On("error", func(data interface{}) {
			m, ok := data.(map[string]interface{})
			if !ok {
				return
			}
			err, _ := m["error"].(string)
			fn(err)
		})
	}
}

func (h *UploadHandle) OnCancelled(fn func()) {
	if h.script != nil {
		h.script.On("cancelled", func(data interface{}) {
			fn()
		})
	}
}

func (h *UploadHandle) Cancel() {
	if h.script != nil {
		h.script.Send("cancel", struct{}{})
	}
}

func (h *UploadHandle) OnComplete(fn func(file multipart.File, header *multipart.FileHeader) error) {
	h.onComplete = fn
}

func UseUpload(ctx Ctx) UploadHandle {
	state, setState := UseState(ctx, UploadProgress{Loaded: 0, Total: 0})
	script := UseScript(ctx, uploadScript)

	handle := &UploadHandle{
		script:         &script,
		progressGetter: state,
	}

	handler := UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {
		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
			return err
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "No file provided", http.StatusBadRequest)
			return err
		}
		defer file.Close()

		if handle.onComplete != nil {
			if err := handle.onComplete(file, header); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return err
			}
		}

		w.WriteHeader(http.StatusOK)
		return nil
	})

	handle.handler = &handler

	script.On("progress", func(data interface{}) {
		m, ok := data.(map[string]interface{})
		if !ok {
			return
		}

		loaded, _ := m["loaded"].(float64)
		total, _ := m["total"].(float64)
		setState(UploadProgress{
			Loaded: int64(loaded),
			Total:  int64(total),
		})
	})

	return *handle
}
