package runtime

import (
	"fmt"
	"reflect"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/view"
	"github.com/eleven-am/pondlive/internal/work"
)

func (s *Session) trackConvertRenderError(inst *Instance) {
	if s == nil || inst == nil || inst.RenderError == nil {
		return
	}
	s.convertRenderErrors = append(s.convertRenderErrors, inst)
}

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

	case *work.PortalNode:
		return s.convertPortalNode(n, parent)

	case *work.PortalTarget:
		return s.convertPortalTarget()

	default:
		return nil
	}
}

func (s *Session) convertPortalNode(portal *work.PortalNode, parent *Instance) view.Node {
	for _, child := range portal.Children {
		if viewChild := s.convertWorkToView(child, parent); viewChild != nil {
			s.PortalViews = append(s.PortalViews, viewChild)
		}
	}
	return nil
}

func (s *Session) convertPortalTarget() view.Node {
	if len(s.PortalViews) == 0 {
		return nil
	}

	if len(s.PortalViews) == 1 {
		node := s.PortalViews[0]
		s.PortalViews = nil
		return node
	}

	frag := &view.Fragment{
		Fragment: true,
		Children: s.PortalViews,
	}
	s.PortalViews = nil
	return frag
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

	if s.flushCtx != nil {
		select {
		case <-s.flushCtx.Done():
			return nil
		default:
		}
	}

	inst := parent.EnsureChild(s, comp.Fn, comp.Key, comp.Props, comp.InputChildren)
	inst.Name = comp.Name
	inst.InputAttrs = comp.InputAttrs
	inst.ParentContextEpoch = parent.CombinedContextEpoch
	inst.CombinedContextEpochs = inst.buildCombinedContextEpochs()

	if inst.errorHandledByBoundary && s.hasAncestorErrorBoundary(inst) {
		propsChanged := !propsEqual(inst.PrevProps, comp.Props)
		if propsChanged {
			inst.mu.Lock()
			inst.errorHandledByBoundary = false
			inst.mu.Unlock()
		} else {
			return nil
		}
	}

	needsRender := false

	if inst.WorkTree == nil {
		needsRender = true
	} else if inst.RenderedThisFlush {
		needsRender = false
	} else if inst.Dirty {
		needsRender = true
	} else if !propsEqual(inst.PrevProps, comp.Props) {
		needsRender = true
	} else if inputChildrenChanged(inst.PrevInputChildren, comp.InputChildren) {
		needsRender = true
	} else if childNeedsContextRender(inst, parent) {
		needsRender = true
	}

	if needsRender {
		s.resetRefsForComponent(inst)
		inst.Render(s)
		inst.snapshotContextDeps(parent)

		if inst.RenderError != nil {
			s.trackConvertRenderError(inst)
		}
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

func inputChildrenChanged(prev, curr []work.Node) bool {
	if len(prev) != len(curr) {
		return true
	}

	for i := range prev {
		if nodeChanged(prev[i], curr[i]) {
			return true
		}
	}

	return false
}

func nodeChanged(prev, curr work.Node) bool {
	if prev == nil && curr == nil {
		return false
	}
	if prev == nil || curr == nil {
		return true
	}

	switch p := prev.(type) {
	case *work.ComponentNode:
		c, ok := curr.(*work.ComponentNode)
		if !ok {
			return true
		}
		if p.Key != c.Key {
			return true
		}
		if !depsValueEqual(p.Props, c.Props) {
			return true
		}
		if inputChildrenChanged(p.InputChildren, c.InputChildren) {
			return true
		}

	case *work.Element:
		c, ok := curr.(*work.Element)
		if !ok {
			return true
		}
		if p.Tag != c.Tag || p.Key != c.Key {
			return true
		}
		if !reflect.DeepEqual(p.Attrs, c.Attrs) {
			return true
		}
		if !reflect.DeepEqual(p.Style, c.Style) {
			return true
		}
		if p.UnsafeHTML != c.UnsafeHTML {
			return true
		}
		if inputChildrenChanged(p.Children, c.Children) {
			return true
		}

	case *work.Text:
		c, ok := curr.(*work.Text)
		if !ok {
			return true
		}
		if p.Value != c.Value {
			return true
		}

	case *work.Comment:
		c, ok := curr.(*work.Comment)
		if !ok {
			return true
		}
		if p.Value != c.Value {
			return true
		}

	case *work.Fragment:
		c, ok := curr.(*work.Fragment)
		if !ok {
			return true
		}
		if inputChildrenChanged(p.Children, c.Children) {
			return true
		}

	default:
		return !reflect.DeepEqual(prev, curr)
	}

	return false
}

func childNeedsContextRender(child *Instance, parent *Instance) bool {
	if child == nil || parent == nil {
		return false
	}

	if len(child.ContextDeps) == 0 {
		return false
	}

	parentCombined := parent.CombinedContextEpochs
	for id := range child.ContextDeps {
		current := 0
		if parentCombined != nil {
			if v, ok := parentCombined[id]; ok {
				current = v
			}
		}

		if child.SeenContextEpochs == nil {
			return true
		}

		if child.SeenContextEpochs[id] != current {
			return true
		}
	}

	return false
}

func (inst *Instance) snapshotContextDeps(parent *Instance) {
	if inst == nil || len(inst.ContextDeps) == 0 {
		return
	}

	if inst.SeenContextEpochs == nil {
		inst.SeenContextEpochs = make(map[contextID]int, len(inst.ContextDeps))
	}

	var source map[contextID]int
	if parent != nil {
		source = parent.CombinedContextEpochs
	}

	for id := range inst.ContextDeps {
		if source != nil {
			inst.SeenContextEpochs[id] = source[id]
		} else {
			inst.SeenContextEpochs[id] = 0
		}
	}
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

	if len(viewFrag.Children) == 0 {
		return nil
	}

	if len(viewFrag.Children) == 1 {
		return viewFrag.Children[0]
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

	s.handlerIDsMu.Lock()
	if s.currentHandlerIDs == nil {
		s.currentHandlerIDs = make(map[string]bool)
	}
	if s.allHandlerSubs == nil {
		s.allHandlerSubs = make(map[string]*protocol.Subscription)
	}

	sub := s.Bus.SubscribeToHandlerInvoke(handlerID, func(data interface{}) {
		var event work.Event
		if payload, ok := data.(map[string]any); ok {
			event.Payload = payload
		} else if payload, ok := data.(map[string]interface{}); ok {
			event.Payload = payload
		} else {
			return
		}

		handler.Fn(event)
	})

	s.currentHandlerIDs[handlerID] = true
	s.allHandlerSubs[handlerID] = sub
	s.handlerIDsMu.Unlock()

	return handlerID
}
