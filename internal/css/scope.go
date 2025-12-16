package css

import (
	"crypto/sha256"
	"encoding/hex"
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
	for _, kf := range parsed.keyframes {
		ss.Keyframes = append(ss.Keyframes, scopeKeyframes(kf, hash))
	}
	ss.OtherBlocks = append(ss.OtherBlocks, parsed.other...)
	return ss
}

func scopeRule(r rule, hash string) []SelectorBlock {
	selectors := splitSelectors(r.selector)
	decls := parseDeclarations(r.declarations)
	blocks := make([]SelectorBlock, 0, len(selectors))
	for _, sel := range selectors {
		sel = strings.TrimSpace(sel)
		if sel == "" {
			continue
		}
		blocks = append(blocks, SelectorBlock{
			Selector: scopeSelector(sel, hash),
			Decls:    decls,
		})
	}
	return blocks
}

func splitSelectors(selector string) []string {
	var selectors []string
	var current strings.Builder
	depth := 0

	for i := 0; i < len(selector); i++ {
		ch := selector[i]
		switch ch {
		case '(':
			depth++
			current.WriteByte(ch)
		case ')':
			depth--
			current.WriteByte(ch)
		case ',':
			if depth == 0 {
				selectors = append(selectors, current.String())
				current.Reset()
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}
	if current.Len() > 0 {
		selectors = append(selectors, current.String())
	}
	return selectors
}

func parseDeclarations(decl string) []Declaration {
	var decls []Declaration

	parts := splitDeclarations(decl)
	for _, part := range parts {
		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}
		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		if key == "" || value == "" {
			continue
		}
		decls = append(decls, Declaration{Property: key, Value: value})
	}
	return decls
}

func splitDeclarations(decl string) []string {
	var parts []string
	var current strings.Builder
	depth := 0
	var quote byte

	flush := func() {
		if current.Len() > 0 {
			parts = append(parts, current.String())
			current.Reset()
		}
	}

	for i := 0; i < len(decl); i++ {
		ch := decl[i]
		switch ch {
		case '\\':

			if i+1 < len(decl) {
				current.WriteByte(ch)
				i++
				current.WriteByte(decl[i])
				continue
			}
		case '\'', '"':
			if quote == 0 {
				quote = ch
			} else if quote == ch {
				quote = 0
			}
		case '(':
			if quote == 0 {
				depth++
			}
		case ')':
			if quote == 0 && depth > 0 {
				depth--
			}
		case ';':
			if quote == 0 && depth == 0 {
				flush()
				continue
			}
		}
		current.WriteByte(ch)
	}
	flush()
	return parts
}

func scopeSelector(selector, hash string) string {
	selector = strings.TrimSpace(selector)
	if selector == "" || selector == "*" || strings.HasPrefix(selector, "@") {
		return selector
	}
	if strings.HasPrefix(selector, ":root") || strings.HasPrefix(selector, "html") || strings.HasPrefix(selector, "body") {
		return selector
	}
	if hash == "" {
		return selector
	}
	parts := tokenize(selector)
	for i, part := range parts {
		trim := strings.TrimSpace(part)
		if strings.Contains(trim, "(") {
			parts[i] = scopePseudoFunction(trim, hash)
		} else if result, ok := scopeSimpleSelector(trim, hash); ok {
			parts[i] = result
		}
	}
	return strings.Join(parts, "")
}

func tokenize(selector string) []string {
	var tokens []string
	var current strings.Builder
	depth := 0

	flush := func() {
		if current.Len() > 0 {
			tokens = append(tokens, current.String())
			current.Reset()
		}
	}

	for i := 0; i < len(selector); i++ {
		ch := selector[i]
		switch ch {
		case '(':
			depth++
			current.WriteByte(ch)
		case ')':
			depth--
			current.WriteByte(ch)
		case ' ', '>', '+', '~':
			if depth == 0 {
				flush()
				if ch == ' ' {
					tokens = append(tokens, " ")
				} else {
					tokens = append(tokens, " "+string(ch)+" ")
				}
			} else {
				current.WriteByte(ch)
			}
		default:
			current.WriteByte(ch)
		}
	}
	flush()
	return tokens
}

func scopeKeyframes(kf keyframesBlock, hash string) KeyframesBlock {
	name := kf.name
	if hash != "" && name != "" {
		name = name + "-" + hash
	}
	steps := make([]KeyframesStep, 0, len(kf.blocks))
	for _, step := range kf.blocks {
		decls := parseDeclarations(step.declarations)
		steps = append(steps, KeyframesStep{
			Selector: step.selector,
			Decls:    decls,
		})
	}
	return KeyframesBlock{Name: name, Steps: steps}
}

func scopeSimpleSelector(sel, hash string) (string, bool) {
	if sel == "" || hash == "" {
		return sel, false
	}

	var result strings.Builder
	i := 0
	scoped := false

	for i < len(sel) {
		ch := sel[i]

		if ch == '[' {
			end := strings.Index(sel[i:], "]")
			if end == -1 {
				result.WriteString(sel[i:])
				break
			}
			result.WriteString(sel[i : i+end+1])
			i += end + 1
			continue
		}

		if ch == ':' {
			result.WriteString(sel[i:])
			break
		}

		if ch == '.' || ch == '#' {
			j := i + 1
			for j < len(sel) && sel[j] != '.' && sel[j] != '#' && sel[j] != '[' && sel[j] != ':' {
				j++
			}
			name := sel[i+1 : j]
			result.WriteByte(ch)
			result.WriteString(name)
			result.WriteByte('-')
			result.WriteString(hash)
			scoped = true
			i = j
			continue
		}

		result.WriteByte(ch)
		i++
	}

	return result.String(), scoped
}

func scopePseudoFunction(sel, hash string) string {
	if hash == "" {
		return sel
	}

	parenIdx := strings.Index(sel, "(")
	if parenIdx == -1 {
		return sel
	}

	colonIdx := strings.LastIndex(sel[:parenIdx], ":")
	if colonIdx == -1 {
		colonIdx = 0
	}

	base := sel[:colonIdx]
	pseudoRest := sel[colonIdx:]

	if result, ok := scopeSimpleSelector(base, hash); ok {
		base = result
	}

	start := strings.Index(pseudoRest, "(")
	end := strings.LastIndex(pseudoRest, ")")
	if start == -1 || end == -1 || end <= start {
		return base + pseudoRest
	}

	prefix := pseudoRest[:start+1]
	inner := pseudoRest[start+1 : end]
	suffix := pseudoRest[end:]

	parts := strings.Split(inner, ",")
	for i, p := range parts {
		p = strings.TrimSpace(p)
		if result, ok := scopeSimpleSelector(p, hash); ok {
			parts[i] = result
		} else {
			parts[i] = p
		}
	}
	return base + prefix + strings.Join(parts, ", ") + suffix
}

func hashComponent(componentID string) string {
	if componentID == "" {
		return ""
	}
	h := sha256.Sum256([]byte(componentID))
	return hex.EncodeToString(h[:])[:8]
}
