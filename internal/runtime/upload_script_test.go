package runtime

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/upload"
)

const uploadJSHarness = `
const fs = require('fs');
const script = fs.readFileSync(process.argv[2], 'utf8');
const scenario = process.argv[3];
const input = JSON.parse(process.argv[4] || '{}');

const tick = async () => { for (let i = 0; i < 5; i++) await Promise.resolve(); };

function makeEnv(preloadTus) {
  const sent = [];
  const handlers = {};
  const listeners = {};
  const appended = [];
  const constructed = [];
  global.window = {};
  if (preloadTus) {
    global.window.tus = { Upload: class { constructor(file, opts) { constructed.push(opts); } start() {} abort() {} } };
  }
  global.document = { createElement: () => ({ remove() {} }), head: { appendChild: (el) => appended.push(el) } };
  const element = { value: 'x', addEventListener: (ev, fn) => { listeners[ev] = fn; }, removeEventListener() {} };
  const transport = { send: (ev, data) => sent.push([ev, data]), on: (ev, fn) => { handlers[ev] = fn; } };
  new Function('element', 'transport', 'return (' + script + ')(element, transport);')(element, transport);
  return { sent, handlers, listeners, appended, constructed };
}

function select(env, files) {
  const el = { files: files.map((f) => ({ name: f.name, size: f.size || 1, type: f.type || '' })), value: 'x' };
  env.listeners.change({ target: el });
}

(async () => {
  let out;
  if (scenario === 'accept') {
    out = [];
    for (const c of input.cases) {
      const env = makeEnv(true);
      select(env, [c.file]);
      env.handlers.start({ token: 't', endpoint: '/e/', tusScript: '/s.js', accept: c.accept, unverifiable: c.unverifiable, multiple: true });
      await tick();
      out.push(!env.sent.some((m) => m[0] === 'error'));
    }
  } else if (scenario === 'cancel-during-load') {
    const env = makeEnv(false);
    select(env, [{ name: 'a.png', type: 'image/png' }]);
    env.handlers.start({ token: 't', endpoint: '/e/', tusScript: '/static/tus.min.js', multiple: false });
    await tick();
    env.handlers.cancel({});
    global.window.tus = { Upload: class { constructor(file, opts) { env.constructed.push(opts); } start() { env.started = true; } abort() {} } };
    env.appended[0].onload();
    await tick();
    out = { sent: env.sent, constructed: env.constructed.length, started: !!env.started, appendedSrc: env.appended.map((e) => e.src) };
  } else if (scenario === 'start') {
    const env = makeEnv(true);
    select(env, [{ name: 'a.png', type: 'image/png' }]);
    env.handlers.start({ token: 'tok', endpoint: '/mounted/', tusScript: '/static/tus.min.js', multiple: false });
    await tick();
    out = { sent: env.sent, constructed: env.constructed.map((o) => ({ endpoint: o.endpoint, headers: o.headers, metadata: o.metadata, resume: o.storeFingerprintForResuming })) };
  } else if (scenario === 'multiple') {
    const env = makeEnv(true);
    select(env, [{ name: 'a.png' }, { name: 'b.png' }]);
    env.handlers.start({ token: 'tok', endpoint: '/mounted/', tusScript: '/s.js', multiple: input.multiple });
    await tick();
    out = { sent: env.sent, constructed: env.constructed.length };
  } else if (scenario === 'cancel-before-start') {
    const env = makeEnv(true);
    select(env, [{ name: 'a.png', type: 'image/png' }]);
    env.handlers.cancel({});
    env.handlers.start({ token: 't', endpoint: '/e/', tusScript: '/s.js', multiple: false });
    await tick();
    out = { constructed: env.constructed.length, sent: env.sent };
  } else if (scenario === 'ready') {
    const env = makeEnv(true);
    select(env, [{ name: 'a.png', size: 5, type: 'image/png' }, { name: 'b.png', size: 6, type: 'image/png' }]);
    out = env.sent;
  } else if (scenario === 'reselect') {
    const env = makeEnv(true);
    select(env, [{ name: 'a.png', type: 'image/png' }]);
    env.handlers.start({ token: 'first', endpoint: '/e/', tusScript: '/s.js', multiple: false });
    await tick();
    env.constructed.length = 0;
    select(env, [{ name: 'b.png', type: 'image/png' }]);
    await tick();
    const startedWithStaleToken = env.constructed.length > 0;
    env.handlers.start({ token: 'second', endpoint: '/e/', tusScript: '/s.js', multiple: false });
    await tick();
    out = { startedWithStaleToken, tokens: env.constructed.map((o) => o.headers['X-Upload-Token']) };
  }
  process.stdout.write(JSON.stringify(out));
})().catch((e) => { process.stderr.write(String(e && e.stack || e)); process.exit(1); });
`

