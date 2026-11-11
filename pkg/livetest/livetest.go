package livetest

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/goroutines"
	rt "github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/pkg"
)

const shutdownBound = 5 * time.Second

func AssertNothingStartsBeforeConnect(t testing.TB, root func() pkg.Node, opts ...pkg.AppOption) {
	t.Helper()
	known := goroutineIDs()
	effectsBefore := rt.EffectRuns()
	app, err := pkg.NewApp(root, opts...)
	if err != nil {
		t.Fatalf("livetest: building the app: %v", err)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code >= http.StatusInternalServerError {
		t.Errorf("livetest: GET / answered %d", rec.Code)
	}
	if n := rt.EffectRuns() - effectsBefore; n != 0 {
		t.Errorf("livetest: %d UseEffect callbacks ran during a request that never connected a socket", n)
	}
	for _, g := range userGoroutines(known) {
		t.Errorf("livetest: a goroutine was started while rendering and is still running (created by %s):\n%s", g.Creator, g.Stack)
	}
	ctx, cancel := context.WithTimeout(context.Background(), shutdownBound)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		t.Errorf("livetest: shutting the app down: %v", err)
	}
	if leaked := userGoroutines(known); len(leaked) > 0 {
		t.Errorf("livetest: %d goroutine(s) started during rendering survived shutdown", len(leaked))
	}
}

func goroutineIDs() map[string]struct{} {
	ids := make(map[string]struct{})
	for _, g := range goroutines.Snapshot() {
		ids[g.ID] = struct{}{}
	}
	return ids
}

func userGoroutines(known map[string]struct{}) []goroutines.Goroutine {
	var out []goroutines.Goroutine
	for _, g := range goroutines.Snapshot() {
		if _, ok := known[g.ID]; ok || g.Creator == "" || ownedByFramework(g.Creator) {
			continue
		}
		out = append(out, g)
	}
	return out
}

func ownedByFramework(creator string) bool {
	if strings.HasPrefix(creator, "github.com/eleven-am/pondlive/internal/") || strings.HasPrefix(creator, "github.com/eleven-am/pondlive/pkg.") {
		return true
	}
	return standardLibrary(creator)
}

func standardLibrary(fn string) bool {
	slash := strings.LastIndex(fn, "/")
	rest := fn[slash+1:]
	dot := strings.Index(rest, ".")
	if dot < 0 {
		return true
	}
	pkgPath := fn[:slash+1] + rest[:dot]
	first, _, _ := strings.Cut(pkgPath, "/")
	return !strings.Contains(first, ".")
}
