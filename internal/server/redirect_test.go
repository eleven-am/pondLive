package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eleven-am/pondlive/internal/router"
	rt "github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestSSRRedirectFromNavigateDuringRender(t *testing.T) {
	app, err := New(Config{Component: func(ctx *rt.Ctx) work.Node {
		router.UseRouter(ctx).Navigate("/elsewhere")
		return work.BuildElement("p", work.NewText("never shown"))
	}})
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/elsewhere" {
		t.Fatalf("status %d location %q, want 302 /elsewhere", rec.Code, rec.Header().Get("Location"))
	}
}
