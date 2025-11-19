package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
	"github.com/eleven-am/pondlive/go/internal/server/pondsocket"
	"github.com/eleven-am/pondlive/go/internal/session"

	pond "github.com/eleven-am/pondsocket/go/pondsocket"
)

// Manager handles HTTP requests and session lifecycle for runtime2.
type Manager[P any] struct {
	registry       *server.SessionRegistry
	component      runtime.Component[P]
	version        int
	idGenerator    func(*http.Request) (session.SessionID, error)
	sessionConfig  *session.Config
	clientAsset    string
	clientConfig   *protocol.ClientConfig
	pubsubProvider runtime.PubsubProvider
}

// Config captures configuration for the HTTP manager.
type Config[P any] struct {
	Registry     *server.SessionRegistry
	Version      int
	IDGenerator  func(*http.Request) (session.SessionID, error)
	Session      *session.Config
	Component    runtime.Component[P]
	ClientAsset  string
	ClientConfig *protocol.ClientConfig
}

const (
	DefaultClientAsset = "/pondlive.js"
	PondSocketPattern  = "/live"
)

// NewManager constructs an HTTP manager for server-side rendering.
func NewManager[P any](cfg *Config[P]) *Manager[P] {
	m := &Manager[P]{
		registry:    server.NewSessionRegistry(),
		version:     1,
		idGenerator: defaultSessionID,
		clientAsset: DefaultClientAsset,
	}
	if cfg == nil {
		return m
	}
	if cfg.Registry != nil {
		m.registry = cfg.Registry
	}
	if cfg.Version > 0 {
		m.version = cfg.Version
	}
	if cfg.IDGenerator != nil {
		m.idGenerator = cfg.IDGenerator
	}
	if cfg.Session != nil {
		clone := *cfg.Session
		m.sessionConfig = &clone
	}
	if cfg.Component != nil {
		m.component = cfg.Component
	}
	if asset := strings.TrimSpace(cfg.ClientAsset); asset != "" {
		m.clientAsset = asset
	}
	if cfg.ClientConfig != nil {
		clone := cloneClientConfig(*cfg.ClientConfig)
		m.clientConfig = &clone
	}
	return m
}

// Registry exposes the session registry.
func (m *Manager[P]) Registry() *server.SessionRegistry {
	if m == nil {
		return nil
	}
	return m.registry
}

// SetClientConfig updates the default client boot configuration for new sessions.
func (m *Manager[P]) SetClientConfig(cfg protocol.ClientConfig) {
	if m == nil {
		return
	}
	clone := cloneClientConfig(cfg)
	m.clientConfig = &clone
}

// RegisterPondSocket wires the manager's registry to the PondSocket endpoint.
func (m *Manager[P]) RegisterPondSocket(srv *pond.Manager) (*pondsocket.Endpoint, error) {
	if m == nil {
		return nil, errors.New("server: manager is nil")
	}
	endpoint, err := pondsocket.Register(srv, PondSocketPattern, m.registry)
	if err != nil {
		return nil, err
	}
	m.pubsubProvider = endpoint.PubsubProvider()
	return endpoint, nil
}

// ServeHTTP satisfies http.Handler using zero-value props.
func (m *Manager[P]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var zero P
	m.ServeHTTPWithProps(w, r, zero)
}

// ServeHTTPWithProps renders the component tree, registers the session, and writes the SSR response.
func (m *Manager[P]) ServeHTTPWithProps(w http.ResponseWriter, r *http.Request, props P) {
	if m == nil {
		http.Error(w, "server: no component configured", http.StatusServiceUnavailable)
		return
	}
	if r == nil || r.URL == nil {
		http.Error(w, "server: invalid request", http.StatusBadRequest)
		return
	}

	component := m.component
	if component == nil {
		http.Error(w, "server: no component configured", http.StatusServiceUnavailable)
		return
	}

	sid, err := m.idGenerator(r)
	if err != nil || sid == "" {
		http.Error(w, "server: failed to allocate session", http.StatusInternalServerError)
		return
	}

	version := m.version
	if version <= 0 {
		version = 1
	}

	cfg := cloneSessionConfig(m.sessionConfig)
	capture := newBootCaptureTransport()
	cfg.Transport = capture

	sess := session.NewLiveSession(sid, version, component, props, &cfg)
	if m.pubsubProvider != nil {
		if adapter := pondsocket.WrapSessionPubsubProvider(sess, m.pubsubProvider); adapter != nil {
			if cs := sess.ComponentSession(); cs != nil {
				cs.SetPubsubProvider(adapter)
			}
		}
	}
	sess.MergeRequest(r)

	if err := sess.Flush(); err != nil {
		http.Error(w, "server: initial render failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	root := sess.Tree()
	if root == nil {
		http.Error(w, "server: render produced nil node", http.StatusInternalServerError)
		return
	}

	documentHTML := root.ToHTML()
	bodyHTML := extractBodyContent(documentHTML)

	initial := sess.InitialLocation()
	location := protocol.Location{
		Path:  initial.Path,
		Query: encodeValues(initial.Query),
		Hash:  initial.Hash,
	}
	if location.Path == "" {
		location.Path = normalizePathValue(r.URL.Path)
	}
	if location.Query == "" {
		location.Query = r.URL.RawQuery
	}
	if location.Hash == "" {
		location.Hash = strings.TrimSpace(r.URL.Fragment)
	}

	clientCfg := cloneOptionalClientConfig(m.clientConfig)
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
		HTML:     bodyHTML,
		Location: location,
		Client:   clientCfg,
	}
	bootJSON, err := json.Marshal(boot)
	if err != nil {
		http.Error(w, "server: failed to encode boot payload", http.StatusInternalServerError)
		return
	}

	sess.SetTransport(nil)
	m.registry.Put(sess)

	document := decorateDocument(documentHTML, bootJSON, m.clientAsset)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(document))
}

