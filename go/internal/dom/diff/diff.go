package diff

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

func intPtr(i int) *int {
	return &i
}

// Diff compares two StructuredNode trees and returns a list of patches.
// Both trees are flattened first, removing fragment/component boundaries,
// so patches operate directly on DOM-equivalent nodes.
// Each patch has a Seq number indicating execution order.
func Diff(prev, next *dom.StructuredNode) []Patch {
	var flatPrev, flatNext *dom.StructuredNode
	if prev != nil {
		flatPrev = prev.Flatten()
	}
	if next != nil {
		flatNext = next.Flatten()
	}

	seq := 0
	patches := make([]Patch, 0)
	diffNode(&patches, &seq, nil, flatPrev, flatNext)
	return patches
}

// DiffRaw diffs without flattening - for testing or when trees are already flat
func DiffRaw(prev, next *dom.StructuredNode) []Patch {
	seq := 0
	patches := make([]Patch, 0)
	diffNode(&patches, &seq, nil, prev, next)
	return patches
}

func emit(patches *[]Patch, seq *int, p Patch) {
	p.Seq = *seq
	*seq++
	*patches = append(*patches, p)
}

func diffNode(patches *[]Patch, seq *int, path []int, a, b *dom.StructuredNode) {
	if a == nil && b == nil {
		return
	}
	if a == nil || b == nil {
		emit(patches, seq, Patch{Path: copyPath(path), Op: OpReplaceNode, Value: b})
		return
	}

	aType := nodeTypeOf(a)
	bType := nodeTypeOf(b)
	if aType != bType {
		emit(patches, seq, Patch{Path: copyPath(path), Op: OpReplaceNode, Value: b})
		return
	}

	if aType == nodeElement && a.Tag != b.Tag {
		emit(patches, seq, Patch{Path: copyPath(path), Op: OpReplaceNode, Value: b})
		return
	}

	switch aType {
	case nodeText:
		if a.Text != b.Text {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetText, Value: b.Text})
		}
	case nodeComment:
		if a.Comment != b.Comment {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetComment, Value: b.Comment})
		}
	case nodeElement:
		diffElement(patches, seq, path, a, b)
	default:
		panic("unhandled default case")
	}
}

func diffElement(patches *[]Patch, seq *int, path []int, a, b *dom.StructuredNode) {
	diffAttrs(patches, seq, path, a, b)
	diffStyle(patches, seq, path, a, b)

	if a.Tag == "style" || b.Tag == "style" {
		diffStylesheet(patches, seq, path, a, b)
	}

	if a.RefID != b.RefID {
		if b.RefID == "" {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpDelRef})
		} else {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetRef, Value: b.RefID})
		}
	}

	if !handlersEqual(a.Handlers, b.Handlers) {
		emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetHandlers, Value: b.Handlers})
	}

	if !routerEqual(a.Router, b.Router) {
		if b.Router == nil {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpDelRouter})
		} else {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetRouter, Value: b.Router})
		}
	}

	if !uploadEqual(a.Upload, b.Upload) {
		if b.Upload == nil {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpDelUpload})
		} else {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetUpload, Value: b.Upload})
		}
	}

	if !scriptEqual(a.Script, b.Script) {
		if b.Script == nil {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpDelScript})
		} else {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetScript, Value: b.Script})
		}
	}

	if a.UnsafeHTML != b.UnsafeHTML {
		emit(patches, seq, Patch{Path: copyPath(path), Op: OpReplaceNode, Value: b})
		return
	}

	if b.UnsafeHTML == "" {
		diffChildren(patches, seq, path, a.Children, b.Children)
	}
}

func diffChildren(patches *[]Patch, seq *int, parentPath []int, a, b []*dom.StructuredNode) {
	if hasKeys(a) || hasKeys(b) {
		diffChildrenKeyed(patches, seq, parentPath, a, b)
	} else {
		diffChildrenIndexed(patches, seq, parentPath, a, b)
	}
}

func diffChildrenIndexed(patches *[]Patch, seq *int, parentPath []int, a, b []*dom.StructuredNode) {
	m := len(a)
	if len(b) > m {
		m = len(b)
	}
	deletionOffset := 0
	for i := 0; i < m; i++ {
		childPath := append(copyPath(parentPath), i)
		var childA, childB *dom.StructuredNode
		if i < len(a) {
			childA = a[i]
		}
		if i < len(b) {
			childB = b[i]
		}
		if childA == nil && childB != nil {
			emit(patches, seq, Patch{
				Path:  copyPath(parentPath),
				Op:    OpAddChild,
				Index: intPtr(i),
				Value: childB,
			})
			continue
		}
		if childA != nil && childB == nil {
			emit(patches, seq, Patch{
				Path:  copyPath(parentPath),
				Op:    OpDelChild,
				Index: intPtr(i - deletionOffset),
			})
			deletionOffset++
			continue
		}
		diffNode(patches, seq, childPath, childA, childB)
	}
}

