package upload

import (
	"bytes"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/exp/slog"
)

type tusEnv struct {
	t        *testing.T
	srv      *httptest.Server
	dir      string
	handler  *Handler
	registry *Registry
	mu       sync.Mutex
	others   map[string]*Registry
	done     chan FileInfo
}

func newTusEnv(t *testing.T, cfg Config) *tusEnv {
	t.Helper()
	env := &tusEnv{t: t, dir: t.TempDir(), registry: NewRegistry(), others: map[string]*Registry{}, done: make(chan FileInfo, 32)}
	cfg.StoragePath = env.dir
	cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	h, err := NewHandler(cfg, func(sessionID string) (*Registry, bool) {
		env.mu.Lock()
		defer env.mu.Unlock()
		if sessionID == "sess" {
			return env.registry, true
		}
		reg, ok := env.others[sessionID]
		return reg, ok
	})
	if err != nil {
		t.Fatal(err)
	}
	env.handler = h
	t.Cleanup(h.Close)
	mux := http.NewServeMux()
	mux.Handle(h.BasePath()+"/", http.StripPrefix(h.BasePath(), h))
	env.srv = httptest.NewServer(mux)
	t.Cleanup(env.srv.Close)
	return env
}

func (e *tusEnv) grant(spec GrantSpec) string {
	if spec.OnComplete == nil {
		spec.OnComplete = func(info FileInfo) error {
			e.done <- info
			return nil
		}
	}
	return e.registry.Issue("sess", spec)
}

func (e *tusEnv) request(method, url, token string, body []byte, headers map[string]string) *http.Response {
	e.t.Helper()
	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		e.t.Fatal(err)
	}
	req.Header.Set("Tus-Resumable", "1.0.0")
	if token != "" {
		req.Header.Set(TokenHeader, token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	e.t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func tusMeta(kv map[string]string) string {
	parts := make([]string, 0, len(kv))
	for k, v := range kv {
		parts = append(parts, k+" "+base64.StdEncoding.EncodeToString([]byte(v)))
	}
	return strings.Join(parts, ",")
}

func (e *tusEnv) create(token string, size int, name, typ string) *http.Response {
	return e.request(http.MethodPost, e.srv.URL+e.handler.BasePath()+"/", token, nil, map[string]string{
		"Upload-Length":   strconv.Itoa(size),
		"Upload-Metadata": tusMeta(map[string]string{"filename": name, "filetype": typ}),
	})
}

func (e *tusEnv) patch(location, token string, offset int, data []byte) *http.Response {
	return e.request(http.MethodPatch, location, token, data, map[string]string{
		"Content-Type":  "application/offset+octet-stream",
		"Upload-Offset": strconv.Itoa(offset),
	})
}

func (e *tusEnv) upload(token string, data []byte, name, typ string) (createResp, patchResp *http.Response) {
	e.t.Helper()
	createResp = e.create(token, len(data), name, typ)
	if createResp.StatusCode != http.StatusCreated {
		return createResp, nil
	}
	return createResp, e.patch(createResp.Header.Get("Location"), token, 0, data)
}

func (e *tusEnv) storedFiles() []string {
	entries, _ := os.ReadDir(e.dir)
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestTusUploadEndToEnd(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1})

	cr, pr := env.upload(token, pngBytes, "pixel.png", "image/png")
	if cr.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d", cr.StatusCode)
	}
	if pr.StatusCode != http.StatusNoContent {
		t.Fatalf("patch: %d", pr.StatusCode)
	}
	if !strings.Contains(cr.Header.Get("Location"), "/tus/") {
		t.Fatalf("location does not use the mount: %q", cr.Header.Get("Location"))
	}

	head := env.request(http.MethodHead, cr.Header.Get("Location"), token, nil, nil)
	if head.StatusCode != http.StatusForbidden {
		t.Fatalf("a completed upload belongs to the application; the client must not reach it again, got %d", head.StatusCode)
	}

	select {
	case info := <-env.done:
		if info.MetaData["filename"] != "pixel.png" || info.Size != int64(len(pngBytes)) {
			t.Fatalf("unexpected completion info %+v", info)
		}
		stored, err := os.ReadFile(info.Storage["Path"])
		if err != nil || !bytes.Equal(stored, pngBytes) {
			t.Fatalf("stored file mismatch: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("completion was not delivered")
	}
}

func TestTusUploadInChunks(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1, Accept: []string{"image/png"}})

	cr := env.create(token, len(pngBytes), "pixel.png", "image/png")
	loc := cr.Header.Get("Location")
	half := len(pngBytes) / 2
	if r := env.patch(loc, token, 0, pngBytes[:half]); r.StatusCode != http.StatusNoContent {
		t.Fatalf("first chunk: %d", r.StatusCode)
	}
	head := env.request(http.MethodHead, loc, token, nil, nil)
	if head.Header.Get("Upload-Offset") != strconv.Itoa(half) {
		t.Fatalf("resume offset %q", head.Header.Get("Upload-Offset"))
	}
	if r := env.patch(loc, token, half, pngBytes[half:]); r.StatusCode != http.StatusNoContent {
		t.Fatalf("second chunk: %d", r.StatusCode)
	}
	select {
	case <-env.done:
	case <-time.After(time.Second):
		t.Fatal("completion was not delivered")
	}
}

