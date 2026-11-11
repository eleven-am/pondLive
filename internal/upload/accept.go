package upload

import (
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

const (
	genericBinary = "application/octet-stream"
	genericText   = "text/plain"
)

var extensionTypes = map[string]string{
	".txt":      "text/plain",
	".text":     "text/plain",
	".log":      "text/plain",
	".md":       "text/markdown",
	".markdown": "text/markdown",
	".csv":      "text/csv",
	".tsv":      "text/tab-separated-values",
	".yaml":     "text/yaml",
	".yml":      "text/yaml",
	".html":     "text/html",
	".htm":      "text/html",
	".css":      "text/css",
	".xml":      "text/xml",
	".js":       "text/javascript",
	".json":     "application/json",
	".pdf":      "application/pdf",
	".zip":      "application/zip",
	".gz":       "application/gzip",
	".png":      "image/png",
	".jpg":      "image/jpeg",
	".jpeg":     "image/jpeg",
	".jpe":      "image/jpeg",
	".gif":      "image/gif",
	".webp":     "image/webp",
	".avif":     "image/avif",
	".bmp":      "image/bmp",
	".tif":      "image/tiff",
	".tiff":     "image/tiff",
	".svg":      "image/svg+xml",
	".ico":      "image/vnd.microsoft.icon",
	".mp3":      "audio/mpeg",
	".wav":      "audio/wav",
	".ogg":      "audio/ogg",
	".mp4":      "video/mp4",
	".webm":     "video/webm",
	".mpg":      "video/mpeg",
	".mpeg":     "video/mpeg",
	".docx":     "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xlsx":     "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".pptx":     "application/vnd.openxmlformats-officedocument.presentationml.presentation",
	".odt":      "application/vnd.oasis.opendocument.text",
	".ods":      "application/vnd.oasis.opendocument.spreadsheet",
	".odp":      "application/vnd.oasis.opendocument.presentation",
	".epub":     "application/epub+zip",
	".tar":      "application/x-tar",
	".7z":       "application/x-7z-compressed",
	".rar":      "application/x-rar-compressed",
	".rtf":      "text/rtf",
	".heic":     "image/heic",
	".woff":     "font/woff",
	".woff2":    "font/woff2",
	".ttf":      "font/ttf",
	".otf":      "font/otf",
}

func normalizeMIME(value string) string {
	if i := strings.IndexByte(value, ';'); i >= 0 {
		value = value[:i]
	}
	return strings.ToLower(strings.TrimSpace(value))
}

func normalizePattern(pattern string) string {
	return strings.ToLower(strings.TrimSpace(pattern))
}

func matchesAny(pattern string) bool {
	return pattern == "*" || pattern == "*/*"
}

func isExtensionPattern(pattern string) bool {
	return strings.HasPrefix(pattern, ".")
}

func isWildcardPattern(pattern string) bool {
	return strings.HasSuffix(pattern, "/*")
}

func extensionType(pattern string) string {
	ext := pattern
	if i := strings.LastIndexByte(ext, '.'); i > 0 {
		ext = ext[i:]
	}
	return extensionTypes[ext]
}

func family(mimeType string) string {
	if i := strings.IndexByte(mimeType, '/'); i > 0 {
		return mimeType[:i]
	}
	return ""
}

func declaredCompatible(declared, want string) bool {
	d := normalizeMIME(declared)
	if d == "" || d == genericBinary || want == "" || d == want {
		return true
	}
	if family(d) == "text" && family(want) == "text" {
		return true
	}
	if m := mimetype.Lookup(want); m != nil && m.Is(d) {
		return true
	}
	return false
}

func matchDeclared(pattern, lowerName, declared string, strictExt bool) bool {
	switch {
	case pattern == "":
		return false
	case matchesAny(pattern):
		return true
	case isExtensionPattern(pattern):
		if !strings.HasSuffix(lowerName, pattern) {
			return false
		}
		if !strictExt {
			return true
		}
		want := extensionType(pattern)
		return want != "" && declaredCompatible(declared, want)
	case isWildcardPattern(pattern):
		return declared != "" && strings.HasPrefix(declared, pattern[:len(pattern)-1])
	default:
		return declared == pattern
	}
}

func acceptsDeclared(accept []string, name, declaredType string, strictExt bool) bool {
	if len(accept) == 0 {
		return true
	}
	lowerName := strings.ToLower(name)
	declared := normalizeMIME(declaredType)
	for _, raw := range accept {
		if matchDeclared(normalizePattern(raw), lowerName, declared, strictExt) {
			return true
		}
	}
	return false
}

func AcceptsDeclared(accept []string, name, declaredType string) bool {
	return acceptsDeclared(accept, name, declaredType, false)
}

func AcceptsUpload(accept []string, name, declaredType string) bool {
	return acceptsDeclared(accept, name, declaredType, true)
}

func AcceptsContent(accept []string, name string, detected *mimetype.MIME) bool {
	if len(accept) == 0 {
		return true
	}
	if detected == nil {
		return false
	}
	chain := detectedChain(detected)
	for _, raw := range accept {
		pattern := normalizePattern(raw)
		switch {
		case pattern == "":
		case matchesAny(pattern):
			return true
		case isExtensionPattern(pattern):
			if want := extensionType(pattern); want != "" && contentSatisfies(chain, want) {
				return true
			}
		case isWildcardPattern(pattern):
			prefix := pattern[:len(pattern)-1]
			for _, m := range chain {
				if strings.HasPrefix(normalizeMIME(m.String()), prefix) {
					return true
				}
			}
		default:
			if contentSatisfies(chain, pattern) {
				return true
			}
		}
	}
	return false
}

func detectedChain(detected *mimetype.MIME) []*mimetype.MIME {
	chain := []*mimetype.MIME{detected}
	for parent := detected.Parent(); parent != nil; parent = parent.Parent() {
		if parent.Is(genericBinary) && !detected.Is(genericBinary) {
			continue
		}
		chain = append(chain, parent)
	}
	return chain
}

func contentSatisfies(chain []*mimetype.MIME, want string) bool {
	for _, m := range chain {
		if m.Is(want) {
			return true
		}
	}
	if family(want) == "text" {
		for _, m := range chain {
			if m.Is(genericText) {
				return true
			}
		}
	}
	return false
}

func UnknownExtensions(accept []string) []string {
	var unknown []string
	for _, raw := range accept {
		pattern := normalizePattern(raw)
		if isExtensionPattern(pattern) && extensionType(pattern) == "" {
			unknown = append(unknown, pattern)
		}
	}
	return unknown
}
