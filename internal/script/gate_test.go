package script

import (
	"fmt"
	"runtime"
	"testing"
	"testing/fstest"
	"time"
)

func gateRegistry(n int) *Registry {
	fsys := fstest.MapFS{
		"lib/util.ts": {Data: []byte("export function fmt(n: number): string { return `#${n}`; }\n")},
	}
	r := NewRegistry()
	for i := 0; i < n; i++ {
		name := fmt.Sprintf("s%02d.ts", i)
		fsys[name] = &fstest.MapFile{Data: []byte(fmt.Sprintf(
			"import { fmt } from \"./lib/util\";\ninterface Msg { v: number }\nexport default function (el: Element, io: { send(m: Msg): void; on(f: (m: Msg) => void): void }) {\n  let t = %d;\n  io.on((m) => { t += m.v; el.textContent = fmt(t); });\n  el.addEventListener(\"click\", () => io.send({ v: t }));\n}\n", i))}
		NewFSIn[none, none](r, 0, fsys, name)
	}
	return r
}

func measure(runs int, fn func()) (allocs, bytes uint64) {
	fn()
	runtime.GC()
	var a, b runtime.MemStats
	runtime.ReadMemStats(&a)
	for i := 0; i < runs; i++ {
		fn()
	}
	runtime.ReadMemStats(&b)
	return (b.Mallocs - a.Mallocs) / uint64(runs), (b.TotalAlloc - a.TotalAlloc) / uint64(runs)
}

func TestGateBundle50Scripts(t *testing.T) {
	if raceEnabled {
		t.Skip("alloc and byte gates are measured without the race detector")
	}
	r := gateRegistry(50)
	if err := r.Seal(); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	allocs, bytes := measure(5, func() {
		if _, err := r.Bundle(Options{}); err != nil {
			t.Fatal(err)
		}
	})
	t.Logf("bundle 50 scripts: %d allocs, %d bytes, %s wall for 6 builds", allocs, bytes, time.Since(start))
	if allocs > gateBundleAllocs || bytes > gateBundleBytes {
		t.Fatalf("bundle 50 scripts: %d allocs (max %d), %d bytes (max %d)", allocs, gateBundleAllocs, bytes, gateBundleBytes)
	}
}

func TestGateAssembleNoScripts(t *testing.T) {
	if raceEnabled {
		t.Skip("alloc and byte gates are measured without the race detector")
	}
	core := make([]byte, 39083)
	for i := range core {
		core[i] = byte('a' + i%23)
	}
	empty := &Bundle{}
	allocs, bytes := measure(20, func() {
		if _, err := Assemble(core, empty); err != nil {
			t.Fatal(err)
		}
	})
	t.Logf("assemble no scripts: %d allocs, %d bytes", allocs, bytes)
	if allocs > gateAssembleAllocs || bytes > gateAssembleBytes {
		t.Fatalf("assemble: %d allocs (max %d), %d bytes (max %d)", allocs, gateAssembleAllocs, bytes, gateAssembleBytes)
	}
}
