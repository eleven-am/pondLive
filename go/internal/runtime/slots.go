package runtime

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/work"
)

const DefaultSlotName = "default"

type SlotMap struct {
	slots     map[string][]work.Node
	slotOrder []string
	setSlots  func(*SlotMap)
}

type SlotContext struct {
	ctx *Context[*SlotMap]
}

func CreateSlotContext() *SlotContext {
	return &SlotContext{
		ctx: CreateContext[*SlotMap](nil).WithEqual(slotMapEqual),
	}
}

func slotMapEqual(a, b *SlotMap) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a.slotOrder) != len(b.slotOrder) {
		return false
	}
	for i, name := range a.slotOrder {
		if b.slotOrder[i] != name {
			return false
		}
	}
	if len(a.slots) != len(b.slots) {
		return false
	}
	for name, nodesA := range a.slots {
		nodesB, exists := b.slots[name]
		if !exists {
			return false
		}
		fpA := fingerprintSlice(nodesA)
		fpB := fingerprintSlice(nodesB)
		if fpA != fpB {
			return false
		}
	}
	return true
}

func fingerprintSlice(nodes []work.Node) string {
	if len(nodes) == 0 {
		return ""
	}
	var parts []string
	for _, n := range nodes {
		parts = append(parts, fingerprintNode(n))
	}
	return strings.Join(parts, ",")
}

func (sc *SlotContext) Provide(ctx *Ctx, children []work.Node) work.Node {
	slots := extractSlotsWithDefault(children, true)
	state, setState := sc.ctx.UseProvider(ctx, slots)
	state.setSlots = func(newSlots *SlotMap) {
		newSlots.setSlots = state.setSlots
		setState(newSlots)
	}
	return &work.Fragment{Children: filterRenderedChildren(children)}
}

func (sc *SlotContext) ProvideWithoutDefault(ctx *Ctx, children []work.Node) work.Node {
	slots := extractSlotsWithDefault(children, false)
	state, setState := sc.ctx.UseProvider(ctx, slots)
	state.setSlots = func(newSlots *SlotMap) {
		newSlots.setSlots = state.setSlots
		setState(newSlots)
	}
	return &work.Fragment{Children: filterRenderedChildren(children)}
}

