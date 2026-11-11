package runtime_test

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestEveryDiagnosticSeverityCarriesTheSessionIDFromOneOwner(t *testing.T) {
	var ctx *runtime.Ctx
	app := runtimetest.New(t, func(c *runtime.Ctx) work.Node {
		ctx = c
		return work.BuildElement("div")
	})
	cases := []struct {
		phase string
		r     runtime.Report
	}{
		{"diag:plain", runtime.Report{Severity: runtime.SeverityDiagnostic, Phase: "diag:plain"}},
		{"diag:withmeta", runtime.Report{Severity: runtime.SeverityDiagnostic, Phase: "diag:withmeta", Meta: map[string]any{"k": "v"}}},
		{"warning:W", runtime.Report{Severity: runtime.SeverityWarning, Phase: "warning:W", Code: "W"}},
		{"error:E", runtime.Report{Severity: runtime.SeverityError, Phase: "error:E", Code: "E"}},
	}
	for _, c := range cases {
		app.Do(func() { runtime.ReportOn(ctx, c.r) })
		diags := app.Diagnostics(c.phase)
		if len(diags) == 0 {
			t.Errorf("%s: no diagnostic", c.phase)
			continue
		}
		if got := diags[len(diags)-1].Metadata["session_id"]; got != "test" {
			t.Errorf("%s: session_id = %v", c.phase, got)
		}
	}
	if got := app.Diagnostics("diag:withmeta")[0].Metadata["k"]; got != "v" {
		t.Errorf("caller metadata was lost: %v", got)
	}
}

func TestADiagnosticDoesNotMutateTheMetaMapItWasGiven(t *testing.T) {
	var ctx *runtime.Ctx
	app := runtimetest.New(t, func(c *runtime.Ctx) work.Node {
		ctx = c
		return work.BuildElement("div")
	})
	meta := map[string]any{"k": "v"}
	app.Do(func() {
		runtime.ReportOn(ctx, runtime.Report{Severity: runtime.SeverityDiagnostic, Phase: "diag:own", Meta: meta})
	})
	if _, ok := meta["session_id"]; ok || len(meta) != 1 {
		t.Fatalf("the caller's map was modified: %v", meta)
	}
}
