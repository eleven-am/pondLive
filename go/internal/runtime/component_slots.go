package runtime

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/render"
)

// childSlot holds the DOM wrapper and structured snapshot for a child
// component. The parent uses these slots to patch the DOM and structured
// tree without re-running its own render function.
type childSlot struct {
	component  *component
	wrapper    *dom.ComponentNode
	structured render.Structured
}

func (c *component) ensureSlot(child *component, wrapper *dom.ComponentNode) *childSlot {
	if c.slots == nil {
		c.slots = make(map[string]*childSlot)
	}
	slot := c.slots[child.id]
	if slot == nil {
		slot = &childSlot{}
		c.slots[child.id] = slot
	}
	slot.component = child
	slot.wrapper = wrapper
	return slot
}

func (slot *childSlot) applyDOMPatch(newNode dom.Node) {
	if slot == nil {
		return
	}
	if slot.wrapper == nil {
		if comp, ok := newNode.(*dom.ComponentNode); ok {
			slot.wrapper = comp
		} else {
			slot.wrapper = dom.WrapComponent("", newNode)
		}
	}

	if comp, ok := newNode.(*dom.ComponentNode); ok {
		if comp == slot.wrapper {

		} else {

			slot.wrapper.Child = comp.Child
			slot.wrapper.Key = comp.Key
			slot.wrapper.ID = comp.ID
		}
	} else {
		slot.wrapper.Child = newNode
	}

	if slot.component != nil {
		slot.structured = slot.component.lastStructured
	}
}
