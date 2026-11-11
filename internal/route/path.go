package route

import "strings"

type PathParts struct {
	Path     string
	RawQuery string
	Hash     string
}

func NormalizeParts(path string) PathParts {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return PathParts{Path: "/"}
	}

	hash := ""
	if idx := strings.Index(trimmed, "#"); idx >= 0 {
		hash = trimmed[idx+1:]
		trimmed = trimmed[:idx]
	}

	rawQuery := ""
	if idx := strings.Index(trimmed, "?"); idx >= 0 {
		rawQuery = trimmed[idx+1:]
		trimmed = trimmed[:idx]
	}

	return PathParts{
		Path:     NormalizePath(trimmed),
		RawQuery: rawQuery,
		Hash:     hash,
	}
}

func NormalizeHash(hash string) string {
	if hash == "" {
		return ""
	}
	return strings.TrimLeft(hash, "#")
}

func NormalizePath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	parts := strings.Split(trimmed, "/")
	segs := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		segs = append(segs, part)
	}
	if len(segs) == 0 {
		return "/"
	}
	return "/" + strings.Join(segs, "/")
}
