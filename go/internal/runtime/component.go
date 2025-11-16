package runtime

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/render"
)

// Component defines a function component with props of type P that returns an HTML node.
type Component[P any] func(Ctx, P) dom.Node

// componentCallable adapts a generic component function for runtime dispatch.
type componentCallable interface {
	call(Ctx, any) dom.Node
	pointer() uintptr
	name() string
}

type componentAdapter[P any] struct {
	fn      Component[P]
	pc      uintptr
	nameStr string
}

type componentMetaResult interface {
	BodyNode() dom.Node
	Metadata() *Meta
}

func newComponentAdapter[P any](fn Component[P]) componentCallable {
	pc := reflect.ValueOf(fn).Pointer()
	name := "<component>"
	if pc != 0 {
		if fnInfo := runtime.FuncForPC(pc); fnInfo != nil {
			name = fnInfo.Name()
		}
	}
	return &componentAdapter[P]{fn: fn, pc: pc, nameStr: name}
}

func (a *componentAdapter[P]) call(ctx Ctx, props any) dom.Node {
	if ctx.sess == nil {
		panic("runtime: component context missing session")
	}
	value, ok := props.(P)
	if !ok {

		if ptr, okPtr := props.(*P); okPtr {
			value = *ptr
		} else {
			panic(fmt.Sprintf("runtime: props type mismatch for component %s", a.name()))
		}
	}
	return a.fn(ctx, value)
}

func (a *componentAdapter[P]) pointer() uintptr { return a.pc }

func (a *componentAdapter[P]) name() string { return a.nameStr }

// component represents a mounted component instance.
type component struct {
	id       string
	key      string
	parent   *component
	depth    int
	callable componentCallable
	props    any
	node     dom.Node
	frame    *hookFrame

	children map[string]*component
	rendered map[string]int
	nextIdx  int
	slots    map[string]*childSlot

	renderEpoch int

	providers map[contextID]any

	sess *ComponentSession
	mu   sync.Mutex

	dirty                  bool
	wasDirtyBeforeRender   bool
	forcedRender           bool // Set when rendering is forced but component wasn't dirty
	rendering              int32
	pendingDescendantDirty int32

	lastStructured render.Structured
	spanIndex      map[string]render.ComponentSpan

	handlers    []dom.EventHandler
	newHandlers []dom.EventHandler // Built during render, swapped atomically after
}

type attachmentResetter interface {
	resetAttachment()
}

func newComponent[P any](sess *ComponentSession, parent *component, key string, fn Component[P], props P) *component {
	adapter := newComponentAdapter(fn)
	return newComponentWithCallable(sess, parent, key, adapter, props)
}

func newComponentWithCallable(sess *ComponentSession, parent *component, key string, callable componentCallable, props any) *component {
	if callable == nil {
		panic("runtime: nil component callable")
	}
	c := &component{
		key:      key,
		parent:   parent,
		depth:    0,
		callable: callable,
		props:    props,
		frame:    &hookFrame{},
		children: make(map[string]*component),
		rendered: make(map[string]int),
		sess:     sess,
		dirty:    true,
	}
	c.id = buildComponentID(parent, callable, key)
	if parent != nil {
		c.depth = parent.depth + 1
	}
	if sess != nil {
		sess.registerComponentInstance(c)
	}
	return c
}

func buildComponentID(parent *component, callable componentCallable, key string) string {
	base := callable.name()
	if base == "" {
		base = fmt.Sprintf("pc:%x", callable.pointer())
	}
	if key == "" {
		key = "_"
	}

	hasher := sha256.New()
	if parent == nil {
		hasher.Write([]byte("root"))
	} else {
		hasher.Write([]byte(parent.id))
	}
	hasher.Write([]byte{0})
	hasher.Write([]byte(base))
	hasher.Write([]byte{0})
	hasher.Write([]byte(key))

	sum := hasher.Sum(nil)
	return fmt.Sprintf("c%x", sum[:12])
}

func (c *component) beginRender() {
	if c.frame == nil {
		c.frame = &hookFrame{}
	}
	for _, cell := range c.frame.cells {
		if resetter, ok := cell.(interface{ resetAttachment() }); ok {
			resetter.resetAttachment()
		}
	}
	c.frame.idx = 0
	c.nextIdx = 0
	c.renderEpoch++
	if c.rendered == nil {
		c.rendered = make(map[string]int, len(c.children))
	}
	if len(c.frame.cells) > 0 {
		for _, cell := range c.frame.cells {
			if resetter, ok := cell.(attachmentResetter); ok {
				resetter.resetAttachment()
			}
		}
	}

	c.newHandlers = nil
}

