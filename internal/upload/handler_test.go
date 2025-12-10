package upload

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/tus/tusd/v2/pkg/handler"
)

func TestNewHandler(t *testing.T) {
	dir := t.TempDir()

	registry := NewRegistry()
	h, err := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		registry.Remove,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestNewHandler_DefaultStoragePath(t *testing.T) {
	registry := NewRegistry()

	defaultPath := "./uploads"
	_ = os.MkdirAll(defaultPath, 0755)
	defer os.RemoveAll(defaultPath)

	h, err := NewHandler(
		Config{},
		registry.Lookup,
		registry.Remove,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if h.config.StoragePath != defaultPath {
		t.Errorf("expected default storage path %q, got %q", defaultPath, h.config.StoragePath)
	}
}

func TestHandler_validateServerLimits_Size(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	h, _ := NewHandler(
		Config{StoragePath: dir, MaxSize: 1000},
		registry.Lookup,
		registry.Remove,
	)

	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{"under limit", 500, false},
		{"at limit", 1000, false},
		{"over limit", 1001, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := handler.FileInfo{Size: tt.size}
			err := h.validateServerLimits(info)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateServerLimits() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_validateServerLimits_FileType(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	h, _ := NewHandler(
		Config{
			StoragePath:  dir,
			AllowedTypes: []string{"image/png", "image/jpeg"},
		},
		registry.Lookup,
		registry.Remove,
	)

	tests := []struct {
		name     string
		fileType string
		wantErr  bool
	}{
		{"allowed png", "image/png", false},
		{"allowed jpeg", "image/jpeg", false},
		{"not allowed gif", "image/gif", true},
		{"empty type", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := handler.FileInfo{
				MetaData: map[string]string{"filetype": tt.fileType},
			}
			err := h.validateServerLimits(info)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateServerLimits() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_validateServerLimits_NoRestrictions(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	h, _ := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		registry.Remove,
	)

	info := handler.FileInfo{
		Size:     999999999,
		MetaData: map[string]string{"filetype": "anything/whatever"},
	}
	err := h.validateServerLimits(info)
	if err != nil {
		t.Errorf("expected no error with no restrictions, got: %v", err)
	}
}

func TestHandler_validateCallbackLimits_Size(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	h, _ := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		registry.Remove,
	)

	cb := UploadCallback{Token: "test", MaxSize: 500}

	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{"under limit", 400, false},
		{"at limit", 500, false},
		{"over limit", 501, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := handler.FileInfo{Size: tt.size}
			err := h.validateCallbackLimits(info, cb)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCallbackLimits() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_validateCallbackLimits_FileType(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	h, _ := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		registry.Remove,
	)

	cb := UploadCallback{
		Token:        "test",
		AllowedTypes: []string{"application/pdf"},
	}

	tests := []struct {
		name     string
		fileType string
		wantErr  bool
	}{
		{"allowed pdf", "application/pdf", false},
		{"not allowed png", "image/png", true},
		{"empty type", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := handler.FileInfo{
				MetaData: map[string]string{"filetype": tt.fileType},
			}
			err := h.validateCallbackLimits(info, cb)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateCallbackLimits() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestHandler_validateCallbackLimits_NoRestrictions(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	h, _ := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		registry.Remove,
	)

	cb := UploadCallback{Token: "test"}

	info := handler.FileInfo{
		Size:     999999999,
		MetaData: map[string]string{"filetype": "anything/whatever"},
	}
	err := h.validateCallbackLimits(info, cb)
	if err != nil {
		t.Errorf("expected no error with no callback restrictions, got: %v", err)
	}
}

func TestHandler_preUpload_MissingToken(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	h, _ := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		registry.Remove,
	)

	event := handler.HookEvent{
		Upload: handler.FileInfo{
			MetaData: map[string]string{},
		},
	}

	_, _, err := h.preUpload(event)
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if err.Error() != "missing upload token" {
		t.Errorf("expected 'missing upload token' error, got: %v", err)
	}
}

func TestHandler_preUpload_InvalidToken(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	h, _ := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		registry.Remove,
	)

	event := handler.HookEvent{
		Upload: handler.FileInfo{
			MetaData: map[string]string{"token": "invalid-token"},
		},
	}

	_, _, err := h.preUpload(event)
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if err.Error() != "invalid upload token" {
		t.Errorf("expected 'invalid upload token' error, got: %v", err)
	}
}

func TestHandler_preUpload_ValidToken(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	registry.Register(UploadCallback{Token: "valid-token"})

	h, _ := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		registry.Remove,
	)

	event := handler.HookEvent{
		Upload: handler.FileInfo{
			MetaData: map[string]string{"token": "valid-token"},
		},
	}

	_, _, err := h.preUpload(event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestHandler_preUpload_ServerLimitExceeded(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	registry.Register(UploadCallback{Token: "valid-token"})

	h, _ := NewHandler(
		Config{StoragePath: dir, MaxSize: 100},
		registry.Lookup,
		registry.Remove,
	)

	event := handler.HookEvent{
		Upload: handler.FileInfo{
			Size:     200,
			MetaData: map[string]string{"token": "valid-token"},
		},
	}

	_, _, err := h.preUpload(event)
	if err == nil {
		t.Fatal("expected error for exceeding server size limit")
	}
}

func TestHandler_preUpload_CallbackLimitExceeded(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	registry.Register(UploadCallback{Token: "valid-token", MaxSize: 50})

	h, _ := NewHandler(
		Config{StoragePath: dir, MaxSize: 100},
		registry.Lookup,
		registry.Remove,
	)

	event := handler.HookEvent{
		Upload: handler.FileInfo{
			Size:     75,
			MetaData: map[string]string{"token": "valid-token"},
		},
	}

	_, _, err := h.preUpload(event)
	if err == nil {
		t.Fatal("expected error for exceeding callback size limit")
	}
}

func TestHandler_handleComplete(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	completeCalled := false
	removeCalled := false

	registry.Register(UploadCallback{
		Token: "complete-token",
		OnComplete: func(info FileInfo) error {
			completeCalled = true
			return nil
		},
	})

	h, _ := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		func(token string) {
			removeCalled = true
			registry.Remove(token)
		},
	)

	event := handler.HookEvent{
		Upload: handler.FileInfo{
			MetaData: map[string]string{"token": "complete-token"},
		},
	}

	h.handleComplete(event)

	if !completeCalled {
		t.Error("expected OnComplete to be called")
	}
	if !removeCalled {
		t.Error("expected onRemove to be called")
	}

	_, ok := registry.Lookup("complete-token")
	if ok {
		t.Error("expected token to be removed from registry")
	}
}

func TestHandler_handleComplete_MissingToken(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	h, _ := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		registry.Remove,
	)

	event := handler.HookEvent{
		Upload: handler.FileInfo{
			MetaData: map[string]string{},
		},
	}

	h.handleComplete(event)
}

func TestHandler_handleComplete_UnknownToken(t *testing.T) {
	dir := t.TempDir()
	registry := NewRegistry()

	h, _ := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		registry.Remove,
	)

	event := handler.HookEvent{
		Upload: handler.FileInfo{
			MetaData: map[string]string{"token": "unknown"},
		},
	}

	h.handleComplete(event)
}

func TestHandler_StoragePathCreated(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "path")
	registry := NewRegistry()

	_, err := NewHandler(
		Config{StoragePath: dir},
		registry.Lookup,
		registry.Remove,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
