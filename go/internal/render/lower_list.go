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
	for _, row := range rows {
		if row == nil {
			continue
		}
		start := len(b.dynamics)
		b.visit(row)
		end := len(b.dynamics)
		if end <= start {
			rowEntries = append(rowEntries, Row{Key: row.Key})
			continue
		}
		slots := make([]int, 0, end-start)
		for i := start; i < end; i++ {
			slots = append(slots, i)
		}
		rowEntries = append(rowEntries, Row{
			Key:   row.Key,
			HTML:  renderFinalizedNode(row),
			Slots: slots,
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