func runUploadJS(t *testing.T, scenario string, input any, out any) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not installed")
	}
	dir := t.TempDir()
	harness := filepath.Join(dir, "harness.js")
	scriptPath := filepath.Join(dir, "upload.js")
	if err := os.WriteFile(harness, []byte(uploadJSHarness), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(scriptPath, []byte(minifyJS(uploadScript)), 0o600); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(input)
	cmd := exec.Command(node, harness, scriptPath, scenario, string(payload))
	cmd.Dir = dir
	raw, err := cmd.Output()
	if err != nil {
		var stderr string
		if ee, ok := err.(*exec.ExitError); ok {
			stderr = string(ee.Stderr)
		}
		t.Fatalf("node failed: %v\n%s", err, stderr)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		t.Fatalf("bad harness output %q: %v", raw, err)
	}
}

func TestUploadScriptDoesNotLoadFromACDN(t *testing.T) {
	if strings.Contains(uploadScript, "jsdelivr") || strings.Contains(uploadScript, "http://") || strings.Contains(uploadScript, "https://") {
		t.Fatal("upload.js must not load code from a third-party host")
	}
	if strings.Contains(uploadScript, "'/tus/'") {
		t.Fatal("upload.js must not hardcode the upload mount")
	}
}

func TestUploadScriptAcceptSemanticsMatchTheServer(t *testing.T) {
	type vector struct {
		Accept       []string       `json:"accept"`
		Unverifiable []string       `json:"unverifiable"`
		File         map[string]any `json:"file"`
	}
	cases := []struct {
		accept []string
		name   string
		typ    string
	}{
		{[]string{"image/png"}, "a.png", "image/png"},
		{[]string{"image/png"}, "a.jpg", "image/jpeg"},
		{[]string{"Text/Plain"}, "a.txt", "text/plain; charset=utf-8"},
		{[]string{"image/*"}, "a.webp", "image/webp"},
		{[]string{"image/*"}, "a.pdf", "application/pdf"},
		{[]string{"image/*"}, "a.png", ""},
		{[]string{".png"}, "photo.PNG", ""},
		{[]string{".png"}, "photo.jpg", "image/png"},
		{[]string{".tar.gz"}, "x.tar.gz", ""},
		{[]string{"*/*"}, "x", ""},
		{[]string{" ", ""}, "x.png", "image/png"},
		{[]string{"application/pdf", ".png"}, "x.png", ""},
		{[]string{"image/svg+xml"}, "x.svg", "image/svg+xml"},
		{[]string{"application/vnd.ms-excel"}, "x.xls", "application/vnd.ms-excel"},
		{[]string{"image/*"}, "x.png", "IMAGE/PNG"},
	}
	var in struct {
		Cases []vector `json:"cases"`
	}
	for _, c := range cases {
		in.Cases = append(in.Cases, vector{Accept: c.accept, File: map[string]any{"name": c.name, "type": c.typ}})
	}
	var got []bool
	runUploadJS(t, "accept", in, &got)
	if len(got) != len(cases) {
		t.Fatalf("harness returned %d results for %d cases", len(got), len(cases))
	}
	for i, c := range cases {
		if want := upload.AcceptsDeclared(c.accept, c.name, c.typ); got[i] != want {
			t.Errorf("accept=%v name=%q type=%q: client says %v, server says %v", c.accept, c.name, c.typ, got[i], want)
		}
	}
}

func TestUploadScriptCancelDuringTusLoadCancels(t *testing.T) {
	var out struct {
		Sent        [][]any  `json:"sent"`
		Constructed int      `json:"constructed"`
		Started     bool     `json:"started"`
		AppendedSrc []string `json:"appendedSrc"`
	}
	runUploadJS(t, "cancel-during-load", nil, &out)

	if out.Started || out.Constructed != 0 {
		t.Fatalf("an upload started after the user cancelled while tus was loading: %+v", out)
	}
	for _, m := range out.Sent {
		if m[0] == "error" || m[0] == "progress" || m[0] == "complete" {
			t.Fatalf("unexpected message after cancel: %v", m)
		}
	}
	if len(out.AppendedSrc) != 1 || out.AppendedSrc[0] != "/static/tus.min.js" {
		t.Fatalf("the tus script must be loaded from the configured self-hosted URL, got %v", out.AppendedSrc)
	}
}

