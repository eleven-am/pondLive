package runtime

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseVarName(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{"var MyButton = pkg.Component(func() {})", "MyButton"},
		{"var Counter = pkg.PropsComponent(func() {})", "Counter"},
		{"MyButton := pkg.Component(func() {})", "MyButton"},
		{"Counter := pkg.PropsComponent(func() {})", "Counter"},
		{"  var MyButton = pkg.Component(func() {})", "MyButton"},
		{"\tvar MyButton = pkg.Component(func() {})", "MyButton"},
		{"var   MyButton   =   pkg.Component(func() {})", "MyButton"},
		{"MyButton:=pkg.Component(func() {})", "MyButton"},
		{"func something() {}", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := parseVarName(tt.line)
			if result != tt.expected {
				t.Errorf("parseVarName(%q) = %q, want %q", tt.line, result, tt.expected)
			}
		})
	}
}

func TestExtractVarNameFromSource(t *testing.T) {
	content := `package main

var FirstComponent = pkg.Component(func() {})
var SecondComponent = pkg.PropsComponent(func() {})
ThirdComponent := pkg.Component(func() {})
`
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.go")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	tests := []struct {
		line     int
		expected string
	}{
		{3, "FirstComponent"},
		{4, "SecondComponent"},
		{5, "ThirdComponent"},
		{1, ""},
		{100, ""},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := extractVarNameFromSource(tmpFile, tt.line)
			if result != tt.expected {
				t.Errorf("extractVarNameFromSource(file, %d) = %q, want %q", tt.line, result, tt.expected)
			}
		})
	}
}

func TestExtractVarNameFromSource_FileNotFound(t *testing.T) {
	result := extractVarNameFromSource("/nonexistent/file.go", 1)
	if result != "" {
		t.Errorf("expected empty string for nonexistent file, got %q", result)
	}
}

func TestInstanceComponentName(t *testing.T) {
	t.Run("returns captured name when set", func(t *testing.T) {
		inst := &Instance{Name: "MyButton"}
		if got := inst.ComponentName(); got != "MyButton" {
			t.Errorf("ComponentName() = %q, want %q", got, "MyButton")
		}
	})

	t.Run("returns empty for nil instance", func(t *testing.T) {
		var inst *Instance
		if got := inst.ComponentName(); got != "" {
			t.Errorf("ComponentName() = %q, want empty", got)
		}
	})

	t.Run("returns empty when name not set and no fn", func(t *testing.T) {
		inst := &Instance{}
		if got := inst.ComponentName(); got != "" {
			t.Errorf("ComponentName() = %q, want empty", got)
		}
	})
}

func TestInstanceBuildComponentNamePath(t *testing.T) {
	root := &Instance{ID: "root", Name: "App"}
	child := &Instance{ID: "child1", Name: "Dashboard", Parent: root}
	grandchild := &Instance{ID: "child2", Name: "Widget", Parent: child}

	t.Run("single component", func(t *testing.T) {
		path := root.BuildComponentNamePath()
		if len(path) != 1 || path[0] != "App" {
			t.Errorf("BuildComponentNamePath() = %v, want [App]", path)
		}
	})

	t.Run("nested components", func(t *testing.T) {
		path := grandchild.BuildComponentNamePath()
		expected := []string{"App", "Dashboard", "Widget"}
		if len(path) != len(expected) {
			t.Errorf("BuildComponentNamePath() = %v, want %v", path, expected)
			return
		}
		for i, v := range expected {
			if path[i] != v {
				t.Errorf("BuildComponentNamePath()[%d] = %q, want %q", i, path[i], v)
			}
		}
	})

	t.Run("falls back to ID when name empty", func(t *testing.T) {
		noName := &Instance{ID: "comp123", Parent: root}
		path := noName.BuildComponentNamePath()
		expected := []string{"App", "comp123"}
		if len(path) != len(expected) {
			t.Errorf("BuildComponentNamePath() = %v, want %v", path, expected)
			return
		}
		for i, v := range expected {
			if path[i] != v {
				t.Errorf("BuildComponentNamePath()[%d] = %q, want %q", i, path[i], v)
			}
		}
	})

	t.Run("nil instance", func(t *testing.T) {
		var inst *Instance
		path := inst.BuildComponentNamePath()
		if path != nil {
			t.Errorf("BuildComponentNamePath() = %v, want nil", path)
		}
	})
}
