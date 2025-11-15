package css

import (
	"regexp"
	"strings"
)

type Rule struct {
	Selector     string
	Declarations string
	StartPos     int
	EndPos       int
}

type ParseResult struct {
	Rules      []Rule
	Selectors  []string
	MediaRules []MediaRule
}

type MediaRule struct {
	Query     string
	Rules     []Rule
	Selectors []string
	StartPos  int
	EndPos    int
}

var (
	ruleRegex    = regexp.MustCompile(`([^{}]+)\{([^{}]*)\}`)
	mediaRegex   = regexp.MustCompile(`@media\s*([^{]+)\{((?:[^{}]|\{[^{}]*\})*)\}`)
	commentRegex = regexp.MustCompile(`/\*.*?\*/`)
)

func Parse(css string) *ParseResult {
	result := &ParseResult{
		Rules:      []Rule{},
		Selectors:  []string{},
		MediaRules: []MediaRule{},
	}

	css = commentRegex.ReplaceAllString(css, "")

	mediaMatches := mediaRegex.FindAllStringSubmatch(css, -1)
	mediaIndexes := mediaRegex.FindAllStringIndex(css, -1)

	for i, match := range mediaMatches {
		if len(match) < 3 {
			continue
		}
		query := strings.TrimSpace(match[1])
		innerCSS := match[2]

		mediaRule := MediaRule{
			Query:     query,
			Rules:     []Rule{},
			Selectors: []string{},
		}

		if len(mediaIndexes) > i {
			mediaRule.StartPos = mediaIndexes[i][0]
			mediaRule.EndPos = mediaIndexes[i][1]
		}

		innerMatches := ruleRegex.FindAllStringSubmatch(innerCSS, -1)
		for _, innerMatch := range innerMatches {
			if len(innerMatch) < 3 {
				continue
			}
			selector := strings.TrimSpace(innerMatch[1])
			declarations := strings.TrimSpace(innerMatch[2])

			mediaRule.Rules = append(mediaRule.Rules, Rule{
				Selector:     selector,
				Declarations: declarations,
			})

			selectors := extractSelectors(selector)
			mediaRule.Selectors = append(mediaRule.Selectors, selectors...)
			result.Selectors = appendUnique(result.Selectors, selectors)
		}

		result.MediaRules = append(result.MediaRules, mediaRule)
	}

	cssWithoutMedia := mediaRegex.ReplaceAllString(css, "")

	matches := ruleRegex.FindAllStringSubmatch(cssWithoutMedia, -1)
	indexes := ruleRegex.FindAllStringIndex(cssWithoutMedia, -1)

	for i, match := range matches {
		if len(match) < 3 {
			continue
		}

		selector := strings.TrimSpace(match[1])
		declarations := strings.TrimSpace(match[2])

		rule := Rule{
			Selector:     selector,
			Declarations: declarations,
		}

		if len(indexes) > i {
			rule.StartPos = indexes[i][0]
			rule.EndPos = indexes[i][1]
		}

		result.Rules = append(result.Rules, rule)

		selectors := extractSelectors(selector)
		result.Selectors = appendUnique(result.Selectors, selectors)
	}

	return result
}

func extractSelectors(selector string) []string {
	parts := strings.Split(selector, ",")
	selectors := make([]string, 0, len(parts))

	for _, part := range parts {
		s := strings.TrimSpace(part)
		if s != "" {
			selectors = append(selectors, s)
		}
	}

	return selectors
}

func appendUnique(slice []string, items []string) []string {
	seen := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		seen[s] = struct{}{}
	}

	for _, item := range items {
		if _, exists := seen[item]; !exists {
			slice = append(slice, item)
			seen[item] = struct{}{}
		}
	}

	return slice
}

// ExtractBaseSelector extracts the base class or ID from a complex selector.
// .card:hover -> .card
// .card::before -> .card
// #header -> #header
// .parent .child -> .parent
func ExtractBaseSelector(selector string) string {
	s := strings.TrimSpace(selector)
	if s == "" {
		return ""
	}

	if idx := strings.IndexAny(s, " \t\n\r>+~"); idx != -1 {
		s = s[:idx]
	}

	if idx := strings.IndexAny(s, ":"); idx != -1 {
		s = s[:idx]
	}

	if strings.HasPrefix(s, ".") {
		parts := strings.Split(s[1:], ".")
		if len(parts) > 0 && parts[0] != "" {
			return "." + parts[0]
		}
	}

	if strings.HasPrefix(s, "#") {
		if idx := strings.IndexAny(s[1:], ".[:"); idx != -1 {
			return s[:idx+1]
		}
		return s
	}

	return s
}
