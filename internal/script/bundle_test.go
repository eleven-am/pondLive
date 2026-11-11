package script

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"
	"testing/fstest"
)

var update = flag.Bool("update", false, "rewrite testdata/bundle.golden.js")

type none = struct{}

func goldenRegistry() *Registry {
	r := NewRegistry()
	fixtures := os.DirFS("testdata/scripts")
	NewFSIn[none, none](r, 0, fixtures, "counter.ts")
	NewFSIn[none, none](r, 0, fixtures, "toggle.js")
	for i := 0; i < 10; i++ {
		NewIn[none, none](r, 0, fmt.Sprintf("export default function (el: Element) { el.setAttribute('data-n', '%d'); }", i))
	}
	return r
}

func TestBundleGolden(t *testing.T) {
	b, err := goldenRegistry().Bundle(Options{Dev: true})
	if err != nil {
		t.Fatal(err)
	}
	if *update {
		if err := os.WriteFile("testdata/bundle.golden.js", b.Code, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile("testdata/bundle.golden.js")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b.Code, want) {
		t.Fatalf("bundle differs from testdata/bundle.golden.js\n--- got ---\n%s\n--- want ---\n%s", b.Code, want)
	}
	if len(b.Keys) != 12 {
		t.Fatalf("keys: %d", len(b.Keys))
	}
	for i := 1; i < len(b.Keys); i++ {
		if b.Keys[i-1] >= b.Keys[i] {
			t.Fatal("keys not sorted")
		}
	}
	last := -1
	for _, k := range b.Keys {
		i := bytes.Index(b.Code, []byte(fmt.Sprintf("define(%q,", k)))
		if i < 0 {
			t.Fatalf("define for %s missing", k)
		}
		if i < last {
			t.Fatalf("define for %s out of order", k)
		}
		last = i
	}
}

func TestMinifiedBundleIsSmallerAndDeterministic(t *testing.T) {
	dev, err := goldenRegistry().Bundle(Options{Dev: true})
	if err != nil {
		t.Fatal(err)
	}
	a, err := goldenRegistry().Bundle(Options{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := goldenRegistry().Bundle(Options{})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Code, b.Code) {
		t.Fatal("minified output is not deterministic")
	}
	if len(a.Code) >= len(dev.Code) {
		t.Fatalf("minified %d >= dev %d", len(a.Code), len(dev.Code))
	}
}

func TestSyntaxErrorNamesDeclarationAndPosition(t *testing.T) {
	r := NewRegistry()
	_, _, line, _ := runtime.Caller(0)
	NewFSIn[none, none](r, 0, os.DirFS("testdata/scripts"), "syntax.ts")
	_, err := r.Bundle(Options{Dev: true})
	if err == nil {
		t.Fatal("syntax error was not reported")
	}
	want := fmt.Sprintf("bundle_test.go:%d syntax.ts 2:13: ", line+1)
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("error %q does not contain %q", err, want)
	}

	r = NewRegistry()
	_, _, line, _ = runtime.Caller(0)
	NewIn[none, none](r, 0, "const a = 1;\nconst b = ;\nexport default function () {}")
	_, err = r.Bundle(Options{})
	want = fmt.Sprintf("bundle_test.go:%d (script) 2:11: ", line+1)
	if err == nil || !strings.Contains(err.Error(), want) {
		t.Fatalf("inline error %v does not contain %q", err, want)
	}
}

func TestFSImportsStayInsideTheFS(t *testing.T) {
	fsys := fstest.MapFS{
		"ok.ts":        {Data: []byte(`import { v } from "./lib/v"; export default function () { return v; }`)},
		"lib/v.ts":     {Data: []byte(`export const v = 1;`)},
		"up.ts":        {Data: []byte(`import { x } from "../x"; export default function () { return x; }`)},
		"deep.ts":      {Data: []byte(`import { x } from "./lib/../../x"; export default function () { return x; }`)},
		"abs.ts":       {Data: []byte(`import x from "/etc/x"; export default function () { return x; }`)},
		"bare.ts":      {Data: []byte(`import _ from "lodash"; export default function () { return _; }`)},
		"missing.ts":   {Data: []byte(`import _ from "./nope"; export default function () { return _; }`)},
		"dynamic.ts":   {Data: []byte(`export default function () { return import("lodash"); }`)},
		"sibling.ts":   {Data: []byte(`import { v } from "./lib/v"; import { w } from "./lib/w"; export default () => v + w;`)},
		"lib/w.ts":     {Data: []byte(`import { v } from "./v"; export const w = v + 1;`)},
		"outside/x.ts": {Data: []byte(`export const x = 1;`)},
	}
	build := func(entry string) error {
		r := NewRegistry()
		NewFSIn[none, none](r, 0, fsys, entry)
		_, err := r.Bundle(Options{Dev: true})
		return err
	}
	for _, entry := range []string{"ok.ts", "sibling.ts"} {
		if err := build(entry); err != nil {
			t.Errorf("%s: %v", entry, err)
		}
	}
	for entry, frag := range map[string]string{
		"up.ts":      "escapes the script FS",
		"deep.ts":    "escapes the script FS",
		"abs.ts":     "absolute import",
		"bare.ts":    "bare import",
		"missing.ts": "not found",
		"dynamic.ts": "bare import",
	} {
		err := build(entry)
		if err == nil || !strings.Contains(err.Error(), frag) {
			t.Errorf("%s: got %v, want %q", entry, err, frag)
		}
	}

	r := NewRegistry()
	NewIn[none, none](r, 0, `import _ from "lodash"; export default function () { return _; }`)
	if _, err := r.Bundle(Options{}); err == nil || !strings.Contains(err.Error(), "inline scripts cannot import") {
		t.Errorf("inline import: %v", err)
	}
}

func TestTypeOnlyImportsAreErased(t *testing.T) {
	r := NewRegistry()
	NewIn[none, none](r, 0, `import type { Channel } from "lodash"; export default function (el: Element, io: Channel) {}`)
	if _, err := r.Bundle(Options{}); err != nil {
		t.Fatal(err)
	}
}

func TestModuleWithoutDefaultExportFails(t *testing.T) {
	for name, make := range map[string]func(*Registry){
		"fs ts": func(r *Registry) { NewFSIn[none, none](r, 0, os.DirFS("testdata/scripts"), "nodefault.ts") },
		"inline": func(r *Registry) {
			NewIn[none, none](r, 0, "export function notDefault() {}")
		},
		"no exports at all": func(r *Registry) { NewIn[none, none](r, 0, "console.log('hi')") },
	} {
		r := NewRegistry()
		make(r)
		_, err := r.Bundle(Options{})
		if err == nil {
			t.Errorf("%s: module without a default export was bundled", name)
			continue
		}
		if !strings.Contains(err.Error(), "no default export") || !strings.Contains(err.Error(), "bundle_test.go:") {
			t.Errorf("%s: unstructured error: %v", name, err)
		}
	}
}

func TestFeatureFlaggedDefExcludedUnlessEnabled(t *testing.T) {
	r := NewRegistry()
	plain := NewIn[none, none](r, 0, "export default function () { /* plain */ }")
	flagged := NewIn[none, none](r, 0, "export default function () { /* flagged */ }", OnlyWith("uploads"))
	off, err := r.Bundle(Options{Dev: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(off.Keys) != 1 || off.Keys[0] != plain.Key() || bytes.Contains(off.Code, []byte(flagged.Key())) {
		t.Fatalf("flagged def present without its feature: %v", off.Keys)
	}
	on, err := r.Bundle(Options{Dev: true, Features: []Feature{"uploads"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(on.Keys) != 2 || !bytes.Contains(on.Code, []byte(flagged.Key())) {
		t.Fatalf("flagged def missing with its feature: %v", on.Keys)
	}
	other, err := r.Bundle(Options{Dev: true, Features: []Feature{"other"}})
	if err != nil || len(other.Keys) != 1 {
		t.Fatalf("unrelated feature enabled the def: %v %v", other.Keys, err)
	}
}

func TestFlaggedDefErrorsAreSkippedWhenDisabled(t *testing.T) {
	r := NewRegistry()
	NewIn[none, none](r, 0, "const = ;", OnlyWith("uploads"))
	if _, err := r.Bundle(Options{}); err != nil {
		t.Fatalf("disabled def was still compiled: %v", err)
	}
}

func TestEmptyRegistryBundlesToNothing(t *testing.T) {
	b, err := NewRegistry().Bundle(Options{})
	if err != nil || len(b.Code) != 0 || len(b.Keys) != 0 {
		t.Fatalf("%v %v", b, err)
	}
}
