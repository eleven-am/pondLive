package pathutil

import "strings"

// PathParts represents the normalized components of a location-oriented path
// string, exposing the canonical path along with any stripped query or hash
// fragments.
type PathParts struct {
	Path     string
	RawQuery string
	Hash     string
}

// NormalizeParts canonicalizes a URL path by trimming whitespace, removing
// query and hash fragments, ensuring a single leading slash, and collapsing
// redundant separators. The removed query and hash fragments are returned for
// callers that need to retain them separately.
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
		Path:     normalizeSegments(trimmed),
		RawQuery: rawQuery,
		Hash:     hash,
	}
}

// Normalize canonicalizes a URL path by trimming whitespace, removing query and
// hash fragments, ensuring a single leading slash, and collapsing redundant
// separators.
func Normalize(path string) string {
	return NormalizeParts(path).Path
}

// NormalizeHash removes any leading hash prefix from the provided fragment,
// returning the canonical value suitable for Location usage.
func NormalizeHash(hash string) string {
	if hash == "" {
		return ""
	}
	return strings.TrimPrefix(hash, "#")
}

func normalizeSegments(path string) string {
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