func TestTusWrongContentTypeIs415AndFileDeleted(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 2, Accept: []string{"image/png"}})

	text := []byte("this is not a png, it only claims to be one")
	cr, pr := env.upload(token, text, "evil.png", "image/png")
	if cr.StatusCode != http.StatusCreated {
		t.Fatalf("the declared type is fine, create should pass: %d", cr.StatusCode)
	}
	if pr.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415 from the content sniff, got %d", pr.StatusCode)
	}
	if files := env.storedFiles(); len(files) != 0 {
		t.Fatalf("the rejected file was not deleted: %v", files)
	}
	select {
	case info := <-env.done:
		t.Fatalf("a rejected upload was completed: %+v", info)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestTusServerAllowedTypesAreSniffedToo(t *testing.T) {
	env := newTusEnv(t, Config{AllowedTypes: []string{"image/png"}})
	token := env.grant(GrantSpec{Files: 1})
	_, pr := env.upload(token, []byte("plain text"), "x.png", "image/png")
	if pr == nil || pr.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %v", pr)
	}
	if files := env.storedFiles(); len(files) != 0 {
		t.Fatalf("file not deleted: %v", files)
	}
}

func TestTusDeclaredTypeMismatchIs415AtCreate(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1, Accept: []string{"image/*"}})
	if cr := env.create(token, 10, "a.pdf", "application/pdf"); cr.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415, got %d", cr.StatusCode)
	}
	if _, _, err := env.registry.Reserve(token, 0, nil); err != nil {
		t.Fatalf("a rejected create burned the grant: %v", err)
	}
}

func TestTusAcceptSemanticsMatchBetweenDeclaredAndSniffed(t *testing.T) {
	for _, accept := range [][]string{{"image/*"}, {".png"}, {"image/png"}} {
		env := newTusEnv(t, Config{})
		token := env.grant(GrantSpec{Files: 1, Accept: accept})
		cr, pr := env.upload(token, pngBytes, "pixel.png", "image/png")
		if cr.StatusCode != http.StatusCreated || pr.StatusCode != http.StatusNoContent {
			t.Fatalf("accept %v: create=%d patch=%d", accept, cr.StatusCode, pr.StatusCode)
		}
	}
}

func TestTusReusedTokenIsRejected(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1})

	if _, pr := env.upload(token, pngBytes, "a.png", "image/png"); pr.StatusCode != http.StatusNoContent {
		t.Fatalf("first upload: %d", pr.StatusCode)
	}
	if cr := env.create(token, len(pngBytes), "b.png", "image/png"); cr.StatusCode != http.StatusForbidden {
		t.Fatalf("reused token: expected 403, got %d", cr.StatusCode)
	}
}

func TestTusConcurrentCreatesOnOneTokenAdmitExactlyOne(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1})

	var created, forbidden atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 24; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			switch env.create(token, 10, "a.png", "image/png").StatusCode {
			case http.StatusCreated:
				created.Add(1)
			case http.StatusForbidden:
				forbidden.Add(1)
			}
		}()
	}
	wg.Wait()
	if created.Load() != 1 || forbidden.Load() != 23 {
		t.Fatalf("created=%d forbidden=%d", created.Load(), forbidden.Load())
	}
}

func TestTusMultipleIsEnforcedByTheServer(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 2})
	for i := 0; i < 2; i++ {
		if cr := env.create(token, 10, "a.png", "image/png"); cr.StatusCode != http.StatusCreated {
			t.Fatalf("file %d: %d", i, cr.StatusCode)
		}
	}
	if cr := env.create(token, 10, "a.png", "image/png"); cr.StatusCode != http.StatusForbidden {
		t.Fatalf("third file: expected 403, got %d", cr.StatusCode)
	}
}

