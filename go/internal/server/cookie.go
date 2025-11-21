package server

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/eleven-am/pondlive/go/internal/session"
)

// CookiePath is the fixed endpoint the client uses to negotiate HttpOnly cookie updates.
const CookiePath = "/pondlive/cookie"

// CookieHandler delivers pending HttpOnly cookie mutations to clients on behalf of a session.
type CookieHandler struct {
	registry *SessionRegistry
}

// NewCookieHandler constructs a cookie negotiation handler bound to the provided session registry.
func NewCookieHandler(reg *SessionRegistry) *CookieHandler {
	return &CookieHandler{registry: reg}
}

type cookiePayload struct {
	SID   string `json:"sid"`
	Token string `json:"token"`
}

// ServeHTTP accepts POST requests that acknowledge a pending cookie batch.
func (h *CookieHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h == nil || h.registry == nil {
		http.Error(w, "server: cookie handler not available", http.StatusServiceUnavailable)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "server: unsupported method", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()

	var payload cookiePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "server: invalid cookie payload", http.StatusBadRequest)
		return
	}

	sid := strings.TrimSpace(payload.SID)
	token := strings.TrimSpace(payload.Token)
	if sid == "" || token == "" {
		http.Error(w, "server: invalid cookie payload", http.StatusBadRequest)
		return
	}

	sess, ok := h.registry.Lookup(session.SessionID(sid))
	if !ok || sess == nil {
		http.Error(w, "server: session not found", http.StatusNotFound)
		return
	}

	batch, ok := sess.ConsumeCookieBatch(token)
	if !ok {
		http.Error(w, "server: cookie batch not found", http.StatusNotFound)
		return
	}

	for _, cookie := range batch.Set {
		if cookie == nil {
			continue
		}
		cookie.HttpOnly = true
		if strings.TrimSpace(cookie.Path) == "" {
			cookie.Path = "/"
		}
		http.SetCookie(w, cookie)
	}

	if len(batch.Delete) > 0 {
		expires := time.Unix(0, 0)
		for _, name := range batch.Delete {
			trimmed := strings.TrimSpace(name)
			if trimmed == "" {
				continue
			}
			http.SetCookie(w, &http.Cookie{
				Name:     trimmed,
				Path:     "/",
				HttpOnly: true,
				Expires:  expires,
				MaxAge:   -1,
			})
		}
	}

	w.WriteHeader(http.StatusNoContent)
}
