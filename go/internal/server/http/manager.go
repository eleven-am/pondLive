package http

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	pond "github.com/eleven-am/pondsocket/go/pondsocket"

	"github.com/eleven-am/go/pondlive/internal/protocol"
	"github.com/eleven-am/go/pondlive/internal/render"
	"github.com/eleven-am/go/pondlive/internal/runtime"
	"github.com/eleven-am/go/pondlive/internal/server"
	"github.com/eleven-am/go/pondlive/internal/server/pondsocket"
	"github.com/eleven-am/go/pondlive/pkg/live/router"
)

type Manager struct {
	registry    *server.SessionRegistry
	component   runtime.Component[struct{}]
	version     int
	idGenerator func(*http.Request) (runtime.SessionID, error)
	sessionCfg  runtime.LiveSessionConfig
	assetURL    string
}

type ManagerConfig struct {
	Registry       *server.SessionRegistry
	Version        int
	IDGenerator    func(*http.Request) (runtime.SessionID, error)
	Session        *runtime.LiveSessionConfig
	Component      runtime.Component[struct{}]
	ClientAssetURL string
}

const (
	defaultClientAssetURL = "/pondlive.js"
	PondSocketPattern     = "/live"
)

// NewManager constructs an HTTP manager responsible for SSR and session registration.
func NewManager(cfg *ManagerConfig) *Manager {
	manager := &Manager{
		registry:    server.NewSessionRegistry(),
		version:     1,
		idGenerator: defaultSessionID,
		assetURL:    defaultClientAssetURL,
	}
	if cfg == nil {
		return manager
	}
	if cfg.Registry != nil {
		manager.registry = cfg.Registry
	}
	if cfg.Version > 0 {
		manager.version = cfg.Version
	}
	if cfg.IDGenerator != nil {
		manager.idGenerator = cfg.IDGenerator
	}
	if cfg.Session != nil {
		manager.sessionCfg = cloneLiveSessionConfig(*cfg.Session)
	}
	if cfg.Component != nil {
		manager.component = cfg.Component
	}
	if url := strings.TrimSpace(cfg.ClientAssetURL); url != "" {
		manager.assetURL = url
	}
	return manager
}

func cloneLiveSessionConfig(in runtime.LiveSessionConfig) runtime.LiveSessionConfig {
	out := in
	if in.DevMode != nil {
		value := *in.DevMode
		out.DevMode = &value
	}
	if in.ClientConfig != nil {
		clone := cloneClientConfig(*in.ClientConfig)
		out.ClientConfig = &clone
	}
	return out
}

func cloneClientConfig(in protocol.ClientConfig) protocol.ClientConfig {
	out := in
	return out
}

// Registry exposes the session registry maintained by the manager.
func (m *Manager) Registry() *server.SessionRegistry {
	if m == nil {
		return nil
	}
	return m.registry
}

// SetClientConfig configures the base client boot payload for all sessions managed by the HTTP server.
func (m *Manager) SetClientConfig(cfg protocol.ClientConfig) {
	if m == nil {
		return
	}
	clone := cloneClientConfig(cfg)
	m.sessionCfg.ClientConfig = &clone
}

// RegisterPondSocket wires the manager's registry to the fixed PondSocket endpoint used by LiveUI.
func (m *Manager) RegisterPondSocket(srv *pond.Manager) (*pondsocket.Endpoint, error) {
	if m == nil {
		return nil, errors.New("live: manager is nil")
	}
	endpoint, err := pondsocket.Register(srv, PondSocketPattern, m.registry)
	if err != nil {
		return nil, err
	}
	if provider := endpoint.PubsubProvider(); provider != nil {
		m.sessionCfg.PubsubProvider = provider
	}
	return endpoint, nil
}

// ServeHTTP renders the matching route, registers the session, and writes the SSR response.
func (m *Manager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m == nil {
		http.Error(w, "live: no component configured", http.StatusServiceUnavailable)
		return
	}
	if r == nil || r.URL == nil {
		http.Error(w, "live: invalid request", http.StatusBadRequest)
		return
	}

	path := canonicalPath(r.URL.Path)
	rawQuery := r.URL.RawQuery
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		http.Error(w, "live: invalid query string", http.StatusBadRequest)
		return
	}

	component := m.component
	if component == nil {
		http.Error(w, "live: no component configured", http.StatusServiceUnavailable)
		return
	}

	sid, err := m.idGenerator(r)
	if err != nil || sid == "" {
		http.Error(w, "live: failed to allocate session", http.StatusInternalServerError)
		return
	}

	version := m.version
	if version <= 0 {
		version = 1
	}

	sessionCfgLocal := cloneLiveSessionConfig(m.sessionCfg)
	session := runtime.NewLiveSession(sid, version, component, struct{}{}, &sessionCfgLocal)
	session.SetRoute(path, rawQuery, nil)
	if sess := session.ComponentSession(); sess != nil {
		router.InternalSeedSessionLocation(sess, buildRouterLocation(path, values))
	}

	node := session.RenderRoot()
	meta := session.Metadata()
	body := render.RenderHTML(node, session.Registry())
	boot := session.BuildBoot(body)

	m.registry.Put(session)

	payload, err := json.Marshal(boot)
	if err != nil {
		http.Error(w, "live: failed to encode boot payload", http.StatusInternalServerError)
		return
	}

	document := buildResponseBody(body, payload, meta, m.assetURL)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(document))
}

func buildRouterLocation(path string, values url.Values) router.Location {
	loc := router.Location{Path: path}
	if values != nil {
		loc.Query = cloneURLValues(values)
	}
	return loc
}

func buildResponseBody(body string, payload []byte, meta *runtime.Meta, assetURL string) string {
	escaped := escapeJSON(string(payload))
	document := buildDocument(body, meta, assetURL)
	script := "<script id=\"live-boot\" type=\"application/json\">" + escaped + "</script>"

	idx := strings.LastIndex(document, "</body>")
	if idx < 0 {
		return document + script
	}

	var builder strings.Builder
	builder.Grow(len(document) + len(script))
	builder.WriteString(document[:idx])
	builder.WriteString(script)
	builder.WriteString(document[idx:])
	return builder.String()
}

func escapeJSON(raw string) string {
	if raw == "" {
		return raw
	}
	raw = strings.ReplaceAll(raw, "</", "<\\/")
	return raw
}

func canonicalPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return trimmed
}

func defaultSessionID(*http.Request) (runtime.SessionID, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(buf[:])
	return runtime.SessionID(id), nil
}

func cloneURLValues(src url.Values) url.Values {
	if len(src) == 0 {
		return url.Values{}
	}
	dst := make(url.Values, len(src))
	for k, values := range src {
		if len(values) == 0 {
			dst[k] = []string{}
			continue
		}
		cp := make([]string, len(values))
		copy(cp, values)
		dst[k] = cp
	}
	return dst
}
