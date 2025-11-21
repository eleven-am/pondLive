package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/dom/diff"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/session"
)

// NavRequest represents the POST body for navigation requests
type NavRequest struct {
	SessionID string `json:"sid"`
	Path      string `json:"path"`
	Query     string `json:"q,omitempty"`
	Hash      string `json:"hash,omitempty"`
}

// NavResponse represents the JSON response containing patches
type NavResponse struct {
	Success bool            `json:"success"`
	Patches []diff.Patch    `json:"patches"`
	Frame   *protocol.Frame `json:"frame"`
	Error   string          `json:"error,omitempty"`
}

// navTransport captures patches instead of sending them over websocket
type navTransport struct {
	mu      sync.Mutex
	frame   *protocol.Frame
	patches []diff.Patch
}

func (t *navTransport) SendBoot(boot protocol.Boot) error {

	panic("implement me")
}

func (t *navTransport) SendInit(init protocol.Init) error {

	panic("implement me")
}

func (t *navTransport) SendResume(res protocol.ResumeOK) error {

	panic("implement me")
}

func (t *navTransport) SendEventAck(ack protocol.EventAck) error {

	panic("implement me")
}

func (t *navTransport) SendServerError(err protocol.ServerError) error {

	panic("implement me")
}

func (t *navTransport) SendDiagnostic(diag protocol.Diagnostic) error {

	panic("implement me")
}

func (t *navTransport) SendDOMRequest(req protocol.DOMRequest) error {

	panic("implement me")
}

func (t *navTransport) SendPubsubControl(ctrl protocol.PubsubControl) error {

	panic("implement me")
}

func (t *navTransport) SendUploadControl(ctrl protocol.UploadControl) error {

	panic("implement me")
}

func (t *navTransport) SendFrame(frame protocol.Frame) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.frame = &frame
	if len(frame.Patch) > 0 {
		t.patches = frame.Patch
	}
	return nil
}

func (t *navTransport) Close() error {
	return nil
}

func (t *navTransport) GetFrame() *protocol.Frame {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.frame
}

func (t *navTransport) GetPatches() []diff.Patch {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.patches
}

// NewNavHandler creates an HTTP handler that processes navigation requests
// and returns patches as JSON
func NewNavHandler(registry *SessionRegistry) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req NavRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(NavResponse{
				Success: false,
				Error:   fmt.Sprintf("Invalid request body: %v", err),
			})
			return
		}

		if req.SessionID == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(NavResponse{
				Success: false,
				Error:   "Session ID is required",
			})
			return
		}

		if req.Path == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(NavResponse{
				Success: false,
				Error:   "Path is required",
			})
			return
		}

		sess, ok := registry.Lookup(session.SessionID(req.SessionID))
		if !ok || sess == nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(NavResponse{
				Success: false,
				Error:   "Session not found",
			})
			return
		}

		transport := &navTransport{}
		sess.SetTransport(transport)

		defer sess.SetTransport(nil)

		if err := sess.HandleNavigation(req.Path, req.Query, req.Hash); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(NavResponse{
				Success: false,
				Error:   fmt.Sprintf("Navigation failed: %v", err),
			})
			return
		}

		if err := sess.Flush(); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(NavResponse{
				Success: false,
				Error:   fmt.Sprintf("Failed to flush: %v", err),
			})
			return
		}

		frame := transport.GetFrame()
		patches := transport.GetPatches()

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(NavResponse{
			Success: true,
			Patches: patches,
			Frame:   frame,
		})
	})
}
