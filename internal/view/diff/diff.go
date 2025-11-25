package diff

import (
	"sort"

	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/view"
)

// OnDuplicateKey is called when duplicate keys are detected during diffing.
// Set this to a custom function to handle warnings (e.g., log in development).
// By default, duplicates are silently ignored.
// Parameters: tree ("old"/"new"), key string, path []int
var OnDuplicateKey func(tree, key string, path []int)

// PanicOnDuplicateKey controls whether to panic on duplicate keys.
// When true, duplicate keys cause a panic with details.
// When false (default), OnDuplicateKey is called if set.
var PanicOnDuplicateKey = false

func intPtr(i int) *int {
	return &i
}

// Diff compares two view node trees and returns a list of patches.
// Both trees are flattened first, removing fragment boundaries,
// so patches operate directly on DOM-equivalent nodes.
// Each patch has a Seq number indicating execution order.
func Diff(prev, next view.Node) []Patch {
	var flatPrev, flatNext view.Node
	if prev != nil {
		flatPrev = Flatten(prev)
	}
	if next != nil {
		flatNext = Flatten(next)
	}

	seq := 0
	patches := make([]Patch, 0)
	diffNode(&patches, &seq, nil, flatPrev, flatNext)
	return patches
}

// DiffRaw diffs without flattening. INTERNAL USE ONLY.
//
// This function is intended for testing and internal use cases where trees
// are already flattened. Production code should always use Diff() which
// flattens both trees before comparison to ensure consistent behavior.
//
// WARNING: Using DiffRaw on trees containing Fragment nodes will produce
// different results than Diff, as fragment boundaries won't be dissolved.
func DiffRaw(prev, next view.Node) []Patch {
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

// nodeType identifies the kind of view node
type nodeType int

const (
	nodeUnknown nodeType = iota
	nodeText
	nodeComment
	nodeElement
	nodeFragment
)

func nodeTypeOf(n view.Node) nodeType {
	if n == nil {
		return nodeUnknown
	}
	switch n.(type) {
	case *view.Element:
		return nodeElement
	case *view.Text:
		return nodeText
	case *view.Comment:
		return nodeComment
	case *view.Fragment:
		return nodeFragment
	default:
		return nodeUnknown
	}
}

func diffNode(patches *[]Patch, seq *int, path []int, a, b view.Node) {
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

	switch aType {
	case nodeElement:
		aElem := a.(*view.Element)
		bElem := b.(*view.Element)
		if aElem.Tag != bElem.Tag {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpReplaceNode, Value: b})
			return
		}
		diffElement(patches, seq, path, aElem, bElem)
	case nodeText:
		aText := a.(*view.Text)
		bText := b.(*view.Text)
		if aText.Text != bText.Text {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetText, Value: bText.Text})
		}
	case nodeComment:
		aComment := a.(*view.Comment)
		bComment := b.(*view.Comment)
		if aComment.Comment != bComment.Comment {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetComment, Value: bComment.Comment})
		}
	case nodeFragment:
		aFrag := a.(*view.Fragment)
		bFrag := b.(*view.Fragment)
		diffChildren(patches, seq, path, aFrag.Children, bFrag.Children)
	}
}

// diffElement compares two elements and emits patches for their differences.
//
// IMPORTANT: UnsafeHTML handling
// When UnsafeHTML changes, the entire node is replaced (OpReplaceNode) rather
// than patching individual attrs/styles. This is intentional because:
// 1. UnsafeHTML content may conflict with programmatic children
// 2. The client needs to completely re-render the inner HTML
// 3. Any attr/style changes alongside UnsafeHTML changes are included in the replacement
//
// Consequence: If you change both attrs AND UnsafeHTML in the same render,
// only the OpReplaceNode patch is emitted (attrs are part of the new node).
func diffElement(patches *[]Patch, seq *int, path []int, a, b *view.Element) {

	if a.UnsafeHTML != b.UnsafeHTML {
		emit(patches, seq, Patch{Path: copyPath(path), Op: OpReplaceNode, Value: b})
		return
	}

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

	if !scriptEqual(a.Script, b.Script) {
		if b.Script == nil {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpDelScript})
		} else {
			emit(patches, seq, Patch{Path: copyPath(path), Op: OpSetScript, Value: b.Script})
		}
	}

	if b.UnsafeHTML == "" {
		diffChildren(patches, seq, path, a.Children, b.Children)
	}
}

func diffChildren(patches *[]Patch, seq *int, parentPath []int, a, b []view.Node) {
	if hasKeys(a) || hasKeys(b) {
		diffChildrenKeyed(patches, seq, parentPath, a, b)
	} else {
		diffChildrenIndexed(patches, seq, parentPath, a, b)
	}
}

