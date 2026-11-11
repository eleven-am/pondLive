package goroutines

import (
	"regexp"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

type Goroutine struct {
	ID      string
	State   string
	Frames  []string
	Creator string
	Created string
	Body    string
	Stack   string
}

var (
	header  = regexp.MustCompile(`^goroutine (\d+) \[([^\]]*)\]`)
	creator = regexp.MustCompile(`^created by (\S+)`)
)

func Snapshot() []Goroutine {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return Parse(string(buf[:n]))
		}
		buf = make([]byte, len(buf)*2)
	}
}

func Parse(dump string) []Goroutine {
	var out []Goroutine
	for _, block := range strings.Split(dump, "\n\n") {
		head, rest, _ := strings.Cut(block, "\n")
		h := header.FindStringSubmatch(head)
		if h == nil {
			continue
		}
		g := Goroutine{ID: h[1], State: h[2], Body: strings.TrimSpace(rest), Stack: strings.TrimSpace(block)}
		lines := strings.Split(rest, "\n")
		for i, line := range lines {
			if c := creator.FindStringSubmatch(line); c != nil {
				g.Creator = c[1]
				g.Created = strings.TrimSpace(strings.Join(lines[i:], "\n"))
				break
			}
			if strings.HasPrefix(line, "\t") {
				g.Frames = append(g.Frames, FrameLocation(line))
			}
		}
		out = append(out, g)
	}
	return out
}

func FrameLocation(line string) string {
	line = strings.TrimSpace(line)
	if i := strings.LastIndex(line, " +0x"); i >= 0 {
		line = line[:i]
	}
	return line
}

func SameFrames(a, b []string) bool {
	return len(a) > 0 && slices.Equal(a, b)
}

type Frame struct {
	Function string
	File     string
	Line     int
}

var fileLine = regexp.MustCompile(`^(.+):(\d+)`)

func ParseFrames(stack string) []Frame {
	if stack == "" {
		return nil
	}
	lines := strings.Split(stack, "\n")
	var frames []Frame
	for i := 0; i < len(lines); i += 2 {
		for i < len(lines) && (strings.TrimSpace(lines[i]) == "" || strings.HasPrefix(strings.TrimSpace(lines[i]), "goroutine ")) {
			i++
		}
		if i+1 >= len(lines) {
			break
		}
		fn := strings.TrimSpace(lines[i])
		frame := Frame{Function: withoutArguments(fn)}
		if m := fileLine.FindStringSubmatch(FrameLocation(lines[i+1])); m != nil {
			frame.File = m[1]
			frame.Line, _ = strconv.Atoi(m[2])
		}
		frames = append(frames, frame)
	}
	return frames
}

func withoutArguments(fn string) string {
	if !strings.HasSuffix(fn, ")") {
		return fn
	}
	depth := 0
	for i := len(fn) - 1; i >= 0; i-- {
		switch fn[i] {
		case ')':
			depth++
		case '(':
			depth--
			if depth == 0 {
				return fn[:i]
			}
		}
	}
	return fn
}
