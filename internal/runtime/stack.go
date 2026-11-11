package runtime

import (
	"strings"

	"github.com/eleven-am/pondlive/internal/goroutines"
)

type FrameCategory int

const (
	FrameUser FrameCategory = iota
	FramePondLive
	FrameRuntime
)

type StackFrame struct {
	Function string
	Package  string
	File     string
	Line     int
	Category FrameCategory
}

var (
	pondLivePrefix  = "github.com/eleven-am/pondlive"
	runtimePrefixes = []string{
		"runtime",
		"runtime/",
		"reflect",
		"sync",
		"testing",
		"panic",
	}
)

func ParseStack(raw string) []StackFrame {
	parsed := goroutines.ParseFrames(raw)
	if len(parsed) == 0 {
		return nil
	}
	frames := make([]StackFrame, len(parsed))
	for i, f := range parsed {
		pkg := extractPackage(f.Function)
		frames[i] = StackFrame{Function: f.Function, Package: pkg, File: f.File, Line: f.Line, Category: categorizePackage(pkg)}
	}
	return frames
}

func extractPackage(funcName string) string {
	if lastSlash := strings.LastIndex(funcName, "/"); lastSlash >= 0 {
		remainder := funcName[lastSlash+1:]
		if dot := strings.Index(remainder, "."); dot >= 0 {
			return funcName[:lastSlash+1+dot]
		}
		return funcName[:lastSlash+1] + remainder
	}

	if dot := strings.Index(funcName, "."); dot >= 0 {
		return funcName[:dot]
	}

	return funcName
}

func categorizePackage(pkg string) FrameCategory {
	if strings.HasPrefix(pkg, pondLivePrefix) {
		return FramePondLive
	}

	for _, prefix := range runtimePrefixes {
		if pkg == prefix || strings.HasPrefix(pkg, prefix+"/") {
			return FrameRuntime
		}
	}

	if strings.HasPrefix(pkg, "runtime") {
		return FrameRuntime
	}

	return FrameUser
}

func filterFrames(frames []StackFrame, category FrameCategory) []StackFrame {
	var result []StackFrame
	for _, f := range frames {
		if f.Category == category {
			result = append(result, f)
		}
	}
	return result
}