func TestTusRequiresTheGrantOnEveryMethod(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1})
	cr := env.create(token, len(pngBytes), "a.png", "image/png")
	loc := cr.Header.Get("Location")
	if r := env.patch(loc, token, 0, pngBytes[:10]); r.StatusCode != http.StatusNoContent {
		t.Fatalf("setup patch: %d", r.StatusCode)
	}

	if r := env.request(http.MethodGet, loc, token, nil, nil); r.StatusCode != http.StatusOK {
		t.Fatalf("the grant holder may download: %d", r.StatusCode)
	}
	if r := env.request(http.MethodHead, loc, token, nil, nil); r.Header.Get("Upload-Offset") != "10" {
		t.Fatalf("the grant holder may HEAD: offset %q", r.Header.Get("Upload-Offset"))
	}

	otherToken := env.grant(GrantSpec{Files: 1})

	for _, method := range []string{http.MethodGet, http.MethodDelete, http.MethodHead, http.MethodPatch} {
		if r := env.request(method, loc, "", nil, nil); r.StatusCode != http.StatusUnauthorized {
			t.Errorf("%s without a token: expected 401, got %d", method, r.StatusCode)
		}
		if r := env.request(method, loc, "sess.forged", nil, nil); r.StatusCode != http.StatusForbidden {
			t.Errorf("%s with a forged token: expected 403, got %d", method, r.StatusCode)
		}
		if r := env.request(method, loc, "nosuchsession.token", nil, nil); r.StatusCode != http.StatusForbidden {
			t.Errorf("%s with an unknown session: expected 403, got %d", method, r.StatusCode)
		}
		if r := env.request(method, loc, otherToken, nil, nil); r.StatusCode != http.StatusForbidden {
			t.Errorf("%s with another grant's token: expected 403, got %d", method, r.StatusCode)
		}
	}
	if files := env.storedFiles(); len(files) == 0 {
		t.Fatal("an unauthorised request deleted the upload")
	}
}

func TestTusCompletedUploadCannotBeDeletedByTheClient(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 2})
	cr, pr := env.upload(token, pngBytes, "a.png", "image/png")
	if pr.StatusCode != http.StatusNoContent {
		t.Fatalf("setup upload: %d", pr.StatusCode)
	}
	loc := cr.Header.Get("Location")

	for _, method := range []string{http.MethodDelete, http.MethodGet} {
		if r := env.request(method, loc, token, nil, nil); r.StatusCode != http.StatusForbidden {
			t.Errorf("%s of a completed upload: expected 403, got %d", method, r.StatusCode)
		}
	}
	if files := env.storedFiles(); len(files) == 0 {
		t.Fatal("the application's file was deleted by the client")
	}
}

func TestTusCreateWithoutTokenIs401(t *testing.T) {
	env := newTusEnv(t, Config{})
	if cr := env.create("", 10, "a.png", "image/png"); cr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", cr.StatusCode)
	}
}

func TestTusGrantFromAnotherSessionCannotTouchUpload(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1})
	cr, _ := env.upload(token, pngBytes, "a.png", "image/png")
	loc := cr.Header.Get("Location")

	other := NewRegistry()
	env.mu.Lock()
	env.others["sess2"] = other
	env.mu.Unlock()
	foreign := other.Issue("sess2", GrantSpec{Files: 1})

	if r := env.request(http.MethodDelete, loc, foreign, nil, nil); r.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", r.StatusCode)
	}
}

func TestTusDeleteByGrantHolderTerminates(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1})
	cr := env.create(token, 100, "a.bin", "application/octet-stream")
	loc := cr.Header.Get("Location")

	if r := env.request(http.MethodDelete, loc, token, nil, nil); r.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", r.StatusCode)
	}
	if files := env.storedFiles(); len(files) != 0 {
		t.Fatalf("terminated upload left files behind: %v", files)
	}
	if r := env.request(http.MethodHead, loc, token, nil, nil); r.StatusCode != http.StatusForbidden {
		t.Fatalf("a terminated upload must no longer be reachable, got %d", r.StatusCode)
	}
}

func TestTusRejectsDeferredLength(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1})
	resp := env.request(http.MethodPost, env.srv.URL+"/tus/", token, nil, map[string]string{"Upload-Defer-Length": "1"})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
	if _, _, err := env.registry.Reserve(token, 0, nil); err != nil {
		t.Fatalf("rejected request burned the grant: %v", err)
	}
}

func TestTusOversizeIs413(t *testing.T) {
	env := newTusEnv(t, Config{MaxSize: 100})
	token := env.grant(GrantSpec{Files: 2, MaxSize: 50})
	if cr := env.create(token, 101, "a", "x/y"); cr.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("server limit: expected 413, got %d", cr.StatusCode)
	}
	if cr := env.create(token, 51, "a", "x/y"); cr.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("grant limit: expected 413, got %d", cr.StatusCode)
	}
	if cr := env.create(token, 50, "a", "x/y"); cr.StatusCode != http.StatusCreated {
		t.Fatalf("within limits: expected 201, got %d", cr.StatusCode)
	}
}

func TestTusPathTraversalIDIs404(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1})
	for _, p := range []string{"/tus/..%2f..%2fetc", "/tus/abc", "/tus/" + strings.Repeat("a", 31), "/tus/" + strings.Repeat("g", 32)} {
		if r := env.request(http.MethodGet, env.srv.URL+p, token, nil, nil); r.StatusCode != http.StatusNotFound && r.StatusCode != http.StatusMovedPermanently {
			t.Errorf("%s: expected 404, got %d", p, r.StatusCode)
		}
	}
}

