package upload

import (
	"errors"
	"net/http"
	"slices"

	"github.com/tus/tusd/v2/pkg/filestore"
	"github.com/tus/tusd/v2/pkg/handler"
)

type Config struct {
	StoragePath  string
	MaxSize      int64
	AllowedTypes []string
}

type LookupFunc func(token string) (UploadCallback, bool)

type Handler struct {
	config   Config
	lookup   LookupFunc
	tusd     *handler.Handler
	store    filestore.FileStore
	onRemove func(token string)
}

func NewHandler(cfg Config, lookup LookupFunc, onRemove func(token string)) (*Handler, error) {
	if cfg.StoragePath == "" {
		cfg.StoragePath = "./uploads"
	}

	store := filestore.New(cfg.StoragePath)

	h := &Handler{
		config:   cfg,
		lookup:   lookup,
		store:    store,
		onRemove: onRemove,
	}

	composer := handler.NewStoreComposer()
	store.UseIn(composer)

	tusConfig := handler.Config{
		BasePath:                "/tus",
		StoreComposer:           composer,
		NotifyCompleteUploads:   true,
		RespectForwardedHeaders: true,
		PreUploadCreateCallback: h.preUpload,
	}

	if cfg.MaxSize > 0 {
		tusConfig.MaxSize = cfg.MaxSize
	}

	tusHandler, err := handler.NewHandler(tusConfig)
	if err != nil {
		return nil, err
	}

	h.tusd = tusHandler

	go h.listenForCompleteUploads()

	return h, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.tusd.ServeHTTP(w, r)
}

func (h *Handler) preUpload(hook handler.HookEvent) (handler.HTTPResponse, handler.FileInfoChanges, error) {
	info := hook.Upload
	resp := handler.HTTPResponse{}
	changes := handler.FileInfoChanges{}

	token := info.MetaData["token"]
	if token == "" {
		return resp, changes, errors.New("missing upload token")
	}

	cb, ok := h.lookup(token)
	if !ok {
		return resp, changes, errors.New("invalid upload token")
	}

	if err := h.validateServerLimits(info); err != nil {
		return resp, changes, err
	}

	if err := h.validateCallbackLimits(info, cb); err != nil {
		return resp, changes, err
	}

	return resp, changes, nil
}

func (h *Handler) validateServerLimits(info handler.FileInfo) error {
	if h.config.MaxSize > 0 && info.Size > h.config.MaxSize {
		return errors.New("file exceeds maximum allowed size")
	}

	if len(h.config.AllowedTypes) > 0 {
		fileType := info.MetaData["filetype"]
		if fileType == "" {
			return errors.New("file type not specified")
		}
		if !slices.Contains(h.config.AllowedTypes, fileType) {
			return errors.New("file type not allowed")
		}
	}

	return nil
}

func (h *Handler) validateCallbackLimits(info handler.FileInfo, cb UploadCallback) error {
	if cb.MaxSize > 0 && info.Size > cb.MaxSize {
		return errors.New("file exceeds maximum allowed size for this upload")
	}

	if len(cb.AllowedTypes) > 0 {
		fileType := info.MetaData["filetype"]
		if fileType == "" {
			return errors.New("file type not specified")
		}
		if !slices.Contains(cb.AllowedTypes, fileType) {
			return errors.New("file type not allowed for this upload")
		}
	}

	return nil
}

func (h *Handler) listenForCompleteUploads() {
	for event := range h.tusd.CompleteUploads {
		h.handleComplete(event)
	}
}

func (h *Handler) handleComplete(event handler.HookEvent) {
	info := event.Upload
	token := info.MetaData["token"]
	if token == "" {
		return
	}

	cb, ok := h.lookup(token)
	if !ok {
		return
	}

	if cb.OnComplete != nil {
		_ = cb.OnComplete(info)
	}

	if h.onRemove != nil {
		h.onRemove(token)
	}
}
