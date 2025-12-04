package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"

	pond "github.com/eleven-am/pondsocket/go/pondsocket"

	"github.com/eleven-am/pondlive/internal/handler"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/view"
	"github.com/eleven-am/pondlive/internal/view/diff"
)

type App struct {
	component     session.Component
	registry      *SessionRegistry
	endpoint      *Endpoint
	version       int
	idGenerator   func(*http.Request) (session.SessionID, error)
	sessionConfig *session.Config
	clientAsset   string
	pondManager   *pond.Manager
	mux           *http.ServeMux
}

type Config struct {
	Component session.Component

	ClientAsset string

	SessionConfig *session.Config

	IDGenerator func(*http.Request) (session.SessionID, error)

	Context context.Context
}

func New(cfg Config) (*App, error) {
	if cfg.Component == nil {
		return nil, &AppError{Code: "missing_component", Message: "component is required"}
	}

	ctx := cfg.Context
	if ctx == nil {
		ctx = context.Background()
	}

	app := &App{
		component:     cfg.Component,
		registry:      NewSessionRegistry(),
		version:       1,
		idGenerator:   defaultSessionID,
		clientAsset:   "/static/pondlive.js",
		pondManager:   pond.NewManager(ctx),
		sessionConfig: &session.Config{},
		mux:           http.NewServeMux(),
	}

	if cfg.IDGenerator != nil {
		app.idGenerator = cfg.IDGenerator
	}

	if cfg.SessionConfig != nil {
		clone := *cfg.SessionConfig
		app.sessionConfig = &clone

		if clone.DevMode {
			app.clientAsset = "/static/pondlive-dev.js"
		}
	}

	if strings.TrimSpace(cfg.ClientAsset) != "" {
		app.clientAsset = cfg.ClientAsset
	}

	app.sessionConfig.ClientAsset = app.clientAsset

	endpoint, err := Register(app.pondManager, "/live", app.registry)
	if err != nil {
		return nil, err
	}
	app.endpoint = endpoint

	app.registerRoutes()

	return app, nil
}

func (a *App) registerRoutes() {
	a.mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(Assets))))
	a.mux.HandleFunc("/live", a.pondManager.HTTPHandler())
	a.mux.Handle(handler.PathPrefix, handler.NewDispatcher(a.registry))
	a.mux.HandleFunc("/", a.serveSSR)
}

func (a *App) Mux() *http.ServeMux {
	return a.mux
}

func (a *App) Handler() http.Handler {
	return a.mux
}

func (a *App) HandlerFunc() http.HandlerFunc {
	return a.mux.ServeHTTP
}

func (a *App) Server(addr string) *http.Server {
	return &http.Server{
		Addr:    addr,
		Handler: a.mux,
	}
}

func (a *App) Registry() *SessionRegistry {
	return a.registry
}

func (a *App) serveSSR(w http.ResponseWriter, r *http.Request) {
	sid, err := a.idGenerator(r)
	if err != nil || sid == "" {
		http.Error(w, "Failed to allocate session", http.StatusInternalServerError)
		return
	}

	version := a.version
	if version <= 0 {
		version = 1
	}

	cfg := cloneSessionConfig(a.sessionConfig)
	cfg.ClientAsset = a.clientAsset

	sess := session.NewLiveSession(sid, version, a.component, &cfg)
	capture := session.NewSSRTransport(r)
	sess.SetTransport(capture)

	if err := sess.Flush(); err != nil {
		http.Error(w, "Initial render failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	if reqState := capture.RequestState(); reqState != nil {
		if redirectURL, redirectCode, hasRedirect := reqState.Redirect(); hasRedirect {
			sess.SetTransport(nil)
			http.Redirect(w, r, redirectURL, redirectCode)
			return
		}
	}

	rtSession := sess.Session()
	if rtSession == nil || rtSession.View == nil {
		http.Error(w, "Render produced nil view", http.StatusInternalServerError)
		return
	}

	documentHTML := view.RenderHTML(rtSession.View)

	pathParts := route.NormalizeParts(r.URL.Path)
	location := route.Location{
		Path:  pathParts.Path,
		Query: r.URL.Query(),
		Hash:  route.NormalizeHash(r.URL.Fragment),
	}

	var clientCfg *protocol.ClientConfig
	if cfg.DevMode {
		clientCfg = &protocol.ClientConfig{}
		value := true
		clientCfg.Debug = &value
	}

	boot := protocol.Boot{
		T:        "boot",
		SID:      string(sid),
		Ver:      version,
		Seq:      int(capture.LastSeq()),
		Patch:    diff.ExtractMetadata(rtSession.View),
		Location: location,
		Client:   clientCfg,
	}

	bootJSON, err := json.Marshal(boot)
	if err != nil {
		http.Error(w, "Failed to encode boot payload", http.StatusInternalServerError)
		return
	}

	a.registry.Put(sess)

	document := decorateDocument(documentHTML, bootJSON)

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(document))
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

func decorateDocument(document string, bootJSON []byte) string {
	escaped := escapeJSON(string(bootJSON))
	bootScript := `<script id="live-boot" type="application/json">` + escaped + `</script>`

	idx := lastIndexFold(document, "</body>")
	if idx < 0 {
		return document + bootScript
	}

	var b strings.Builder
	b.Grow(len(document) + len(bootScript))
	b.WriteString(document[:idx])
	b.WriteString(bootScript)
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

type AppError struct {
	Code    string
	Message string
}

func (e *AppError) Error() string {
	return e.Message
}