func TestTusExpiredGrantIs403(t *testing.T) {
	env := newTusEnv(t, Config{})
	now := time.Unix(5000, 0)
	env.registry.now = func() time.Time { return now }
	token := env.grant(GrantSpec{Files: 1})
	now = now.Add(DefaultGrantTTL + time.Second)
	if cr := env.create(token, 10, "a", "x/y"); cr.StatusCode != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", cr.StatusCode)
	}
}

func TestTusSlowCompletionDoesNotBlockOtherUploads(t *testing.T) {
	env := newTusEnv(t, Config{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	entered := make(chan struct{}, 1)
	slow := env.grant(GrantSpec{Files: 1, OnComplete: func(FileInfo) error {
		entered <- struct{}{}
		<-release
		return nil
	}})
	fast := env.grant(GrantSpec{Files: 1})

	slowDone := make(chan struct{})
	go func() {
		defer close(slowDone)
		env.upload(slow, pngBytes, "slow.png", "image/png")
	}()
	<-entered

	fastDone := make(chan int, 1)
	go func() {
		_, pr := env.upload(fast, pngBytes, "fast.png", "image/png")
		fastDone <- pr.StatusCode
	}()

	select {
	case code := <-fastDone:
		if code != http.StatusNoContent {
			t.Fatalf("fast upload: %d", code)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("a slow OnComplete blocked another upload's completion")
	}
	unblock()
	<-slowDone
}

func TestTusCompletionFailureIs503AndFileDeleted(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1, OnComplete: func(FileInfo) error { return io.ErrClosedPipe }})
	_, pr := env.upload(token, pngBytes, "a.png", "image/png")
	if pr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", pr.StatusCode)
	}
	if files := env.storedFiles(); len(files) != 0 {
		t.Fatalf("file not deleted: %v", files)
	}
}

func TestTusCORSIsDisabled(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1})
	resp := env.request(http.MethodPost, env.srv.URL+"/tus/", token, nil, map[string]string{
		"Origin":          "https://evil.example",
		"Upload-Length":   "1",
		"Upload-Metadata": tusMeta(map[string]string{"filename": "a", "filetype": "x/y"}),
	})
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("cross-origin access must not be granted, got %q", got)
	}
}

func TestTusForwardedHeadersOnlyWhenConfigured(t *testing.T) {
	for _, respect := range []bool{false, true} {
		env := newTusEnv(t, Config{RespectForwardedHeaders: respect})
		token := env.grant(GrantSpec{Files: 1})
		resp := env.request(http.MethodPost, env.srv.URL+"/tus/", token, nil, map[string]string{
			"X-Forwarded-Host":  "public.example",
			"X-Forwarded-Proto": "https",
			"Upload-Length":     "1",
			"Upload-Metadata":   tusMeta(map[string]string{"filename": "a", "filetype": "x/y"}),
		})
		loc := resp.Header.Get("Location")
		if respect != strings.HasPrefix(loc, "https://public.example/") {
			t.Fatalf("respect=%v location=%q", respect, loc)
		}
	}
}

func TestCustomBasePath(t *testing.T) {
	env := newTusEnv(t, Config{BasePath: "files/up/"})
	if env.handler.BasePath() != "/files/up" {
		t.Fatalf("got %q", env.handler.BasePath())
	}
	token := env.grant(GrantSpec{Files: 1})
	resp := env.request(http.MethodPost, env.srv.URL+"/files/up/", token, nil, map[string]string{
		"Upload-Length":   "1",
		"Upload-Metadata": tusMeta(map[string]string{"filename": "a", "filetype": "x/y"}),
	})
	if resp.StatusCode != http.StatusCreated || !strings.Contains(resp.Header.Get("Location"), "/files/up/") {
		t.Fatalf("status=%d location=%q", resp.StatusCode, resp.Header.Get("Location"))
	}
}

func TestNewHandlerRequiresResolver(t *testing.T) {
	if _, err := NewHandler(Config{StoragePath: t.TempDir()}, nil); err == nil {
		t.Fatal("expected an error without a resolver")
	}
}

func TestNewHandlerDefaults(t *testing.T) {
	dir := t.TempDir()
	h, err := NewHandler(Config{StoragePath: dir}, func(string) (*Registry, bool) { return nil, false })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	if h.BasePath() != DefaultBasePath {
		t.Fatalf("got %q", h.BasePath())
	}
	if _, err := os.Stat(filepath.Join(dir)); err != nil {
		t.Fatal(err)
	}
}

