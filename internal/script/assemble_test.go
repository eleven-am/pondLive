package script

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"io"
	"os"
	"regexp"
	"strings"
	"testing"
)

var coreStub = []byte("var LiveUIModule={define(){},start(){}};")

func TestHashIsDeterministicAndContentAddressed(t *testing.T) {
	build := func(src string) *Asset {
		r := NewRegistry()
		NewIn[none, none](r, 0, src)
		a, err := r.Asset(coreStub, Options{})
		if err != nil {
			t.Fatal(err)
		}
		return a
	}
	one := build("export default function () { return 1; }")
	again := build("export default function () { return 1; }")
	changed := build("export default function () { return 2; }")
	if one.Hash != again.Hash || !bytes.Equal(one.Body, again.Body) {
		t.Fatal("two builds of the same source differ")
	}
	if one.Hash == changed.Hash {
		t.Fatal("a one-byte source change kept the hash")
	}
	r := NewRegistry()
	NewIn[none, none](r, 0, "export default function () { return 1; }")
	other, err := r.Asset([]byte("var LiveUIModule={define(){},start(){}};//core2"), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if other.Hash == one.Hash {
		t.Fatal("a core change kept the hash")
	}
}

func TestAssembleOrderAndSRI(t *testing.T) {
	r := NewRegistry()
	keys := map[string]bool{}
	for _, s := range []string{"a", "b", "c"} {
		d := NewIn[none, none](r, 0, "export default function () { return '"+s+"'; }")
		keys[d.Key()] = true
	}
	b, err := r.Bundle(Options{Dev: true})
	if err != nil {
		t.Fatal(err)
	}
	a, err := Assemble(coreStub, b)
	if err != nil {
		t.Fatal(err)
	}
	body := string(a.Body)

	if strings.Count(body, "LiveUIModule.start(") != 1 {
		t.Fatalf("start appears %d times", strings.Count(body, "LiveUIModule.start("))
	}
	coreAt := strings.Index(body, string(coreStub))
	wrapAt := strings.Index(body, "(function(define){")
	startAt := strings.Index(body, "LiveUIModule.start(")
	if coreAt != 0 || wrapAt < coreAt || startAt < wrapAt {
		t.Fatalf("order: core %d wrapper %d start %d", coreAt, wrapAt, startAt)
	}
	for k := range keys {
		at := strings.Index(body, `define("`+k+`"`)
		if at < wrapAt || at > startAt {
			t.Fatalf("define %s at %d outside [%d,%d]", k, at, wrapAt, startAt)
		}
	}
	if !strings.HasSuffix(body, "\n") || !strings.Contains(body, "})(LiveUIModule.define);\nLiveUIModule.start(") {
		t.Fatal("wrapper is not invoked with LiveUIModule.define directly before start")
	}

	sum := sha512.Sum384(a.Body)
	if a.SRI != "sha384-"+base64.StdEncoding.EncodeToString(sum[:]) {
		t.Fatalf("SRI is not sha384 of the final bytes: %s", a.SRI)
	}
	pre := sha256.Sum256([]byte(body[:startAt]))
	if a.Hash != hex.EncodeToString(pre[:])[:12] {
		t.Fatalf("hash is not sha256 of everything above the start line")
	}
	if a.Name != "pondlive."+a.Hash+".js" || a.ETag != `"`+a.Hash+`"` {
		t.Fatalf("name %q etag %q", a.Name, a.ETag)
	}
	if !strings.Contains(body, `LiveUIModule.start("`+a.Hash+`");`) {
		t.Fatal("start does not carry the hash")
	}
	zr, err := gzip.NewReader(bytes.NewReader(a.Gzip))
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(zr)
	if !bytes.Equal(raw, a.Body) {
		t.Fatal("gzip does not round-trip")
	}
	again, _ := Assemble(coreStub, b)
	if !bytes.Equal(again.Gzip, a.Gzip) {
		t.Fatal("gzip is not deterministic")
	}
}

func TestAssembleWithoutScripts(t *testing.T) {
	a, err := Assemble(coreStub, &Bundle{})
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`LiveUIModule\.start\("[0-9a-f]{12}"\);\n$`).Match(a.Body) {
		t.Fatalf("tail: %q", a.Body[len(a.Body)-60:])
	}
	if _, err := Assemble(coreStub, nil); err != nil {
		t.Fatal(err)
	}
}