func filterRenderedChildren(children []work.Node) []work.Node {
	hasSlots := false
	for _, child := range children {
		if frag, ok := child.(*work.Fragment); ok && frag.Metadata != nil {
			if _, ok := frag.Metadata["slot:name"]; ok {
				hasSlots = true
				break
			}
			if _, ok := frag.Metadata["scoped-slot:name"]; ok {
				hasSlots = true
				break
			}
		}
	}
	if !hasSlots {
		return children
	}

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

func renderSlotNodes(nodes []work.Node, fallback []work.Node) work.Node {
	if len(nodes) == 0 {
		if len(fallback) == 0 {
			return &work.Fragment{Children: nil}
		}
		if len(fallback) == 1 {
			return fallback[0]
		}
		return &work.Fragment{Children: fallback}
	}
	if len(nodes) == 1 {
		return nodes[0]
	}
	return &work.Fragment{Children: nodes}
}

func (sc *SlotContext) Render(ctx *Ctx, name string, fallback ...work.Node) work.Node {
	slots := sc.ctx.UseContextValue(ctx)
	if slots == nil {
		return renderSlotNodes(nil, fallback)
	}
	return renderSlotNodes(slots.slots[name], fallback)
}

func (sc *SlotContext) Has(ctx *Ctx, name string) bool {
	slots := sc.ctx.UseContextValue(ctx)
	if slots == nil {
		return false
	}
	return len(slots.slots[name]) > 0
}

func (sc *SlotContext) SetSlot(ctx *Ctx, name string, content ...work.Node) {
	slots := sc.ctx.UseContextValue(ctx)
	if slots == nil {
		return
	}

	existing := slots.slots[name]
	if len(existing) == len(content) {
		same := true
		for i := range existing {
			if existing[i] != content[i] {
				same = false
				break
			}
		}
		if same {
			return
		}
	}

	newSlots := make(map[string][]work.Node)
	for k, v := range slots.slots {
		newSlots[k] = v
	}
	newSlots[name] = content

	newOrder := slots.slotOrder
	_, existed := slots.slots[name]
	if !existed {
		newOrder = append(append([]string{}, slots.slotOrder...), name)
	}

	if slots.setSlots != nil {
		slots.setSlots(&SlotMap{
			slots:     newSlots,
			slotOrder: newOrder,
		})
	}
}

func (sc *SlotContext) AppendSlot(ctx *Ctx, name string, content ...work.Node) {
	slots := sc.ctx.UseContextValue(ctx)
	if slots == nil {
		return
	}

	newSlots := make(map[string][]work.Node)
	for k, v := range slots.slots {
		newSlots[k] = v
	}
	newSlots[name] = append(newSlots[name], content...)

	newOrder := slots.slotOrder
	_, existed := slots.slots[name]
	if !existed {
		newOrder = append(append([]string{}, slots.slotOrder...), name)
	}

	if slots.setSlots != nil {
		slots.setSlots(&SlotMap{
			slots:     newSlots,
			slotOrder: newOrder,
		})
	}
}

func (sc *SlotContext) Names(ctx *Ctx) []string {
	slots := sc.ctx.UseContextValue(ctx)
	if slots == nil {
		return nil
	}
	return slots.slotOrder
}

func UseSlots(ctx *Ctx, children []work.Node) *SlotRenderer {
	slots := extractSlots(children)
	return &SlotRenderer{slots: slots}
}

type SlotRenderer struct {
	slots *SlotMap
}

func (sr *SlotRenderer) Render(name string, fallback ...work.Node) work.Node {
	if sr.slots == nil {
		return renderSlotNodes(nil, fallback)
	}
	return renderSlotNodes(sr.slots.slots[name], fallback)
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

func (sr *SlotRenderer) RequireSlots(names ...string) error {
	var missing []string
	for _, name := range names {
		if !sr.Has(name) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required slots: %v", missing)
	}
	return nil
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
		slots[DefaultSlotName] = defaultContent
		if seenDefault {
			slotOrder = append(slotOrder, DefaultSlotName)
		}
	}

	return &SlotMap{
		slots:     slots,
		slotOrder: slotOrder,
	}
}

type ScopedSlotMap[T any] struct {
	slots     map[string][]func(T) work.Node
	slotOrder []string
}

type ScopedSlotRenderer[T any] struct {
	slots *ScopedSlotMap[T]
}

func (sr *ScopedSlotRenderer[T]) Render(name string, data T, fallback ...work.Node) work.Node {
	if sr.slots == nil || len(sr.slots.slots[name]) == 0 {
		return renderSlotNodes(nil, fallback)
	}

	fns := sr.slots.slots[name]
	results := make([]work.Node, len(fns))
	for i, fn := range fns {
		results[i] = fn(data)
	}

	return renderSlotNodes(results, nil)
}

func (sr *ScopedSlotRenderer[T]) Has(name string) bool {
	if sr.slots == nil {
		return false
	}
	return len(sr.slots.slots[name]) > 0
}

func (sr *ScopedSlotRenderer[T]) Names() []string {
	if sr.slots == nil {
		return nil
	}
	return sr.slots.slotOrder
}

func UseScopedSlots[T any](ctx *Ctx, children []work.Node) *ScopedSlotRenderer[T] {
	slots := UseMemo(ctx, func() *ScopedSlotMap[T] {
		return extractScopedSlots[T](children)
	}, fingerprintChildren(children))

	return &ScopedSlotRenderer[T]{slots: slots}
}

func extractScopedSlots[T any](children []work.Node) *ScopedSlotMap[T] {
	slots := make(map[string][]func(T) work.Node)
	var slotOrder []string

	for _, child := range children {
		if frag, ok := child.(*work.Fragment); ok && frag.Metadata != nil {
			if slotName, ok := frag.Metadata["scoped-slot:name"].(string); ok {
				if fn, ok := frag.Metadata["scoped-slot:fn"].(func(T) work.Node); ok {
					if _, exists := slots[slotName]; !exists {
						slotOrder = append(slotOrder, slotName)
					}
					slots[slotName] = append(slots[slotName], fn)
				}
			}
		}
	}

	return &ScopedSlotMap[T]{slots: slots, slotOrder: slotOrder}
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
		h := fnv.New32a()
		h.Write([]byte(n.Value))
		return fmt.Sprintf("t:%d:%x", len(n.Value), h.Sum32())

	case *work.Fragment:
		if n.Metadata != nil {
			if name, ok := n.Metadata["slot:name"].(string); ok {
				return fmt.Sprintf("slot:%s:%d", name, len(n.Children))
			}
			if name, ok := n.Metadata["scoped-slot:name"].(string); ok {
				if fn, ok := n.Metadata["scoped-slot:fn"]; ok {
					return fmt.Sprintf("scoped:%s:%p:%d", name, fn, len(n.Children))
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
