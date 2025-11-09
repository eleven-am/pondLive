package render

import (
	"github.com/eleven-am/pondlive/go/pkg/live/html"
)

func (b *structuredBuilder) tryKeyedChildren(parent *html.Element, children []html.Node, parentAttrSlot int) bool {
	rows := collectKeyedElements(children)
	if len(rows) == 0 {
		return false
	}
	listSlot := b.addDyn(Dyn{Kind: DynList})
	if parent != nil {
		if path := b.elementPaths[parent]; len(path) > 0 {
			b.registerSlotAnchor(listSlot, path)
		}
	}
	rowEntries := make([]Row, 0, len(rows))
	for idx, row := range rows {
		if row == nil {
			continue
		}
		start := len(b.dynamics)
		bindingStart := len(b.handlerBindings)
		orderStart := len(b.componentOrder)
		b.pushChildIndex(idx)
		b.visit(row)
		b.popChildIndex()
		end := len(b.dynamics)
		bindingEnd := len(b.handlerBindings)
		orderEnd := len(b.componentOrder)
		if end <= start {
			rowEntries = append(rowEntries, Row{Key: row.Key})
			continue
		}
		slots := make([]int, 0, end-start)
		for i := start; i < end; i++ {
			slots = append(slots, i)
		}
		var bindings []HandlerBinding
		if bindingEnd > bindingStart {
			bindings = append([]HandlerBinding(nil), b.handlerBindings[bindingStart:bindingEnd]...)
		}
		var markers map[string]ComponentBoundary
		if orderEnd > orderStart {
			markers = make(map[string]ComponentBoundary, orderEnd-orderStart)
			rowPath := b.elementPaths[row]
			for _, id := range b.componentOrder[orderStart:orderEnd] {
				span := b.components[id]
				if boundary, ok := relativeComponentBoundary(span.Boundary, rowPath); ok {
					markers[id] = boundary
				}
			}
		}
		var anchors map[int]NodeAnchor
		if len(slots) > 0 {
			rowPath := b.elementPaths[row]
			for _, slot := range slots {
				anchor, ok := b.slotAnchors[slot]
				if !ok {
					continue
				}
				relative, ok := RelativeNodeAnchor(anchor, rowPath)
				if !ok {
					continue
				}
				if anchors == nil {
					anchors = make(map[int]NodeAnchor)
				}
				anchors[slot] = relative
			}
		}
		rowEntries = append(rowEntries, Row{
			Key:      row.Key,
			HTML:     renderFinalizedNode(row),
			Slots:    slots,
			Bindings: bindings,
			Markers:  markers,
			Anchors:  anchors,
		})
	}
	if listSlot >= 0 && listSlot < len(b.dynamics) {
		b.dynamics[listSlot].List = rowEntries
	}
	return true
}

func collectKeyedElements(children []html.Node) []*html.Element {
	rows := make([]*html.Element, 0, len(children))
	for _, child := range children {
		if child == nil {
			continue
		}
		el, ok := child.(*html.Element)
		if !ok || el.Key == "" {
			return nil
		}
		rows = append(rows, el)
	}
	if len(rows) == 0 {
		return nil
	}
	return rows
}

func relativeComponentBoundary(boundary ComponentBoundary, rowPath []int) (ComponentBoundary, bool) {
	if len(rowPath) == 0 {
		return ComponentBoundary{
			ParentPath: append([]int(nil), boundary.ParentPath...),
			StartIndex: boundary.StartIndex,
			EndIndex:   boundary.EndIndex,
		}, true
	}
	if len(boundary.ParentPath) < len(rowPath) {
		return ComponentBoundary{}, false
	}
	for i := range rowPath {
		if boundary.ParentPath[i] != rowPath[i] {
			return ComponentBoundary{}, false
		}
	}
	rel := make([]int, 1+len(boundary.ParentPath)-len(rowPath))
	rel[0] = 0
	copy(rel[1:], boundary.ParentPath[len(rowPath):])
	return ComponentBoundary{
		ParentPath: rel,
		StartIndex: boundary.StartIndex,
		EndIndex:   boundary.EndIndex,
	}, true
}
