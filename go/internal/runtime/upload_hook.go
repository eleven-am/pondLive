package runtime

import (
	_ "embed"
	"io"
	"mime/multipart"
	"net/http"
	"os"

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
	token          string
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
		token:          "",
	}

	handler := UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {
		token := r.Header.Get("X-Upload-Token")
		if token == "" || token != handle.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return nil
		}

		if handle.config != nil && handle.config.MaxSize > 0 {
			r.Body = http.MaxBytesReader(w, r.Body, handle.config.MaxSize)
		}

		reader, err := r.MultipartReader()
		if err != nil {
			http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
			return err
		}

		var processed bool
		for {
			part, errPart := reader.NextPart()
			if errPart != nil {
				if errPart == io.EOF {
					break
				}
				http.Error(w, "Failed to read multipart", http.StatusBadRequest)
				return errPart
			}
			if part.FormName() != "file" {
				_ = part.Close()
				continue
			}
			header := &multipart.FileHeader{
				Filename: part.FileName(),
				Header:   part.Header,
				Size:     0,
			}

			tmp, tmpErr := os.CreateTemp("", "pond-upload-*")
			if tmpErr != nil {
				_ = part.Close()
				http.Error(w, "Failed to stage file", http.StatusInternalServerError)
				return tmpErr
			}
			tmpPath := tmp.Name()

			defer func() {
				_ = tmp.Close()
				_ = os.Remove(tmpPath)
			}()

			var written int64
			if handle.config != nil && handle.config.MaxSize > 0 {
				written, err = io.Copy(tmp, io.LimitReader(part, handle.config.MaxSize+1))
				if err != nil && err != io.EOF {
					_ = part.Close()
					http.Error(w, "Failed to read file", http.StatusBadRequest)
					return err
				}
				if written > handle.config.MaxSize {
					_ = part.Close()
					http.Error(w, "File exceeds maximum size", http.StatusRequestEntityTooLarge)
					return nil
				}
			} else {
				written, err = io.Copy(tmp, part)
				if err != nil && err != io.EOF {
					_ = part.Close()
					http.Error(w, "Failed to read file", http.StatusBadRequest)
					return err
				}
			}

			header.Size = written

			if _, seekErr := tmp.Seek(0, io.SeekStart); seekErr != nil {
				_ = part.Close()
				http.Error(w, "Failed to rewind file", http.StatusInternalServerError)
				return seekErr
			}

			var fileReader multipart.File = tmp

			if handle.onComplete != nil {
				if err := handle.onComplete(fileReader, header); err != nil {
					_ = part.Close()
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return err
				}
			}
			processed = true
			_ = part.Close()
		}

		if !processed {
			http.Error(w, "No file provided", http.StatusBadRequest)
			return nil
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

	script.On("ready", func(data interface{}) {
		handle.sendStart()
	})

	return *handle
}

func (h *UploadHandle) sendStart() {
	if h.script == nil || h.handler == nil {
		return
	}

	h.token = h.handler.GenerateToken()
	url := h.handler.URL()

	if url == "" {
		return
	}

	payload := map[string]interface{}{
		"url": url,
	}
	if h.config != nil {
		if h.config.MaxSize > 0 {
			payload["maxSize"] = h.config.MaxSize
		}
		if len(h.config.Accept) > 0 {
			payload["accept"] = h.config.Accept
		}
		if h.config.Multiple {
			payload["multiple"] = true
		}
	}
	if h.token != "" {
		payload["token"] = h.token
	}

	h.script.Send("start", payload)
}