func TestTusRevokingAGrantDeletesItsIncompleteUploads(t *testing.T) {
	env := newTusEnv(t, Config{})
	chunk := bytes.Repeat([]byte("p"), 1023)
	prev := ""
	for i := 0; i < 50; i++ {
		env.registry.Revoke(prev)
		prev = env.grant(GrantSpec{Files: 1})
		cr := env.create(prev, 1024, "a.bin", "application/octet-stream")
		if cr.StatusCode != http.StatusCreated {
			t.Fatalf("round %d create: %d", i, cr.StatusCode)
		}
		if pr := env.patch(cr.Header.Get("Location"), prev, 0, chunk); pr.StatusCode != http.StatusNoContent {
			t.Fatalf("round %d patch: %d", i, pr.StatusCode)
		}
	}
	env.registry.Revoke(prev)
	if files := env.storedFiles(); len(files) != 0 {
		t.Fatalf("orphaned partial uploads remain after revoke: %d files", len(files))
	}
}

func TestTusGrantFloodIsBoundedByTheGrantCap(t *testing.T) {
	env := newTusEnv(t, Config{MaxGrantsPerSession: 4})
	chunk := bytes.Repeat([]byte("p"), 1023)
	for i := 0; i < 100; i++ {
		token := env.grant(GrantSpec{Files: 1})
		cr := env.create(token, 1024, "a.bin", "application/octet-stream")
		if cr.StatusCode == http.StatusCreated {
			env.patch(cr.Header.Get("Location"), token, 0, chunk)
		}
	}
	if files := env.storedFiles(); len(files) > 2*4 {
		t.Fatalf("a grant flood left %d files, want at most %d", len(files), 2*4)
	}
	if env.registry.Len() > 4 {
		t.Fatalf("live grants exceed the cap: %d", env.registry.Len())
	}
}

func TestTusClosingTheSessionDeletesIncompleteUploads(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 3})
	for i := 0; i < 3; i++ {
		cr := env.create(token, 1024, "a.bin", "application/octet-stream")
		env.patch(cr.Header.Get("Location"), token, 0, bytes.Repeat([]byte("p"), 100))
	}
	if len(env.storedFiles()) == 0 {
		t.Fatal("setup left no partial uploads")
	}
	env.registry.Close()
	if files := env.storedFiles(); len(files) != 0 {
		t.Fatalf("closing the session left %d files", len(files))
	}
}

func TestTusCompletedUploadsSurviveRevoke(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 2})
	if _, pr := env.upload(token, pngBytes, "a.png", "image/png"); pr.StatusCode != http.StatusNoContent {
		t.Fatalf("setup: %d", pr.StatusCode)
	}
	env.registry.Revoke(token)
	if len(env.storedFiles()) != 2 {
		t.Fatalf("the application's completed file was deleted: %v", env.storedFiles())
	}
}

func TestTusDefaultMaxSizeRejectsHugeDeclarations(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 100})
	if cr := env.create(token, DefaultMaxSize+1, "a.bin", "application/octet-stream"); cr.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 over the default limit, got %d", cr.StatusCode)
	}
	huge := env.request(http.MethodPost, env.srv.URL+"/tus/", token, nil, map[string]string{"Upload-Length": "1099511627776"})
	if huge.StatusCode != http.StatusRequestEntityTooLarge {
		t.Fatalf("1 TiB declaration: expected 413, got %d", huge.StatusCode)
	}
	if cr := env.create(token, DefaultMaxSize, "a.bin", "application/octet-stream"); cr.StatusCode != http.StatusCreated {
		t.Fatalf("exactly the default limit must be allowed, got %d", cr.StatusCode)
	}
}

func TestTusNegativeMaxSizeMeansUnlimited(t *testing.T) {
	env := newTusEnv(t, Config{MaxSize: -1})
	token := env.grant(GrantSpec{Files: 1})
	if cr := env.create(token, DefaultMaxSize*4, "a.bin", "application/octet-stream"); cr.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 with MaxSize disabled, got %d", cr.StatusCode)
	}
}

func TestTusSessionBudgetIs429(t *testing.T) {
	env := newTusEnv(t, Config{SessionBudget: 1000})
	token := env.grant(GrantSpec{Files: 5})
	if cr := env.create(token, 600, "a.bin", "application/octet-stream"); cr.StatusCode != http.StatusCreated {
		t.Fatalf("first: %d", cr.StatusCode)
	}
	if cr := env.create(token, 600, "a.bin", "application/octet-stream"); cr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("over budget: expected 429, got %d", cr.StatusCode)
	}
}

