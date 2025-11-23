package slot

import (
	"fmt"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// ScopedSlotContext provides scoped slot functionality.
// Allows components to pass data to slot content.
// Generic parameter T is the type of data passed to slots.
type ScopedSlotContext[T any] struct {
	ctx *runtime.Context[*scopedSlotMap[T]]
}

// CreateScopedSlotContext creates a new scoped slot context.
// The generic type T defines what data will be passed to slots.
//
// Example:
//
//	type TableRow struct { ID int; Name string }
//	var tableSlotCtx = CreateScopedSlotContext[TableRow]()
func CreateScopedSlotContext[T any]() *ScopedSlotContext[T] {
	return &ScopedSlotContext[T]{
		ctx: runtime.CreateContext[*scopedSlotMap[T]](nil),
	}
}

// scopedSlotFingerprint creates a structural fingerprint of scoped slots
// based on slot names, function pointers, and child structure. This allows memoization to work
// while detecting when function implementations or slot structure changes.
func scopedSlotFingerprint[T any](children []dom.Item) string {
	temp := dom.FragmentNode()
	for _, child := range children {
		if child != nil {
			child.ApplyTo(temp)
		}
	}

	var parts []string
	for _, node := range temp.Children {
		if node != nil && node.Metadata != nil {
			if name, ok := node.Metadata[scopedSlotNameKey].(string); ok {

				part := name
				if fn, ok := node.Metadata[scopedSlotFuncKey].(func(T) *dom.StructuredNode); ok {
					part += fmt.Sprintf(":%p", fn)
				}

				structHash := nodeStructureHash(node)
				part += ":" + structHash
				parts = append(parts, part)
			}
		}
	}

	return strings.Join(parts, ",")
}

// Provide wraps children in a scoped slot context provider.
// Extracts scoped slot functions from children and makes them available.
// Uses structural fingerprinting to avoid context churn from stable functions.
//
// Example:
//
//	return tableSlotCtx.Provide(ctx, children, func(sctx runtime.Ctx) *dom.StructuredNode {
//	    rows := make([]*dom.StructuredNode, len(props.Rows))
//	    for i, row := range props.Rows {
//	        rows[i] = h.Tr(
//	            h.Td(h.Text(row.Name)),
//	            h.Td(tableSlotCtx.Render(sctx, "actions", row)),
//	        )
//	    }
//	    return h.Table(rows...)
//	})
func (sc *ScopedSlotContext[T]) Provide(
	ctx runtime.Ctx,
	children []dom.Item,
	render func(runtime.Ctx) *dom.StructuredNode,
) *dom.StructuredNode {

	fingerprint := scopedSlotFingerprint[T](children)

	slots := runtime.UseMemo(ctx, func() *scopedSlotMap[T] {
		return extractScopedSlots[T](children)
	}, fingerprint)

	return sc.ctx.Provide(ctx, slots, render)
}

// Render calls the scoped slot function(s) with provided data.
// If multiple functions were provided for the same slot name, calls all of them
// and combines results in a fragment (consistent with regular slot behavior).
// Clones results to prevent shared nodes if functions reuse nodes internally.
// Returns an empty fragment if the slot doesn't exist.
//
// Example:
//
//	tableSlotCtx.Render(ctx, "actions", row)
func (sc *ScopedSlotContext[T]) Render(ctx runtime.Ctx, name string, data T) *dom.StructuredNode {
	slots := sc.ctx.Use(ctx)
	if slots == nil || slots.slots == nil {
		return dom.FragmentNode()
	}

	fns, ok := slots.slots[name]
	if !ok || len(fns) == 0 {
		return dom.FragmentNode()
	}

	results := make([]*dom.StructuredNode, len(fns))
	for i, fn := range fns {
		result := fn(data)

		results[i] = runtime.CloneTree(result)
	}

	if len(results) == 1 {
		return results[0]
	}

	fragment := dom.FragmentNode()
	fragment.Children = results
	return fragment
}

// Has checks if a scoped slot was provided.
//
// Example:
//
//	if tableSlotCtx.Has(ctx, "actions") {
//	    return tableSlotCtx.Render(ctx, "actions", row)
//	}
func (sc *ScopedSlotContext[T]) Has(ctx runtime.Ctx, name string) bool {
	slots := sc.ctx.Use(ctx)
	if slots == nil || slots.slots == nil {
		return false
	}
	_, ok := slots.slots[name]
	return ok
}
