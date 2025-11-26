package runtime

import (
	"fmt"
	"reflect"

	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/view"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// convertWorkToView converts a work tree to a view tree.
// Handles components by rendering them, assigns handler IDs, and flattens fragments.
func (s *Session) convertWorkToView(node work.Node, parent *Instance) view.Node {
	if node == nil {
		return nil
	}

	switch n := node.(type) {
	case *work.Element:
		return s.convertElement(n, parent)

	case *work.Text:
		return &view.Text{
			Text: n.Value,
		}

	case *work.Comment:
		return &view.Comment{
			Comment: n.Value,
		}

	case *work.ComponentNode:
		return s.convertComponent(n, parent)

	case *work.Fragment:
		return s.convertFragment(n, parent)

	default:
		return nil
	}
}

// convertElement converts a work.Element to a view.Element.
func (s *Session) convertElement(elem *work.Element, parent *Instance) *view.Element {
	if elem == nil {
		return nil
	}

	viewElem := &view.Element{
		Tag:        elem.Tag,
		Attrs:      elem.Attrs,
		Style:      elem.Style,
		UnsafeHTML: elem.UnsafeHTML,
		Key:        elem.Key,
		RefID:      elem.RefID,
		Script:     elem.Script,
		Stylesheet: elem.Stylesheet,
	}

	if len(elem.Handlers) > 0 {
		viewElem.Handlers = make([]metadata.HandlerMeta, 0, len(elem.Handlers))
		for event, handler := range elem.Handlers {
			handlerID := s.registerHandler(elem, event, handler)
			viewElem.Handlers = append(viewElem.Handlers, metadata.HandlerMeta{
				Event:        event,
				Handler:      handlerID,
				EventOptions: handler.EventOptions,
			})
		}
	}

	if len(elem.Children) > 0 {
		viewElem.Children = make([]view.Node, 0, len(elem.Children))
		for _, child := range elem.Children {
			if viewChild := s.convertWorkToView(child, parent); viewChild != nil {
				viewElem.Children = append(viewElem.Children, viewChild)
			}
		}
	}

	return viewElem
}

// convertComponent converts a component instance's work tree to view tree.
// Renders the component if it's new, props changed, or context changed.
// Skips render if props are unchanged (memoization) and component already has output.
func (s *Session) convertComponent(comp *work.ComponentNode, parent *Instance) view.Node {
	if comp == nil {
		return nil
	}

	inst := parent.EnsureChild(s, comp.Fn, comp.Key, comp.Props, comp.InputChildren)

	needsRender := false

	if inst.WorkTree == nil {
		needsRender = true
	} else if inst.RenderedThisFlush {
		needsRender = false
	} else if !propsEqual(inst.PrevProps, comp.Props) {
		needsRender = true
	} else if inst.ParentContextEpoch != parent.CombinedContextEpoch {
		needsRender = true
	}

	if needsRender {
		s.resetRefsForComponent(inst)
		inst.Render(s)
	}

	return s.convertWorkToView(inst.WorkTree, inst)
}

// propsEqual compares two props values for equality.
// Returns true if props are deeply equal.
func propsEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

// convertFragment flattens a fragment's children into the parent's child list.
func (s *Session) convertFragment(frag *work.Fragment, parent *Instance) view.Node {
	if frag == nil || len(frag.Children) == 0 {
		return nil
	}

	if len(frag.Children) == 1 {
		return s.convertWorkToView(frag.Children[0], parent)
	}

	viewFrag := &view.Fragment{
		Fragment: true,
		Children: make([]view.Node, 0, len(frag.Children)),
	}

	for _, child := range frag.Children {
		if viewChild := s.convertWorkToView(child, parent); viewChild != nil {
			viewFrag.Children = append(viewFrag.Children, viewChild)
		}
	}

	return viewFrag
}

// registerHandler registers a handler using the bus and returns its ID.
// Uses stable IDs (refID:event) when RefID is available, otherwise auto-increments.
func (s *Session) registerHandler(elem *work.Element, event string, handler work.Handler) string {
	var handlerID string

	if elem.RefID != "" {
		handlerID = fmt.Sprintf("%s:%s", elem.RefID, event)
	} else {

		s.handlersMu.Lock()
		s.nextHandlerID++
		handlerID = fmt.Sprintf("h-%d", s.nextHandlerID)
		s.handlersMu.Unlock()
	}

	if s.Bus == nil {
		s.Bus = protocol.NewBus()
	}

	sub := s.Bus.SubscribeToHandlerInvoke(handlerID, func(data interface{}) {
		if event, ok := data.(work.Event); ok {
			handler.Fn(event)
		}
	})

	s.handlerIDsMu.Lock()
	if s.currentHandlerIDs == nil {
		s.currentHandlerIDs = make(map[string]bool)
	}
	if s.allHandlerSubs == nil {
		s.allHandlerSubs = make(map[string]*protocol.Subscription)
	}
	s.currentHandlerIDs[handlerID] = true
	s.allHandlerSubs[handlerID] = sub
	s.handlerIDsMu.Unlock()

	return handlerID
}
