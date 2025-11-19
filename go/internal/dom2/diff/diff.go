package diff

import (
	"reflect"

	"github.com/eleven-am/pondlive/go/internal/dom2"
)

func Diff(prev, next *dom2.StructuredNode) []Patch {
	patches := make([]Patch, 0)
	diffNode(&patches, nil, prev, next)
	return patches
}

func diffNode(patches *[]Patch, path []int, a, b *dom2.StructuredNode) {
	if a == nil && b == nil {
		return
	}
	if a == nil || b == nil {
		*patches = append(*patches, Patch{Path: copyPath(path), Op: OpReplaceNode, Value: b})
		return
	}

	aType := nodeTypeOf(a)
	bType := nodeTypeOf(b)
	if aType != bType {
		*patches = append(*patches, Patch{Path: copyPath(path), Op: OpReplaceNode, Value: b})
		return
	}

	if aType == nodeElement && a.Tag != b.Tag {
		*patches = append(*patches, Patch{Path: copyPath(path), Op: OpReplaceNode, Value: b})
		return
	}

	switch aType {
	case nodeText:
		if a.Text != b.Text {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpSetText, Value: b.Text})
		}
	case nodeComment:
		if a.Comment != b.Comment {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpSetComment, Value: b.Comment})
		}
	case nodeElement:
		diffElement(patches, path, a, b)
	case nodeComponent:
		if a.ComponentID != b.ComponentID {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpSetComponent, Value: b.ComponentID})
		}
		diffChildren(patches, path, a.Children, b.Children)
	case nodeFragment:
		diffChildren(patches, path, a.Children, b.Children)
	}
}

func diffElement(patches *[]Patch, path []int, a, b *dom2.StructuredNode) {
	diffAttrs(patches, path, a, b)
	diffStyle(patches, path, a, b)

	if a.Tag == "style" || b.Tag == "style" {
		diffStyles(patches, path, a, b)
	}

	if a.RefID != b.RefID {
		if b.RefID == "" {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpDelRef})
		} else {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpSetRef, Value: b.RefID})
		}
	}

	if !handlersEqual(a.Handlers, b.Handlers) {
		*patches = append(*patches, Patch{Path: copyPath(path), Op: OpSetHandlers, Value: b.Handlers})
	}

	if !routerEqual(a.Router, b.Router) {
		if b.Router == nil {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpDelRouter})
		} else {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpSetRouter, Value: b.Router})
		}
	}

	if !uploadEqual(a.Upload, b.Upload) {
		if b.Upload == nil {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpDelUpload})
		} else {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpSetUpload, Value: b.Upload})
		}
	}

	if a.UnsafeHTML != b.UnsafeHTML {
		*patches = append(*patches, Patch{Path: copyPath(path), Op: OpReplaceNode, Value: b})
		return
	}

	if b.UnsafeHTML == "" {
		diffChildren(patches, path, a.Children, b.Children)
	}
}

func diffChildren(patches *[]Patch, parentPath []int, a, b []*dom2.StructuredNode) {
	if hasKeys(a) || hasKeys(b) {
		diffChildrenKeyed(patches, parentPath, a, b)
	} else {
		diffChildrenIndexed(patches, parentPath, a, b)
	}
}

func diffChildrenIndexed(patches *[]Patch, parentPath []int, a, b []*dom2.StructuredNode) {
	max := len(a)
	if len(b) > max {
		max = len(b)
	}
	for i := 0; i < max; i++ {
		childPath := append(copyPath(parentPath), i)
		var childA, childB *dom2.StructuredNode
		if i < len(a) {
			childA = a[i]
		}
		if i < len(b) {
			childB = b[i]
		}
		if childA == nil && childB != nil {
			*patches = append(*patches, Patch{
				Path:  copyPath(parentPath),
				Op:    OpAddChild,
				Index: i,
				Value: childB,
			})
			continue
		}
		if childA != nil && childB == nil {
			*patches = append(*patches, Patch{
				Path:  copyPath(parentPath),
				Op:    OpDelChild,
				Index: i,
			})
			continue
		}
		diffNode(patches, childPath, childA, childB)
	}
}

func diffChildrenKeyed(patches *[]Patch, parentPath []int, a, b []*dom2.StructuredNode) {
	oldKeys := make(map[string]int)
	newKeys := make(map[string]int)

	for i, child := range a {
		if child != nil && child.Key != "" {
			oldKeys[child.Key] = i
		}
	}
	for i, child := range b {
		if child != nil && child.Key != "" {
			newKeys[child.Key] = i
		}
	}

	processed := make(map[int]bool)

	for newIdx := range b {
		newChild := b[newIdx]
		if newChild == nil || newChild.Key == "" {
			childPath := append(copyPath(parentPath), newIdx)
			if newIdx < len(a) && (a[newIdx] == nil || a[newIdx].Key == "") {
				diffNode(patches, childPath, a[newIdx], newChild)
				processed[newIdx] = true
			} else {
				*patches = append(*patches, Patch{
					Path:  copyPath(parentPath),
					Op:    OpAddChild,
					Index: newIdx,
					Value: newChild,
				})
			}
			continue
		}

		oldIdx, existedBefore := oldKeys[newChild.Key]
		if !existedBefore {
			*patches = append(*patches, Patch{
				Path:  copyPath(parentPath),
				Op:    OpAddChild,
				Index: newIdx,
				Value: newChild,
			})
			continue
		}

		processed[oldIdx] = true

		if oldIdx != newIdx {
			*patches = append(*patches, Patch{
				Path:  copyPath(parentPath),
				Op:    OpMoveChild,
				Index: newIdx,
				Value: map[string]interface{}{
					"key":    newChild.Key,
					"oldIdx": oldIdx,
					"newIdx": newIdx,
				},
			})
		}

		childPath := append(copyPath(parentPath), newIdx)
		diffNode(patches, childPath, a[oldIdx], newChild)
	}

	for oldIdx, oldChild := range a {
		if processed[oldIdx] {
			continue
		}
		if oldChild == nil || oldChild.Key == "" {
			if oldIdx >= len(b) || (b[oldIdx] != nil && b[oldIdx].Key != "") {
				*patches = append(*patches, Patch{
					Path:  copyPath(parentPath),
					Op:    OpDelChild,
					Index: oldIdx,
				})
			}
			continue
		}
		if _, stillExists := newKeys[oldChild.Key]; !stillExists {
			*patches = append(*patches, Patch{
				Path:  copyPath(parentPath),
				Op:    OpDelChild,
				Index: oldIdx,
				Value: map[string]interface{}{
					"key": oldChild.Key,
				},
			})
		}
	}
}

