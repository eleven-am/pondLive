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
		slotPathStart := len(b.slotPaths)
		listPathStart := len(b.listPaths)
		componentPathStart := len(b.componentPaths)
		b.pushChildIndex(idx)
		b.visit(row)
		b.popChildIndex()
		end := len(b.dynamics)
		bindingEnd := len(b.handlerBindings)
		slotPathEnd := len(b.slotPaths)
		listPathEnd := len(b.listPaths)
		componentPathEnd := len(b.componentPaths)
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
		var slotPaths []SlotPath
		if slotPathEnd > slotPathStart {
			slotPaths = append([]SlotPath(nil), b.slotPaths[slotPathStart:slotPathEnd]...)
		}
		var listPaths []ListPath
		if listPathEnd > listPathStart {
			listPaths = append([]ListPath(nil), b.listPaths[listPathStart:listPathEnd]...)
		}
		var componentPaths []ComponentPath
		if componentPathEnd > componentPathStart {
			componentPaths = append([]ComponentPath(nil), b.componentPaths[componentPathStart:componentPathEnd]...)
		}
		rowEntries = append(rowEntries, Row{
			Key:            row.Key,
			HTML:           renderFinalizedNode(row),
			Slots:          slots,
			Bindings:       bindings,
			SlotPaths:      slotPaths,
			ListPaths:      listPaths,
			ComponentPaths: componentPaths,
		})
	}
	if listSlot >= 0 && listSlot < len(b.dynamics) {
		b.dynamics[listSlot].List = rowEntries
	}
	if parent != nil && len(b.stack) > 0 {
		frame := b.stack[len(b.stack)-1]
		if frame.componentID != "" {
			b.listPaths = append(b.listPaths, ListPath{
				Slot:        listSlot,
				ComponentID: frame.componentID,
				ElementPath: append([]int(nil), frame.componentPath...),
			})
		}
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