func TestTusAcceptByExtensionUsesContent(t *testing.T) {
	md := []byte("# Title\n\nhello world\n")
	for _, tc := range []struct {
		name   string
		accept []string
		file   string
		typ    string
		data   []byte
		want   int
	}{
		{"markdown by extension", []string{".md"}, "readme.md", "text/markdown", md, http.StatusNoContent},
		{"markdown declared as plain text", []string{".md"}, "readme.md", "text/plain", md, http.StatusNoContent},
		{"csv by extension", []string{".csv"}, "a.csv", "text/csv", []byte("a,b,c\n1,2,3\n4,5,6\n"), http.StatusNoContent},
		{"markdown by type", []string{"text/markdown"}, "readme.md", "text/markdown", md, http.StatusNoContent},
		{"png renamed to .md", []string{".md"}, "x.md", "text/markdown", pngBytes, http.StatusUnsupportedMediaType},
		{"exe-like binary renamed to .png", []string{".png"}, "a.png", "image/png", append([]byte("MZ\x90\x00\x03\x00"), make([]byte, 100)...), http.StatusUnsupportedMediaType},
		{"jpeg by .jpeg", []string{".jpeg"}, "a.jpeg", "image/jpeg", []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00"), http.StatusNoContent},
		{"name must still carry the accepted extension at create", []string{".jpeg"}, "a.jpg", "image/jpeg", []byte("\xff\xd8\xff\xe0\x00\x10JFIF\x00\x01\x01\x00\x00\x01\x00\x01\x00\x00"), http.StatusUnsupportedMediaType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newTusEnv(t, Config{})
			token := env.grant(GrantSpec{Files: 1, Accept: tc.accept})
			cr, pr := env.upload(token, tc.data, tc.file, tc.typ)
			got := cr.StatusCode
			if pr != nil {
				got = pr.StatusCode
			}
			if got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got)
			}
		})
	}
}

func TestTusIncompatibleDeclaredTypeIsRejectedBeforeAnyBytesAreSent(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1, Accept: []string{".png"}})
	if cr := env.create(token, 1<<20, "a.png", "application/pdf"); cr.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("expected 415 at create, got %d", cr.StatusCode)
	}
	if files := env.storedFiles(); len(files) != 0 {
		t.Fatalf("nothing may be stored for an early rejection: %v", files)
	}
}

func TestSweepRemovesStaleIncompleteUploadsOnly(t *testing.T) {
	env := newTusEnv(t, Config{SweepInterval: -1})
	token := env.grant(GrantSpec{Files: 4})

	partial := func() string {
		cr := env.create(token, 1024, "a.bin", "application/octet-stream")
		env.patch(cr.Header.Get("Location"), token, 0, bytes.Repeat([]byte("p"), 100))
		return strings.TrimPrefix(cr.Header.Get("Location"), env.srv.URL+"/tus/")
	}
	stale := partial()
	fresh := partial()
	cr, pr := env.upload(token, pngBytes, "a.png", "image/png")
	if pr.StatusCode != http.StatusNoContent {
		t.Fatal(pr.StatusCode)
	}
	complete := strings.TrimPrefix(cr.Header.Get("Location"), env.srv.URL+"/tus/")

	env.handler.untrack(env.registry)

	old := time.Now().Add(-2 * DefaultGrantTTL)
	for _, id := range []string{stale, complete} {
		for _, name := range []string{id, id + ".info"} {
			if err := os.Chtimes(filepath.Join(env.dir, name), old, old); err != nil {
				t.Fatal(err)
			}
		}
	}
	orphanBin := filepath.Join(env.dir, strings.Repeat("ab", 16))
	if err := os.WriteFile(orphanBin, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(orphanBin, old, old); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(env.dir, "notes.txt")
	if err := os.WriteFile(foreign, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(foreign, old, old); err != nil {
		t.Fatal(err)
	}

	env.handler.sweep(time.Now())

	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(env.dir, name))
		return err == nil
	}
	if exists(stale) || exists(stale+".info") {
		t.Error("a stale incomplete upload survived the sweep")
	}
	if !exists(fresh) || !exists(fresh+".info") {
		t.Error("a fresh incomplete upload was swept")
	}
	if !exists(complete) || !exists(complete+".info") {
		t.Error("a completed upload was swept")
	}
	if exists(filepath.Base(orphanBin)) {
		t.Error("a stale bin without an info file survived")
	}
	if !exists("notes.txt") {
		t.Error("an unrelated file was deleted")
	}
}

func TestSweepRunsAtStartup(t *testing.T) {
	dir := t.TempDir()
	id := strings.Repeat("cd", 16)
	info := `{"ID":"` + id + `","Size":1024,"Offset":0,"MetaData":{},"Storage":{"Type":"filestore","Path":"` + filepath.Join(dir, id) + `"}}`
	for name, content := range map[string]string{id: "partial", id + ".info": info} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		old := time.Now().Add(-2 * DefaultGrantTTL)
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	h, err := NewHandler(Config{StoragePath: dir, SweepInterval: -1}, func(string) (*Registry, bool) { return nil, false })
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("startup sweep left %d entries", len(entries))
	}
}

