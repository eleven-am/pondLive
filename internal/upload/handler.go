package upload

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gabriel-vasile/mimetype"
	"github.com/tus/tusd/v2/pkg/filestore"
	"github.com/tus/tusd/v2/pkg/handler"
	"golang.org/x/exp/slog"
)

const (
	DefaultBasePath      = "/tus"
	DefaultMaxSize       = 32 << 20
	DefaultSweepInterval = 5 * time.Minute
	sniffBytes           = 3072
)

var uploadIDPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)

type Config struct {
	StoragePath             string
	MaxSize                 int64
	AllowedTypes            []string
	Logger                  *slog.Logger
	BasePath                string
	RespectForwardedHeaders bool
	MaxGrantsPerSession     int
	SessionBudget           int64
	GlobalBudget            int64
	SweepInterval           time.Duration
}

type Resolver func(sessionID string) (*Registry, bool)

type Handler struct {
	config  Config
	resolve Resolver
	tusd    *handler.Handler
	store   filestore.FileStore
	pool    *Pool
	regMu   sync.Mutex
	regs    map[*Registry]struct{}
	done    chan struct{}
	stopped sync.WaitGroup
	once    sync.Once
}

func NewHandler(cfg Config, resolve Resolver) (*Handler, error) {
	if resolve == nil {
		return nil, errors.New("upload: resolver is required")
	}
	if cfg.StoragePath == "" {
		cfg.StoragePath = "./uploads"
	}
	if cfg.BasePath == "" {
		cfg.BasePath = DefaultBasePath
	}
	cfg.BasePath = "/" + strings.Trim(cfg.BasePath, "/")
	if cfg.MaxSize == 0 {
		cfg.MaxSize = DefaultMaxSize
	}
	if cfg.SweepInterval == 0 {
		cfg.SweepInterval = DefaultSweepInterval
	}

	store := filestore.New(cfg.StoragePath)
	h := &Handler{config: cfg, resolve: resolve, store: store, done: make(chan struct{}), regs: make(map[*Registry]struct{})}
	if cfg.GlobalBudget > 0 {
		h.pool = NewPool(cfg.GlobalBudget)
	}

	composer := handler.NewStoreComposer()
	composer.UseCore(store)
	composer.UseTerminater(store)

	tusConfig := handler.Config{
		BasePath:                   cfg.BasePath,
		StoreComposer:              composer,
		MaxSize:                    max(cfg.MaxSize, 0),
		RespectForwardedHeaders:    cfg.RespectForwardedHeaders,
		Cors:                       &handler.CorsConfig{Disable: true},
		PreUploadCreateCallback:    h.preUpload,
		PreFinishResponseCallback:  h.preFinish,
		PreUploadTerminateCallback: h.preTerminate,
		Logger:                     cfg.Logger,
	}

	tusHandler, err := handler.NewHandler(tusConfig)
	if err != nil {
		return nil, err
	}
	h.tusd = tusHandler

	h.sweep(time.Now())
	if cfg.SweepInterval > 0 {
		h.stopped.Add(1)
		go h.sweepLoop()
	}
	return h, nil
}

func (h *Handler) Close() {
	h.once.Do(func() { close(h.done) })
	h.stopped.Wait()
}

func (h *Handler) BasePath() string {
	return h.config.BasePath
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := r.Header.Get(TokenHeader)
	if token == "" {
		http.Error(w, "upload token required", http.StatusUnauthorized)
		return
	}
	reg, ok := h.registryFor(token)
	if !ok {
		http.Error(w, "upload not permitted", http.StatusForbidden)
		return
	}

	if id := strings.Trim(r.URL.Path, "/"); id != "" {
		if !uploadIDPattern.MatchString(id) {
			http.Error(w, "upload not found", http.StatusNotFound)
			return
		}
		release, err := reg.Acquire(token, id)
		if err != nil {
			http.Error(w, "upload not permitted", http.StatusForbidden)
			return
		}
		defer release()
	} else if r.Header.Get("Upload-Defer-Length") != "" {
		http.Error(w, "deferred upload length is not supported", http.StatusBadRequest)
		return
	}

	h.tusd.ServeHTTP(w, r)
}

func (h *Handler) registryFor(token string) (*Registry, bool) {
	sessionID, ok := SessionIDFromToken(token)
	if !ok {
		return nil, false
	}
	reg, ok := h.resolve(sessionID)
	if !ok || reg == nil {
		return nil, false
	}
	if reg.Bind(BindOptions{
		Limits:  Limits{MaxGrants: h.config.MaxGrantsPerSession, SessionBudget: h.config.SessionBudget},
		Reap:    h.reap,
		Pool:    h.pool,
		OnClose: func() { h.untrack(reg) },
	}) {
		h.track(reg)
	}
	return reg, true
}