func TestUploadScriptStartUsesGrantEndpointAndHeader(t *testing.T) {
	var out struct {
		Constructed []struct {
			Endpoint string            `json:"endpoint"`
			Headers  map[string]string `json:"headers"`
			Metadata map[string]string `json:"metadata"`
			Resume   bool              `json:"resume"`
		} `json:"constructed"`
	}
	runUploadJS(t, "start", nil, &out)
	if len(out.Constructed) != 1 {
		t.Fatalf("expected one tus upload, got %d", len(out.Constructed))
	}
	c := out.Constructed[0]
	if c.Endpoint != "/mounted/" {
		t.Errorf("endpoint %q ignores the configured mount", c.Endpoint)
	}
	if c.Headers[upload.TokenHeader] != "tok" {
		t.Errorf("the grant is not sent on every tus request: %v", c.Headers)
	}
	if _, leaked := c.Metadata["token"]; leaked {
		t.Errorf("the grant must travel in the header, not the metadata: %v", c.Metadata)
	}
	if c.Resume {
		t.Error("resuming across grants must be disabled")
	}
}

func TestUploadScriptRejectsMultipleFilesWhenNotMultiple(t *testing.T) {
	for _, multiple := range []bool{false, true} {
		var out struct {
			Sent        [][]any `json:"sent"`
			Constructed int     `json:"constructed"`
		}
		runUploadJS(t, "multiple", map[string]any{"multiple": multiple}, &out)
		hasError := false
		for _, m := range out.Sent {
			if m[0] == "error" {
				hasError = true
			}
		}
		if multiple && (hasError || out.Constructed != 1) {
			t.Errorf("multiple=true must start the queue: %+v", out)
		}
		if !multiple && (!hasError || out.Constructed != 0) {
			t.Errorf("multiple=false must reject two files client-side: %+v", out)
		}
	}
}

func TestUploadScriptReadyReportsFileCount(t *testing.T) {
	var out [][]any
	runUploadJS(t, "ready", nil, &out)
	if len(out) != 1 || out[0][0] != "ready" {
		t.Fatalf("unexpected messages %v", out)
	}
	data := out[0][1].(map[string]any)
	if data["count"] != float64(2) || data["name"] != "a.png" {
		t.Fatalf("ready must carry the first file and the selection count: %v", data)
	}
}

func TestUploadScriptReselectWaitsForTheFreshGrant(t *testing.T) {
	var out struct {
		StartedWithStaleToken bool     `json:"startedWithStaleToken"`
		Tokens                []string `json:"tokens"`
	}
	runUploadJS(t, "reselect", nil, &out)
	if out.StartedWithStaleToken {
		t.Fatal("a new selection started uploading with the previous (revoked) grant")
	}
	if len(out.Tokens) != 1 || out.Tokens[0] != "second" {
		t.Fatalf("expected the upload to use the fresh grant, got %v", out.Tokens)
	}
}

func TestUploadScriptCancelBeforeStartDropsTheSelection(t *testing.T) {
	var out struct {
		Constructed int     `json:"constructed"`
		Sent        [][]any `json:"sent"`
	}
	runUploadJS(t, "cancel-before-start", nil, &out)
	if out.Constructed != 0 {
		t.Fatalf("a cancelled selection was uploaded anyway: %+v", out)
	}
}

func TestUploadScriptRejectsWhatTheServerCannotVerify(t *testing.T) {
	type vector struct {
		Accept       []string       `json:"accept"`
		Unverifiable []string       `json:"unverifiable"`
		File         map[string]any `json:"file"`
	}
	cases := []struct {
		accept []string
		name   string
		typ    string
	}{
		{[]string{".mov"}, "clip.mov", "video/quicktime"},
		{[]string{".mov", ".png"}, "a.png", "image/png"},
		{[]string{".mov", ".png"}, "clip.mov", ""},
		{[]string{".mov", "video/quicktime"}, "clip.mov", "video/quicktime"},
		{[]string{".SQL"}, "dump.sql", ""},
		{[]string{".md"}, "readme.md", "text/markdown"},
		{[]string{"image/*"}, "a.png", "image/png"},
	}
	var in struct {
		Cases []vector `json:"cases"`
	}
	for _, c := range cases {
		in.Cases = append(in.Cases, vector{Accept: c.accept, Unverifiable: upload.UnknownExtensions(c.accept), File: map[string]any{"name": c.name, "type": c.typ}})
	}
	var got []bool
	runUploadJS(t, "accept", in, &got)
	for i, c := range cases {
		if want := upload.AcceptsUpload(c.accept, c.name, c.typ); got[i] != want {
			t.Errorf("accept=%v name=%q: client says %v, the server's create check says %v", c.accept, c.name, got[i], want)
		}
	}
}
