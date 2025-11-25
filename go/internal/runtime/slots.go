package runtime

import (
	"fmt"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/work"
)

// SlotMap organizes slot content by name.
type SlotMap struct {
	slots     map[string][]work.Node
	slotOrder []string
}

// SlotContext provides slot functionality using the Context API.
// Used for patterns where the slot provider and consumer are separate components.
type SlotContext struct {
	ctx *Context[*SlotMap]
}

// CreateSlotContext creates a new slot context instance.
func CreateSlotContext() *SlotContext {
	return &SlotContext{
		ctx: CreateContext[*SlotMap](nil),
	}
}

// Provide extracts slots from children (including default content) and provides them via context.
func (sc *SlotContext) Provide(ctx *Ctx, children []work.Node) work.Node {
	slots := extractSlotsWithDefault(children, true)
	sc.ctx.UseProvider(ctx, slots)
	return &work.Fragment{Children: filterRenderedChildren(children)}
}

// ProvideWithoutDefault extracts only named slots and provides them via context without
// populating the default slot. This is useful when children should render normally but
// an empty default slot is desired until explicitly set.
func (sc *SlotContext) ProvideWithoutDefault(ctx *Ctx, children []work.Node) work.Node {
	slots := extractSlotsWithDefault(children, false)
	sc.ctx.UseProvider(ctx, slots)
	return &work.Fragment{Children: filterRenderedChildren(children)}
}

func filterRenderedChildren(children []work.Node) []work.Node {
	filtered := make([]work.Node, 0, len(children))

	for _, child := range children {
		if frag, ok := child.(*work.Fragment); ok && frag.Metadata != nil {
			if _, ok := frag.Metadata["slot:name"]; ok {
				continue
			}
			if _, ok := frag.Metadata["scoped-slot:name"]; ok {
				continue
			}
		}

		filtered = append(filtered, child)
	}

	return filtered
}

// Render retrieves and renders a named slot from the context.
func (sc *SlotContext) Render(ctx *Ctx, name string, fallback ...work.Node) work.Node {
	slots := sc.ctx.UseContextValue(ctx)
	if slots == nil || len(slots.slots[name]) == 0 {
		if len(fallback) == 0 {
			return &work.Fragment{Children: nil}
		}
		if len(fallback) == 1 {
			return fallback[0]
		}
		return &work.Fragment{Children: fallback}
	}

	nodes := slots.slots[name]
	if len(nodes) == 1 {
		return nodes[0]
	}

	return &work.Fragment{Children: nodes}
}

// Has checks if a named slot was provided.
func (sc *SlotContext) Has(ctx *Ctx, name string) bool {
	slots := sc.ctx.UseContextValue(ctx)
	if slots == nil {
		return false
	}
	return len(slots.slots[name]) > 0
}

// SetSlot adds content to a named slot in the current context.
func (sc *SlotContext) SetSlot(ctx *Ctx, name string, content ...work.Node) {
	slots := sc.ctx.UseContextValue(ctx)
	if slots == nil {
		return
	}

	if slots.slots == nil {
		slots.slots = make(map[string][]work.Node)
	}

	slots.slots[name] = content

	found := false
	for _, n := range slots.slotOrder {
		if n == name {
			found = true
			break
		}
	}
	if !found {
		slots.slotOrder = append(slots.slotOrder, name)
	}
}

// Names returns all slot names in declaration order.
func (sc *SlotContext) Names(ctx *Ctx) []string {
	slots := sc.ctx.UseContextValue(ctx)
	if slots == nil {
		return nil
	}
	return slots.slotOrder
}

// UseSlots extracts slots from children using context.
// This is a convenience function for components that both provide and consume slots.
func UseSlots(ctx *Ctx, children []work.Node) *SlotRenderer {
	slots := extractSlots(children)
	return &SlotRenderer{slots: slots}
}

// SlotRenderer provides methods to render extracted slots.
type SlotRenderer struct {
	slots *SlotMap
}

func (sr *SlotRenderer) Render(name string, fallback ...work.Node) work.Node {
	if sr.slots == nil || len(sr.slots.slots[name]) == 0 {
		if len(fallback) == 0 {
			return &work.Fragment{Children: nil}
		}
		if len(fallback) == 1 {
			return fallback[0]
		}
		return &work.Fragment{Children: fallback}
	}

	nodes := sr.slots.slots[name]
	if len(nodes) == 1 {
		return nodes[0]
	}

	return &work.Fragment{Children: nodes}
}

