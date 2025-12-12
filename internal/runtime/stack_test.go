package runtime

import (
	"testing"
)

func TestParseStack_BasicParsing(t *testing.T) {
	raw := `goroutine 1 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:24 +0x5e
github.com/eleven-am/pondlive/internal/runtime.(*Instance).Render.func1()
	/Users/dev/pondlive/internal/runtime/render.go:54 +0x1a5
main.MyComponent(0x140000b8020, {0x0, 0x0})
	/app/components/widget.go:42 +0x45
main.Dashboard(0x140000b8030)
	/app/components/dashboard.go:18 +0x32
`

	frames := ParseStack(raw)

	if len(frames) != 4 {
		t.Fatalf("expected 4 frames, got %d", len(frames))
	}

	if frames[0].Package != "runtime/debug" {
		t.Errorf("frame 0: expected package 'runtime/debug', got '%s'", frames[0].Package)
	}
	if frames[0].Category != FrameRuntime {
		t.Errorf("frame 0: expected FrameRuntime, got %d", frames[0].Category)
	}

	if frames[1].Package != "github.com/eleven-am/pondlive/internal/runtime" {
		t.Errorf("frame 1: expected pondlive package, got '%s'", frames[1].Package)
	}
	if frames[1].Category != FramePondLive {
		t.Errorf("frame 1: expected FramePondLive, got %d", frames[1].Category)
	}

	if frames[2].Package != "main" {
		t.Errorf("frame 2: expected package 'main', got '%s'", frames[2].Package)
	}
	if frames[2].Category != FrameUser {
		t.Errorf("frame 2: expected FrameUser, got %d", frames[2].Category)
	}
	if frames[2].Function != "main.MyComponent" {
		t.Errorf("frame 2: expected function 'main.MyComponent', got '%s'", frames[2].Function)
	}
	if frames[2].File != "/app/components/widget.go" {
		t.Errorf("frame 2: expected file '/app/components/widget.go', got '%s'", frames[2].File)
	}
	if frames[2].Line != 42 {
		t.Errorf("frame 2: expected line 42, got %d", frames[2].Line)
	}
}

func TestParseStack_EmptyInput(t *testing.T) {
	frames := ParseStack("")
	if frames != nil {
		t.Errorf("expected nil for empty input, got %v", frames)
	}
}

func TestParseStack_FilterUserFrames(t *testing.T) {
	raw := `goroutine 1 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:24 +0x5e
github.com/eleven-am/pondlive/internal/runtime.Render()
	/Users/dev/pondlive/internal/runtime/render.go:54 +0x1a5
main.MyComponent()
	/app/components/widget.go:42 +0x45
reflect.Value.Call()
	/usr/local/go/src/reflect/value.go:339 +0xc5
main.App()
	/app/app.go:10 +0x20
`

	frames := ParseStack(raw)
	userFrames := filterFrames(frames, FrameUser)

	if len(userFrames) != 2 {
		t.Fatalf("expected 2 user frames, got %d", len(userFrames))
	}

	if userFrames[0].Function != "main.MyComponent" {
		t.Errorf("expected first user frame to be 'main.MyComponent', got '%s'", userFrames[0].Function)
	}
	if userFrames[1].Function != "main.App" {
		t.Errorf("expected second user frame to be 'main.App', got '%s'", userFrames[1].Function)
	}
}

func TestParseStack_CategorizePackages(t *testing.T) {
	tests := []struct {
		pkg      string
		expected FrameCategory
	}{
		{"runtime", FrameRuntime},
		{"runtime/debug", FrameRuntime},
		{"reflect", FrameRuntime},
		{"sync", FrameRuntime},
		{"testing", FrameRuntime},
		{"github.com/eleven-am/pondlive/internal/runtime", FramePondLive},
		{"github.com/eleven-am/pondlive/pkg", FramePondLive},
		{"main", FrameUser},
		{"github.com/user/myapp", FrameUser},
		{"mypackage", FrameUser},
	}

	for _, tt := range tests {
		got := categorizePackage(tt.pkg)
		if got != tt.expected {
			t.Errorf("categorizePackage(%q): expected %d, got %d", tt.pkg, tt.expected, got)
		}
	}
}

func TestParseStack_ExtractPackage(t *testing.T) {
	tests := []struct {
		funcName string
		expected string
	}{
		{"main.MyComponent", "main"},
		{"main.(*App).Render", "main"},
		{"github.com/eleven-am/pondlive/internal/runtime.Render", "github.com/eleven-am/pondlive/internal/runtime"},
		{"github.com/eleven-am/pondlive/internal/runtime.(*Instance).Render", "github.com/eleven-am/pondlive/internal/runtime"},
		{"runtime/debug.Stack", "runtime/debug"},
		{"reflect.Value.Call", "reflect"},
	}

	for _, tt := range tests {
		got := extractPackage(tt.funcName)
		if got != tt.expected {
			t.Errorf("extractPackage(%q): expected %q, got %q", tt.funcName, tt.expected, got)
		}
	}
}

func TestError_ParsedStack(t *testing.T) {
	raw := `goroutine 1 [running]:
main.MyComponent()
	/app/components/widget.go:42 +0x45
main.App()
	/app/app.go:10 +0x20
`

	err := &Error{
		ErrorCode:  ErrCodeRender,
		Message:    "test error",
		StackTrace: raw,
	}

	frames := err.ParsedStack()
	if len(frames) != 2 {
		t.Fatalf("expected 2 frames, got %d", len(frames))
	}
}

func TestError_UserFrames(t *testing.T) {
	raw := `goroutine 1 [running]:
runtime/debug.Stack()
	/usr/local/go/src/runtime/debug/stack.go:24 +0x5e
github.com/eleven-am/pondlive/internal/runtime.Render()
	/Users/dev/pondlive/internal/runtime/render.go:54 +0x1a5
main.MyComponent()
	/app/components/widget.go:42 +0x45
`

	err := &Error{
		ErrorCode:  ErrCodeRender,
		Message:    "test error",
		StackTrace: raw,
	}

	userFrames := err.UserFrames()
	if len(userFrames) != 1 {
		t.Fatalf("expected 1 user frame, got %d", len(userFrames))
	}

	if userFrames[0].Function != "main.MyComponent" {
		t.Errorf("expected 'main.MyComponent', got '%s'", userFrames[0].Function)
	}
}

func TestError_NilSafety(t *testing.T) {
	var err *Error

	if frames := err.ParsedStack(); frames != nil {
		t.Error("expected nil ParsedStack for nil error")
	}

	if frames := err.UserFrames(); frames != nil {
		t.Error("expected nil UserFrames for nil error")
	}
}
