package slot

import (
	"fmt"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// SlotContext provides slot functionality using the Context API.
// Each component that uses slots should create its own SlotContext instance.
type SlotContext struct {
	ctx *runtime.Context[*SlotMap]
}

// CreateSlotContext creates a new slot context instance.
// Typically defined at package level for a component.
//
// Example:
//
//	var cardSlotCtx = CreateSlotContext()
func CreateSlotContext() *SlotContext {
	return &SlotContext{
		ctx: runtime.CreateContext[*SlotMap](nil),
	}
}

// nodeStructureHash creates a lightweight structural hash of a node tree
// including tag names, text lengths, and child count to detect structural changes.
// Uses FNV-1a hash for all child tags to keep fingerprint bounded while detecting all changes.
func nodeStructureHash(node *dom.StructuredNode) string {
	if node == nil {
		return "nil"
	}

	var parts []string

	if node.Tag != "" {
		parts = append(parts, node.Tag)
	}

	if node.Text != "" {
		parts = append(parts, fmt.Sprintf("t%d", len(node.Text)))
	}

	if len(node.Children) > 0 {
		parts = append(parts, fmt.Sprintf("c%d", len(node.Children)))

		hash := uint32(2166136261)
		for _, child := range node.Children {
			if child != nil && child.Tag != "" {

				for i := 0; i < len(child.Tag); i++ {
					hash ^= uint32(child.Tag[i])
					hash *= 16777619
				}
			}
		}
		if hash != 2166136261 {
			parts = append(parts, fmt.Sprintf("h%x", hash))
		}
	}

	return strings.Join(parts, ".")
}

// slotFingerprint creates a structural fingerprint of regular slots
// based on slot names, child counts, and child structure. Avoids re-extraction when structure is stable.
// Uses strings.Builder for efficient string construction with many slots.
func slotFingerprint(children []dom.Item) string {
	temp := dom.FragmentNode()
	for _, child := range children {
		if child != nil {
			child.ApplyTo(temp)
		}
	}

	var builder strings.Builder
	var defaultNodes []*dom.StructuredNode
	first := true

	for _, node := range temp.Children {
		if node == nil {
			continue
		}

		if node.Metadata != nil {
			if name, ok := node.Metadata[slotNameKey].(string); ok {
				if !first {
					builder.WriteByte(',')
				}
				first = false

				childCount := len(node.Children)
				structHash := nodeStructureHash(node)
				fmt.Fprintf(&builder, "%s:%d:%s", name, childCount, structHash)
				continue
			}
		}

		defaultNodes = append(defaultNodes, node)
	}

	if len(defaultNodes) > 0 {
		if !first {
			builder.WriteByte(',')
		}

		builder.WriteString("default:")
		fmt.Fprintf(&builder, "%d:", len(defaultNodes))

		for i, node := range defaultNodes {
			if i > 0 {
				builder.WriteByte('.')
			}
			builder.WriteString(nodeStructureHash(node))
		}
	}

	return builder.String()
}

// Provide wraps children in a slot context provider.
// Extracts slots from children and makes them available to the render function.
// Uses structural fingerprinting to avoid re-extraction when stable.
//
// Example:
//
//	return cardSlotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
//	    return h.Div(
//	        h.Header(cardSlotCtx.Render(sctx, "header")),
//	        h.Div(cardSlotCtx.Render(sctx, "default")),
//	    )
//	})
func (sc *SlotContext) Provide(
	ctx runtime.Ctx,
	children []dom.Item,
	render func(runtime.Ctx) *dom.StructuredNode,
) *dom.StructuredNode {
	fingerprint := slotFingerprint(children)

	slots := runtime.UseMemo(ctx, func() *SlotMap {
		return extractSlots(children)
	}, fingerprint)

	return sc.ctx.Provide(ctx, slots, render)
}

// Render retrieves and renders a named slot.
// Returns an empty fragment if the slot doesn't exist.
//
// Example:
//
//	cardSlotCtx.Render(ctx, "header")
func (sc *SlotContext) Render(ctx runtime.Ctx, name string) *dom.StructuredNode {
	slots := sc.ctx.Use(ctx)
	if slots == nil {
		return dom.FragmentNode()
	}

	if content := slots.slots[name]; content != nil {
		return renderSlotContent(content)
	}

	if fallback := slots.fallbacks[name]; fallback != nil {
		return renderSlotContent(fallback)
	}

	return dom.FragmentNode()
}

// Has checks if a named slot was provided by the user.
// Useful for conditional rendering.
//
// Example:
//
//	if cardSlotCtx.Has(ctx, "footer") {
//	    return h.Footer(cardSlotCtx.Render(ctx, "footer"))
//	}
func (sc *SlotContext) Has(ctx runtime.Ctx, name string) bool {
	slots := sc.ctx.Use(ctx)
	if slots == nil {
		return false
	}
	return slots.slots[name] != nil
}

// SetFallback defines default content for a slot.
// Only used if the slot was not provided by the user.
//
// Example:
//
//	if !cardSlotCtx.Has(ctx, "footer") {
//	    cardSlotCtx.SetFallback(ctx, "footer",
//	        h.Button(h.Text("Default Close")),
//	    )
//	}
func (sc *SlotContext) SetFallback(ctx runtime.Ctx, name string, content *dom.StructuredNode) {
	slots := sc.ctx.Use(ctx)
	if slots == nil {
		return
	}

	if slots.slots[name] == nil {
		if slots.fallbacks == nil {
			slots.fallbacks = make(map[string]*SlotContent)
		}
		slots.fallbacks[name] = nodeToSlotContent(content)
	}
}

// GetSlotNames returns all slot names that were provided in declaration order.
// Useful for dynamic slot rendering.
//
// Example:
//
//	for _, name := range slotCtx.GetSlotNames(ctx) {
//	    sections = append(sections, h.Section(slotCtx.Render(ctx, name)))
//	}
func (sc *SlotContext) GetSlotNames(ctx runtime.Ctx) []string {
	slots := sc.ctx.Use(ctx)
	if slots == nil || slots.slotOrder == nil {
		return nil
	}

	names := make([]string, len(slots.slotOrder))
	copy(names, slots.slotOrder)
	return names
}
