package runtime

import (
	"regexp"
	"strconv"
	"strings"
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
	fileLineRegex   = regexp.MustCompile(`^\s*(.+):(\d+)`)
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
	if raw == "" {
		return nil
	}

	lines := strings.Split(raw, "\n")
	var frames []StackFrame

	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		if line == "" || strings.HasPrefix(line, "goroutine ") {
			i++
			continue
		}

		if i+1 >= len(lines) {
			break
		}

		funcLine := line
		fileLine := strings.TrimSpace(lines[i+1])

		frame := parseFramePair(funcLine, fileLine)
		if frame.Function != "" {
			frames = append(frames, frame)
		}

		i += 2
	}

	return frames
}

func parseFramePair(funcLine, fileLine string) StackFrame {
	var frame StackFrame

	if idx := strings.Index(funcLine, "("); idx > 0 {
		funcLine = funcLine[:idx]
	}

	frame.Function = funcLine
	frame.Package = extractPackage(funcLine)
	frame.Category = categorizePackage(frame.Package)

	matches := fileLineRegex.FindStringSubmatch(fileLine)
	if len(matches) >= 3 {
		frame.File = matches[1]
		frame.Line, _ = strconv.Atoi(matches[2])
	}

	return frame
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
