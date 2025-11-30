package router

import (
	"net/http"
	"net/url"
	"sync"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func TestSSRRedirectStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		replace    bool
		wantStatus int
	}{
		{
			name:       "Navigate uses 302 Found",
			replace:    false,
			wantStatus: http.StatusFound,
		},
		{
			name:       "Replace uses 303 See Other",
			replace:    true,
			wantStatus: http.StatusSeeOther,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state := headers.NewRequestState(&headers.RequestInfo{
				Path: "/current",
			})

			inst := &runtime.Instance{
				ID:        "test",
				Providers: make(map[any]any),
			}

			setProviderValue(inst, headersCtxID(), state)
			setProviderValue(inst, locationCtxID(), &Location{
				Path:  "/current",
				Query: url.Values{},
			})

			ctx := makeCtx(inst)

			navigate(ctx, "/target", tt.replace)

			redirectURL, code, hasRedirect := state.Redirect()
			if !hasRedirect {
				t.Fatal("expected redirect to be set")
			}
			if redirectURL != "/target" {
				t.Errorf("expected redirect URL '/target', got %q", redirectURL)
			}
			if code != tt.wantStatus {
				t.Errorf("expected status %d, got %d", tt.wantStatus, code)
			}
		})
	}
}

func TestSSRRedirectWithQueryAndHash(t *testing.T) {
	state := headers.NewRequestState(&headers.RequestInfo{
		Path:  "/search",
		Query: url.Values{"q": []string{"old"}},
	})

	inst := &runtime.Instance{
		ID:        "test",
		Providers: make(map[any]any),
	}

	setProviderValue(inst, headersCtxID(), state)
	setProviderValue(inst, locationCtxID(), &Location{
		Path:  "/search",
		Query: url.Values{"q": []string{"old"}},
	})

	ctx := makeCtx(inst)

	navigate(ctx, "/results?q=new#top", false)

	redirectURL, code, hasRedirect := state.Redirect()
	if !hasRedirect {
		t.Fatal("expected redirect to be set")
	}
	if redirectURL != "/results?q=new#top" {
		t.Errorf("expected redirect URL '/results?q=new#top', got %q", redirectURL)
	}
	if code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, code)
	}
}

func TestNavigateAndReplaceHelpers(t *testing.T) {
	t.Run("Navigate calls navigate with replace=false", func(t *testing.T) {
		state := headers.NewRequestState(&headers.RequestInfo{Path: "/"})
		inst := &runtime.Instance{
			ID:        "test",
			Providers: make(map[any]any),
		}
		setProviderValue(inst, headersCtxID(), state)
		setProviderValue(inst, locationCtxID(), &Location{Path: "/", Query: url.Values{}})
		ctx := makeCtx(inst)

		Navigate(ctx, "/page")

		_, code, _ := state.Redirect()
		if code != http.StatusFound {
			t.Errorf("Navigate should use 302, got %d", code)
		}
	})

	t.Run("Replace calls navigate with replace=true", func(t *testing.T) {
		state := headers.NewRequestState(&headers.RequestInfo{Path: "/"})
		inst := &runtime.Instance{
			ID:        "test",
			Providers: make(map[any]any),
		}
		setProviderValue(inst, headersCtxID(), state)
		setProviderValue(inst, locationCtxID(), &Location{Path: "/", Query: url.Values{}})
		ctx := makeCtx(inst)

		Replace(ctx, "/page")

		_, code, _ := state.Redirect()
		if code != http.StatusSeeOther {
			t.Errorf("Replace should use 303, got %d", code)
		}
	})
}

var (
	testHeadersCtxID  any
	testLocationCtxID any
	ctxIDOnce         sync.Once
)

func initCtxIDs() {
	ctxIDOnce.Do(func() {
		tempInst := &runtime.Instance{
			ID:        "ctx-id-probe",
			Providers: make(map[any]any),
		}
		ctx := makeCtx(tempInst)

		headers.UseProvideRequestState(ctx, nil)
		LocationContext.UseProvider(ctx, nil)

		for id := range tempInst.Providers {
			val := tempInst.Providers[id]
			switch val.(type) {
			case *headers.RequestState:
				testHeadersCtxID = id
			case *Location:
				testLocationCtxID = id
			}
		}
	})
}

func headersCtxID() any {
	initCtxIDs()
	return testHeadersCtxID
}

func locationCtxID() any {
	initCtxIDs()
	return testLocationCtxID
}

func setProviderValue(inst *runtime.Instance, ctxID any, value any) {
	if inst.Providers == nil {
		inst.Providers = make(map[any]any)
	}
	inst.Providers[ctxID] = value
}

func makeCtx(inst *runtime.Instance) *runtime.Ctx {
	return runtime.NewCtxForTest(inst, nil)
}
