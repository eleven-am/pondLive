package router

import (
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/eleven-am/pondlive/internal/route"
)

func canonicalizeLocation(loc Location) Location {
	if loc.Path == "" {
		loc.Path = "/"
	}

	parts := route.NormalizeParts(loc.Path)

	var canonQuery url.Values
	if len(loc.Query) > 0 {
		canonQuery = canonicalizeValues(loc.Query)
	} else {
		canonQuery = url.Values{}
	}

	canon := Location{
		Path:  parts.Path,
		Query: canonQuery,
		Hash:  normalizeHash(loc.Hash),
	}

	if canon.Hash == "" && parts.Hash != "" {
		canon.Hash = route.NormalizeHash(parts.Hash)
	}

	return canon
}

func cloneLocation(loc Location) Location {
	return Location{
		Path:  loc.Path,
		Query: cloneValues(loc.Query),
		Hash:  loc.Hash,
	}
}

func normalizeHash(hash string) string {
	return route.NormalizeHash(hash)
}

func cloneValues(q url.Values) url.Values {
	if len(q) == 0 {
		return url.Values{}
	}
	out := make(url.Values, len(q))
	for k, values := range q {
		cp := make([]string, len(values))
		copy(cp, values)
		out[k] = cp
	}
	return out
}

func canonicalizeValues(q url.Values) url.Values {
	if len(q) == 0 {
		return url.Values{}
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(url.Values, len(keys))
	for _, key := range keys {
		out[key] = canonicalizeList(q[key])
	}
	return out
}

func canonicalizeList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	cleaned := make([]string, 0, len(values))
	for _, v := range values {
		cleaned = append(cleaned, strings.TrimSpace(v))
	}
	sort.Strings(cleaned)
	return cleaned
}

func buildHref(p string, query url.Values, hash string) string {
	if p == "" {
		p = "/"
	}

	href := p

	if len(query) > 0 {
		encoded := query.Encode()
		if encoded != "" {
			href += "?" + encoded
		}
	}

	if hash != "" {
		if !strings.HasPrefix(hash, "#") {
			href += "#" + hash
		} else {
			href += hash
		}
	}

	return href
}

func resolveHref(current Location, href string) Location {
	if href == "" {
		return cloneLocation(current)
	}

	if strings.HasPrefix(href, "/") {
		parsed, err := url.Parse(href)
		if err != nil {
			return cloneLocation(current)
		}
		return Location{
			Path:  parsed.Path,
			Query: parsed.Query(),
			Hash:  normalizeHash(parsed.Fragment),
		}
	}

	if strings.HasPrefix(href, "#") {
		return Location{
			Path:  current.Path,
			Query: cloneValues(current.Query),
			Hash:  normalizeHash(strings.TrimPrefix(href, "#")),
		}
	}

	if strings.HasPrefix(href, "./") || strings.HasPrefix(href, "../") {
		basePath := current.Path
		if !strings.HasSuffix(basePath, "/") {
			if idx := strings.LastIndex(basePath, "/"); idx >= 0 {
				basePath = basePath[:idx+1]
			}
		}

		resolved := basePath + href
		parsed, err := url.Parse(resolved)
		if err != nil {
			return cloneLocation(current)
		}

		cleanPath := path.Clean(parsed.Path)

		return Location{
			Path:  cleanPath,
			Query: parsed.Query(),
			Hash:  normalizeHash(parsed.Fragment),
		}
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return cloneLocation(current)
	}

	p := parsed.Path
	if p == "" {
		p = current.Path
	}

	return Location{
		Path:  p,
		Query: parsed.Query(),
		Hash:  normalizeHash(parsed.Fragment),
	}
}

func matchesPrefix(currentPath, targetPath string) bool {
	if targetPath == "/" {
		return currentPath == "/"
	}

	if len(currentPath) < len(targetPath) {
		return false
	}

	if currentPath[:len(targetPath)] != targetPath {
		return false
	}

	if len(currentPath) > len(targetPath) {
		return currentPath[len(targetPath)] == '/'
	}
	return true
}

func resolveRoutePattern(raw, base string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		trimmed = "/"
	}
	if strings.HasPrefix(trimmed, "./") {
		rel := strings.TrimPrefix(trimmed, ".")
		return normalizePath(joinRelativePath(base, rel))
	}
	return normalizePath(trimmed)
}

func joinRelativePath(base, rel string) string {
	rel = normalizePath(rel)
	base = normalizePath(base)
	base = trimWildcardSuffix(base)
	if base == "/" {
		return rel
	}
	if rel == "/" {
		return base
	}
	return normalizePath(strings.TrimSuffix(base, "/") + rel)
}

func trimWildcardSuffix(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	normalized := normalizePath(trimmed)
	if normalized == "/" {
		return "/"
	}
	segments := strings.Split(strings.Trim(normalized, "/"), "/")
	if len(segments) == 0 {
		return "/"
	}
	last := segments[len(segments)-1]
	if strings.HasPrefix(last, "*") {
		segments = segments[:len(segments)-1]
	}
	if len(segments) == 0 {
		return "/"
	}
	return "/" + strings.Join(segments, "/")
}

func normalizePath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	for strings.Contains(path, "//") {
		path = strings.ReplaceAll(path, "//", "/")
	}
	if path != "/" && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}
