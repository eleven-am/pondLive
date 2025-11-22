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
	if node.Stylesheet != nil {
		clone.Stylesheet = cloneStylesheet(node.Stylesheet)
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
	if node.Script != nil {
		s := *node.Script
		clone.Script = &s
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

func cloneStylesheet(src *dom.Stylesheet) *dom.Stylesheet {
	if src == nil {
		return nil
	}
	dst := &dom.Stylesheet{
		Hash: src.Hash,
	}

	if len(src.Rules) > 0 {
		dst.Rules = make([]dom.StyleRule, len(src.Rules))
		for i, rule := range src.Rules {
			dst.Rules[i] = dom.StyleRule{
				Selector: rule.Selector,
				Props:    cloneStyle(rule.Props),
			}
		}
	}

	if len(src.MediaBlocks) > 0 {
		dst.MediaBlocks = make([]dom.MediaBlock, len(src.MediaBlocks))
		for i, media := range src.MediaBlocks {
			dst.MediaBlocks[i] = dom.MediaBlock{
				Query: media.Query,
			}
			if len(media.Rules) > 0 {
				dst.MediaBlocks[i].Rules = make([]dom.StyleRule, len(media.Rules))
				for j, rule := range media.Rules {
					dst.MediaBlocks[i].Rules[j] = dom.StyleRule{
						Selector: rule.Selector,
						Props:    cloneStyle(rule.Props),
					}
				}
			}
		}
	}

	return dst
}
