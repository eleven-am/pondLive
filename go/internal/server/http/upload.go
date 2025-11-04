package http

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
)

const UploadPathPrefix = "/pondlive/upload/"

// UploadHandler streams multipart uploads into the component session that registered the slot.
type UploadHandler struct {
	registry *server.SessionRegistry
}

// NewUploadHandler constructs an upload handler bound to the provided session registry.
func NewUploadHandler(reg *server.SessionRegistry) *UploadHandler {
	return &UploadHandler{registry: reg}
}

// ServeHTTP accepts POST uploads addressed to /pondlive/upload/{sid}/{uploadId}.
func (h *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.registry == nil {
		http.Error(w, "live: upload handler not available", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "live: unsupported method", http.StatusMethodNotAllowed)
		return
	}
	sid, uploadID := extractUploadTarget(r.URL.Path)
	if sid == "" || uploadID == "" {
		http.Error(w, "live: invalid upload target", http.StatusBadRequest)
		return
	}

	session, ok := h.registry.Lookup(runtime.SessionID(sid))
	if !ok || session == nil {
		http.Error(w, "live: session not found", http.StatusNotFound)
		return
	}

	component := session.ComponentSession()
	if component == nil {
		http.Error(w, "live: session unavailable", http.StatusGone)
		return
	}

	limit, _ := component.UploadMaxSize(uploadID)

	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "live: expected multipart form", http.StatusBadRequest)
		return
	}

	var (
		staged   runtime.UploadedFile
		stageErr error
	)

	for {
		part, errPart := reader.NextPart()
		if errPart == io.EOF {
			break
		}
		if errPart != nil {
			stageErr = errPart
			break
		}
		if part.FormName() != "file" {
			_ = part.Close()
			continue
		}
		staged, stageErr = runtime.StageUploadedFile(part, part.FileName(), part.Header.Get("Content-Type"), limit)
		_ = part.Close()
		break
	}

	if stageErr != nil {
		component.HandleUploadError(uploadID, stageErr)
		_ = session.Flush()
		status := http.StatusInternalServerError
		switch {
		case errors.Is(stageErr, runtime.ErrUploadTooLarge):
			status = http.StatusRequestEntityTooLarge
		case errors.Is(stageErr, io.EOF):
			status = http.StatusBadRequest
		}
		cleanupUploadedFile(staged)
		http.Error(w, "live: failed to process upload", status)
		return
	}
	if staged.Reader == nil {
		component.HandleUploadError(uploadID, errors.New("runtime: missing upload payload"))
		_ = session.Flush()
		cleanupUploadedFile(staged)
		http.Error(w, "live: missing file", http.StatusBadRequest)
		return
	}

	if _, err := component.CompleteUpload(uploadID, staged); err != nil {
		component.HandleUploadError(uploadID, err)
		_ = session.Flush()
		cleanupUploadedFile(staged)
		http.Error(w, "live: upload handler failed", http.StatusInternalServerError)
		return
	}

	if component.Dirty() {
		if err := session.Flush(); err != nil {
			http.Error(w, "live: failed to flush upload", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Status string              `json:"status"`
		File   protocol.UploadMeta `json:"file"`
	}{Status: "ok", File: protocol.UploadMeta{Name: staged.FileMeta.Name, Size: staged.FileMeta.Size, Type: staged.FileMeta.Type}})
}

func extractUploadTarget(p string) (string, string) {
	trimmed := strings.TrimPrefix(p, UploadPathPrefix)
	if trimmed == p {
		return "", ""
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		return "", ""
	}
	sid := strings.TrimSpace(parts[0])
	uploadID := strings.TrimSpace(parts[1])
	return sid, uploadID
}

func cleanupUploadedFile(file runtime.UploadedFile) {
	if file.Reader != nil {
		_ = file.Reader.Close()
	}
	if file.TempPath != "" {
		_ = os.Remove(file.TempPath)
	}
}