func escapeJSON(raw string) string {
	if raw == "" {
		return raw
	}

	return strings.ReplaceAll(raw, "</", "<\\/")
}

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

func cloneClientConfig(cfg protocol.ClientConfig) protocol.ClientConfig {
	out := cfg
	if cfg.Debug != nil {
		value := *cfg.Debug
		out.Debug = &value
	}
	return out
}

func cloneOptionalClientConfig(cfg *protocol.ClientConfig) *protocol.ClientConfig {
	if cfg == nil {
		return nil
	}
	clone := cloneClientConfig(*cfg)
	return &clone
}

func decorateDocument(document string, bootJSON []byte, assetURL string) string {
	if strings.Contains(strings.ToLower(document), "<html") {
		return injectScripts(document, bootJSON, assetURL)
	}
	return buildFallbackDocument(document, bootJSON, assetURL)
}

func buildFallbackDocument(body string, bootJSON []byte, assetURL string) string {
	escaped := escapeJSON(string(bootJSON))
	var b strings.Builder
	b.Grow(len(body) + len(escaped) + 256)
	b.WriteString("<!DOCTYPE html><html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\"></head><body>")
	b.WriteString(body)
	b.WriteString(`<script id="live-boot" type="application/json">`)
	b.WriteString(escaped)
	b.WriteString("</script>")
	if assetURL != "" {
		b.WriteString(`<script src="`)
		b.WriteString(assetURL)
		b.WriteString(`" defer></script>`)
	}
	b.WriteString("</body></html>")
	return b.String()
}

func injectScripts(document string, bootJSON []byte, assetURL string) string {
	escaped := escapeJSON(string(bootJSON))
	bootScript := `<script id="live-boot" type="application/json">` + escaped + `</script>`
	clientScript := ""
	if strings.TrimSpace(assetURL) != "" {
		clientScript = `<script src="` + assetURL + `" defer></script>`
	}
	insert := bootScript + clientScript
	idx := lastIndexFold(document, "</body>")
	if idx < 0 {
		return document + insert
	}
	var b strings.Builder
	b.Grow(len(document) + len(insert))
	b.WriteString(document[:idx])
	b.WriteString(insert)
	b.WriteString(document[idx:])
	return b.String()
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

func extractBodyContent(document string) string {
	start := indexFold(document, "<body")
	if start < 0 {
		return document
	}
	open := strings.Index(document[start:], ">")
	if open < 0 {
		return document
	}
	bodyStart := start + open + 1
	end := lastIndexFold(document, "</body>")
	if end < 0 || end <= bodyStart {
		return document[bodyStart:]
	}
	return document[bodyStart:end]
}

func indexFold(haystack, needle string) int {
	needleLen := len(needle)
	if needleLen == 0 || len(haystack) < needleLen {
		return -1
	}
	for i := 0; i <= len(haystack)-needleLen; i++ {
		if strings.EqualFold(haystack[i:i+needleLen], needle) {
			return i
		}
	}
	return -1
}

func encodeValues(values url.Values) string {
	if values == nil {
		return ""
	}
	return values.Encode()
}

func normalizePathValue(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		return "/" + trimmed
	}
	return trimmed
}

type bootCaptureTransport struct {
	lastSeq int
}

func newBootCaptureTransport() *bootCaptureTransport {
	return &bootCaptureTransport{}
}

func (b *bootCaptureTransport) LastSeq() int {
	return b.lastSeq
}

func (b *bootCaptureTransport) SendBoot(protocol.Boot) error {
	return nil
}

func (b *bootCaptureTransport) SendInit(protocol.Init) error {
	return nil
}

func (b *bootCaptureTransport) SendResume(protocol.ResumeOK) error {
	return nil
}

func (b *bootCaptureTransport) SendFrame(frame protocol.Frame) error {
	b.lastSeq = frame.Seq
	return nil
}

func (b *bootCaptureTransport) SendEventAck(protocol.EventAck) error {
	return nil
}

func (b *bootCaptureTransport) SendServerError(protocol.ServerError) error {
	return nil
}

func (b *bootCaptureTransport) SendDiagnostic(protocol.Diagnostic) error {
	return nil
}

func (b *bootCaptureTransport) SendDOMRequest(protocol.DOMRequest) error {
	return nil
}

func (b *bootCaptureTransport) SendPubsubControl(protocol.PubsubControl) error {
	return nil
}

func (b *bootCaptureTransport) SendUploadControl(protocol.UploadControl) error {
	return nil
}

func (b *bootCaptureTransport) Close() error {
	return nil
}
