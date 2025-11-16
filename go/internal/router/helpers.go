package router

import (
	"github.com/eleven-am/pondlive/go/internal/route"
)

func normalizePath(path string) string {
	return route.NormalizeParts(path).Path
}
