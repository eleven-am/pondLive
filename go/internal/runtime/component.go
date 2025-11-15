package runtime

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/eleven-am/pondlive/go/internal/dom"
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
	callable componentCallable
	props    any
	node     dom.Node
	frame    *hookFrame

	children map[string]*component
	rendered map[string]int
	nextIdx  int

	renderEpoch int

	providers map[contextID]any

	sess *ComponentSession
	mu   sync.Mutex

	dirty                  bool
	rendering              int32
	pendingDescendantDirty int32

	handlers []dom.EventHandler
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
		callable: callable,
		props:    props,
		frame:    &hookFrame{},
		children: make(map[string]*component),
		rendered: make(map[string]int),
		sess:     sess,
		dirty:    true,
	}
	c.id = buildComponentID(parent, callable, key)
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
	c.handlers = nil
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
	if !c.shouldRenderLocked() {
		node := c.node
		c.mu.Unlock()
		return node
	}
	defer c.mu.Unlock()
	c.beginRender()
	if c.sess != nil {
		c.sess.beginComponentMetadata(c)
		defer c.sess.finishComponentMetadata(c)
	}
	atomic.StoreInt32(&c.rendering, 1)
	defer atomic.StoreInt32(&c.rendering, 0)
	ctx := Ctx{sess: c.sess, comp: c, frame: c.frame}
	node := c.callable.call(ctx, c.props)
	if carrier, ok := node.(componentMetaResult); ok {
		if sess := c.sess; sess != nil {
			sess.SetMetadata(carrier.Metadata())
		}
		node = carrier.BodyNode()
	}
	node = ensureComponentWrapper(c.id, node)
	if compNode, ok := node.(*dom.ComponentNode); ok {
		compNode.Key = c.key
	}
	c.endRender()
	if atomic.SwapInt32(&c.pendingDescendantDirty, 0) == 1 {
		c.markDescendantsDirtyLocked()
	}
	c.node = node
	c.dirty = false
	return node
}

func (c *component) shouldRenderLocked() bool {
	if c.callable == nil {
		return true
	}
	if c.node == nil {
		return true
	}
	return c.dirty
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
	c.mu.Lock()
	children := make([]*component, 0, len(c.children))
	for _, child := range c.children {
		children = append(children, child)
	}
	c.dirty = true
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

func (c *component) RegisterHandler(handler dom.EventHandler) int {
	if c == nil {
		return -1
	}
	c.mu.Lock()
	defer c.mu.Unlock()
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
