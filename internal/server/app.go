package server

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/eleven-am/pondlive/internal/handler"
	"github.com/eleven-am/pondlive/internal/htmlutil"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/upload"
)

const (
	staticPath = "/static/"
	tusAsset   = "tus.min.js"

	DefaultReadHeaderTimeout = 10 * time.Second
	DefaultReadTimeout       = 60 * time.Second
	DefaultWriteTimeout      = 60 * time.Second
	DefaultIdleTimeout       = 120 * time.Second
	DefaultMaxHeaderBytes    = 64 << 10

	DefaultWatchdogInterval = time.Second
	DefaultSlowStep         = time.Second
	DefaultWedgeAfter       = 30 * time.Second
)

type HTTPConfig struct {
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
	MaxHeaderBytes    int
}

func (c HTTPConfig) withDefaults() HTTPConfig {
	if c.ReadHeaderTimeout == 0 {
		c.ReadHeaderTimeout = DefaultReadHeaderTimeout
	}
	if c.ReadTimeout == 0 {
		c.ReadTimeout = DefaultReadTimeout
	}
	if c.WriteTimeout == 0 {
		c.WriteTimeout = DefaultWriteTimeout
	}
	if c.IdleTimeout == 0 {
		c.IdleTimeout = DefaultIdleTimeout
	}
	if c.MaxHeaderBytes == 0 {
		c.MaxHeaderBytes = DefaultMaxHeaderBytes
	}
	return c
}

func enabled(d time.Duration) time.Duration {
	if d < 0 {
		return 0
	}
	return d
}

type WatchdogConfig struct {
	Interval  time.Duration
	SlowStep  time.Duration
	WedgeStep time.Duration
}

func (c WatchdogConfig) withDefaults() WatchdogConfig {
	if c.Interval <= 0 {
		c.Interval = DefaultWatchdogInterval
	}
	if c.SlowStep <= 0 {
		c.SlowStep = DefaultSlowStep
	}
	if c.WedgeStep <= 0 {
		c.WedgeStep = DefaultWedgeAfter
	}
	return c
}

type App struct {
	component     session.Component
	registry      *SessionRegistry
	endpoint      *Endpoint
	version       int
	idGenerator   func(*http.Request) (session.SessionID, error)
	sessionConfig *session.Config
	clientAsset   string
	mux           *http.ServeMux
	uploadHandler *upload.Handler
	httpConfig    HTTPConfig
	watchdog      *watchdog
}

type Config struct {
	Component session.Component

	ClientAsset string

	SessionConfig *session.Config

	IDGenerator func(*http.Request) (session.SessionID, error)

	Context context.Context

	UploadConfig *upload.Config

	Transport TransportConfig

	RetainedHeaders []string

	MaxSessions int

	MaxPendingSessions int

	SessionLifetime time.Duration

	HTTP HTTPConfig

	Watchdog WatchdogConfig
}

