package dom

// Flatten converts a StructuredNode tree into a flattened tree where
// fragments and components are removed, leaving only real DOM nodes
// (Element, Text, Comment). The resulting tree directly maps to
// what exists in the actual browser DOM.
func (n *StructuredNode) Flatten() *StructuredNode {
	if n == nil {
		return nil
	}

	if n.Fragment || n.ComponentID != "" {
		flattened := flattenChildren(n.Children)
		if len(flattened) == 0 {
			return nil
		}
		if len(flattened) == 1 {
			return flattened[0]
		}

		return &StructuredNode{
			Fragment: true,
			Children: flattened,
		}
	}

	if n.Text != "" {
		return &StructuredNode{
			Text: n.Text,
		}
	}

	if n.Comment != "" {
		return &StructuredNode{
			Comment: n.Comment,
		}
	}

	if n.Tag != "" {
		result := &StructuredNode{
			Tag:        n.Tag,
			Key:        n.Key,
			Attrs:      copyAttrs(n.Attrs),
			Style:      copyStyle(n.Style),
			Stylesheet: n.Stylesheet,
			RefID:      n.RefID,
			Handlers:   copyHandlers(n.Handlers),
			Events:     copyEvents(n.Events),
			Router:     n.Router,
			Upload:     n.Upload,
			UnsafeHTML: n.UnsafeHTML,
		}

		if n.UnsafeHTML == "" {
			result.Children = flattenChildren(n.Children)
		}

		return result
	}

	return nil
}

// flattenChildren processes a slice of children, flattening any
// fragments/components and returning only real DOM nodes.
func flattenChildren(children []*StructuredNode) []*StructuredNode {
	if len(children) == 0 {
		return nil
	}

	result := make([]*StructuredNode, 0, len(children))

	for _, child := range children {
		if child == nil {
			continue
		}

		if child.Fragment || child.ComponentID != "" {
			nested := flattenChildren(child.Children)
			result = append(result, nested...)
			continue
		}

		flattened := child.Flatten()
		if flattened != nil {
			result = append(result, flattened)
		}
	}

	return result
}

func copyAttrs(attrs map[string][]string) map[string][]string {
	if attrs == nil {
		return nil
	}
	result := make(map[string][]string, len(attrs))
	for k, v := range attrs {
		if len(v) > 0 {
			cp := make([]string, len(v))
			copy(cp, v)
			result[k] = cp
		}
	}
	return result
}

func copyStyle(style map[string]string) map[string]string {
	if style == nil {
		return nil
	}
	result := make(map[string]string, len(style))
	for k, v := range style {
		result[k] = v
	}
	return result
}

func copyHandlers(handlers []HandlerMeta) []HandlerMeta {
	if handlers == nil {
		return nil
	}
	result := make([]HandlerMeta, len(handlers))
	copy(result, handlers)
	return result
}

func copyEvents(events map[string]EventBinding) map[string]EventBinding {
	if events == nil {
		return nil
	}
	result := make(map[string]EventBinding, len(events))
	for k, v := range events {
		result[k] = v
	}
	return result
}
