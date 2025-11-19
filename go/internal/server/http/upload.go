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
	"github.com/eleven-am/pondlive/go/internal/session"
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
		http.Error(w, "server2: upload handler not available", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "server2: unsupported method", http.StatusMethodNotAllowed)
		return
	}

	sid, uploadID := extractUploadTarget(r.URL.Path)
	if sid == "" || uploadID == "" {
		http.Error(w, "server2: invalid upload target", http.StatusBadRequest)
		return
	}

	sess, ok := h.registry.Lookup(session.SessionID(sid))
	if !ok || sess == nil {
		http.Error(w, "server2: session not found", http.StatusNotFound)
		return
	}

	component := sess.ComponentSession()
	if component == nil {
		http.Error(w, "server2: session unavailable", http.StatusGone)
		return
	}

	limit, _ := component.UploadMaxSize(uploadID)

	reader, err := r.MultipartReader()
	if err != nil {
		http.Error(w, "server2: expected multipart form", http.StatusBadRequest)
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
		_ = sess.Flush()
		status := http.StatusInternalServerError
		switch {
		case errors.Is(stageErr, runtime.ErrUploadTooLarge):
			status = http.StatusRequestEntityTooLarge
		case errors.Is(stageErr, io.EOF):
			status = http.StatusBadRequest
		}
		cleanupUploadedFile(staged)
		http.Error(w, "server2: failed to process upload", status)
		return
	}

	if staged.Reader == nil {
		component.HandleUploadError(uploadID, errors.New("runtime2: missing upload payload"))
		_ = sess.Flush()
		cleanupUploadedFile(staged)
		http.Error(w, "server2: missing file", http.StatusBadRequest)
		return
	}

	component.HandleUploadComplete(uploadID, staged.FileMeta)

	if err := sess.Flush(); err != nil {
		cleanupUploadedFile(staged)
		http.Error(w, "server2: failed to flush upload", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Status string              `json:"status"`
		File   protocol.UploadMeta `json:"file"`
	}{
		Status: "ok",
		File: protocol.UploadMeta{
			Name: staged.FileMeta.Name,
			Size: staged.FileMeta.Size,
			Type: staged.FileMeta.Type,
		},
	})
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
