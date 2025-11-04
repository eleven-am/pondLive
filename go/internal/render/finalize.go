package render

import (
	"sort"
	"strings"

	"github.com/eleven-am/go/pondlive/internal/handlers"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
)

var voidElements = map[string]struct{}{
	"area": {}, "base": {}, "br": {}, "col": {}, "embed": {}, "hr": {}, "img": {}, "input": {},
	"link": {}, "meta": {}, "param": {}, "source": {}, "track": {}, "wbr": {},
}

// Finalize normalizes a node tree by folding metadata into attributes.
func Finalize(n h.Node) h.Node {
	finalizeNode(n, nil)
	return n
}

// FinalizeWithHandlers normalizes the tree and attaches handler IDs using the
// provided registry.
func FinalizeWithHandlers(n h.Node, reg handlers.Registry) h.Node {
	finalizeNode(n, reg)
	return n
}

func finalizeNode(n h.Node, reg handlers.Registry) {
	switch v := n.(type) {
	case *h.Element:
		finalizeElement(v, reg)
		if v.Unsafe != nil {
			return
		}
		for _, child := range v.Children {
			if child == nil {
				continue
			}
			finalizeNode(child, reg)
		}
	case *h.FragmentNode:
		for _, child := range v.Children {
			if child == nil {
				continue
			}
			finalizeNode(child, reg)
		}
	}
}

func finalizeElement(e *h.Element, reg handlers.Registry) {
	if e == nil {
		return
	}
	if _, ok := voidElements[strings.ToLower(e.Tag)]; ok {
		e.Children = nil
		e.Unsafe = nil
	}
	if len(e.Class) > 0 {
		existing := ""
		if e.Attrs != nil {
			existing = e.Attrs["class"]
		}
		joined := joinClasses(existing, e.Class)
		if joined != "" {
			if e.Attrs == nil {
				e.Attrs = map[string]string{}
			}
			e.Attrs["class"] = joined
		}
		e.Class = nil
	}
	if len(e.Style) > 0 {
		existing := ""
		if e.Attrs != nil {
			existing = e.Attrs["style"]
		}
		joined := joinStyles(e.Style, existing)
		if joined != "" {
			if e.Attrs == nil {
				e.Attrs = map[string]string{}
			}
			e.Attrs["style"] = joined
		}
		e.Style = nil
	}
	if e.Unsafe != nil {
		e.Children = nil
	}

	attachHandlers(e, reg)

	if e.Key != "" {
		if e.Attrs == nil {
			e.Attrs = map[string]string{}
		}
		if _, exists := e.Attrs["data-row-key"]; !exists {
			e.Attrs["data-row-key"] = e.Key
		}
	}
}

func attachHandlers(e *h.Element, reg handlers.Registry) {
	if e == nil || len(e.Events) == 0 || reg == nil {
		return
	}
	if e.Attrs == nil {
		e.Attrs = map[string]string{}
	}
	keys := make([]string, 0, len(e.Events))
	for name := range e.Events {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		handler := e.Events[name]
		id := reg.Ensure(handler)
		if id == "" {
			continue
		}
		attrName := "data-on" + name
		e.Attrs[attrName] = string(id)
	}
}

func joinClasses(existing string, classes []string) string {
	seen := map[string]struct{}{}
	ordered := make([]string, 0, len(classes))
	for _, token := range strings.Fields(existing) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		ordered = append(ordered, token)
	}
	for _, c := range classes {
		for _, token := range strings.Fields(c) {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			ordered = append(ordered, token)
		}
	}
	return strings.Join(ordered, " ")
}

func joinStyles(styleMap map[string]string, existing string) string {
	if len(styleMap) == 0 && strings.TrimSpace(existing) == "" {
		return ""
	}
	merged := map[string]string{}
	if existing != "" {
		for _, decl := range strings.Split(existing, ";") {
			decl = strings.TrimSpace(decl)
			if decl == "" {
				continue
			}
			parts := strings.SplitN(decl, ":", 2)
			key := strings.TrimSpace(parts[0])
			if key == "" {
				continue
			}
			val := ""
			if len(parts) > 1 {
				val = strings.TrimSpace(parts[1])
			}
			merged[key] = val
		}
	}
	for k, v := range styleMap {
		if strings.TrimSpace(k) == "" {
			continue
		}
		merged[k] = v
	}
	if len(merged) == 0 {
		return ""
	}
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(":")
		b.WriteString(merged[k])
		b.WriteString(";")
	}
	return b.String()
}
