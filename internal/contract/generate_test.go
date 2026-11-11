package contract

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

const outputEnv = "CONTRACT_FIXTURES_DIR"

type fixtureStep struct {
	Edits    []protocol.Edit `json:"edits"`
	Expected string          `json:"expected"`
	Live     string          `json:"live"`
	Reset    string          `json:"reset"`
	Keep     []uint32        `json:"keep"`
	Fresh    bool            `json:"fresh"`
}

type fixtureCase struct {
	Name  string        `json:"name"`
	SSR   string        `json:"ssr"`
	Steps []fixtureStep `json:"steps"`
}

type fixtureFile struct {
	Scenario    string        `json:"scenario"`
	Description string        `json:"description"`
	Cases       []fixtureCase `json:"cases"`
}

type instance struct {
	root  func(ctx *runtime.Ctx) work.Node
	steps []func(a *runtimetest.App)
	fresh func() func(ctx *runtime.Ctx) work.Node
}

type scenarioCase struct {
	name  string
	prod  bool
	build func() instance
}

type scenario struct {
	file        string
	description string
	cases       []scenarioCase
}

func freshNormalized(t testing.TB, root func(ctx *runtime.Ctx) work.Node, prod bool) string {
	t.Helper()
	s := runtime.NewSession(runtime.Config{SessionID: "fresh", Root: root, DevMode: !prod})
	snap, err := s.RenderSnapshot()
	if err != nil {
		t.Fatalf("fresh render: %v", err)
	}
	if snap.Crashed {
		t.Fatalf("fresh render crashed: %v", snap.Crash)
	}
	norm, err := runtimetest.NormalizedOf(snap.HTML)
	if err != nil {
		t.Fatalf("fresh render is not parseable: %v", err)
	}
	return norm
}

func idsOf(a *runtimetest.App) []uint32 {
	ids := a.Model.IDs()
	slices.Sort(ids)
	return ids
}

func intersect(before, after []uint32) []uint32 {
	in := make(map[uint32]struct{}, len(after))
	for _, id := range after {
		in[id] = struct{}{}
	}
	out := []uint32{}
	for _, id := range before {
		if _, ok := in[id]; ok {
			out = append(out, id)
		}
	}
	return out
}

func runCase(t *testing.T, c scenarioCase) fixtureCase {
	t.Helper()
	inst := c.build()
	var opts []runtimetest.Option
	if c.prod {
		opts = append(opts, runtimetest.Dev(false))
	}
	app := runtimetest.New(t, inst.root, opts...)
	fc := fixtureCase{Name: c.name, SSR: app.SSR, Steps: []fixtureStep{}}
	for i, step := range inst.steps {
		before := idsOf(app)
		frames := len(app.Frames)
		step(app)
		var edits []protocol.Edit
		for _, f := range app.Frames[frames:] {
			edits = append(edits, f...)
		}
		if edits == nil {
			edits = []protocol.Edit{}
		}
		reset := app.ResetHTML()
		canon, err := runtimetest.Canonical(reset)
		if err != nil {
			t.Fatalf("step %d: reset html is not parseable: %v", i, err)
		}
		live := app.Model.HTML()
		if live != canon {
			t.Fatalf("step %d: serialize pass differs from the live DOM\nlive:      %s\nserialize: %s", i, live, canon)
		}
		fs := fixtureStep{Edits: edits, Live: live, Reset: reset, Keep: intersect(before, idsOf(app)), Fresh: app.Clock.Pending() == 0}
		fs.Expected = app.Model.Normalized()
		if fs.Fresh {
			want := freshNormalized(t, inst.fresh(), c.prod)
			if fs.Expected != want {
				t.Fatalf("step %d: edits applied to the SSR differ from a fresh render\nedits: %v\nlive:  %s\nfresh: %s", i, edits, fs.Expected, want)
			}
		}
		fc.Steps = append(fc.Steps, fs)
	}
	return fc
}

func encodeFile(ff fixtureFile) []byte {
	var buf bytes.Buffer
	head, err := json.Marshal(struct {
		Scenario    string `json:"scenario"`
		Description string `json:"description"`
	}{ff.Scenario, ff.Description})
	if err != nil {
		panic(err)
	}
	buf.Write(head[:len(head)-1])
	buf.WriteString(`,"cases":[` + "\n")
	for i, c := range ff.Cases {
		var line bytes.Buffer
		enc := json.NewEncoder(&line)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(c); err != nil {
			panic(err)
		}
		buf.Write(bytes.TrimRight(line.Bytes(), "\n"))
		if i < len(ff.Cases)-1 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\n')
	}
	buf.WriteString("]}\n")
	return buf.Bytes()
}

func generate(t *testing.T) map[string][]byte {
	t.Helper()
	files, _ := generateAll(t)
	return files
}

func generateAll(t *testing.T) (map[string][]byte, map[protocol.Op]int) {
	t.Helper()
	files := map[string][]byte{}
	counts := map[protocol.Op]int{}
	for _, sc := range allScenarios() {
		ff := fixtureFile{Scenario: sc.file, Description: sc.description}
		for _, c := range sc.cases {
			t.Run(sc.file+"/"+c.name, func(t *testing.T) {
				fc := runCase(t, c)
				for _, st := range fc.Steps {
					for _, e := range st.Edits {
						counts[e.Op]++
					}
				}
				ff.Cases = append(ff.Cases, fc)
			})
		}
		files[sc.file+".json"] = encodeFile(ff)
	}
	if t.Failed() {
		t.FailNow()
	}
	return files, counts
}

func TestGenerateFixtures(t *testing.T) {
	dir := os.Getenv(outputEnv)
	if dir == "" {
		t.Skipf("%s not set; run `make contract-fixtures` to write fixtures", outputEnv)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, f := range existing {
		if err := os.Remove(f); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range generate(t) {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGenerationIsDeterministic(t *testing.T) {
	first := generate(t)
	again := generate(t)
	if len(again) != len(first) {
		t.Fatalf("file count changed between runs: %d vs %d", len(first), len(again))
	}
	for name, data := range first {
		if !bytes.Equal(data, again[name]) {
			t.Fatalf("fixture %s differs between generations: the generator is nondeterministic", name)
		}
	}
}

func TestCommittedFixturesAreFresh(t *testing.T) {
	dir := filepath.Join("..", "..", "client", "test", "fixtures", "contract")
	committed, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	fresh := generate(t)
	if len(committed) != len(fresh) {
		t.Fatalf("committed fixture count %d differs from generated %d; run `make contract-fixtures`", len(committed), len(fresh))
	}
	for name, data := range fresh {
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("missing fixture %s; run `make contract-fixtures`: %v", name, err)
		}
		if !bytes.Equal(got, data) {
			t.Fatalf("fixture %s is stale; run `make contract-fixtures`", name)
		}
	}
}

func TestEveryEditOpIsExercised(t *testing.T) {
	_, counts := generateAll(t)
	for _, op := range []protocol.Op{protocol.OpText, protocol.OpAttr, protocol.OpRemoveAttr, protocol.OpStyle, protocol.OpHTML,
		protocol.OpInsert, protocol.OpMove, protocol.OpRemove, protocol.OpClear} {
		if counts[op] == 0 {
			t.Errorf("no scenario produced a %q edit: %v", op, counts)
		}
	}
	t.Log(fmt.Sprint(counts))
}
