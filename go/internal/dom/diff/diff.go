package diff

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

func intPtr(i int) *int {
	return &i
}

func Diff(prev, next *dom.StructuredNode) []Patch {
	patches := make([]Patch, 0)
	diffNode(&patches, nil, prev, next)
	return patches
}

func diffNode(patches *[]Patch, path []int, a, b *dom.StructuredNode) {
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

func diffElement(patches *[]Patch, path []int, a, b *dom.StructuredNode) {
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

func diffChildren(patches *[]Patch, parentPath []int, a, b []*dom.StructuredNode) {
	if hasKeys(a) || hasKeys(b) {
		diffChildrenKeyed(patches, parentPath, a, b)
	} else {
		diffChildrenIndexed(patches, parentPath, a, b)
	}
}

func diffChildrenIndexed(patches *[]Patch, parentPath []int, a, b []*dom.StructuredNode) {
	max := len(a)
	if len(b) > max {
		max = len(b)
	}
	deletionOffset := 0
	for i := 0; i < max; i++ {
		childPath := append(copyPath(parentPath), i)
		var childA, childB *dom.StructuredNode
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
				Index: intPtr(i),
				Value: childB,
			})
			continue
		}
		if childA != nil && childB == nil {
			*patches = append(*patches, Patch{
				Path:  copyPath(parentPath),
				Op:    OpDelChild,
				Index: intPtr(i - deletionOffset),
			})
			deletionOffset++
			continue
		}
		diffNode(patches, childPath, childA, childB)
	}
}

func diffChildrenKeyed(patches *[]Patch, parentPath []int, a, b []*dom.StructuredNode) {
	// Build key→index maps for old and new children
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

	// Track which old indices are retained (exist in new list)
	retained := make(map[int]bool)
	for _, child := range b {
		if child != nil && child.Key != "" {
			if oldIdx, ok := oldKeys[child.Key]; ok {
				retained[oldIdx] = true
			}
		}
	}

	// ============================================================
	// PHASE 1: DELETIONS (emit in reverse order for stable indexes)
	// ============================================================
	// Collect indices to delete, then emit in reverse order so each
	// deletion index is valid at the time of application.
	var toDelete []int
	for oldIdx, oldChild := range a {
		if retained[oldIdx] {
			continue
		}
		// Unkeyed node at this position that's being replaced
		if oldChild == nil || oldChild.Key == "" {
			if oldIdx >= len(b) || (b[oldIdx] != nil && b[oldIdx].Key != "") {
				toDelete = append(toDelete, oldIdx)
			}
			continue
		}
		// Keyed node that no longer exists
		if _, stillExists := newKeys[oldChild.Key]; !stillExists {
			toDelete = append(toDelete, oldIdx)
		}
	}

	// Emit deletions in reverse index order (highest first)
	for i := len(toDelete) - 1; i >= 0; i-- {
		idx := toDelete[i]
		var value interface{}
		if a[idx] != nil && a[idx].Key != "" {
			value = map[string]interface{}{"key": a[idx].Key}
		}
		*patches = append(*patches, Patch{
			Path:  copyPath(parentPath),
			Op:    OpDelChild,
			Index: intPtr(idx),
			Value: value,
		})
	}

	// ============================================================
	// PHASE 2: Build intermediate state after deletions
	// ============================================================
	// This represents the child array state after deletions are applied.
	// We need this to calculate correct move source indices.
	intermediate := make([]*dom.StructuredNode, 0, len(a)-len(toDelete))
	deleteSet := make(map[int]bool)
	for _, idx := range toDelete {
		deleteSet[idx] = true
	}
	for i, child := range a {
		if !deleteSet[i] {
			intermediate = append(intermediate, child)
		}
	}

	// Build key→index map for intermediate state
	intermediateKeys := make(map[string]int)
	for i, child := range intermediate {
		if child != nil && child.Key != "" {
			intermediateKeys[child.Key] = i
		}
	}

	// ============================================================
	// PHASE 3: MOVES AND ADDITIONS
	// ============================================================
	// Process new children in order. Use key-based addressing for moves
	// so the client can resolve current position at application time.
	for newIdx, newChild := range b {
		if newChild == nil {
			continue
		}

		if newChild.Key == "" {
			// Unkeyed node: if same position exists in intermediate with no key, diff it
			if newIdx < len(intermediate) && (intermediate[newIdx] == nil || intermediate[newIdx].Key == "") {
				childPath := append(copyPath(parentPath), newIdx)
				diffNode(patches, childPath, intermediate[newIdx], newChild)
			} else {
				// Otherwise add new unkeyed node
				*patches = append(*patches, Patch{
					Path:  copyPath(parentPath),
					Op:    OpAddChild,
					Index: intPtr(newIdx),
					Value: newChild,
				})
			}
			continue
		}

		// Keyed node
		oldIdx, existedBefore := oldKeys[newChild.Key]
		if !existedBefore {
			// New keyed node - add it
			*patches = append(*patches, Patch{
				Path:  copyPath(parentPath),
				Op:    OpAddChild,
				Index: intPtr(newIdx),
				Value: newChild,
			})
			continue
		}

		// Existing keyed node - check if it needs to move
		intermediateIdx := intermediateKeys[newChild.Key]
		if intermediateIdx != newIdx {
			// Emit move with key for client-side resolution
			*patches = append(*patches, Patch{
				Path:  copyPath(parentPath),
				Op:    OpMoveChild,
				Index: intPtr(newIdx),
				Value: map[string]interface{}{
					"key":    newChild.Key,
					"newIdx": newIdx,
				},
			})
		}

		// Diff the content
		childPath := append(copyPath(parentPath), newIdx)
		diffNode(patches, childPath, a[oldIdx], newChild)
	}
}