func (h *Handler) track(reg *Registry) {
	h.regMu.Lock()
	h.regs[reg] = struct{}{}
	h.regMu.Unlock()
}

func (h *Handler) untrack(reg *Registry) {
	h.regMu.Lock()
	delete(h.regs, reg)
	h.regMu.Unlock()
}

func (h *Handler) liveUploads() map[string]struct{} {
	h.regMu.Lock()
	regs := make([]*Registry, 0, len(h.regs))
	for reg := range h.regs {
		regs = append(regs, reg)
	}
	h.regMu.Unlock()

	live := make(map[string]struct{})
	for _, reg := range regs {
		reg.Expire()
		for _, id := range reg.LiveUploadIDs() {
			live[id] = struct{}{}
		}
	}
	return live
}

func (h *Handler) grantFor(hook handler.HookEvent) (string, *Registry, error) {
	token := hook.HTTPRequest.Header.Get(TokenHeader)
	reg, ok := h.registryFor(token)
	if !ok {
		return "", nil, errForbidden
	}
	return token, reg, nil
}

var (
	errForbidden    = handler.NewError("ERR_UPLOAD_FORBIDDEN", "upload not permitted", http.StatusForbidden)
	errUnsupported  = handler.NewError("ERR_UPLOAD_UNSUPPORTED", "partial, final and deferred uploads are not supported", http.StatusBadRequest)
	errTooLarge     = handler.NewError("ERR_UPLOAD_TOO_LARGE", "file exceeds the maximum allowed size", http.StatusRequestEntityTooLarge)
	errTypeNotAllow = handler.NewError("ERR_UPLOAD_TYPE_NOT_ALLOWED", "file type not allowed", http.StatusUnsupportedMediaType)
	errUnavailable  = handler.NewError("ERR_UPLOAD_UNAVAILABLE", "upload target is unavailable", http.StatusServiceUnavailable)
)

func grantError(err error) error {
	var detailed handler.Error
	if errors.As(err, &detailed) {
		return err
	}
	if errors.Is(err, ErrGrantUnknown) || errors.Is(err, ErrGrantExpired) || errors.Is(err, ErrGrantExhausted) || errors.Is(err, ErrUploadNotGranted) {
		return handler.NewError("ERR_UPLOAD_GRANT", err.Error(), http.StatusForbidden)
	}
	if errors.Is(err, ErrUploadFinished) {
		return handler.NewError("ERR_UPLOAD_FINISHED", err.Error(), http.StatusConflict)
	}
	if errors.Is(err, ErrBudgetExceeded) {
		return handler.NewError("ERR_UPLOAD_BUDGET", err.Error(), http.StatusTooManyRequests)
	}
	return err
}

func (h *Handler) preUpload(hook handler.HookEvent) (handler.HTTPResponse, handler.FileInfoChanges, error) {
	token, reg, err := h.grantFor(hook)
	if err != nil {
		return handler.HTTPResponse{}, handler.FileInfoChanges{}, err
	}
	info := hook.Upload
	if info.SizeIsDeferred || info.IsPartial || info.IsFinal {
		return handler.HTTPResponse{}, handler.FileInfoChanges{}, errUnsupported
	}

	id, _, err := reg.Reserve(token, info.Size, func(spec GrantSpec) error {
		return h.checkNew(info, spec)
	})
	if err != nil {
		return handler.HTTPResponse{}, handler.FileInfoChanges{}, grantError(err)
	}
	return handler.HTTPResponse{}, handler.FileInfoChanges{ID: id}, nil
}

func (h *Handler) checkNew(info handler.FileInfo, spec GrantSpec) error {
	if h.config.MaxSize > 0 && info.Size > h.config.MaxSize {
		return errTooLarge
	}
	if spec.MaxSize > 0 && info.Size > spec.MaxSize {
		return errTooLarge
	}
	name := info.MetaData["filename"]
	declared := info.MetaData["filetype"]
	if !AcceptsUpload(h.config.AllowedTypes, name, declared) || !AcceptsUpload(spec.Accept, name, declared) {
		return errTypeNotAllow
	}
	return nil
}