func TestPeriodicSweepRunsAndCloseStopsIt(t *testing.T) {
	dir := t.TempDir()
	h, err := NewHandler(Config{StoragePath: dir, SweepInterval: 10 * time.Millisecond}, func(string) (*Registry, bool) { return nil, false })
	if err != nil {
		t.Fatal(err)
	}
	id := strings.Repeat("ef", 16)
	bin := filepath.Join(dir, id)
	if err := os.WriteFile(bin, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-2 * DefaultGrantTTL)
	if err := os.Chtimes(bin, old, old); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, err := os.Stat(bin); err != nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the periodic sweep never removed the stale file")
		}
		time.Sleep(5 * time.Millisecond)
	}
	done := make(chan struct{})
	go func() { h.Close(); h.Close(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Close did not stop the sweep goroutine")
	}
}

func TestSweepAgreesWithTheGrantAboutLiveness(t *testing.T) {
	env := newTusEnv(t, Config{SweepInterval: -1})
	token := env.grant(GrantSpec{Files: 2})
	cr := env.create(token, 1024, "a.bin", "application/octet-stream")
	id := strings.TrimPrefix(cr.Header.Get("Location"), env.srv.URL+"/tus/")
	env.patch(cr.Header.Get("Location"), token, 0, bytes.Repeat([]byte("p"), 100))

	old := time.Now().Add(-2 * DefaultGrantTTL)
	for _, name := range []string{id, id + ".info"} {
		if err := os.Chtimes(filepath.Join(env.dir, name), old, old); err != nil {
			t.Fatal(err)
		}
	}
	before := env.registry.Inflight()

	env.handler.sweep(time.Now())

	if _, err := os.Stat(filepath.Join(env.dir, id)); err != nil {
		t.Fatal("the sweep deleted an upload whose grant is still alive")
	}
	if env.registry.Inflight() != before || before == 0 {
		t.Fatalf("budget charge changed (%d to %d) although the upload survived", before, env.registry.Inflight())
	}
	if r := env.request(http.MethodHead, cr.Header.Get("Location"), token, nil, nil); r.Header.Get("Upload-Offset") != "100" {
		t.Fatalf("the upload must still be resumable, got offset %q", r.Header.Get("Upload-Offset"))
	}
}

func TestSweepExpiresIdleGrantsAndReapsTheirUploads(t *testing.T) {
	env := newTusEnv(t, Config{SweepInterval: -1})
	now := time.Now()
	env.registry.now = func() time.Time { return now }
	token := env.grant(GrantSpec{Files: 1})
	cr := env.create(token, 1024, "a.bin", "application/octet-stream")
	env.patch(cr.Header.Get("Location"), token, 0, bytes.Repeat([]byte("p"), 100))

	now = now.Add(DefaultGrantTTL + time.Minute)
	env.handler.sweep(now)

	if files := env.storedFiles(); len(files) != 0 {
		t.Fatalf("an expired grant's partial upload survived the sweep: %v", files)
	}
	if env.registry.Inflight() != 0 || env.registry.Len() != 0 {
		t.Fatalf("expired grant still holds budget: inflight=%d grants=%d", env.registry.Inflight(), env.registry.Len())
	}
}

func TestCompletionRacingRevokeNeverDeletesTheCompletedFile(t *testing.T) {
	env := newTusEnv(t, Config{})
	entered := make(chan struct{})
	release := make(chan struct{})
	completed := make(chan FileInfo, 1)
	token := env.registry.Issue("sess", GrantSpec{Files: 1, OnComplete: func(info FileInfo) error {
		close(entered)
		<-release
		completed <- info
		return nil
	}})
	res := make(chan int, 1)
	go func() {
		_, p := env.upload(token, []byte("hello"), "a.txt", "text/plain")
		res <- p.StatusCode
	}()
	<-entered
	env.registry.Revoke(token)
	env.registry.Close()
	close(release)
	if code := <-res; code != http.StatusNoContent {
		t.Fatalf("expected the finished upload to be acknowledged, got %d", code)
	}
	info := <-completed
	if _, err := os.Stat(info.Storage["Path"]); err != nil {
		t.Fatalf("the application was told the file is complete but it was deleted: %v", err)
	}
}

func TestCompletionRacingRevokeStatistically(t *testing.T) {
	env := newTusEnv(t, Config{})
	data := bytes.Repeat([]byte("x"), 64<<10)
	for i := 0; i < 150; i++ {
		got := make(chan FileInfo, 1)
		token := env.registry.Issue("sess", GrantSpec{Files: 1, Accept: []string{"text/plain"}, OnComplete: func(info FileInfo) error {
			got <- info
			return nil
		}})
		cr := env.create(token, len(data), "a.txt", "text/plain")
		if cr.StatusCode != http.StatusCreated {
			t.Fatalf("create: %d", cr.StatusCode)
		}
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			env.patch(cr.Header.Get("Location"), token, 0, data)
		}()
		go func() {
			defer wg.Done()
			time.Sleep(time.Duration(100+i*4) * time.Microsecond)
			env.registry.Revoke(token)
		}()
		wg.Wait()
		select {
		case info := <-got:
			if _, err := os.Stat(info.Storage["Path"]); err != nil {
				t.Fatalf("round %d: OnComplete fired but the file was deleted", i)
			}
		default:
		}
	}
}

