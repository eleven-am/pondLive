package css

import (
	"regexp"
	"strings"
)

var (
	ruleRegex      = regexp.MustCompile(`([^{}]+)\{([^{}]*)\}`)
	mediaRegex     = regexp.MustCompile(`@media\s*([^\{]+)\{((?:[^{}]|\{[^{}]*\})*)\}`)
	commentRegex   = regexp.MustCompile(`(?s)/\*.*?\*/`)
	otherAtRegex   = regexp.MustCompile(`@(?:[a-zA-Z-]+)\s*[^{]*\{(?:[^{}]|\{[^{}]*\})*\}`)
	keyframesRegex = regexp.MustCompile(`@keyframes\s+([a-zA-Z0-9_-]+)\s*\{((?:[^{}]|\{[^{}]*\})*)\}`)
)

func ParseAndScope(css string, componentID string) *Stylesheet {
	parsed := parse(css)
	hash := hashComponent(componentID)
	return scope(parsed, hash)
}

type parsedCSS struct {
	rules      []rule
	mediaRules []mediaRule
	other      []string
	keyframes  []keyframesBlock
}

type keyframesBlock struct {
	name   string
	blocks []rule
}

func parseKeyframes(body string) []rule {
	return parseRules(body)
}

type rule struct {
	selector     string
	declarations string
}

type mediaRule struct {
	query string
	rules []rule
}

func parse(css string) *parsedCSS {
	css = commentRegex.ReplaceAllString(css, "")
	result := &parsedCSS{}
	mediaMatches := mediaRegex.FindAllStringSubmatch(css, -1)
	for _, match := range mediaMatches {
		if len(match) < 3 {
			continue
		}
		inner := parseRules(match[2])
		result.mediaRules = append(result.mediaRules, mediaRule{query: strings.TrimSpace(match[1]), rules: inner})
	}
	css = mediaRegex.ReplaceAllString(css, "")

	kfMatches := keyframesRegex.FindAllStringSubmatch(css, -1)
	for _, match := range kfMatches {
		if len(match) < 3 {
			continue
		}
		blocks := parseKeyframes(match[2])
		result.keyframes = append(result.keyframes, keyframesBlock{name: strings.TrimSpace(match[1]), blocks: blocks})
	}
	css = keyframesRegex.ReplaceAllString(css, "")

	otherMatches := otherAtRegex.FindAllString(css, -1)
	if len(otherMatches) > 0 {
		result.other = append(result.other, otherMatches...)
		css = otherAtRegex.ReplaceAllString(css, "")
	}
	result.rules = parseRules(css)
	return result
}

func parseRules(css string) []rule {
	matches := ruleRegex.FindAllStringSubmatch(css, -1)
	var rules []rule
	for _, match := range matches {
		if len(match) < 3 {
			continue
		}
		selector := strings.TrimSpace(match[1])
		decl := strings.TrimSpace(match[2])
		if selector == "" || decl == "" {
			continue
		}
		rules = append(rules, rule{selector: selector, declarations: decl})
	}
	return rules
}
