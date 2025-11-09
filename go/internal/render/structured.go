package render

import (
	"html"
	"sort"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/handlers"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// DynKind enumerates dynamic slot kinds.
type DynKind uint8

const (
	DynText DynKind = iota
	DynAttrs
	DynList
)

// Dyn represents a dynamic slot value.
type Dyn struct {
	Kind DynKind

	Text  string
	Attrs map[string]string
	List  []Row
}

// NodeAnchor describes how to locate a rendered node relative to the render root.
// ParentPath identifies the parent node by walking the render tree using
// renderable child indexes, while ChildIndex points at the target child within
// that parent. If HasIndex is false the anchor could not be determined.
type NodeAnchor struct {
	ParentPath []int
	ChildIndex int
	HasIndex   bool
}

// Row represents a keyed row inside a dynamic list slot.
type Row struct {
	Key      string
	HTML     string
	Slots    []int
	Bindings []HandlerBinding
	Markers  map[string]ComponentBoundary
	Anchors  map[int]NodeAnchor
}

// Structured represents the structured render output of a node tree.
type Structured struct {
	S          []string
	D          []Dyn
	Components map[string]ComponentSpan
	Bindings   []HandlerBinding
	Anchors    map[int]NodeAnchor
}

// PromotionTracker allows callers to control when static nodes should become dynamic.
type PromotionTracker interface {
	ResolveTextPromotion(componentID string, path []int, value string, mutable bool) bool
	ResolveAttrPromotion(componentID string, path []int, attrs map[string]string, mutable map[string]bool) bool
}

// StructuredOptions configures structured rendering.
type StructuredOptions struct {
	Handlers   handlers.Registry
	Promotions PromotionTracker
}

// ComponentSpan captures the statics and dynamic slot ranges associated with a component subtree.
type ComponentSpan struct {
	StaticsStart  int
	StaticsEnd    int
	DynamicsStart int
	DynamicsEnd   int
	Boundary      ComponentBoundary
}

// ComponentBoundary describes where a component's rendered DOM should be
// anchored within its parent. The ParentPath identifies the parent node by its
// child indices starting from the render root, while StartIndex and EndIndex
// indicate the range of child nodes that belong to the component (EndIndex is
// exclusive).
type ComponentBoundary struct {
	ParentPath []int
	StartIndex int
	EndIndex   int
}

// ToStructured lowers a node tree into structured statics and dynamics.
func ToStructured(n h.Node) Structured {
	return ToStructuredWithOptions(n, StructuredOptions{})
}

// ToStructuredWithHandlers lowers a node tree into structured statics and
// dynamics while registering handlers in the provided registry.
func ToStructuredWithHandlers(n h.Node, reg handlers.Registry) Structured {
	return ToStructuredWithOptions(n, StructuredOptions{Handlers: reg})
}

// ToStructuredWithOptions lowers a node tree into structured statics and dynamics with custom behaviour.
func ToStructuredWithOptions(n h.Node, opts StructuredOptions) Structured {
	if n == nil {
		return Structured{}
	}
	FinalizeWithHandlers(n, opts.Handlers)
	b := &structuredBuilder{tracker: opts.Promotions}
	b.ensureDomRoot()
	b.visit(n)
	b.flush()
	return Structured{
		S:          b.statics,
		D:          b.dynamics,
		Components: b.components,
		Bindings:   append([]HandlerBinding(nil), b.handlerBindings...),
		Anchors:    b.anchors(),
	}
}

type structuredBuilder struct {
	statics         []string
	current         strings.Builder
	dynamics        []Dyn
	stack           []elementFrame
	components      map[string]ComponentSpan
	handlerBindings []HandlerBinding

	tracker        PromotionTracker
	componentStack []componentFrame
	componentPath  []int
	componentOrder []string

	domStack     []domFrame
	elementPaths map[*h.Element][]int
	slotAnchors  map[int]NodeAnchor
}

type elementFrame struct {
	attrSlot    int
	element     *h.Element
	startStatic int
	void        bool
	staticAttrs bool
}

type componentFrame struct {
	id              string
	prevPath        []int
	parentPath      []int
	anchorIndex     int
	startChildCount int
}

type domFrame struct {
	path       []int
	childCount int
}

type domFrame struct {
	path       []int
	childCount int
}

// HandlerBinding captures the association between a dynamic slot anchor and a handler ID.
type HandlerBinding struct {
	Slot    int
	Event   string
	Handler string
	Listen  []string
	Props   []string
}

func (b *structuredBuilder) writeStatic(s string) {
	if s == "" {
		return
	}
	b.current.WriteString(s)
}

func (b *structuredBuilder) flush() {
	b.statics = append(b.statics, b.current.String())
	b.current.Reset()
}

func (b *structuredBuilder) ensureDomRoot() {
	if len(b.domStack) == 0 {
		b.domStack = append(b.domStack, domFrame{path: nil})
	}
}

func (b *structuredBuilder) currentDomFrame() *domFrame {
	b.ensureDomRoot()
	return &b.domStack[len(b.domStack)-1]
}

func (b *structuredBuilder) appendDomNode() ([]int, int) {
	frame := b.currentDomFrame()
	index := frame.childCount
	frame.childCount++
	path := append([]int(nil), frame.path...)
	path = append(path, index)
	return path, index
}

func (b *structuredBuilder) pushDomFrame(path []int) {
	b.domStack = append(b.domStack, domFrame{path: append([]int(nil), path...)})
}

func (b *structuredBuilder) popDomFrame() {
	if len(b.domStack) <= 1 {
		return
	}
	b.domStack = b.domStack[:len(b.domStack)-1]
}

func (b *structuredBuilder) addDyn(d Dyn) int {
	b.flush()
	b.dynamics = append(b.dynamics, d)
	return len(b.dynamics) - 1
}

func (b *structuredBuilder) visit(n h.Node) {
	switch v := n.(type) {
	case *h.TextNode:
		b.visitText(v)
	case *h.Element:
		b.visitElement(v)
	case *h.FragmentNode:
		b.visitFragment(v)
	case *h.CommentNode:
		b.writeStatic("<!--")
		b.writeStatic(escapeComment(v.Value))
		b.writeStatic("-->")
	case *h.ComponentNode:
		b.visitComponentNode(v)
	}
}

func (b *structuredBuilder) visitText(t *h.TextNode) {
	if t == nil {
		return
	}
	path, _ := b.appendDomNode()
	dynamic := t.Mutable
	if !dynamic {
		if tracker := b.tracker; tracker != nil {
			componentID := b.currentComponentID()
			path := b.currentComponentPath()
			dynamic = tracker.ResolveTextPromotion(componentID, path, t.Value, t.Mutable)
		}
	}
	if dynamic {
		idx := b.addDyn(Dyn{Kind: DynText, Text: t.Value})
		b.registerSlotAnchor(idx, path)
		return
	}
	b.writeStatic(html.EscapeString(t.Value))
}

func (b *structuredBuilder) visitComponentNode(v *h.ComponentNode) {
	if v == nil || v.ID == "" {
		if v != nil && v.Child != nil {
			b.visit(v.Child)
		}
		return
	}
	b.flush()
	staticsStart := len(b.statics)
	dynamicsStart := len(b.dynamics)
	parentFrame := b.currentDomFrame()
	parentPath := []int(nil)
	anchorIndex := 0
	startChildCount := 0
	if parentFrame != nil {
		parentPath = append([]int(nil), parentFrame.path...)
		anchorIndex = parentFrame.childCount
		startChildCount = parentFrame.childCount
	}
	b.pushComponent(v.ID, parentPath, anchorIndex, startChildCount)
	if v.Child != nil {
		b.visit(v.Child)
	}
	frame := b.popComponent()
	parentFrame = b.currentDomFrame()
	endChildCount := anchorIndex
	if parentFrame != nil {
		endChildCount = parentFrame.childCount
	}
	if endChildCount < frame.startChildCount {
		endChildCount = frame.startChildCount
	}
	b.flush()
	span := ComponentSpan{
		StaticsStart:  staticsStart,
		StaticsEnd:    len(b.statics),
		DynamicsStart: dynamicsStart,
		DynamicsEnd:   len(b.dynamics),
		Boundary: ComponentBoundary{
			ParentPath: append([]int(nil), frame.parentPath...),
			StartIndex: frame.anchorIndex,
			EndIndex:   frame.anchorIndex + (endChildCount - frame.startChildCount),
		},
	}
	if b.components == nil {
		b.components = make(map[string]ComponentSpan)
	}
	if _, exists := b.components[v.ID]; !exists {
		b.componentOrder = append(b.componentOrder, v.ID)
	}
	b.components[v.ID] = span
}

func (b *structuredBuilder) visitElement(v *h.Element) {
	if v == nil {
		return
	}
	path, _ := b.appendDomNode()
	if b.elementPaths == nil {
		b.elementPaths = make(map[*h.Element][]int)
	}
	b.elementPaths[v] = path
	b.pushDomFrame(path)
	defer b.popDomFrame()
	void := dom.IsVoidElement(v.Tag)
	dynamicAttrs := b.shouldUseDynamicAttrs(v)
	attrSlot := -1
	startStatic := -1
	if dynamicAttrs {
		b.writeStatic("<")
		b.writeStatic(v.Tag)
		attrs := copyAttrs(v.Attrs)
		if attrs == nil {
			attrs = map[string]string{}
		}
		attrSlot = b.addDyn(Dyn{Kind: DynAttrs, Attrs: attrs})
		b.registerSlotAnchor(attrSlot, path)
		startStatic = len(b.statics) - 1
	} else {
		start := renderStartTag(v, void)
		b.writeStatic(start)
		b.flush()
		startStatic = len(b.statics) - 1
	}
	b.pushFrame(v, attrSlot, startStatic, void, !dynamicAttrs)
	defer b.popFrame()
	if dynamicAttrs {
		if void {
			b.writeStatic("/>")
			return
		}
		b.writeStatic(">")
	} else if void {
		return
	}
	if v.Unsafe != nil {
		b.writeStatic(*v.Unsafe)
	} else if !b.tryKeyedChildren(v, v.Children, attrSlot) {
		for idx, child := range v.Children {
			if child == nil {
				continue
			}
			b.pushChildIndex(idx)
			b.visit(child)
			b.popChildIndex()
		}
	}
	b.writeStatic("</")
	b.writeStatic(v.Tag)
	b.writeStatic(">")
}

func (b *structuredBuilder) shouldUseDynamicAttrs(el *h.Element) bool {
	if el == nil {
		return false
	}
	if len(el.HandlerAssignments) > 0 {
		return true
	}
	mutable := el.MutableAttrs
	if tracker := b.tracker; tracker != nil {
		componentID := b.currentComponentID()
		path := b.currentComponentPath()
		return tracker.ResolveAttrPromotion(componentID, path, el.Attrs, mutable)
	}
	return shouldForceDynamicAttrs(mutable, el.Attrs)
}

func (b *structuredBuilder) visitFragment(f *h.FragmentNode) {
	if f == nil {
		return
	}
	if b.tryKeyedChildren(nil, f.Children, -1) {
		return
	}
	for idx, child := range f.Children {
		if child == nil {
			continue
		}
		b.pushChildIndex(idx)
		b.visit(child)
		b.popChildIndex()
	}
}

func (b *structuredBuilder) pushFrame(el *h.Element, attrSlot, startStatic int, void, staticAttrs bool) {
	frame := elementFrame{attrSlot: attrSlot, element: el, startStatic: startStatic, void: void, staticAttrs: staticAttrs}
	b.stack = append(b.stack, frame)
}

func (b *structuredBuilder) registerSlotAnchor(slot int, path []int) {
	if slot < 0 || len(path) == 0 {
		return
	}
	anchor := NodeAnchor{ChildIndex: path[len(path)-1], HasIndex: true}
	if len(path) > 1 {
		anchor.ParentPath = append([]int(nil), path[:len(path)-1]...)
	}
	if b.slotAnchors == nil {
		b.slotAnchors = make(map[int]NodeAnchor)
	}
	b.slotAnchors[slot] = anchor
}

func (b *structuredBuilder) pushChildIndex(idx int) {
	if len(b.componentStack) == 0 {
		return
	}
	b.componentPath = append(b.componentPath, idx)
}

func (b *structuredBuilder) popChildIndex() {
	if len(b.componentStack) == 0 {
		return
	}
	if l := len(b.componentPath); l > 0 {
		b.componentPath = b.componentPath[:l-1]
	}
}

func (b *structuredBuilder) pushComponent(id string, parentPath []int, anchorIndex, startChildCount int) {
	frame := componentFrame{
		id:              id,
		prevPath:        append([]int(nil), b.componentPath...),
		parentPath:      append([]int(nil), parentPath...),
		anchorIndex:     anchorIndex,
		startChildCount: startChildCount,
	}
	b.componentStack = append(b.componentStack, frame)
	b.componentPath = b.componentPath[:0]
}

func (b *structuredBuilder) popComponent() componentFrame {
	if len(b.componentStack) == 0 {
		return componentFrame{}
	}
	lastIdx := len(b.componentStack) - 1
	frame := b.componentStack[lastIdx]
	b.componentStack = b.componentStack[:lastIdx]
	b.componentPath = append([]int(nil), frame.prevPath...)
	return frame
}

func (b *structuredBuilder) currentComponentID() string {
	if len(b.componentStack) == 0 {
		return ""
	}
	return b.componentStack[len(b.componentStack)-1].id
}

func (b *structuredBuilder) currentComponentPath() []int {
	if len(b.componentStack) == 0 {
		return nil
	}
	return append([]int(nil), b.componentPath...)
}

func (b *structuredBuilder) popFrame() {
	if len(b.stack) == 0 {
		return
	}
	lastIdx := len(b.stack) - 1
	frame := b.stack[lastIdx]
	b.stack = b.stack[:lastIdx]
	b.assignSlotIndices(frame)
}

func (b *structuredBuilder) assignSlotIndices(frame elementFrame) {
	if frame.staticAttrs && frame.startStatic >= 0 && frame.startStatic < len(b.statics) {
		b.statics[frame.startStatic] = renderStartTag(frame.element, frame.void)
	}
	if len(frame.element.HandlerAssignments) > 0 && frame.attrSlot >= 0 {
		if attrs := frame.element.HandlerAssignments; len(attrs) > 0 {
			for event, assignment := range attrs {
				b.handlerBindings = append(b.handlerBindings, HandlerBinding{
					Slot:    frame.attrSlot,
					Event:   event,
					Handler: assignment.ID,
					Listen:  append([]string(nil), assignment.Listen...),
					Props:   append([]string(nil), assignment.Props...),
				})
			}
		}
	}
}

func escapeComment(value string) string {
	return strings.ReplaceAll(value, "--", "- -")
}

func copyAttrs(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	keys := make([]string, 0, len(src))
	for k := range src {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	m := make(map[string]string, len(src))
	for _, k := range keys {
		m[k] = src[k]
	}
	return m
}

func (b *structuredBuilder) anchors() map[int]NodeAnchor {
	if len(b.slotAnchors) == 0 {
		return nil
	}
	out := make(map[int]NodeAnchor, len(b.slotAnchors))
	for slot, anchor := range b.slotAnchors {
		cloned := NodeAnchor{ChildIndex: anchor.ChildIndex, HasIndex: anchor.HasIndex}
		if len(anchor.ParentPath) > 0 {
			cloned.ParentPath = append([]int(nil), anchor.ParentPath...)
		}
		out[slot] = cloned
	}
	return out
}

// RelativeNodeAnchor computes a node anchor relative to a base parent path. If the
// provided anchor does not share the same prefix, the returned boolean will be false.
func RelativeNodeAnchor(anchor NodeAnchor, basePath []int) (NodeAnchor, bool) {
	if len(basePath) == 0 {
		cloned := NodeAnchor{ChildIndex: anchor.ChildIndex, HasIndex: anchor.HasIndex}
		if len(anchor.ParentPath) > 0 {
			cloned.ParentPath = append([]int(nil), anchor.ParentPath...)
		}
		return cloned, true
	}
	if len(anchor.ParentPath) < len(basePath) {
		return NodeAnchor{}, false
	}
	for i := range basePath {
		if anchor.ParentPath[i] != basePath[i] {
			return NodeAnchor{}, false
		}
	}
	rel := NodeAnchor{ChildIndex: anchor.ChildIndex, HasIndex: anchor.HasIndex}
	if len(anchor.ParentPath) > len(basePath) {
		rel.ParentPath = append([]int(nil), anchor.ParentPath[len(basePath):]...)
	}
	return rel, true
}

func shouldForceDynamicAttrs(mutable map[string]bool, attrs map[string]string) bool {
	if len(mutable) == 0 {
		return false
	}
	if mutable["*"] {
		return true
	}
	if len(attrs) == 0 {
		return false
	}
	for key := range attrs {
		if mutable[key] {
			return true
		}
	}
	return false
}

func renderStartTag(el *h.Element, void bool) string {
	if el == nil {
		return ""
	}
	var b strings.Builder
	b.WriteByte('<')
	b.WriteString(el.Tag)
	if len(el.Attrs) > 0 {
		keys := make([]string, 0, len(el.Attrs))
		for k := range el.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := el.Attrs[k]
			if v == "" {
				continue
			}
			if strings.HasPrefix(k, "data-on") || strings.HasPrefix(k, "data-router-") {
				continue
			}
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteString("=\"")
			b.WriteString(html.EscapeString(v))
			b.WriteString("\"")
		}
	}
	if void {
		b.WriteString("/>")
	} else {
		b.WriteByte('>')
	}
	return b.String()
}