func TestAssetIsCachedPerVariant(t *testing.T) {
	r := NewRegistry()
	NewIn[none, none](r, 0, "export default function () {}", OnlyWith("f"))
	NewIn[none, none](r, 0, "export default function () { 1 }")
	a, _ := r.Asset(coreStub, Options{})
	b, _ := r.Asset(coreStub, Options{})
	if a != b {
		t.Fatal("same variant rebuilt")
	}
	c, _ := r.Asset(coreStub, Options{Features: []Feature{"f"}})
	d, _ := r.Asset(coreStub, Options{Features: []Feature{"f"}})
	if c != d || c == a || c.Hash == a.Hash {
		t.Fatal("feature variant not cached separately")
	}
	e, _ := r.Asset(coreStub, Options{Dev: true})
	if e == a {
		t.Fatal("dev variant shares the production asset")
	}
}

func TestEsbuildPinnedExactly(t *testing.T) {
	data, err := os.ReadFile("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if err := checkGoModPin(string(data)); err != nil {
		t.Error(err)
	}
	for name, content := range map[string]string{
		"older":       "require github.com/evanw/esbuild v0.28.1\n",
		"newer":       "require github.com/evanw/esbuild v0.28.3\n",
		"prefix only": "require github.com/evanw/esbuild v0.28.20\n",
		"absent":      "require github.com/other/x v0.28.2\n",
	} {
		if checkGoModPin(content) == nil {
			t.Errorf("go.mod %s accepted", name)
		}
	}
	if checkGoModPin("require (\n\tgithub.com/evanw/esbuild v0.28.2\n)\n") != nil {
		t.Error("exact pin in a require block rejected")
	}
	for name, content := range map[string]string{
		"caret": `{"devDependencies":{"esbuild":"^0.28.2"}}`,
		"tilde": `{"devDependencies":{"esbuild":"~0.28.2"}}`,
		"old":   `{"devDependencies":{"esbuild":"0.25.11"}}`,
		"none":  `{"devDependencies":{}}`,
	} {
		if checkPackageJSONPin(content) == nil {
			t.Errorf("package.json %s accepted", name)
		}
	}
	if checkPackageJSONPin(`{"devDependencies":{"esbuild":"0.28.2"}}`) != nil {
		t.Error("exact package.json pin rejected")
	}
	// TODO: assert client/package.json with checkPackageJSONPin once the esbuild 0.28.2 pin lands in the client
}

const esbuildVersion = "0.28.2"

func checkGoModPin(goMod string) error {
	re := regexp.MustCompile(`(?m)^\s*(?:require\s+)?github\.com/evanw/esbuild\s+(\S+)`)
	m := re.FindStringSubmatch(goMod)
	if m == nil {
		return errPin("go.mod does not require github.com/evanw/esbuild")
	}
	if m[1] != "v"+esbuildVersion {
		return errPin("go.mod pins esbuild " + m[1] + ", want v" + esbuildVersion)
	}
	return nil
}

func checkPackageJSONPin(pkg string) error {
	re := regexp.MustCompile(`"esbuild"\s*:\s*"([^"]*)"`)
	m := re.FindStringSubmatch(pkg)
	if m == nil {
		return errPin("package.json does not list esbuild")
	}
	if m[1] != esbuildVersion {
		return errPin("package.json pins esbuild " + m[1] + ", want exactly " + esbuildVersion)
	}
	return nil
}

type errPin string

func (e errPin) Error() string { return string(e) }
