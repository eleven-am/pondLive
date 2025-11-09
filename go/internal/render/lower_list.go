package render

import (
	"strconv"

	"github.com/eleven-am/pondlive/go/pkg/live/html"
)

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
		markerBase := b.markerCounter
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
		var markers map[string]ComponentMarker
		if orderEnd > orderStart {
			markers = make(map[string]ComponentMarker, orderEnd-orderStart)
			for _, id := range b.componentOrder[orderStart:orderEnd] {
				span := b.components[id]
				markers[id] = ComponentMarker{
					Start: span.MarkerStart - markerBase,
					End:   span.MarkerEnd - markerBase,
				}
			}
		}
		rowEntries = append(rowEntries, Row{
			Key:      row.Key,
			HTML:     renderFinalizedNode(row),
			Slots:    slots,
			Bindings: bindings,
			Markers:  markers,
		})
	}
	if listSlot >= 0 && listSlot < len(b.dynamics) {
		b.dynamics[listSlot].List = rowEntries
	}
	if parent != nil {
		if parent.Attrs == nil {
			parent.Attrs = map[string]string{}
		}
		parent.Attrs["data-list-slot"] = intToString(listSlot)
	}
	if parentAttrSlot >= 0 && parentAttrSlot < len(b.dynamics) {
		attrs := b.dynamics[parentAttrSlot].Attrs
		if attrs == nil {
			attrs = map[string]string{}
		}
		attrs["data-list-slot"] = intToString(listSlot)
		b.dynamics[parentAttrSlot].Attrs = attrs
	}
	return true
}

func intToString(i int) string {
	return strconv.Itoa(i)
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