func hasKeys(children []*dom2.StructuredNode) bool {
	for _, child := range children {
		if child != nil && child.Key != "" {
			return true
		}
	}
	return false
}

func diffAttrs(patches *[]Patch, path []int, a, b *dom2.StructuredNode) {
	set := make(map[string][]string)
	for k, v := range b.Attrs {
		if !sliceEqual(a.Attrs[k], v) {
			set[k] = v
		}
	}
	for k := range a.Attrs {
		if _, ok := b.Attrs[k]; !ok {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpDelAttr, Name: k})
		}
	}
	if len(set) > 0 {
		*patches = append(*patches, Patch{Path: copyPath(path), Op: OpSetAttr, Value: set})
	}
}

func diffStyle(patches *[]Patch, path []int, a, b *dom2.StructuredNode) {
	if a.Style == nil && b.Style == nil {
		return
	}
	if a.Style == nil {
		a.Style = make(map[string]string)
	}
	if b.Style == nil {
		b.Style = make(map[string]string)
	}

	set := make(map[string]string)
	for k, v := range b.Style {
		if a.Style[k] != v {
			set[k] = v
		}
	}
	for k := range a.Style {
		if _, ok := b.Style[k]; !ok {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpDelStyle, Name: k})
		}
	}
	if len(set) > 0 {
		*patches = append(*patches, Patch{Path: copyPath(path), Op: OpSetStyle, Value: set})
	}
}

func diffStyles(patches *[]Patch, path []int, a, b *dom2.StructuredNode) {
	if a.Styles == nil && b.Styles == nil {
		return
	}
	if a.Styles == nil {
		a.Styles = make(map[string]map[string]string)
	}
	if b.Styles == nil {
		b.Styles = make(map[string]map[string]string)
	}

	allSelectors := make(map[string]struct{})
	for sel := range a.Styles {
		allSelectors[sel] = struct{}{}
	}
	for sel := range b.Styles {
		allSelectors[sel] = struct{}{}
	}

	for sel := range allSelectors {
		oldProps := a.Styles[sel]
		newProps := b.Styles[sel]

		if len(oldProps) == 0 && len(newProps) == 0 {
			continue
		}
		if len(oldProps) == 0 {
			for prop, val := range newProps {
				*patches = append(*patches, Patch{
					Path:     copyPath(path),
					Op:       OpSetStyleDecl,
					Selector: sel,
					Name:     prop,
					Value:    val,
				})
			}
			continue
		}
		if len(newProps) == 0 {
			for prop := range oldProps {
				*patches = append(*patches, Patch{
					Path:     copyPath(path),
					Op:       OpDelStyleDecl,
					Selector: sel,
					Name:     prop,
				})
			}
			continue
		}

		for prop, val := range newProps {
			if oldProps[prop] != val {
				*patches = append(*patches, Patch{
					Path:     copyPath(path),
					Op:       OpSetStyleDecl,
					Selector: sel,
					Name:     prop,
					Value:    val,
				})
			}
		}
		for prop := range oldProps {
			if _, ok := newProps[prop]; !ok {
				*patches = append(*patches, Patch{
					Path:     copyPath(path),
					Op:       OpDelStyleDecl,
					Selector: sel,
					Name:     prop,
				})
			}
		}
	}
}

type nodeType int

const (
	nodeUnknown nodeType = iota
	nodeText
	nodeComment
	nodeElement
	nodeComponent
	nodeFragment
)

func nodeTypeOf(n *dom2.StructuredNode) nodeType {
	switch {
	case n == nil:
		return nodeUnknown
	case n.ComponentID != "":
		return nodeComponent
	case n.Tag != "":
		return nodeElement
	case n.Text != "":
		return nodeText
	case n.Comment != "":
		return nodeComment
	case n.Fragment:
		return nodeFragment
	default:
		return nodeUnknown
	}
}

func copyPath(path []int) []int {
	if path == nil {
		return nil
	}
	cp := make([]int, len(path))
	copy(cp, path)
	return cp
}

func sliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func handlersEqual(a, b []dom2.HandlerMeta) bool {
	return reflect.DeepEqual(a, b)
}

func routerEqual(a, b *dom2.RouterMeta) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

func uploadEqual(a, b *dom2.UploadMeta) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}
