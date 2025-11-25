package server

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/session"
	"github.com/eleven-am/pondlive/go/internal/view"
	"github.com/eleven-am/pondlive/go/internal/view/diff"
)

// SSRHandler handles server-side rendering of components.
type SSRHandler struct {
	registry      *SessionRegistry
	component     session.Component
	version       int
	idGenerator   func(*http.Request) (session.SessionID, error)
	sessionConfig *session.Config
	clientAsset   string
	clientConfig  *protocol.ClientConfig
}

// SSRConfig configures the SSR handler.
type SSRConfig struct {
	Registry      *SessionRegistry
	Component     session.Component
	IDGenerator   func(*http.Request) (session.SessionID, error)
	SessionConfig *session.Config
	ClientAsset   string
	ClientConfig  *protocol.ClientConfig
}

// NewSSRHandler creates a new SSR handler.
func NewSSRHandler(cfg SSRConfig) *SSRHandler {
	h := &SSRHandler{
		registry:    NewSessionRegistry(),
		version:     1,
		idGenerator: defaultSessionID,
		clientAsset: "/pondlive.js",
	}
	if cfg.Registry != nil {
		h.registry = cfg.Registry
	}
	if cfg.Component != nil {
		h.component = cfg.Component
	}
	if cfg.IDGenerator != nil {
		h.idGenerator = cfg.IDGenerator
	}
	if cfg.SessionConfig != nil {
		clone := *cfg.SessionConfig
		h.sessionConfig = &clone
	}
	if strings.TrimSpace(cfg.ClientAsset) != "" {
		h.clientAsset = cfg.ClientAsset
	}
	if cfg.ClientConfig != nil {
		clone := *cfg.ClientConfig
		h.clientConfig = &clone
	}
	return h
}

// Registry returns the session registry.
func (h *SSRHandler) Registry() *SessionRegistry {
	return h.registry
}

// ServeHTTP handles SSR rendering.
func (h *SSRHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.component == nil {
		http.Error(w, "no component configured", http.StatusServiceUnavailable)
		return
	}

	sid, err := h.idGenerator(r)
	if err != nil || sid == "" {
		http.Error(w, "failed to allocate session", http.StatusInternalServerError)
		return
	}

	version := h.version
	if version <= 0 {
		version = 1
	}

	cfg := cloneSessionConfig(h.sessionConfig)
	cfg.ClientAsset = h.clientAsset

	requestInfo := headers.NewRequestInfo(r)

	sess := session.NewLiveSession(sid, version, h.component, &cfg)

	capture := newBootCaptureTransport()
	capture.requestInfo = requestInfo
	sess.SetTransport(capture)

	if err := sess.Flush(); err != nil {
		http.Error(w, "initial render failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rtSession := sess.Session()
	if rtSession == nil || rtSession.View == nil {
		http.Error(w, "render produced nil view", http.StatusInternalServerError)
		return
	}

	documentHTML := view.RenderHTML(rtSession.View)

	location := protocol.Location{
		Path:  normalizePath(r.URL.Path),
		Query: r.URL.RawQuery,
		Hash:  strings.TrimSpace(r.URL.Fragment),
	}

	clientCfg := cloneOptionalClientConfig(h.clientConfig)
	if cfg.DevMode {
		if clientCfg == nil {
			clientCfg = &protocol.ClientConfig{}
		}
		if clientCfg.Debug == nil {
			value := true
			clientCfg.Debug = &value
		}
	}

	boot := protocol.Boot{
		T:        "boot",
		SID:      string(sid),
		Ver:      version,
		Seq:      capture.LastSeq(),
		Patch:    diff.ExtractMetadata(rtSession.View),
		Location: location,
		Client:   clientCfg,
	}

	bootJSON, err := json.Marshal(boot)
	if err != nil {
		http.Error(w, "failed to encode boot payload", http.StatusInternalServerError)
		return
	}

	sess.SetTransport(nil)
	h.registry.Put(sess)

	document := decorateDocument(documentHTML, bootJSON, h.clientAsset)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(document))
}

// Helper functions

func defaultSessionID(*http.Request) (session.SessionID, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(buf[:])
	return session.SessionID(id), nil
}

func cloneSessionConfig(cfg *session.Config) session.Config {
	if cfg == nil {
		return session.Config{}
	}
	return *cfg
}

func cloneOptionalClientConfig(cfg *protocol.ClientConfig) *protocol.ClientConfig {
	if cfg == nil {
		return nil
	}
	clone := *cfg
	if cfg.Debug != nil {
		value := *cfg.Debug
		clone.Debug = &value
	}
	return &clone
}

func decorateDocument(document string, bootJSON []byte, clientAsset string) string {
	escaped := escapeJSON(string(bootJSON))
	bootScript := `<script id="live-boot" type="application/json">` + escaped + `</script>`

	clientScript := ""
	if clientAsset != "" {
		clientScript = `<script src="` + clientAsset + `" defer></script>`
	}

	idx := lastIndexFold(document, "</body>")
	if idx < 0 {
		return document + bootScript + clientScript
	}

	var b strings.Builder
	b.Grow(len(document) + len(bootScript) + len(clientScript))
	b.WriteString(document[:idx])
	b.WriteString(bootScript)
	b.WriteString(clientScript)
	b.WriteString(document[idx:])
	return b.String()
}

func escapeJSON(raw string) string {
	return strings.ReplaceAll(raw, "</", "<\\/")
}

func lastIndexFold(haystack, needle string) int {
	needleLen := len(needle)
	if needleLen == 0 || len(haystack) < needleLen {
		return -1
	}
	for i := len(haystack) - needleLen; i >= 0; i-- {
		if strings.EqualFold(haystack[i:i+needleLen], needle) {
			return i
		}
	}
	return -1
}

func normalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}

// bootCaptureTransport captures the last sequence number during initial render.
type bootCaptureTransport struct {
	lastSeq     int
	requestInfo *headers.RequestInfo
}

func newBootCaptureTransport() *bootCaptureTransport {
	return &bootCaptureTransport{}
}

func (b *bootCaptureTransport) LastSeq() int {
	return b.lastSeq
}

func (b *bootCaptureTransport) IsLive() bool { return false }

func (b *bootCaptureTransport) RequestInfo() *headers.RequestInfo { return b.requestInfo }

func (b *bootCaptureTransport) Close() error { return nil }

func (b *bootCaptureTransport) Send(topic, event string, data any) error {

	if topic == "frame" && event == "patch" {
		b.lastSeq++
	}
	return nil
}
