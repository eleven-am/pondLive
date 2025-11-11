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

// Row represents a keyed row inside a dynamic list slot.
type Row struct {
	Key            string
	HTML           string
	Slots          []int
	Bindings       []HandlerBinding
	SlotPaths      []SlotPath
	ListPaths      []ListPath
	ComponentPaths []ComponentPath
}

// SlotPath captures the structural location of a dynamic slot anchor relative to a component root.
type SlotPath struct {
	Slot           int
	ComponentID    string
	ElementPath    []int
	TextChildIndex int
}

// ListPath captures the structural location of a keyed list container relative to a component root.
type ListPath struct {
	Slot        int
	ComponentID string
	ElementPath []int
}

// ComponentPath captures the structural bounds of a component subtree.
type ComponentPath struct {
	ComponentID string
	ParentID    string
	ParentPath  []int
	FirstChild  []int
	LastChild   []int
}

// Structured represents the structured render output of a node tree.
type Structured struct {
	S          []string
	D          []Dyn
	Components map[string]ComponentSpan
	Bindings   []HandlerBinding

	SlotPaths      []SlotPath
	ListPaths      []ListPath
	ComponentPaths []ComponentPath
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
	b.visit(n)
	b.flush()
	return Structured{
		S:              b.statics,
		D:              b.dynamics,
		Components:     b.components,
		Bindings:       append([]HandlerBinding(nil), b.handlerBindings...),
		SlotPaths:      append([]SlotPath(nil), b.slotPaths...),
		ListPaths:      append([]ListPath(nil), b.listPaths...),
		ComponentPaths: append([]ComponentPath(nil), b.componentPaths...),
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

	slotPaths      []SlotPath
	listPaths      []ListPath
	componentPaths []ComponentPath
}

type elementFrame struct {
	attrSlot      int
	element       *h.Element
	bindings      []slotBinding
	startStatic   int
	void          bool
	staticAttrs   bool
	componentID   string
	componentPath []int
}

type componentFrame struct {
	id        string
	parentID  string
	prevPath  []int
	startPath []int
	endPath   []int
}

type slotBinding struct {
	slot       int
	childIndex int
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
		b.appendSlotToCurrent(idx, t)
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
	b.pushComponent(v.ID)
	if v.Child != nil {
		b.visit(v.Child)
	}
	b.popComponent()
	b.flush()
	span := ComponentSpan{
		StaticsStart:  staticsStart,
		StaticsEnd:    len(b.statics),
		DynamicsStart: dynamicsStart,
		DynamicsEnd:   len(b.dynamics),
	}
	if b.components == nil {
		b.components = make(map[string]ComponentSpan)
	}
	b.components[v.ID] = span
}

func (b *structuredBuilder) visitElement(v *h.Element) {
	if v == nil {
		return
	}
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
	frame := elementFrame{
		attrSlot:      attrSlot,
		element:       el,
		startStatic:   startStatic,
		void:          void,
		staticAttrs:   staticAttrs,
		componentID:   b.currentComponentID(),
		componentPath: b.currentComponentPath(),
	}
	if attrSlot >= 0 {
		frame.bindings = append(frame.bindings, slotBinding{slot: attrSlot, childIndex: -1})
	}
	b.stack = append(b.stack, frame)
}

func (b *structuredBuilder) appendSlotToCurrent(slot int, node h.Node) {
	if len(b.stack) == 0 {
		return
	}
	frame := &b.stack[len(b.stack)-1]
	binding := slotBinding{slot: slot, childIndex: -1}
	if txt, ok := node.(*h.TextNode); ok {
		binding.childIndex = childIndexOf(frame.element, txt)
	}
	frame.bindings = append(frame.bindings, binding)
}

func (b *structuredBuilder) pushChildIndex(idx int) {
	if len(b.componentStack) == 0 {
		return
	}
	b.componentPath = append(b.componentPath, idx)
	b.recordComponentTraversal()
}

func (b *structuredBuilder) popChildIndex() {
	if len(b.componentStack) == 0 {
		return
	}
	if l := len(b.componentPath); l > 0 {
		b.componentPath = b.componentPath[:l-1]
	}
}

func (b *structuredBuilder) recordComponentTraversal() {
	if len(b.componentStack) == 0 {
		return
	}
	frame := &b.componentStack[len(b.componentStack)-1]
	if frame.startPath == nil {
		frame.startPath = append([]int(nil), b.componentPath...)
	}
	frame.endPath = append([]int(nil), b.componentPath...)
}

func (b *structuredBuilder) pushComponent(id string) {
	parentID := ""
	if len(b.componentStack) > 0 {
		parentID = b.componentStack[len(b.componentStack)-1].id
	}
	frame := componentFrame{
		id:       id,
		parentID: parentID,
		prevPath: append([]int(nil), b.componentPath...),
	}
	b.componentStack = append(b.componentStack, frame)
	b.componentPath = b.componentPath[:0]
}

func (b *structuredBuilder) popComponent() {
	if len(b.componentStack) == 0 {
		return
	}
	lastIdx := len(b.componentStack) - 1
	frame := b.componentStack[lastIdx]
	b.componentStack = b.componentStack[:lastIdx]
	b.componentPath = append([]int(nil), frame.prevPath...)
	if frame.id != "" {
		b.componentPaths = append(b.componentPaths, ComponentPath{
			ComponentID: frame.id,
			ParentID:    frame.parentID,
			ParentPath:  append([]int(nil), frame.prevPath...),
			FirstChild:  append([]int(nil), frame.startPath...),
			LastChild:   append([]int(nil), frame.endPath...),
		})
	}
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
	if frame.componentID != "" {
		seen := make(map[[2]int]struct{}, len(frame.bindings))
		for _, binding := range frame.bindings {
			key := [2]int{binding.slot, binding.childIndex}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			anchor := SlotPath{
				Slot:           binding.slot,
				ComponentID:    frame.componentID,
				ElementPath:    append([]int(nil), frame.componentPath...),
				TextChildIndex: -1,
			}
			if binding.childIndex >= 0 {
				anchor.TextChildIndex = binding.childIndex
			}
			b.slotPaths = append(b.slotPaths, anchor)
		}
	}
}

func escapeComment(value string) string {
	return strings.ReplaceAll(value, "--", "- -")
}

func childIndexOf(parent *h.Element, target h.Node) int {
	if parent == nil || target == nil {
		return -1
	}
	for idx, child := range parent.Children {
		if child == target {
			return idx
		}
	}
	return -1
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
