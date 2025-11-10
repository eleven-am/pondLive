package render

import "github.com/eleven-am/pondlive/go/pkg/live/html"

func (b *structuredBuilder) tryKeyedChildren(parent *html.Element, children []html.Node, parentAttrSlot int) bool {
	rows := collectKeyedElements(children)
	if len(rows) == 0 {
		return false
	}
	listSlot := b.addDyn(Dyn{Kind: DynList})
	rowEntries := make([]Row, 0, len(rows))
	for idx, row := range rows {
		if row == nil {
			continue
		}
		start := len(b.dynamics)
		bindingStart := len(b.handlerBindings)
		orderStart := len(b.componentOrder)
		anchorStart := len(b.slotAnchors)
		b.pushChildIndex(idx)
		rowPath := append([]int(nil), b.nodePath...)
		b.visit(row)
		b.popChildIndex()
		end := len(b.dynamics)
		bindingEnd := len(b.handlerBindings)
		orderEnd := len(b.componentOrder)
		anchorEnd := len(b.slotAnchors)
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
		var markers map[string]ComponentMarker
		if orderEnd > orderStart {
			rowParent := append([]int(nil), rowPath...)
			rowIndex := 0
			if len(rowParent) > 0 {
				rowIndex = rowParent[len(rowParent)-1]
				rowParent = rowParent[:len(rowParent)-1]
			}
			markers = make(map[string]ComponentMarker, orderEnd-orderStart)
			for _, id := range b.componentOrder[orderStart:orderEnd] {
				span := b.components[id]
				if len(span.ContainerPath) < len(rowParent)+1 {
					continue
				}
				match := true
				for i, v := range rowParent {
					if span.ContainerPath[i] != v {
						match = false
						break
					}
				}
				if !match {
					continue
				}
				relative := append([]int(nil), span.ContainerPath[len(rowParent):]...)
				if len(relative) == 0 {
					continue
				}
				if relative[0] != rowIndex {
					continue
				}
				relative[0] = 0
				markers[id] = ComponentMarker{
					ContainerPath: relative,
					StartIndex:    span.StartIndex,
					EndIndex:      span.EndIndex,
				}
			}
		}
		var anchors []SlotAnchor
		if anchorEnd > anchorStart {
			anchors = append([]SlotAnchor(nil), b.slotAnchors[anchorStart:anchorEnd]...)
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
	b.recordListAnchor(listSlot)
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
