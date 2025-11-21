package css

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

func scope(parsed *parsedCSS, hash string) *Stylesheet {
	ss := &Stylesheet{SelectorHash: hash}
	for _, r := range parsed.rules {
		blocks := scopeRule(r, hash)
		ss.Rules = append(ss.Rules, blocks...)
	}
	for _, mr := range parsed.mediaRules {
		media := MediaRule{Query: mr.query}
		for _, r := range mr.rules {
			media.Rules = append(media.Rules, scopeRule(r, hash)...)
		}
		ss.MediaRules = append(ss.MediaRules, media)
	}
	return ss
}

func scopeRule(r rule, hash string) []SelectorBlock {
	selectors := strings.Split(r.selector, ",")
	props := parseDeclarations(r.declarations)
	blocks := make([]SelectorBlock, 0, len(selectors))
	for _, sel := range selectors {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		blocks = append(blocks, SelectorBlock{
			Selector: scopeSelector(sel, hash),
			Props:    props,
		})
	}
	return blocks
}

func parseDeclarations(decl string) PropertyMap {
	props := PropertyMap{}
	parts := strings.Split(decl, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		if key == "" || value == "" {
			continue
		}
		props[key] = value
	}
	return props
}

func scopeSelector(selector, hash string) string {
	selector = strings.TrimSpace(selector)
	if selector == "" || selector == "*" || strings.HasPrefix(selector, "@") {
		return selector
	}
	if strings.HasPrefix(selector, ":root") || strings.HasPrefix(selector, "html") || strings.HasPrefix(selector, "body") {
		return selector
	}
	parts := tokenize(selector)
	for i, part := range parts {
		trim := strings.TrimSpace(part)
		if strings.HasPrefix(trim, ".") || strings.HasPrefix(trim, "#") {
			parts[i] = scopeToken(trim, hash)
			break
		}
	}
	return strings.Join(parts, "")
}

func tokenize(selector string) []string {
	var tokens []string
	var current strings.Builder
	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}
	for _, ch := range selector {
		switch ch {
		case ' ', '>', '+', '~':
			flush()
			if ch == ' ' {
				tokens = append(tokens, " ")
			} else {
				tokens = append(tokens, " "+string(ch)+" ")
			}
		case ':':
			flush()
			rest := selector[strings.IndexRune(selector, ch):]
			tokens = append(tokens, rest)
			return tokens
		default:
			current.WriteRune(ch)
		}
	}
	flush()
	return tokens
}

func scopeToken(token, hash string) string {
	if strings.HasPrefix(token, ".") {
		return "." + strings.TrimPrefix(token, ".") + "-" + hash
	}
	if strings.HasPrefix(token, "#") {
		return "#" + strings.TrimPrefix(token, "#") + "-" + hash
	}
	return token
}

func hashComponent(componentID string) string {
	if componentID == "" {
		return ""
	}
	h := sha256.Sum256([]byte(componentID))
	return hex.EncodeToString(h[:])[:8]
}

// Serialize converts the structured stylesheet back into CSS text (for SSR).
func (ss *Stylesheet) Serialize() string {
	var b strings.Builder
	for _, rule := range ss.Rules {
		b.WriteString(rule.Selector)
		b.WriteString(" { ")
		b.WriteString(joinProps(rule.Props))
		b.WriteString(" }\n")
	}
	for _, media := range ss.MediaRules {
		b.WriteString("@media ")
		b.WriteString(media.Query)
		b.WriteString(" {\n")
		for _, rule := range media.Rules {
			b.WriteString("  ")
			b.WriteString(rule.Selector)
			b.WriteString(" { ")
			b.WriteString(joinProps(rule.Props))
			b.WriteString(" }\n")
		}
		b.WriteString("}\n")
	}
	return b.String()
}

func joinProps(props PropertyMap) string {
	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(k)
		b.WriteString(": ")
		b.WriteString(props[k])
	}
	return b.String()
}