func (c *component) endRender() {
	if len(c.children) == 0 {
		return
	}
	for id, child := range c.children {
		if epoch, ok := c.rendered[id]; ok && epoch == c.renderEpoch {
			continue
		}
		child.unmount()
		delete(c.children, id)
		delete(c.rendered, id)
	}
}

func (c *component) ensureChild(callable componentCallable, key string, props any) *component {
	if callable == nil {
		panic("runtime: nil component callable")
	}
	if key == "" {
		key = fmt.Sprintf("%d", c.nextIdx)
	}
	c.nextIdx++
	childID := buildComponentID(c, callable, key)
	if epoch, seen := c.rendered[childID]; seen && epoch == c.renderEpoch {
		panic(fmt.Sprintf("runtime: duplicate child key %q in component %s", key, c.callable.name()))
	}
	c.rendered[childID] = c.renderEpoch
	if existing, ok := c.children[childID]; ok {
		existing.parent = c
		shouldRender := !callableEqual(existing.callable, callable) || !propsEqual(existing.props, props)
		existing.callable = callable
		if shouldRender {
			existing.markSelfDirty()
		}
		existing.props = props
		return existing
	}
	child := &component{
		id:       childID,
		key:      key,
		parent:   c,
		depth:    c.depth + 1,
		callable: callable,
		props:    props,
		frame:    &hookFrame{},
		children: make(map[string]*component),
		rendered: make(map[string]int),
		sess:     c.sess,
		dirty:    true,
	}
	c.children[childID] = child
	return child
}

func (c *component) render() dom.Node {
	if c.sess != nil {
		c.sess.pushComponent(c)
		defer c.sess.popComponent()
	}
	c.mu.Lock()

	if c.forcedRender && !c.dirty {
		node := c.node
		c.forcedRender = false
		c.mu.Unlock()
		return node
	}
	if !c.shouldRenderLocked() {
		node := c.node
		c.mu.Unlock()
		return node
	}

	c.wasDirtyBeforeRender = c.dirty
	defer func() {
		c.forcedRender = false
		c.mu.Unlock()
	}()

	atomic.StoreInt32(&c.rendering, 1)
	defer atomic.StoreInt32(&c.rendering, 0)

	c.beginRender()
	if c.sess != nil {
		c.sess.beginComponentMetadata(c)
		defer c.sess.finishComponentMetadata(c)
	}
	ctx := Ctx{sess: c.sess, comp: c, frame: c.frame}
	node := c.callable.call(ctx, c.props)
	if carrier, ok := node.(componentMetaResult); ok {
		if sess := c.sess; sess != nil {
			sess.SetMetadata(carrier.Metadata())
		}
		node = carrier.BodyNode()
	}
	wrapper := c.wrapComponentNode(node)
	c.endRender()
	if atomic.SwapInt32(&c.pendingDescendantDirty, 0) == 1 {
		c.markDescendantsDirtyLocked()
	}

	if c.newHandlers != nil {
		c.handlers = c.newHandlers
		c.newHandlers = nil
	}

	c.node = wrapper
	c.captureChildSlots(wrapper)
	c.patchParentSlot(wrapper)
	c.dirty = false
	return wrapper
}

func (c *component) shouldRenderLocked() bool {
	if c.callable == nil {
		return true
	}
	if c.node == nil {
		return true
	}
	if c.dirty {
		return true
	}
	if c.forcedRender {
		return true
	}

	if c.parent != nil && atomic.LoadInt32(&c.parent.rendering) == 1 && c.parent.wasDirtyBeforeRender && !c.parent.forcedRender {
		return true
	}
	return false
}

func (c *component) shouldRender() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.shouldRenderLocked()
}

func (c *component) markDirtyChain() {
	for cur := c; cur != nil; {
		cur.mu.Lock()
		next := cur.parent
		cur.dirty = true
		cur.mu.Unlock()
		cur = next
	}
}

func (c *component) markSelfDirty() {
	if c == nil {
		return
	}
	c.mu.Lock()
	c.dirty = true
	c.mu.Unlock()
}

func (c *component) markClean() {
	c.mu.Lock()
	c.dirty = false
	c.mu.Unlock()
}

func (c *component) markDescendantsDirty() {
	if c == nil {
		return
	}
	c.mu.Lock()
	children := make([]*component, 0, len(c.children))
	for _, child := range c.children {
		children = append(children, child)
	}
	c.mu.Unlock()
	for _, child := range children {
		child.markSubtreeDirty()
	}
}

