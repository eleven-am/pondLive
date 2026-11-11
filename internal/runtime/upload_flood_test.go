package runtime

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/upload"
)

type floodEnv struct {
	f   *uploadFixture
	srv *httptest.Server
	dir string
}

func newFloodEnv(t *testing.T) *floodEnv {
	t.Helper()
	f := newUploadFixture(t)
	dir := t.TempDir()
	h, err := upload.NewHandler(upload.Config{StoragePath: dir, SweepInterval: -1}, func(id string) (*upload.Registry, bool) {
		if id == f.sess.SessionID {
			return f.sess.UploadRegistry, true
		}
		return nil, false
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(h.Close)
	mux := http.NewServeMux()
	mux.Handle("/tus/", http.StripPrefix("/tus", h))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &floodEnv{f: f, srv: srv, dir: dir}
}

func (e *floodEnv) do(method, url, token string, body []byte, hdr map[string]string) (int, http.Header) {
	req, _ := http.NewRequest(method, url, bytes.NewReader(body))
	req.Header.Set("Tus-Resumable", "1.0.0")
	req.Header.Set(upload.TokenHeader, token)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return -1, nil
	}
	resp.Body.Close()
	return resp.StatusCode, resp.Header
}

func (e *floodEnv) entries() int {
	list, _ := os.ReadDir(e.dir)
	return len(list)
}

func (e *floodEnv) partialUpload(token string, size int) {
	code, h := e.do("POST", e.srv.URL+"/tus/", token, nil, map[string]string{"Upload-Length": strconv.Itoa(size)})
	if code != http.StatusCreated {
		return
	}
	e.do("PATCH", h.Get("Location"), token, bytes.Repeat([]byte("z"), size-1), map[string]string{"Content-Type": "application/offset+octet-stream", "Upload-Offset": "0"})
}

func TestReadyFloodLeavesBoundedDiskAndCloseRemovesIt(t *testing.T) {
	e := newFloodEnv(t)
	for i := 0; i < 300; i++ {
		e.f.fromClient("ready", map[string]interface{}{"name": "z", "size": float64(1 << 16), "count": float64(1)})
		e.partialUpload(e.f.nextStart()["token"].(string), 1<<16)
	}

	if n := e.entries(); n > 2 {
		t.Fatalf("each new selection revokes the previous grant, so at most one partial upload (2 files) may remain, got %d entries", n)
	}
	if live := e.f.sess.UploadRegistry.Len(); live > upload.DefaultMaxGrants {
		t.Fatalf("live grants exceed the cap: %d", live)
	}

	e.f.sess.Close()
	if n := e.entries(); n != 0 {
		t.Fatalf("closing the session left %d entries on disk", n)
	}
}

func TestUnmountRevokesGrantAndDeletesPartialUploads(t *testing.T) {
	e := newFloodEnv(t)
	e.f.fromClient("ready", map[string]interface{}{"name": "z", "size": float64(1 << 12), "count": float64(1)})
	e.partialUpload(e.f.nextStart()["token"].(string), 1<<12)
	if e.entries() == 0 {
		t.Fatal("setup left no partial upload")
	}
	e.f.sess.Close()
	if n := e.entries(); n != 0 {
		t.Fatalf("unmount left %d entries", n)
	}
}

func TestStartPayloadReportsTheClampedFileCount(t *testing.T) {
	f := newUploadFixture(t)
	f.handle.Accept(UploadConfig{Multiple: true})
	f.fromClient("ready", map[string]interface{}{"name": "m", "size": float64(1), "count": float64(1e9)})
	if got := f.nextStart()["files"]; got != upload.MaxGrantFiles {
		t.Fatalf("start advertised %v files, the grant allows %d", got, upload.MaxGrantFiles)
	}
	f.fromClient("ready", map[string]interface{}{"name": "m", "size": float64(1), "count": float64(-3)})
	if got := f.nextStart()["files"]; got != 1 {
		t.Fatalf("negative count must report 1, got %v", got)
	}
}

func TestAcceptWithUnverifiableExtensionWarnsOnceAndTellsTheClient(t *testing.T) {
	f := newUploadFixture(t)
	diags := make(chan protocol.Diagnostic, 8)
	f.sess.Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) { diags <- d })

	cfg := UploadConfig{Accept: []string{".png", ".mov", ".SQL"}}
	onActor(t, f.sess, func() {
		f.handle.Accept(cfg)
		f.handle.Accept(cfg)
		f.handle.Accept(cfg)
	})

	select {
	case d := <-diags:
		if d.Phase != "upload:accept" || !strings.Contains(d.Message, ".mov") || !strings.Contains(d.Message, ".sql") || strings.Contains(d.Message, ".png") {
			t.Fatalf("unexpected diagnostic %+v", d)
		}
	case <-time.After(time.Second):
		t.Fatal("no diagnostic for unverifiable accept extensions")
	}
	select {
	case d := <-diags:
		t.Fatalf("the warning must be emitted once per configuration, got a second: %+v", d)
	case <-time.After(100 * time.Millisecond):
	}

	f.fromClient("ready", map[string]interface{}{"name": "a.mov", "size": float64(1)})
	start := f.nextStart()
	got, _ := start["unverifiable"].([]string)
	if len(got) != 2 || got[0] != ".mov" || got[1] != ".sql" {
		t.Fatalf("start must tell the client which extensions the server cannot verify: %v", start["unverifiable"])
	}

	onActor(t, f.sess, func() { f.handle.Accept(UploadConfig{Accept: []string{".png"}}) })
	select {
	case d := <-diags:
		t.Fatalf("a fully verifiable accept list must not warn: %+v", d)
	case <-time.After(100 * time.Millisecond):
	}
}