func New(cfg Config) (*App, error) {
	if cfg.Component == nil {
		return nil, runtime.NewError(runtime.ErrCodeApp, "component is required")
	}

	ctx := cfg.Context
	if ctx == nil {
		ctx = context.Background()
	}

	app := &App{
		component:     cfg.Component,
		registry:      NewSessionRegistry(cfg.MaxSessions, cfg.MaxPendingSessions),
		version:       2,
		idGenerator:   defaultSessionID,
		clientAsset:   "/static/pondlive.js",
		sessionConfig: &session.Config{},
		mux:           http.NewServeMux(),
		httpConfig:    cfg.HTTP.withDefaults(),
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
	app.sessionConfig.RetainedHeaders = append([]string(nil), cfg.RetainedHeaders...)
	app.sessionConfig.MaxLifetime = cfg.SessionLifetime
	app.sessionConfig.OnClose = app.registry.sessionClosed
	app.sessionConfig.OnLeavePending = app.registry.leftPending
	userOnReset := app.sessionConfig.OnReset
	app.sessionConfig.OnReset = func(s *session.LiveSession, cause string) {
		app.registry.sessionReset(cause)
		if userOnReset != nil {
			userOnReset(s, cause)
		}
	}

	transport := cfg.Transport
	transport.RetainedHeaders = app.sessionConfig.RetainedHeaders
	endpoint, err := NewEndpoint(ctx, app.registry, transport)
	if err != nil {
		return nil, err
	}
	app.endpoint = endpoint
	app.sessionConfig.PendingTTL = endpoint.disconnectTTL
	app.sessionConfig.DisconnectTTL = endpoint.disconnectTTL
	clampReplay(app.sessionConfig, endpoint.cfg.OutboundBudget)

	if cfg.UploadConfig != nil {
		uploadHandler, err := upload.NewHandler(*cfg.UploadConfig, app.resolveUploadRegistry)
		if err != nil {
			endpoint.cancel()
			return nil, err
		}
		app.uploadHandler = uploadHandler
		app.sessionConfig.UploadEndpoint = uploadHandler.BasePath() + "/"
		app.sessionConfig.UploadScriptURL = staticPath + tusAsset
	}

	app.watchdog = newWatchdog(app.registry, cfg.Watchdog.withDefaults())
	app.watchdog.start()
	context.AfterFunc(ctx, app.watchdog.stop)

	app.registerRoutes()

	return app, nil
}

func (a *App) registerRoutes() {
	a.mux.Handle(staticPath, http.StripPrefix(staticPath, http.FileServer(http.FS(Assets))))
	a.mux.Handle(livePath, a.endpoint)
	a.mux.Handle(handler.PathPrefix, handler.NewDispatcher(a.registry))
	if a.uploadHandler != nil {
		base := a.uploadHandler.BasePath()
		a.mux.Handle(base+"/", http.StripPrefix(base, a.uploadHandler))
	}
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
		Addr:              addr,
		Handler:           a.mux,
		ReadHeaderTimeout: enabled(a.httpConfig.ReadHeaderTimeout),
		ReadTimeout:       enabled(a.httpConfig.ReadTimeout),
		WriteTimeout:      enabled(a.httpConfig.WriteTimeout),
		IdleTimeout:       enabled(a.httpConfig.IdleTimeout),
		MaxHeaderBytes:    int(enabled(time.Duration(a.httpConfig.MaxHeaderBytes))),
	}
}

func (a *App) Stats() Stats {
	return a.registry.Stats()
}

type ShutdownError struct {
	Stuck int
	Err   error
}

func (e *ShutdownError) Error() string {
	return fmt.Sprintf("server: shutdown incomplete: %d session(s) still running (%v)", e.Stuck, e.Err)
}

func (e *ShutdownError) Unwrap() error {
	return e.Err
}

func (a *App) Shutdown(ctx context.Context) error {
	a.registry.Close()

	sessions := a.registry.Snapshot()
	for _, sess := range sessions {
		sess.RequestClose(session.ReasonShutdown)
	}
	sessions = append(sessions, a.registry.stuckSessions()...)

	var err error
	for _, sess := range sessions {
		select {
		case <-sess.Done():
		case <-ctx.Done():
			err = ctx.Err()
		}
		if err != nil {
			break
		}
	}

	if endpointErr := a.endpoint.Shutdown(ctx); err == nil {
		err = endpointErr
	}
	if a.uploadHandler != nil {
		a.uploadHandler.Close()
	}
	a.watchdog.stop()

	if err != nil {
		running := 0
		for _, sess := range sessions {
			select {
			case <-sess.Done():
			default:
				running++
			}
		}
		return &ShutdownError{Stuck: running, Err: err}
	}
	return nil
}

func (a *App) resolveUploadRegistry(sessionID string) (*upload.Registry, bool) {
	sess, ok := a.registry.Lookup(session.SessionID(sessionID))
	if !ok || sess == nil {
		return nil, false
	}
	reg := sess.UploadRegistry()
	return reg, reg != nil
}

