package slot

import (
	"fmt"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

// extractSlots parses children items and organizes them by slot name.
// - Nodes with "slot:name" metadata go to named slots
// - Regular nodes go to "default" slot
// - Multiple nodes with same slot name are appended together
func extractSlots(children []dom.Item) *SlotMap {
	temp := dom.FragmentNode()
	for _, child := range children {
		if child != nil {
			child.ApplyTo(temp)
		}
	}

	slots := make(map[string]*SlotContent)
	var slotOrder []string
	var defaultContent []*dom.StructuredNode
	seenDefault := false

	for _, node := range temp.Children {
		if node == nil {
			continue
		}

		if node.Metadata != nil {

			hasRegular := node.Metadata[slotNameKey] != nil
			hasScoped := node.Metadata[scopedSlotNameKey] != nil
			if hasRegular && hasScoped {
				panic(fmt.Sprintf("slot: node has conflicting regular and scoped slot markers (found both %q and %q keys)", slotNameKey, scopedSlotNameKey))
			}

			if slotName, ok := node.Metadata[slotNameKey].(string); ok {

				slotNodes := node.Children

				if existing := slots[slotName]; existing != nil {
					existing.nodes = append(existing.nodes, slotNodes...)
				} else {
					slots[slotName] = &SlotContent{nodes: slotNodes}
					slotOrder = append(slotOrder, slotName)
				}
				continue
			}
		}

		if !seenDefault {
			seenDefault = true
		}
		defaultContent = append(defaultContent, node)
	}

	if len(defaultContent) > 0 {
		slots["default"] = &SlotContent{nodes: defaultContent}

		if seenDefault {
			slotOrder = append(slotOrder, "default")
		}
	}

	return &SlotMap{
		slots:     slots,
		fallbacks: make(map[string]*SlotContent),
		slotOrder: slotOrder,
	}
}

// extractScopedSlots parses children items and extracts scoped slot functions.
// Only processes nodes marked with "scoped-slot:name" metadata.
func extractScopedSlots[T any](children []dom.Item) *scopedSlotMap[T] {
	temp := dom.FragmentNode()
	for _, child := range children {
		if child != nil {
			child.ApplyTo(temp)
		}
	}

	slots := make(map[string][]func(T) *dom.StructuredNode)

	for _, node := range temp.Children {
		if node == nil || node.Metadata == nil {
			continue
		}

		hasRegular := node.Metadata[slotNameKey] != nil
		hasScoped := node.Metadata[scopedSlotNameKey] != nil
		if hasRegular && hasScoped {
			panic(fmt.Sprintf("slot: node has conflicting regular and scoped slot markers (found both %q and %q keys)", slotNameKey, scopedSlotNameKey))
		}

		if slotName, ok := node.Metadata[scopedSlotNameKey].(string); ok {
			if fn, ok := node.Metadata[scopedSlotFuncKey].(func(T) *dom.StructuredNode); ok {
				slots[slotName] = append(slots[slotName], fn)
			}
		}
	}

	return &scopedSlotMap[T]{slots: slots}
}
