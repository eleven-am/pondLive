package render

import (
	"sort"
	"strconv"
	"strings"

	"github.com/eleven-am/go/pondlive/internal/handlers"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
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
	Key   string
	HTML  string
	Slots []int
}

// Structured represents the structured render output of a node tree.
type Structured struct {
	S []string
	D []Dyn
}

// ToStructured lowers a node tree into structured statics and dynamics.
func ToStructured(n h.Node) Structured {
	return ToStructuredWithHandlers(n, nil)
}

// ToStructuredWithHandlers lowers a node tree into structured statics and
// dynamics while registering handlers in the provided registry.
func ToStructuredWithHandlers(n h.Node, reg handlers.Registry) Structured {
	if n == nil {
		return Structured{}
	}
	FinalizeWithHandlers(n, reg)
	b := &structuredBuilder{}
	b.visit(n)
	b.flush()
	return Structured{S: b.statics, D: b.dynamics}
}

type structuredBuilder struct {
	statics  []string
	current  strings.Builder
	dynamics []Dyn
	stack    []elementFrame
}

type elementFrame struct {
	attrSlot int
	element  *h.Element
	bindings []slotBinding
}

type slotBinding struct {
	slot       int
	childIndex int
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
		idx := b.addDyn(Dyn{Kind: DynText, Text: v.Value})
		b.appendSlotToCurrent(idx, v)
	case *h.Element:
		b.visitElement(v)
	case *h.FragmentNode:
		b.visitFragment(v)
	}
}

func (b *structuredBuilder) visitElement(v *h.Element) {
	if v == nil {
		return
	}
	b.writeStatic("<")
	b.writeStatic(v.Tag)
	d := Dyn{Kind: DynAttrs}
	if len(v.Attrs) > 0 {
		d.Attrs = copyAttrs(v.Attrs)
	} else {
		d.Attrs = map[string]string{}
	}
	attrSlot := b.addDyn(d)
	b.pushFrame(v, attrSlot)
	defer b.popFrame()
	if _, ok := voidElements[strings.ToLower(v.Tag)]; ok {
		b.writeStatic("/>")
		return
	}
	b.writeStatic(">")
	if v.Unsafe != nil {
		b.writeStatic(*v.Unsafe)
	} else if !b.tryKeyedChildren(v, v.Children, attrSlot) {
		for _, child := range v.Children {
			if child == nil {
				continue
			}
			b.visit(child)
		}
	}
	b.writeStatic("</")
	b.writeStatic(v.Tag)
	b.writeStatic(">")
}

func (b *structuredBuilder) visitFragment(f *h.FragmentNode) {
	if f == nil {
		return
	}
	if b.tryKeyedChildren(nil, f.Children, -1) {
		return
	}
	for _, child := range f.Children {
		if child == nil {
			continue
		}
		b.visit(child)
	}
}

func (b *structuredBuilder) pushFrame(el *h.Element, attrSlot int) {
	frame := elementFrame{attrSlot: attrSlot, element: el}
	frame.bindings = append(frame.bindings, slotBinding{slot: attrSlot, childIndex: -1})
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
	if frame.attrSlot < 0 || frame.attrSlot >= len(b.dynamics) {
		return
	}
	value := joinSlotBindings(frame.bindings)
	if value == "" {
		return
	}
	attrs := frame.element.Attrs
	if attrs == nil {
		attrs = map[string]string{}
	}
	attrs["data-slot-index"] = value
	frame.element.Attrs = attrs

	dynAttrs := b.dynamics[frame.attrSlot].Attrs
	if dynAttrs == nil {
		dynAttrs = map[string]string{}
	}
	dynAttrs["data-slot-index"] = value
	b.dynamics[frame.attrSlot].Attrs = dynAttrs
}

func joinSlotBindings(bindings []slotBinding) string {
	if len(bindings) == 0 {
		return ""
	}
	parts := make([]string, 0, len(bindings))
	seen := make(map[int]struct{}, len(bindings))
	for _, binding := range bindings {
		if _, ok := seen[binding.slot]; ok {
			continue
		}
		seen[binding.slot] = struct{}{}
		token := strconv.Itoa(binding.slot)
		if binding.childIndex >= 0 {
			token = token + "@" + strconv.Itoa(binding.childIndex)
		}
		parts = append(parts, token)
	}
	return strings.Join(parts, " ")
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