func (c *component) markDescendantsDirtyLocked() {
	if c == nil {
		return
	}
	children := make([]*component, 0, len(c.children))
	for _, child := range c.children {
		children = append(children, child)
	}
	for _, child := range children {
		if atomic.LoadInt32(&child.rendering) == 1 {
			atomic.StoreInt32(&child.pendingDescendantDirty, 1)
			continue
		}
		child.markSubtreeDirty()
	}
}

func (c *component) markSubtreeDirty() {
	if c == nil {
		return
	}
	if c.sess != nil {
		c.sess.markDirty(c)
	} else {
		c.markSelfDirty()
	}
	c.mu.Lock()
	children := make([]*component, 0, len(c.children))
	for _, child := range c.children {
		children = append(children, child)
	}
	c.mu.Unlock()
	for _, child := range children {
		child.markSubtreeDirty()
	}
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

func callableEqual(a, b componentCallable) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.pointer() == b.pointer()
}

func (c *component) unmount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sess != nil {
		c.sess.unregisterComponentInstance(c)
		c.sess.releaseUploadSlots(c)
		c.sess.removeMetadataForComponent(c)
	}
	for _, child := range c.children {
		child.unmount()
	}
	c.children = nil
	c.rendered = nil
	c.providers = nil
	if c.frame != nil {
		for _, cell := range c.frame.cells {
			if eff, ok := cell.(*effectCell); ok {
				if eff.cleanup != nil {
					eff.cleanup()
					eff.cleanup = nil
				}
			}
		}
	}
}

func ensureComponentWrapper(id string, node dom.Node) dom.Node {
	if existing, ok := node.(*dom.ComponentNode); ok {
		if existing.ID == id {
			return existing
		}

		node = existing.Child
	}
	return dom.WrapComponent(id, node)
}

func (c *component) captureChildSlots(node dom.Node) {
	if len(c.children) == 0 {
		c.slots = nil
		return
	}
	if c.slots == nil {
		c.slots = make(map[string]*childSlot, len(c.children))
	} else {
		for id := range c.slots {
			delete(c.slots, id)
		}
	}
	wrapperByID := make(map[string]*dom.ComponentNode, len(c.children))
	collectComponentNodes(node, func(n *dom.ComponentNode) {
		if n == nil || n.ID == "" || n.ID == c.id {
			return
		}
		if _, exists := wrapperByID[n.ID]; !exists {
			wrapperByID[n.ID] = n
		}
	})
	for id, child := range c.children {
		wrapper := wrapperByID[id]
		if wrapper == nil {
			continue
		}
		slot := c.slots[id]
		if slot == nil {
			slot = &childSlot{}
			c.slots[id] = slot
		}
		slot.component = child
		slot.wrapper = wrapper
	}
}

func collectComponentNodes(node dom.Node, fn func(*dom.ComponentNode)) {
	switch v := node.(type) {
	case *dom.ComponentNode:
		fn(v)
		if v.Child != nil {
			collectComponentNodes(v.Child, fn)
		}
	case *dom.Element:
		for _, child := range v.Children {
			collectComponentNodes(child, fn)
		}
	case *dom.FragmentNode:
		for _, child := range v.Children {
			collectComponentNodes(child, fn)
		}
	}
}

func (c *component) parentSlot() *childSlot {
	if c == nil || c.parent == nil {
		return nil
	}
	if c.parent.slots == nil {
		return nil
	}
	return c.parent.slots[c.id]
}

func (c *component) wrapComponentNode(node dom.Node) *dom.ComponentNode {
	if existing, ok := c.node.(*dom.ComponentNode); ok && existing.ID == c.id {

		existing.Child = node
		existing.Key = c.key
		return existing
	}
	if comp, ok := node.(*dom.ComponentNode); ok && comp.ID == c.id {
		comp.Key = c.key
		return comp
	}

	wrapped := dom.WrapComponent(c.id, node)
	wrapped.Key = c.key
	return wrapped
}

func (c *component) patchParentSlot(node dom.Node) {
	slot := c.parentSlot()
	if slot == nil {
		return
	}
	if compNode, ok := node.(*dom.ComponentNode); ok {
		slot.component = c
		slot.applyDOMPatch(compNode)
	} else {
		slot.component = c
		slot.applyDOMPatch(node)
	}

	if c.parent != nil && c.lastStructured.S != nil {
		c.parent.patchChildSpan(c.id, c.lastStructured)
	}
}

func unwrapComponentChild(node dom.Node) dom.Node {
	if comp, ok := node.(*dom.ComponentNode); ok {
		return comp.Child
	}
	return node
}