func (a *App) serveSSR(w http.ResponseWriter, r *http.Request) {
	sid, err := a.idGenerator(r)
	if err != nil || sid == "" {
		http.Error(w, "Failed to allocate session", http.StatusInternalServerError)
		return
	}

	version := a.version
	if version <= 0 {
		version = 2
	}

	if err := a.registry.Admit(sid); err != nil {
		a.rejectAdmission(w, err)
		return
	}

	cfg := cloneSessionConfig(a.sessionConfig)
	cfg.ClientAsset = a.clientAsset

	sess := session.NewLiveSession(sid, version, a.component, &cfg, session.WithRequest(r))

	if err := a.registry.Put(sess); err != nil {
		a.rejectSession(w, sess, err)
		return
	}

	res, err := sess.Render(r.Context())
	if err != nil {
		writeRenderError(w, r, err)
		return
	}

	h := w.Header()
	h.Set("Cache-Control", "no-store")
	for name, values := range res.Header {
		h[name] = append([]string(nil), values...)
	}

	switch {
	case res.Redirect != nil:
		http.Redirect(w, r, res.Redirect.URL, res.Redirect.Code)
	case res.Status >= http.StatusInternalServerError:
		a.writeCrash(w, res, cfg.DevMode)
	default:
		boot := res.Boot
		boot.WS = livePath
		boot.Tok = a.endpoint.tokens.issue(string(sid))
		bootJSON, err := json.Marshal(boot)
		if err != nil {
			sess.RequestClose(session.ReasonRenderFailed)
			http.Error(w, "Failed to encode boot payload", http.StatusInternalServerError)
			return
		}
		h.Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(decorateDocument(res.HTML, bootJSON)))
	}
}

func (a *App) rejectAdmission(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrDuplicateSession) {
		a.logger().Error("pondlive diagnostic", "phase", "ssr:duplicate-session", "message", "the id generator returned a session id that is already registered; the request was refused")
		http.Error(w, "Failed to allocate session", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Retry-After", "1")
	http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
}

func (a *App) logger() *slog.Logger {
	if a.sessionConfig.Logger != nil {
		return a.sessionConfig.Logger
	}
	return slog.Default()
}

func (a *App) rejectSession(w http.ResponseWriter, sess *session.LiveSession, err error) {
	switch {
	case errors.Is(err, ErrDuplicateSession):
		sess.Report(protocol.Diagnostic{
			Phase:   "ssr:duplicate-session",
			Message: "the id generator returned a session id that is already registered; the request was refused",
		})
		sess.RequestClose(session.ReasonRenderFailed)
		http.Error(w, "Failed to allocate session", http.StatusInternalServerError)
	default:
		sess.RequestClose(session.ReasonRenderFailed)
		w.Header().Set("Retry-After", "1")
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
	}
}

func writeRenderError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, context.Canceled) && r.Context().Err() != nil:
		return
	case errors.Is(err, runtime.ErrNotSettled):
		http.Error(w, "Initial render failed: "+err.Error(), http.StatusInternalServerError)
	default:
		w.Header().Set("Retry-After", "1")
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
	}
}

func (a *App) writeCrash(w http.ResponseWriter, res session.RenderResult, devMode bool) {
	if res.HTML != "" {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(res.HTML))
		return
	}
	if devMode && res.Crash != nil {
		http.Error(w, "render crashed: "+res.Crash.Message+"\n\n"+res.Crash.StackTrace, http.StatusInternalServerError)
		return
	}
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func defaultSessionID(*http.Request) (session.SessionID, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	id := base64.RawURLEncoding.EncodeToString(buf[:])
	return session.SessionID(id), nil
}

func clampReplay(cfg *session.Config, budget int) {
	maxReplay := budget / 4
	switch {
	case cfg.ReplayBytes <= 0:
		cfg.ReplayBytes = max(1, min(session.DefaultReplayBytes, maxReplay))
	case cfg.ReplayBytes > maxReplay:
		cfg.ReplayBytes = max(1, maxReplay)
	}
	maxReset := max(1, budget/2)
	if cfg.ResetLimit <= 0 || cfg.ResetLimit > maxReset {
		cfg.ResetLimit = maxReset
	}
}

func cloneSessionConfig(cfg *session.Config) session.Config {
	if cfg == nil {
		return session.Config{}
	}
	return *cfg
}

func decorateDocument(document string, bootJSON []byte) string {
	escaped := htmlutil.EscapeClose(string(bootJSON))
	bootScript := `<script id="live-boot" type="application/json">` + escaped + `</script>`

	idx := htmlutil.LastIndexFold(document, "</body>")
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