func hasKeys(children []*dom.StructuredNode) bool {
	for _, child := range children {
		if child != nil && child.Key != "" {
			return true
		}
	}
	return false
}

func diffAttrs(patches *[]Patch, path []int, a, b *dom.StructuredNode) {
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

func diffStyle(patches *[]Patch, path []int, a, b *dom.StructuredNode) {
	if a.Style == nil && b.Style == nil {
		return
	}

	// Use local references to avoid mutating the original nodes
	aStyle := a.Style
	bStyle := b.Style
	if aStyle == nil {
		aStyle = map[string]string{}
	}
	if bStyle == nil {
		bStyle = map[string]string{}
	}

	set := make(map[string]string)
	for k, v := range bStyle {
		if aStyle[k] != v {
			set[k] = v
		}
	}
	for k := range aStyle {
		if _, ok := bStyle[k]; !ok {
			*patches = append(*patches, Patch{Path: copyPath(path), Op: OpDelStyle, Name: k})
		}
	}
	if len(set) > 0 {
		*patches = append(*patches, Patch{Path: copyPath(path), Op: OpSetStyle, Value: set})
	}
}

func diffStyles(patches *[]Patch, path []int, a, b *dom.StructuredNode) {
	if a.Styles == nil && b.Styles == nil {
		return
	}

	// Use local references to avoid mutating the original nodes
	aStyles := a.Styles
	bStyles := b.Styles
	if aStyles == nil {
		aStyles = map[string]map[string]string{}
	}
	if bStyles == nil {
		bStyles = map[string]map[string]string{}
	}

	allSelectors := make(map[string]struct{})
	for sel := range aStyles {
		allSelectors[sel] = struct{}{}
	}
	for sel := range bStyles {
		allSelectors[sel] = struct{}{}
	}

	for sel := range allSelectors {
		oldProps := aStyles[sel]
		newProps := bStyles[sel]

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

func nodeTypeOf(n *dom.StructuredNode) nodeType {
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

func handlersEqual(a, b []dom.HandlerMeta) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Event != b[i].Event ||
			a[i].Handler != b[i].Handler ||
			!sliceEqual(a[i].Listen, b[i].Listen) ||
			!sliceEqual(a[i].Props, b[i].Props) {
			return false
		}
	}
	return true
}

func routerEqual(a, b *dom.RouterMeta) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.PathValue == b.PathValue &&
		a.Query == b.Query &&
		a.Hash == b.Hash &&
		a.Replace == b.Replace
}

func uploadEqual(a, b *dom.UploadMeta) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.UploadID == b.UploadID &&
		a.Multiple == b.Multiple &&
		a.MaxSize == b.MaxSize &&
		sliceEqual(a.Accept, b.Accept)
}