func (h *Handler) preFinish(hook handler.HookEvent) (handler.HTTPResponse, error) {
	token, reg, err := h.grantFor(hook)
	if err != nil {
		h.discard(hook.Context, hook.Upload.ID)
		return handler.HTTPResponse{}, err
	}
	info := hook.Upload
	spec, err := reg.Finish(token, info.ID)
	if err != nil {
		h.discard(hook.Context, info.ID)
		return handler.HTTPResponse{}, grantError(err)
	}

	if err := h.verifyContent(info, spec); err != nil {
		h.discard(hook.Context, info.ID)
		reg.Settle(token, info.ID)
		return handler.HTTPResponse{}, err
	}

	if spec.OnComplete != nil {
		if err := spec.OnComplete(info); err != nil {
			h.discard(hook.Context, info.ID)
			reg.Settle(token, info.ID)
			return handler.HTTPResponse{}, errUnavailable
		}
	}
	reg.Settle(token, info.ID)
	return handler.HTTPResponse{}, nil
}

func (h *Handler) verifyContent(info handler.FileInfo, spec GrantSpec) error {
	if len(h.config.AllowedTypes) == 0 && len(spec.Accept) == 0 {
		return nil
	}
	path := info.Storage["Path"]
	if path == "" {
		return errUnavailable
	}
	detected, err := sniffFile(path)
	if err != nil {
		return errUnavailable
	}
	name := info.MetaData["filename"]
	if !AcceptsContent(h.config.AllowedTypes, name, detected) || !AcceptsContent(spec.Accept, name, detected) {
		return errTypeNotAllow
	}
	return nil
}

func sniffFile(path string) (*mimetype.MIME, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	buf := make([]byte, sniffBytes)
	n, err := io.ReadFull(f, buf)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}
	return mimetype.Detect(buf[:n]), nil
}

func (h *Handler) preTerminate(hook handler.HookEvent) (handler.HTTPResponse, error) {
	token, reg, err := h.grantFor(hook)
	if err != nil {
		return handler.HTTPResponse{}, err
	}
	if err := reg.Terminate(token, hook.Upload.ID); err != nil {
		return handler.HTTPResponse{}, grantError(err)
	}
	return handler.HTTPResponse{}, nil
}

func (h *Handler) discard(ctx context.Context, id string) {
	if id == "" {
		return
	}
	up, err := h.store.GetUpload(ctx, id)
	if err != nil {
		if !isNotFound(err) {
			h.logDiscardFailure(id, err)
		}
		return
	}
	if err := h.store.AsTerminatableUpload(up).Terminate(ctx); err != nil && !isNotFound(err) {
		h.logDiscardFailure(id, err)
	}
}

func isNotFound(err error) bool {
	return errors.Is(err, handler.ErrNotFound) || errors.Is(err, os.ErrNotExist)
}

func (h *Handler) reap(ids []string) {
	for _, id := range ids {
		h.discard(context.Background(), id)
	}
}

func (h *Handler) sweepLoop() {
	defer h.stopped.Done()
	ticker := time.NewTicker(h.config.SweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-h.done:
			return
		case now := <-ticker.C:
			h.sweep(now)
		}
	}
}

func (h *Handler) sweep(now time.Time) {
	live := h.liveUploads()
	entries, err := os.ReadDir(h.config.StoragePath)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		id := strings.TrimSuffix(name, ".info")
		if !uploadIDPattern.MatchString(id) {
			continue
		}
		if _, alive := live[id]; alive {
			continue
		}
		infoPath := filepath.Join(h.config.StoragePath, id+".info")
		binPath := filepath.Join(h.config.StoragePath, id)
		if name == id {
			if _, err := os.Stat(infoPath); err == nil {
				continue
			}
			if staleSince(binPath, now) {
				_ = os.Remove(binPath)
			}
			continue
		}
		h.sweepIncomplete(infoPath, binPath, now)
	}
}

func (h *Handler) sweepIncomplete(infoPath, binPath string, now time.Time) {
	raw, err := os.ReadFile(infoPath)
	if err != nil {
		return
	}
	var info handler.FileInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return
	}
	if p := info.Storage["Path"]; p != "" {
		binPath = p
	}
	var binSize int64 = -1
	if st, err := os.Stat(binPath); err == nil {
		binSize = st.Size()
		if !staleSince(binPath, now) {
			return
		}
	}
	if !staleSince(infoPath, now) {
		return
	}
	if binSize >= info.Size {
		return
	}
	_ = os.Remove(binPath)
	_ = os.Remove(infoPath)
}

func staleSince(path string, now time.Time) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return now.Sub(st.ModTime()) >= DefaultGrantTTL
}

func (h *Handler) logDiscardFailure(id string, err error) {
	logger := h.config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger.Error("upload: failed to discard rejected upload", "id", id, "error", fmt.Sprint(err))
}
