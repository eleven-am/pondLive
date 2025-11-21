package runtime

import "github.com/eleven-am/pondlive/go/internal/dom"

func cloneTree(node *dom.StructuredNode) *dom.StructuredNode {
	if node == nil {
		return nil
	}

	clone := &dom.StructuredNode{
		ComponentID: node.ComponentID,
		Tag:         node.Tag,
		Text:        node.Text,
		Comment:     node.Comment,
		Fragment:    node.Fragment,
		Key:         node.Key,
		UnsafeHTML:  node.UnsafeHTML,
		RefID:       node.RefID,
	}

	if len(node.Children) > 0 {
		clone.Children = make([]*dom.StructuredNode, len(node.Children))
		for i, child := range node.Children {
			clone.Children[i] = cloneTree(child)
		}
	}

	if len(node.Attrs) > 0 {
		clone.Attrs = cloneAttrs(node.Attrs)
	}
	if len(node.Style) > 0 {
		clone.Style = cloneStyle(node.Style)
	}
	if len(node.Styles) > 0 {
		clone.Styles = cloneStyles(node.Styles)
	}
	if len(node.Handlers) > 0 {
		clone.Handlers = append([]dom.HandlerMeta(nil), node.Handlers...)
	}
	if node.Router != nil {
		r := *node.Router
		clone.Router = &r
	}
	if node.Upload != nil {
		u := *node.Upload
		clone.Upload = &u
	}
	if len(node.UploadBindings) > 0 {
		clone.UploadBindings = append([]dom.UploadBinding(nil), node.UploadBindings...)
	}
	if len(node.Events) > 0 {
		clone.Events = make(map[string]dom.EventBinding, len(node.Events))
		for k, v := range node.Events {
			clone.Events[k] = v
		}
	}

	return clone
}

func cloneAttrs(src map[string][]string) map[string][]string {
	if src == nil {
		return nil
	}
	dst := make(map[string][]string, len(src))
	for k, v := range src {
		copied := append([]string(nil), v...)
		dst[k] = copied
	}
	return dst
}

func cloneStyle(src map[string]string) map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func cloneStyles(src map[string]map[string]string) map[string]map[string]string {
	if src == nil {
		return nil
	}
	dst := make(map[string]map[string]string, len(src))
	for selector, props := range src {
		copied := make(map[string]string, len(props))
		for k, v := range props {
			copied[k] = v
		}
		dst[selector] = copied
	}
	return dst
}