func TestCompletionOnAClosedSessionDeletesTheFile(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1, OnComplete: func(FileInfo) error { return io.ErrClosedPipe }})
	_, pr := env.upload(token, pngBytes, "a.png", "image/png")
	if pr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", pr.StatusCode)
	}
	if files := env.storedFiles(); len(files) != 0 {
		t.Fatalf("a completion that could not be delivered must not leave the file behind: %v", files)
	}
}

func TestLongPatchOutlivesTheGrantTTL(t *testing.T) {
	env := newTusEnv(t, Config{})
	var mu sync.Mutex
	clock := time.Now()
	env.registry.now = func() time.Time {
		mu.Lock()
		defer mu.Unlock()
		return clock
	}
	token := env.grant(GrantSpec{Files: 1})
	cr := env.create(token, 10, "a.bin", "application/octet-stream")
	pr, pw := io.Pipe()
	res := make(chan int, 1)
	go func() {
		req, _ := http.NewRequest(http.MethodPatch, cr.Header.Get("Location"), pr)
		req.Header.Set("Tus-Resumable", "1.0.0")
		req.Header.Set(TokenHeader, token)
		req.Header.Set("Content-Type", "application/offset+octet-stream")
		req.Header.Set("Upload-Offset", "0")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			res <- -1
			return
		}
		resp.Body.Close()
		res <- resp.StatusCode
	}()
	pw.Write([]byte("12345"))
	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	clock = clock.Add(DefaultGrantTTL + time.Minute)
	mu.Unlock()
	env.registry.Issue("sess", GrantSpec{})
	env.handler.sweep(clock)
	pw.Write([]byte("67890"))
	pw.Close()
	if code := <-res; code != http.StatusNoContent {
		t.Fatalf("a PATCH that was still streaming was killed by grant expiry: %d", code)
	}
}

func TestUnknownExtensionIsRejectedAtCreate(t *testing.T) {
	env := newTusEnv(t, Config{})
	token := env.grant(GrantSpec{Files: 1, Accept: []string{".zzqq"}})
	if cr := env.create(token, 1<<20, "a.zzqq", ""); cr.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("an extension that cannot be verified must be refused before any bytes are sent, got %d", cr.StatusCode)
	}
}

func TestGlobalBudgetBoundsIncompleteBytesAcrossSessions(t *testing.T) {
	env := newTusEnv(t, Config{GlobalBudget: 1000})
	other := NewRegistry()
	env.mu.Lock()
	env.others["sess2"] = other
	env.mu.Unlock()
	a := env.grant(GrantSpec{Files: 3})
	b := other.Issue("sess2", GrantSpec{Files: 3})
	if cr := env.create(a, 600, "a.bin", ""); cr.StatusCode != http.StatusCreated {
		t.Fatalf("first: %d", cr.StatusCode)
	}
	if cr := env.create(b, 600, "a.bin", ""); cr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("a second session must hit the global cap, got %d", cr.StatusCode)
	}
	env.registry.Close()
	if cr := env.create(b, 600, "a.bin", ""); cr.StatusCode != http.StatusCreated {
		t.Fatalf("closing the first session must free the global budget, got %d", cr.StatusCode)
	}
}

func TestClientDeleteDuringCompletionIsRefusedAndTheFileSurvives(t *testing.T) {
	env := newTusEnv(t, Config{})
	entered := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	completed := make(chan FileInfo, 1)
	token := env.registry.Issue("sess", GrantSpec{Files: 1, OnComplete: func(info FileInfo) error {
		close(entered)
		<-release
		completed <- info
		return nil
	}})
	cr := env.create(token, len(pngBytes), "a.png", "image/png")
	loc := cr.Header.Get("Location")
	res := make(chan int, 1)
	go func() { res <- env.patch(loc, token, 0, pngBytes).StatusCode }()
	<-entered

	del := env.request(http.MethodDelete, loc, token, nil, nil)
	if del.StatusCode != http.StatusConflict {
		t.Fatalf("DELETE of a finishing upload: expected 409, got %d", del.StatusCode)
	}
	unblock()
	if code := <-res; code != http.StatusNoContent {
		t.Fatalf("expected the completion to be acknowledged, got %d", code)
	}
	info := <-completed
	if _, err := os.Stat(info.Storage["Path"]); err != nil {
		t.Fatalf("a client DELETE removed the file the application was told about: %v", err)
	}
}
