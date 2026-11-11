package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/upload"
	"github.com/eleven-am/pondlive/internal/work"
)

func uploadTestApp(t *testing.T, cfg upload.Config) *App {
	t.Helper()
	cfg.StoragePath = t.TempDir()
	app, err := New(Config{
		Component:    func(ctx *runtime.Ctx) work.Node { return &work.Element{Tag: "div"} },
		UploadConfig: &cfg,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(app.uploadHandler.Close)
	return app
}

func TestResolveUploadRegistryFindsTheSessionBySessionID(t *testing.T) {
	app := uploadTestApp(t, upload.Config{})
	component := func(ctx *runtime.Ctx) work.Node { return &work.Element{Tag: "div"} }
	sess := session.NewLiveSession("sess-1", 1, component, nil)
	defer sess.Close()
	app.registry.Put(sess)

	reg, ok := app.resolveUploadRegistry("sess-1")
	if !ok || reg != sess.UploadRegistry() {
		t.Fatalf("expected the session's own registry, got %v %v", reg, ok)
	}
	if _, ok := app.resolveUploadRegistry("missing"); ok {
		t.Fatal("an unknown session must not resolve")
	}
}

func TestUploadTokenOfOneSessionDoesNotWorkOnAnother(t *testing.T) {
	app := uploadTestApp(t, upload.Config{})
	component := func(ctx *runtime.Ctx) work.Node { return &work.Element{Tag: "div"} }
	a := session.NewLiveSession("sess-a", 1, component, nil)
	b := session.NewLiveSession("sess-b", 1, component, nil)
	defer a.Close()
	defer b.Close()
	app.registry.Put(a)
	app.registry.Put(b)

	token := a.UploadRegistry().Issue("sess-a", upload.GrantSpec{Files: 1})

	reg, ok := app.resolveUploadRegistry("sess-b")
	if !ok {
		t.Fatal("sess-b should resolve")
	}
	if _, _, err := reg.Reserve(token, 0, nil); err == nil {
		t.Fatal("a token issued for sess-a was accepted by sess-b's registry")
	}
}

func TestTusAssetIsSelfHostedWithoutSourceMapAndWithLicense(t *testing.T) {
	app := uploadTestApp(t, upload.Config{})

	for path, want := range map[string]string{
		"/static/tus.min.js":  "tus",
		"/static/tus.LICENSE": "MIT License",
		"/static/tus.NOTICE":  "tus-js-client 4.3.1",
	} {
		rr := httptest.NewRecorder()
		app.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodGet, path, nil))
		if rr.Code != http.StatusOK {
			t.Fatalf("%s: expected 200, got %d", path, rr.Code)
		}
		body, _ := io.ReadAll(rr.Body)
		if !strings.Contains(string(body), want) {
			t.Fatalf("%s: missing %q", path, want)
		}
		if strings.Contains(string(body), "sourceMappingURL") {
			t.Fatalf("%s still references a source map", path)
		}
	}
}

func TestUploadMountFollowsConfigAndIsAdvertisedToSessions(t *testing.T) {
	app := uploadTestApp(t, upload.Config{BasePath: "/files/in"})

	if app.sessionConfig.UploadEndpoint != "/files/in/" {
		t.Fatalf("session endpoint %q does not follow the mount", app.sessionConfig.UploadEndpoint)
	}
	if app.sessionConfig.UploadScriptURL != "/static/tus.min.js" {
		t.Fatalf("session tus script %q", app.sessionConfig.UploadScriptURL)
	}

	rr := httptest.NewRecorder()
	app.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/files/in/", nil))
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("the configured mount must be served by the upload handler (401 without token), got %d", rr.Code)
	}

	rr = httptest.NewRecorder()
	app.Handler().ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/tus/", nil))
	if rr.Code == http.StatusUnauthorized {
		t.Fatal("the default /tus/ mount must not exist when another base path is configured")
	}
}

func TestUploadDefaultMountIsTus(t *testing.T) {
	app := uploadTestApp(t, upload.Config{})
	if app.sessionConfig.UploadEndpoint != "/tus/" {
		t.Fatalf("got %q", app.sessionConfig.UploadEndpoint)
	}
}

func TestNoUploadConfigAdvertisesNothing(t *testing.T) {
	app, err := New(Config{Component: func(ctx *runtime.Ctx) work.Node { return &work.Element{Tag: "div"} }})
	if err != nil {
		t.Fatal(err)
	}
	if app.sessionConfig.UploadEndpoint != "" || app.sessionConfig.UploadScriptURL != "" {
		t.Fatal("upload endpoints must only be advertised when uploads are configured")
	}
}
