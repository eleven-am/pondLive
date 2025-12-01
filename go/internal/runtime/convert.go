package runtime

import (
	"fmt"
	"reflect"

	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/view"
	"github.com/eleven-am/pondlive/go/internal/work"
)

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
			handlerID := s.registerHandler(parent, elem, event, handler)
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
	} else if inst.Dirty {
		needsRender = true
	} else if !propsEqual(inst.PrevProps, comp.Props) {
		needsRender = true
	} else if inst.ParentContextEpoch != parent.CombinedContextEpoch {
		needsRender = true
	}

	if needsRender {
		s.resetRefsForComponent(inst)
		inst.Render(s)
	}

	inst.NextHandlerIndex = 0
	return s.convertWorkToView(inst.WorkTree, inst)
}

func propsEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a, b)
}

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

func (s *Session) registerHandler(inst *Instance, elem *work.Element, event string, handler work.Handler) string {
	var handlerID string

	if elem.RefID != "" {
		handlerID = fmt.Sprintf("%s:%s", elem.RefID, event)
	} else {
		handlerID = fmt.Sprintf("%s:h%d", inst.ID, inst.NextHandlerIndex)
		inst.NextHandlerIndex++
	}

	if s.Bus == nil {
		s.Bus = protocol.NewBus()
	}

	sub := s.Bus.SubscribeToHandlerInvoke(handlerID, func(data interface{}) {
		event, ok := protocol.DecodePayload[work.Event](data)
		if !ok {
			return
		}

		handler.Fn(event)
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