func cloneStructured(s render.Structured) render.Structured {
	result := render.Structured{}

	if s.S != nil {
		result.S = make([]string, len(s.S))
		copy(result.S, s.S)
	}

	if s.D != nil {
		result.D = make([]render.DynamicSlot, len(s.D))
		copy(result.D, s.D)
	}

	if s.Components != nil {
		result.Components = make(map[string]render.ComponentSpan, len(s.Components))
		for k, v := range s.Components {
			result.Components[k] = v
		}
	}

	if s.Bindings != nil {
		result.Bindings = make([]render.HandlerBinding, len(s.Bindings))
		copy(result.Bindings, s.Bindings)
	}

	if s.UploadBindings != nil {
		result.UploadBindings = make([]render.UploadBinding, len(s.UploadBindings))
		copy(result.UploadBindings, s.UploadBindings)
	}

	if s.RefBindings != nil {
		result.RefBindings = make([]render.RefBinding, len(s.RefBindings))
		copy(result.RefBindings, s.RefBindings)
	}

	if s.RouterBindings != nil {
		result.RouterBindings = make([]render.RouterBinding, len(s.RouterBindings))
		copy(result.RouterBindings, s.RouterBindings)
	}

	if s.SlotPaths != nil {
		result.SlotPaths = make([]render.SlotPath, len(s.SlotPaths))
		copy(result.SlotPaths, s.SlotPaths)
	}

	if s.ListPaths != nil {
		result.ListPaths = make([]render.ListPath, len(s.ListPaths))
		copy(result.ListPaths, s.ListPaths)
	}

	if s.ComponentPaths != nil {
		result.ComponentPaths = make([]render.ComponentPath, len(s.ComponentPaths))
		copy(result.ComponentPaths, s.ComponentPaths)
	}

	return result
}

func (c *component) assignStructuredSnapshot(frame render.Structured) {
	if c == nil {
		return
	}
	c.lastStructured = cloneStructured(frame)
	c.buildSpanIndex(frame)
	c.updateSlotStructured()
}

func (c *component) buildSpanIndex(frame render.Structured) {
	if c.spanIndex == nil {
		c.spanIndex = make(map[string]render.ComponentSpan)
	}
	for id := range c.spanIndex {
		delete(c.spanIndex, id)
	}
	for id, span := range frame.Components {
		c.spanIndex[id] = span
	}
}

func (c *component) updateSlotStructured() {
	if len(c.slots) == 0 {
		return
	}
	for id, slot := range c.slots {
		if slot == nil {
			continue
		}
		child := slot.component
		if child == nil {
			child = c.children[id]
			slot.component = child
		}
		if child == nil {
			continue
		}
		childStructured := child.lastStructured
		slot.structured = childStructured
		c.patchChildSpan(id, childStructured)
	}
}

func (c *component) patchChildSpan(childID string, childStructured render.Structured) {
	if c == nil || c.spanIndex == nil {
		return
	}
	_span, ok := c.spanIndex[childID]
	if !ok {
		return
	}
	span := _span

	if span.StaticsStart >= 0 && span.StaticsEnd <= len(c.lastStructured.S) {
		before := append([]string(nil), c.lastStructured.S[:span.StaticsStart]...)
		after := append([]string(nil), c.lastStructured.S[span.StaticsEnd:]...)
		c.lastStructured.S = append(before, childStructured.S...)
		c.lastStructured.S = append(c.lastStructured.S, after...)
		span.StaticsEnd = span.StaticsStart + len(childStructured.S)
	}

	if span.DynamicsStart >= 0 && span.DynamicsEnd <= len(c.lastStructured.D) {
		before := append([]render.DynamicSlot(nil), c.lastStructured.D[:span.DynamicsStart]...)
		after := append([]render.DynamicSlot(nil), c.lastStructured.D[span.DynamicsEnd:]...)
		c.lastStructured.D = append(before, childStructured.D...)
		c.lastStructured.D = append(c.lastStructured.D, after...)
		span.DynamicsEnd = span.DynamicsStart + len(childStructured.D)
	}

	c.spanIndex[childID] = span
}

func (c *component) RegisterHandler(handler dom.EventHandler) int {
	if c == nil {
		return -1
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if atomic.LoadInt32(&c.rendering) == 1 {
		slotIdx := len(c.newHandlers)
		c.newHandlers = append(c.newHandlers, handler)
		return slotIdx
	}

	slotIdx := len(c.handlers)
	c.handlers = append(c.handlers, handler)
	return slotIdx
}

func (c *component) ComponentID() string {
	if c == nil {
		return ""
	}
	return c.id
}

// hookFrame tracks per-component hook state.
type hookFrame struct {
	cells []any
	idx   int
}