func diffChildrenKeyed(patches *[]Patch, seq *int, parentPath []int, a, b []*dom.StructuredNode) {
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

	retained := make(map[int]bool)
	for _, child := range b {
		if child != nil && child.Key != "" {
			if oldIdx, ok := oldKeys[child.Key]; ok {
				retained[oldIdx] = true
			}
		}
	}

	var toDelete []int
	for oldIdx, oldChild := range a {
		if retained[oldIdx] {
			continue
		}

		if oldChild == nil || oldChild.Key == "" {
			if oldIdx >= len(b) || (b[oldIdx] != nil && b[oldIdx].Key != "") {
				toDelete = append(toDelete, oldIdx)
			}
			continue
		}

		if _, stillExists := newKeys[oldChild.Key]; !stillExists {
			toDelete = append(toDelete, oldIdx)
		}
	}

	for i := len(toDelete) - 1; i >= 0; i-- {
		idx := toDelete[i]
		var value interface{}
		if a[idx] != nil && a[idx].Key != "" {
			value = map[string]interface{}{"key": a[idx].Key}
		}
		emit(patches, seq, Patch{
			Path:  copyPath(parentPath),
			Op:    OpDelChild,
			Index: intPtr(idx),
			Value: value,
		})
	}

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

	intermediateKeys := make(map[string]int)
	for i, child := range intermediate {
		if child != nil && child.Key != "" {
			intermediateKeys[child.Key] = i
		}
	}

	for newIdx, newChild := range b {
		if newChild == nil {
			continue
		}

		if newChild.Key == "" {
			if newIdx < len(intermediate) && (intermediate[newIdx] == nil || intermediate[newIdx].Key == "") {
				childPath := append(copyPath(parentPath), newIdx)
				diffNode(patches, seq, childPath, intermediate[newIdx], newChild)
			} else {
				emit(patches, seq, Patch{
					Path:  copyPath(parentPath),
					Op:    OpAddChild,
					Index: intPtr(newIdx),
					Value: newChild,
				})
			}
			continue
		}

		oldIdx, existedBefore := oldKeys[newChild.Key]
		if !existedBefore {
			emit(patches, seq, Patch{
				Path:  copyPath(parentPath),
				Op:    OpAddChild,
				Index: intPtr(newIdx),
				Value: newChild,
			})
			continue
		}

		intermediateIdx := intermediateKeys[newChild.Key]
		if intermediateIdx != newIdx {
			emit(patches, seq, Patch{
				Path:  copyPath(parentPath),
				Op:    OpMoveChild,
				Index: intPtr(newIdx),
				Value: map[string]interface{}{
					"key":    newChild.Key,
					"newIdx": newIdx,
				},
			})
		}

		childPath := append(copyPath(parentPath), newIdx)
		diffNode(patches, seq, childPath, a[oldIdx], newChild)
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

func diffAttrs(patches *[]Patch, seq *int, path []int, a, b *dom.StructuredNode) {

	for k := range a.Attrs {
		if _, ok := b.Attrs[k]; !ok {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpDelAttr, Name: k})
		}
	}

	set := make(map[string][]string)
	for k, v := range b.Attrs {
		if !sliceEqual(a.Attrs[k], v) {
			set[k] = v
		}
	}
	if len(set) > 0 {
		emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetAttr, Value: set})
	}
}

func diffStyle(patches *[]Patch, seq *int, path []int, a, b *dom.StructuredNode) {
	if a.Style == nil && b.Style == nil {
		return
	}

	aStyle := a.Style
	bStyle := b.Style
	if aStyle == nil {
		aStyle = map[string]string{}
	}
	if bStyle == nil {
		bStyle = map[string]string{}
	}

	for k := range aStyle {
		if _, ok := bStyle[k]; !ok {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpDelStyle, Name: k})
		}
	}

	set := make(map[string]string)
	for k, v := range bStyle {
		if aStyle[k] != v {
			set[k] = v
		}
	}
	if len(set) > 0 {
		emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetStyle, Value: set})
	}
}

