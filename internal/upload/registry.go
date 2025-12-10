package upload

import (
	"sync"

	"github.com/tus/tusd/v2/pkg/handler"
)

type FileInfo = handler.FileInfo

type UploadCallback struct {
	Token        string
	MaxSize      int64
	AllowedTypes []string
	OnComplete   func(FileInfo) error
}

type Registry struct {
	mu        sync.RWMutex
	callbacks map[string]UploadCallback
}

func NewRegistry() *Registry {
	return &Registry{
		callbacks: make(map[string]UploadCallback),
	}
}

func (r *Registry) Register(cb UploadCallback) {
	if cb.Token == "" {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.callbacks[cb.Token] = cb
}

func (r *Registry) Lookup(token string) (UploadCallback, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cb, ok := r.callbacks[token]
	return cb, ok
}

func (r *Registry) Remove(token string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	delete(r.callbacks, token)
}
