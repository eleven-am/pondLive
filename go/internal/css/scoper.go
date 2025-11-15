package css

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

type ScopedResult struct {
	CSS           string
	SelectorMap   map[string]string
	RuleMap       map[string]string
	ComponentHash string
}

func Scope(css string, componentID string) *ScopedResult {
	parsed := Parse(css)

	hash := generateHash(componentID)
	selectorMap := make(map[string]string)
	ruleMap := make(map[string]string)

	var builder strings.Builder

	for _, rule := range parsed.Rules {
		scopedSelector := scopeSelector(rule.Selector, hash, selectorMap)
		ruleText := scopedSelector + " { " + rule.Declarations + " }"
		builder.WriteString(ruleText)
		builder.WriteString("\n")

		base := ExtractBaseSelector(rule.Selector)
		if base != "" {
			if existing, ok := ruleMap[base]; ok {
				ruleMap[base] = existing + "\n" + ruleText
			} else {
				ruleMap[base] = ruleText
			}
		}
	}

	for _, media := range parsed.MediaRules {
		builder.WriteString("@media ")
		builder.WriteString(media.Query)
		builder.WriteString(" {\n")

		for _, rule := range media.Rules {
			scopedSelector := scopeSelector(rule.Selector, hash, selectorMap)
			ruleText := "  " + scopedSelector + " { " + rule.Declarations + " }"
			builder.WriteString(ruleText)
			builder.WriteString("\n")

			base := ExtractBaseSelector(rule.Selector)
			if base != "" {
				mediaRuleText := "@media " + media.Query + " { " + scopedSelector + " { " + rule.Declarations + " } }"
				if existing, ok := ruleMap[base]; ok {
					ruleMap[base] = existing + "\n" + mediaRuleText
				} else {
					ruleMap[base] = mediaRuleText
				}
			}
		}

		builder.WriteString("}\n")
	}

	return &ScopedResult{
		CSS:           builder.String(),
		SelectorMap:   selectorMap,
		RuleMap:       ruleMap,
		ComponentHash: hash,
	}
}

func scopeSelector(selector string, hash string, selectorMap map[string]string) string {
	parts := strings.Split(selector, ",")
	scoped := make([]string, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		scopedPart := scopeSingleSelector(part, hash)
		scoped = append(scoped, scopedPart)

		base := ExtractBaseSelector(part)
		if base != "" {
			scopedBase := ExtractBaseSelector(scopedPart)
			selectorMap[base] = scopedBase
		}
	}

	return strings.Join(scoped, ", ")
}

func scopeSingleSelector(selector string, hash string) string {
	selector = strings.TrimSpace(selector)

	if selector == "" || selector == "*" {
		return selector
	}

	if strings.HasPrefix(selector, "@") {
		return selector
	}

	if strings.HasPrefix(selector, ":root") ||
		strings.HasPrefix(selector, "html") ||
		strings.HasPrefix(selector, "body") {
		return selector
	}

	parts := tokenizeSelector(selector)

	for i, part := range parts {
		if isClassOrID(part) {
			parts[i] = scopeToken(part, hash)
			break
		}
	}

	return strings.Join(parts, "")
}

func tokenizeSelector(selector string) []string {
	var tokens []string
	var current strings.Builder

	for i, ch := range selector {
		switch ch {
		case ' ', '>', '+', '~':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			if ch == ' ' {
				tokens = append(tokens, " ")
			} else {
				tokens = append(tokens, " "+string(ch)+" ")
			}
		case ':':
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			rest := selector[i:]
			tokens = append(tokens, rest)
			return tokens
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens
}

func isClassOrID(token string) bool {
	token = strings.TrimSpace(token)
	return strings.HasPrefix(token, ".") || strings.HasPrefix(token, "#")
}

func scopeToken(token string, hash string) string {
	if strings.HasPrefix(token, ".") {
		className := token[1:]
		return "." + className + "-" + hash
	}

	if strings.HasPrefix(token, "#") {
		idName := token[1:]
		return "#" + idName + "-" + hash
	}

	return token
}

func generateHash(componentID string) string {
	h := sha256.New()
	h.Write([]byte(componentID))
	hash := hex.EncodeToString(h.Sum(nil))
	return hash[:8]
}

type StyleLookup struct {
	selectorMap map[string]string
	ruleMap     map[string]string
}

func NewStyleLookup(selectorMap map[string]string) *StyleLookup {
	return &StyleLookup{selectorMap: selectorMap, ruleMap: make(map[string]string)}
}

func NewStyleLookupWithRules(selectorMap, ruleMap map[string]string) *StyleLookup {
	return &StyleLookup{selectorMap: selectorMap, ruleMap: ruleMap}
}

func (s *StyleLookup) Get(selector string) string {
	selector = strings.TrimSpace(selector)

	if scoped, ok := s.selectorMap[selector]; ok {
		if strings.HasPrefix(scoped, ".") {
			return scoped[1:]
		}
		if strings.HasPrefix(scoped, "#") {
			return scoped[1:]
		}
		return scoped
	}

	return ""
}

func (s *StyleLookup) Class(className string) string {
	if !strings.HasPrefix(className, ".") {
		className = "." + className
	}
	return s.Get(className)
}

func (s *StyleLookup) ID(idName string) string {
	if !strings.HasPrefix(idName, "#") {
		idName = "#" + idName
	}
	result := s.Get(idName)
	if result == "" {
		return ""
	}
	if strings.HasPrefix(result, "#") {
		return result[1:]
	}
	return result
}

func (s *StyleLookup) Selector(sel string) string {
	return s.Get(sel)
}

func (s *StyleLookup) Call(args ...interface{}) string {
	if len(args) == 0 {
		return ""
	}

	selector := fmt.Sprint(args[0])
	return s.Get(selector)
}

func (s *StyleLookup) Rule(selector string) string {
	return s.ruleMap[selector]
}

func (s *StyleLookup) AllRules() map[string]string {
	return s.ruleMap
}