func diffStylesheet(patches *[]Patch, seq *int, path []int, a, b *dom.StructuredNode) {
	if a.Stylesheet == nil && b.Stylesheet == nil {
		return
	}

	aStyles := stylesheetToMap(a.Stylesheet)
	bStyles := stylesheetToMap(b.Stylesheet)

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
				emit(patches, seq, Patch{
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
				emit(patches, seq, Patch{
					Path:     copyPath(path),
					Op:       OpDelStyleDecl,
					Selector: sel,
					Name:     prop,
				})
			}
			continue
		}

		for prop := range oldProps {
			if _, ok := newProps[prop]; !ok {
				emit(patches, seq, Patch{
					Path:     copyPath(path),
					Op:       OpDelStyleDecl,
					Selector: sel,
					Name:     prop,
				})
			}
		}

		for prop, val := range newProps {
			if oldProps[prop] != val {
				emit(patches, seq, Patch{
					Path:     copyPath(path),
					Op:       OpSetStyleDecl,
					Selector: sel,
					Name:     prop,
					Value:    val,
				})
			}
		}
	}
}

// stylesheetToMap converts a Stylesheet to a selector->props map for diffing.
// Media block rules are prefixed with their query for unique identification.
func stylesheetToMap(ss *dom.Stylesheet) map[string]map[string]string {
	if ss == nil {
		return map[string]map[string]string{}
	}

	result := make(map[string]map[string]string)

	for _, rule := range ss.Rules {
		result[rule.Selector] = rule.Props
	}

	for _, media := range ss.MediaBlocks {
		for _, rule := range media.Rules {
			key := "@media " + media.Query + " " + rule.Selector
			result[key] = rule.Props
		}
	}

	return result
}

type nodeType int

const (
	nodeUnknown nodeType = iota
	nodeText
	nodeComment
	nodeElement
)

func nodeTypeOf(n *dom.StructuredNode) nodeType {
	switch {
	case n == nil:
		return nodeUnknown
	case n.Tag != "":
		return nodeElement
	case n.Text != "":
		return nodeText
	case n.Comment != "":
		return nodeComment
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

func scriptEqual(a, b *dom.ScriptMeta) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.ScriptID == b.ScriptID &&
		a.Script == b.Script
}

// ExtractMetadata recursively walks the tree and extracts metadata patches
// (setHandlers, setRef, setRouter, setUpload, setScript) for initial client setup.
// Returns patches in sequence order for applying to existing SSR'd DOM.
func ExtractMetadata(n *dom.StructuredNode) []Patch {
	if n == nil {
		return nil
	}

	flattened := n.Flatten()
	if flattened == nil {
		return nil
	}

	var patches []Patch
	seq := 0
	extractMetadataRecursive(flattened, &patches, &seq, nil)
	return patches
}

func extractMetadataRecursive(n *dom.StructuredNode, patches *[]Patch, seq *int, path []int) {
	if n == nil {
		return
	}

	if n.Fragment {
		for i, child := range n.Children {
			childPath := append(copyPath(path), i)
			extractMetadataRecursive(child, patches, seq, childPath)
		}
		return
	}

	if n.Tag != "" {
		handlers := n.Handlers
		if len(handlers) == 0 && len(n.Events) > 0 {
			handlers = make([]dom.HandlerMeta, 0, len(n.Events))
			for event, binding := range n.Events {
				meta := dom.HandlerMeta{
					Event:   event,
					Handler: binding.Key,
					Listen:  binding.Listen,
					Props:   binding.Props,
				}
				handlers = append(handlers, meta)
			}
		}

		if len(handlers) > 0 {
			*patches = append(*patches, Patch{
				Seq:   *seq,
				Path:  copyPath(path),
				Op:    OpSetHandlers,
				Value: handlers,
			})
			*seq++
		}

		if n.RefID != "" {
			*patches = append(*patches, Patch{
				Seq:   *seq,
				Path:  copyPath(path),
				Op:    OpSetRef,
				Value: n.RefID,
			})
			*seq++
		}

		if n.Router != nil {
			*patches = append(*patches, Patch{
				Seq:   *seq,
				Path:  copyPath(path),
				Op:    OpSetRouter,
				Value: n.Router,
			})
			*seq++
		}

		if n.Upload != nil {
			*patches = append(*patches, Patch{
				Seq:   *seq,
				Path:  copyPath(path),
				Op:    OpSetUpload,
				Value: n.Upload,
			})
			*seq++
		}

		if n.Script != nil {
			*patches = append(*patches, Patch{
				Seq:   *seq,
				Path:  copyPath(path),
				Op:    OpSetScript,
				Value: n.Script,
			})
			*seq++
		}

		if n.UnsafeHTML == "" {
			for i, child := range n.Children {
				childPath := append(copyPath(path), i)
				extractMetadataRecursive(child, patches, seq, childPath)
			}
		}
	}
}
