package runtime

import (
	"bufio"
	"os"
	"regexp"
	"runtime"
)

var varNameRegex = regexp.MustCompile(`(?:var\s+)?(\w+)\s*(?:=|:=)`)

func captureComponentName(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		return ""
	}
	return extractVarNameFromSource(file, line)
}

func extractVarNameFromSource(file string, lineNum int) string {
	f, err := os.Open(file)
	if err != nil {
		return ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	current := 0
	for scanner.Scan() {
		current++
		if current == lineNum {
			return parseVarName(scanner.Text())
		}
	}
	return ""
}

func parseVarName(line string) string {
	if matches := varNameRegex.FindStringSubmatch(line); len(matches) > 1 {
		return matches[1]
	}
	return ""
}
