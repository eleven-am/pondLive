package upload

import (
	"sync"
	"testing"
)

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("expected non-nil registry")
	}
	if r.callbacks == nil {
		t.Fatal("expected non-nil callbacks map")
	}
}

func TestRegistry_Register(t *testing.T) {
	r := NewRegistry()

	cb := UploadCallback{
		Token:   "test-token",
		MaxSize: 1024,
	}
	r.Register(cb)

	got, ok := r.Lookup("test-token")
	if !ok {
		t.Fatal("expected callback to be registered")
	}
	if got.Token != "test-token" {
		t.Errorf("expected token %q, got %q", "test-token", got.Token)
	}
	if got.MaxSize != 1024 {
		t.Errorf("expected MaxSize %d, got %d", 1024, got.MaxSize)
	}
}

func TestRegistry_Register_EmptyToken(t *testing.T) {
	r := NewRegistry()

	cb := UploadCallback{
		Token:   "",
		MaxSize: 1024,
	}
	r.Register(cb)

	_, ok := r.Lookup("")
	if ok {
		t.Fatal("expected empty token to not be registered")
	}
}

func TestRegistry_Lookup_NotFound(t *testing.T) {
	r := NewRegistry()

	_, ok := r.Lookup("nonexistent")
	if ok {
		t.Fatal("expected lookup to return false for nonexistent token")
	}
}

func TestRegistry_Remove(t *testing.T) {
	r := NewRegistry()

	cb := UploadCallback{Token: "test-token"}
	r.Register(cb)

	_, ok := r.Lookup("test-token")
	if !ok {
		t.Fatal("expected callback to be registered")
	}

	r.Remove("test-token")

	_, ok = r.Lookup("test-token")
	if ok {
		t.Fatal("expected callback to be removed")
	}
}

func TestRegistry_Remove_Nonexistent(t *testing.T) {
	r := NewRegistry()
	r.Remove("nonexistent")
}

func TestRegistry_Concurrent(t *testing.T) {
	r := NewRegistry()
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			token := string(rune('a' + i%26))
			r.Register(UploadCallback{Token: token, MaxSize: int64(i)})
			r.Lookup(token)
			r.Remove(token)
		}(i)
	}

	wg.Wait()
}

func TestRegistry_OverwriteCallback(t *testing.T) {
	r := NewRegistry()

	cb1 := UploadCallback{Token: "test-token", MaxSize: 100}
	r.Register(cb1)

	cb2 := UploadCallback{Token: "test-token", MaxSize: 200}
	r.Register(cb2)

	got, ok := r.Lookup("test-token")
	if !ok {
		t.Fatal("expected callback to exist")
	}
	if got.MaxSize != 200 {
		t.Errorf("expected MaxSize %d, got %d", 200, got.MaxSize)
	}
}

func TestRegistry_AllowedTypes(t *testing.T) {
	r := NewRegistry()

	cb := UploadCallback{
		Token:        "test-token",
		AllowedTypes: []string{"image/png", "image/jpeg"},
	}
	r.Register(cb)

	got, ok := r.Lookup("test-token")
	if !ok {
		t.Fatal("expected callback to be registered")
	}
	if len(got.AllowedTypes) != 2 {
		t.Errorf("expected 2 allowed types, got %d", len(got.AllowedTypes))
	}
}

func TestRegistry_OnComplete(t *testing.T) {
	r := NewRegistry()

	called := false
	cb := UploadCallback{
		Token: "test-token",
		OnComplete: func(info FileInfo) error {
			called = true
			return nil
		},
	}
	r.Register(cb)

	got, ok := r.Lookup("test-token")
	if !ok {
		t.Fatal("expected callback to be registered")
	}
	if got.OnComplete == nil {
		t.Fatal("expected OnComplete to be set")
	}

	_ = got.OnComplete(FileInfo{})
	if !called {
		t.Fatal("expected OnComplete to be called")
	}
}