func (sr *SlotRenderer) Has(name string) bool {
	if sr.slots == nil {
		return false
	}
	return len(sr.slots.slots[name]) > 0
}

func (sr *SlotRenderer) Names() []string {
	if sr.slots == nil {
		return nil
	}
	return sr.slots.slotOrder
}

func extractSlots(children []work.Node) *SlotMap {
	return extractSlotsWithDefault(children, true)
}

func extractSlotsWithDefault(children []work.Node, includeDefault bool) *SlotMap {
	slots := make(map[string][]work.Node)
	var slotOrder []string
	var defaultContent []work.Node
	seenDefault := false

	for _, child := range children {
		if child == nil {
			continue
		}

		if frag, ok := child.(*work.Fragment); ok && frag.Metadata != nil {
			if slotName, ok := frag.Metadata["slot:name"].(string); ok {
				if _, exists := slots[slotName]; !exists {
					slotOrder = append(slotOrder, slotName)
				}
				slots[slotName] = append(slots[slotName], frag.Children...)
				continue
			}
		}

		if includeDefault {
			if !seenDefault {
				seenDefault = true
			}
			defaultContent = append(defaultContent, child)
		}
	}

	if includeDefault && len(defaultContent) > 0 {
		slots["default"] = defaultContent
		if seenDefault {
			slotOrder = append(slotOrder, "default")
		}
	}

	return &SlotMap{
		slots:     slots,
		slotOrder: slotOrder,
	}
}

type ScopedSlotMap[T any] struct {
	slots map[string][]func(T) work.Node
}

type ScopedSlotRenderer[T any] struct {
	slots *ScopedSlotMap[T]
}

func (sr *ScopedSlotRenderer[T]) Render(name string, data T, fallback ...work.Node) work.Node {
	if sr.slots == nil || len(sr.slots.slots[name]) == 0 {
		if len(fallback) == 0 {
			return &work.Fragment{Children: nil}
		}
		if len(fallback) == 1 {
			return fallback[0]
		}
		return &work.Fragment{Children: fallback}
	}

	fns := sr.slots.slots[name]
	results := make([]work.Node, len(fns))
	for i, fn := range fns {
		results[i] = fn(data)
	}

	if len(results) == 1 {
		return results[0]
	}

	return &work.Fragment{Children: results}
}

func (sr *ScopedSlotRenderer[T]) Has(name string) bool {
	if sr.slots == nil {
		return false
	}
	return len(sr.slots.slots[name]) > 0
}

func UseScopedSlots[T any](ctx *Ctx, children []work.Node) *ScopedSlotRenderer[T] {
	slots := UseMemo(ctx, func() *ScopedSlotMap[T] {
		return extractScopedSlots[T](children)
	}, fingerprintChildren(children))

	return &ScopedSlotRenderer[T]{slots: slots}
}

func extractScopedSlots[T any](children []work.Node) *ScopedSlotMap[T] {
	slots := make(map[string][]func(T) work.Node)

	for _, child := range children {
		if frag, ok := child.(*work.Fragment); ok && frag.Metadata != nil {
			if slotName, ok := frag.Metadata["scoped-slot:name"].(string); ok {
				if fn, ok := frag.Metadata["scoped-slot:fn"].(func(T) work.Node); ok {
					slots[slotName] = append(slots[slotName], fn)
				}
			}
		}
	}

	return &ScopedSlotMap[T]{slots: slots}
}

func fingerprintChildren(children []work.Node) string {
	var parts []string

	for _, child := range children {
		parts = append(parts, fingerprintNode(child))
	}

	return strings.Join(parts, "|")
}

func fingerprintNode(node work.Node) string {
	if node == nil {
		return "nil"
	}

	switch n := node.(type) {
	case *work.Element:
		return fmt.Sprintf("e:%s:%d", n.Tag, len(n.Children))

	case *work.Text:
		return fmt.Sprintf("t:%d", len(n.Value))

	case *work.Fragment:
		if n.Metadata != nil {
			if name, ok := n.Metadata["slot:name"].(string); ok {
				return fmt.Sprintf("slot:%s:%d", name, len(n.Children))
			}
			if name, ok := n.Metadata["scoped-slot:name"].(string); ok {
				if fn, ok := n.Metadata["scoped-slot:fn"]; ok {
					return fmt.Sprintf("scoped:%s:%p", name, fn)
				}
			}
		}
		return fmt.Sprintf("f:%d", len(n.Children))

	case *work.ComponentNode:
		return fmt.Sprintf("c:%p:%s", n.Fn, n.Key)

	default:
		return "unknown"
	}
}
