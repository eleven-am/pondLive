package pkg

import (
	"os/exec"
	"strings"
	"testing"
)

func buildSnippet(t *testing.T, dir string) (string, error) {
	t.Helper()
	cmd := exec.Command("go", "build", "-o", "/dev/null", "./testdata/compilefail/"+dir)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func TestComponentValuesAreAcceptedWhereUIIsTaken(t *testing.T) {
	if out, err := buildSnippet(t, "ok"); err != nil {
		t.Fatalf("component values must compile: %v\n%s", err, out)
	}
}

func TestRawCtxFunctionsDoNotCompileWhereUIIsTaken(t *testing.T) {
	for _, dir := range []string{"newapp", "route", "forrow"} {
		t.Run(dir, func(t *testing.T) {
			out, err := buildSnippet(t, dir)
			if err == nil {
				t.Fatalf("a raw func(*Ctx) Node must not compile in %s", dir)
			}
			if !strings.Contains(out, "cannot use") && !strings.Contains(out, "does not match") {
				t.Fatalf("expected a type error, got:\n%s", out)
			}
		})
	}
}

func TestARouteDefCannotBeUsedAsANode(t *testing.T) {
	out, err := buildSnippet(t, "routedefnode")
	if err == nil {
		t.Fatal("a RouteDef must not compile where a Node or Item is expected")
	}
	if !strings.Contains(out, "RouteDef") {
		t.Fatalf("expected the error to name RouteDef, got:\n%s", out)
	}
}
