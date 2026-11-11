package route

import (
	"net/url"
	"path"
	"strings"
)

func ResolveHref(current Location, href string) Location {
	if href == "" {
		return current.Clone()
	}

	if strings.HasPrefix(href, "/") {
		parsed, err := url.Parse(href)
		if err != nil {
			return current.Clone()
		}
		return Location{Path: parsed.Path, Query: parsed.Query(), Hash: NormalizeHash(parsed.Fragment)}
	}

	if strings.HasPrefix(href, "#") {
		return Location{Path: current.Path, Query: cloneValues(current.Query), Hash: NormalizeHash(strings.TrimPrefix(href, "#"))}
	}

	if strings.HasPrefix(href, "./") || strings.HasPrefix(href, "../") {
		basePath := current.Path
		if !strings.HasSuffix(basePath, "/") {
			if idx := strings.LastIndex(basePath, "/"); idx >= 0 {
				basePath = basePath[:idx+1]
			}
		}
		parsed, err := url.Parse(basePath + href)
		if err != nil {
			return current.Clone()
		}
		return Location{Path: path.Clean(parsed.Path), Query: parsed.Query(), Hash: NormalizeHash(parsed.Fragment)}
	}

	parsed, err := url.Parse(href)
	if err != nil {
		return current.Clone()
	}
	p := parsed.Path
	if p == "" {
		p = current.Path
	}
	return Location{Path: p, Query: parsed.Query(), Hash: NormalizeHash(parsed.Fragment)}
}