func diffChildrenIndexed(patches *[]Patch, seq *int, parentPath []int, a, b []view.Node) {
	m := len(a)
	if len(b) > m {
		m = len(b)
	}
	deletionOffset := 0
	for i := 0; i < m; i++ {
		childPath := append(copyPath(parentPath), i)
		var childA, childB view.Node
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

// diffChildrenKeyed handles keyed child reconciliation.
//
// Key uniqueness: Keys MUST be unique within siblings. Duplicate keys cause
// undefined behavior - the last occurrence wins in the key map, potentially
// causing incorrect moves/deletes. Set PanicOnDuplicateKey=true or provide
// an OnDuplicateKey callback to detect duplicates during development.
//
// Mixed keyed/unkeyed: When some children have keys and others don't,
// unkeyed children are matched by position among other unkeyed siblings.
// This can lead to unexpected results - prefer all-keyed or all-unkeyed.
func diffChildrenKeyed(patches *[]Patch, seq *int, parentPath []int, a, b []view.Node) {
	oldKeys := make(map[string]int)
	newKeys := make(map[string]int)

	for i, child := range a {
		if key := getKey(child); key != "" {
			if _, exists := oldKeys[key]; exists {
				handleDuplicateKey("old", key, parentPath)
			}
			oldKeys[key] = i
		}
	}
	for i, child := range b {
		if key := getKey(child); key != "" {
			if _, exists := newKeys[key]; exists {
				handleDuplicateKey("new", key, parentPath)
			}
			newKeys[key] = i
		}
	}

	retained := make(map[int]bool)
	for _, child := range b {
		if key := getKey(child); key != "" {
			if oldIdx, ok := oldKeys[key]; ok {
				retained[oldIdx] = true
			}
		}
	}

	var toDelete []int
	for oldIdx, oldChild := range a {
		if retained[oldIdx] {
			continue
		}

		key := getKey(oldChild)
		if key == "" {
			if oldIdx >= len(b) || getKey(b[oldIdx]) != "" {
				toDelete = append(toDelete, oldIdx)
			}
			continue
		}

		if _, stillExists := newKeys[key]; !stillExists {
			toDelete = append(toDelete, oldIdx)
		}
	}

	for i := len(toDelete) - 1; i >= 0; i-- {
		idx := toDelete[i]
		var value interface{}
		if key := getKey(a[idx]); key != "" {
			value = map[string]interface{}{"key": key}
		}
		emit(patches, seq, Patch{
			Path:  copyPath(parentPath),
			Op:    OpDelChild,
			Index: intPtr(idx),
			Value: value,
		})
	}

	intermediate := make([]view.Node, 0, len(a)-len(toDelete))
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
		if key := getKey(child); key != "" {
			intermediateKeys[key] = i
		}
	}

	for newIdx, newChild := range b {
		if newChild == nil {
			continue
		}

		key := getKey(newChild)
		if key == "" {
			if newIdx < len(intermediate) && getKey(intermediate[newIdx]) == "" {
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

		oldIdx, existedBefore := oldKeys[key]
		if !existedBefore {
			emit(patches, seq, Patch{
				Path:  copyPath(parentPath),
				Op:    OpAddChild,
				Index: intPtr(newIdx),
				Value: newChild,
			})
			continue
		}

		intermediateIdx := intermediateKeys[key]
		if intermediateIdx != newIdx {
			emit(patches, seq, Patch{
				Path:  copyPath(parentPath),
				Op:    OpMoveChild,
				Index: intPtr(newIdx),
				Value: map[string]interface{}{
					"key":       key,
					"newIdx":    newIdx,
					"fromIndex": intermediateIdx,
				},
			})
		}

		childPath := append(copyPath(parentPath), newIdx)
		diffNode(patches, seq, childPath, a[oldIdx], newChild)
	}
}

func hasKeys(children []view.Node) bool {
	for _, child := range children {
		if getKey(child) != "" {
			return true
		}
	}
	return false
}

func getKey(n view.Node) string {
	if elem, ok := n.(*view.Element); ok {
		return elem.Key
	}
	return ""
}

// handleDuplicateKey reports duplicate key issues.
// If PanicOnDuplicateKey is true, panics with details.
// Otherwise, calls OnDuplicateKey callback if set.
func handleDuplicateKey(tree, key string, path []int) {
	if PanicOnDuplicateKey {
		panic("view/diff: duplicate key '" + key + "' in " + tree + " tree at path " + formatPath(path))
	}
	if OnDuplicateKey != nil {
		OnDuplicateKey(tree, key, path)
	}
}

func formatPath(path []int) string {
	if len(path) == 0 {
		return "[]"
	}
	result := "["
	for i, p := range path {
		if i > 0 {
			result += ","
		}
		result += intToStr(p)
	}
	return result + "]"
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	if n < 0 {
		return "-" + intToStr(-n)
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}

func diffAttrs(patches *[]Patch, seq *int, path []int, a, b *view.Element) {
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

func diffStyle(patches *[]Patch, seq *int, path []int, a, b *view.Element) {
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

func diffStylesheet(patches *[]Patch, seq *int, path []int, a, b *view.Element) {
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
func stylesheetToMap(ss *metadata.Stylesheet) map[string]map[string]string {
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

// handlersEqual compares two handler slices for equality.
// Comparison is order-independent - handlers are sorted by Event+Handler
// before comparison to avoid spurious patches when handlers are reordered.
func handlersEqual(a, b []metadata.HandlerMeta) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}

	aSorted := make([]metadata.HandlerMeta, len(a))
	bSorted := make([]metadata.HandlerMeta, len(b))
	copy(aSorted, a)
	copy(bSorted, b)

	sortHandlers(aSorted)
	sortHandlers(bSorted)

	for i := range aSorted {
		if !handlerMetaEqual(aSorted[i], bSorted[i]) {
			return false
		}
	}
	return true
}

func sortHandlers(handlers []metadata.HandlerMeta) {
	sort.Slice(handlers, func(i, j int) bool {
		if handlers[i].Event != handlers[j].Event {
			return handlers[i].Event < handlers[j].Event
		}
		return handlers[i].Handler < handlers[j].Handler
	})
}

func handlerMetaEqual(a, b metadata.HandlerMeta) bool {
	return a.Event == b.Event &&
		a.Handler == b.Handler &&
		a.Prevent == b.Prevent &&
		a.Stop == b.Stop &&
		a.Passive == b.Passive &&
		a.Once == b.Once &&
		a.Capture == b.Capture &&
		a.Debounce == b.Debounce &&
		a.Throttle == b.Throttle &&
		sliceEqual(a.Listen, b.Listen) &&
		sliceEqual(a.Props, b.Props)
}

func scriptEqual(a, b *metadata.ScriptMeta) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.ScriptID == b.ScriptID &&
		a.Script == b.Script
}

// Flatten removes Fragment wrappers, promoting children to parent level.
// Returns a single Element, Text, Comment, or nil (fragments dissolved).
func Flatten(n view.Node) view.Node {
	if n == nil {
		return nil
	}

	switch node := n.(type) {
	case *view.Fragment:

		flattened := make([]view.Node, 0, len(node.Children))
		for _, child := range node.Children {
			if flat := Flatten(child); flat != nil {
				if frag, ok := flat.(*view.Fragment); ok {

					flattened = append(flattened, frag.Children...)
				} else {
					flattened = append(flattened, flat)
				}
			}
		}
		if len(flattened) == 0 {
			return nil
		}
		if len(flattened) == 1 {
			return flattened[0]
		}
		return &view.Fragment{Fragment: true, Children: flattened}

	case *view.Element:

		if len(node.Children) == 0 {
			return node
		}
		flattened := make([]view.Node, 0, len(node.Children))
		for _, child := range node.Children {
			if flat := Flatten(child); flat != nil {
				if frag, ok := flat.(*view.Fragment); ok {

					flattened = append(flattened, frag.Children...)
				} else {
					flattened = append(flattened, flat)
				}
			}
		}

		return &view.Element{
			Tag:        node.Tag,
			Attrs:      node.Attrs,
			Style:      node.Style,
			Children:   flattened,
			UnsafeHTML: node.UnsafeHTML,
			Key:        node.Key,
			RefID:      node.RefID,
			Handlers:   node.Handlers,
			Script:     node.Script,
			Stylesheet: node.Stylesheet,
		}

	default:

		return n
	}
}

// ExtractMetadata recursively walks the tree and extracts metadata patches
// (setHandlers, setRef, setScript) for initial client setup.
// Returns patches in sequence order for applying to existing SSR'd DOM.
func ExtractMetadata(n view.Node) []Patch {
	if n == nil {
		return nil
	}

	flattened := Flatten(n)
	if flattened == nil {
		return nil
	}

	var patches []Patch
	seq := 0
	extractMetadataRecursive(flattened, &patches, &seq, nil)
	return patches
}

func extractMetadataRecursive(n view.Node, patches *[]Patch, seq *int, path []int) {
	if n == nil {
		return
	}

	switch node := n.(type) {
	case *view.Fragment:
		for i, child := range node.Children {
			childPath := append(copyPath(path), i)
			extractMetadataRecursive(child, patches, seq, childPath)
		}

	case *view.Element:
		if len(node.Handlers) > 0 {
			*patches = append(*patches, Patch{
				Seq:   *seq,
				Path:  copyPath(path),
				Op:    OpSetHandlers,
				Value: node.Handlers,
			})
			*seq++
		}

		if node.RefID != "" {
			*patches = append(*patches, Patch{
				Seq:   *seq,
				Path:  copyPath(path),
				Op:    OpSetRef,
				Value: node.RefID,
			})
			*seq++
		}

		if node.Script != nil {
			*patches = append(*patches, Patch{
				Seq:   *seq,
				Path:  copyPath(path),
				Op:    OpSetScript,
				Value: node.Script,
			})
			*seq++
		}

		if node.UnsafeHTML == "" {
			for i, child := range node.Children {
				childPath := append(copyPath(path), i)
				extractMetadataRecursive(child, patches, seq, childPath)
			}
		}
	}
}
