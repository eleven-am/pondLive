package router2

import "github.com/eleven-am/pondlive/go/internal/pathutil"

func normalizePath(path string) string {
	return pathutil.NormalizeParts(path).Path
}
