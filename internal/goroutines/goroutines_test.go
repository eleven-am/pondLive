package goroutines

import (
	"runtime/debug"
	"strings"
	"testing"
)

func TestParseFramesReadsFunctionFileAndLine(t *testing.T) {
	raw := "goroutine 7 [running]:\n" +
		"example.com/app/pkg.(*T).Run(0xc000012345, {0x1, 0x2})\n" +
		"\t/src/app/pkg/t.go:42 +0x1f\n" +
		"main.main()\n" +
		"\t/src/app/main.go:9 +0x2a\n"
	got := ParseFrames(raw)
	want := []Frame{
		{Function: "example.com/app/pkg.(*T).Run", File: "/src/app/pkg/t.go", Line: 42},
		{Function: "main.main", File: "/src/app/main.go", Line: 9},
	}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("frames = %+v", got)
	}
}

func TestParseFramesOfARealStackNamesThisTest(t *testing.T) {
	frames := ParseFrames(string(debug.Stack()))
	for _, f := range frames {
		if strings.HasSuffix(f.Function, "TestParseFramesOfARealStackNamesThisTest") && strings.HasSuffix(f.File, "goroutines_test.go") && f.Line > 0 {
			return
		}
	}
	t.Fatalf("frames = %+v", frames)
}

func TestParseFramesOfNothingIsNil(t *testing.T) {
	if ParseFrames("") != nil {
		t.Fatal("expected nil")
	}
}

func TestSameFramesNeverMatchesAnEmptyStack(t *testing.T) {
	if SameFrames(nil, nil) || SameFrames([]string{}, []string{}) {
		t.Fatal("empty stacks matched")
	}
	if !SameFrames([]string{"a:1", "b:2"}, []string{"a:1", "b:2"}) || SameFrames([]string{"a:1"}, []string{"a:2"}) || SameFrames([]string{"a:1"}, []string{"a:1", "b:2"}) {
		t.Fatal("SameFrames compares frames wrongly")
	}
}
