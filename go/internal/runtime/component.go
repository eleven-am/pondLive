package runtime

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// Component defines a function component with props of type P that returns an HTML node.
type Component[P any] func(Ctx, P) h.Node

// componentCallable adapts a generic component function for runtime dispatch.
type componentCallable interface {
	call(Ctx, any) h.Node
	pointer() uintptr
	name() string
}

type componentAdapter[P any] struct {
	fn      Component[P]
	pc      uintptr
	nameStr string
}

type componentMetaResult interface {
	BodyNode() h.Node
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

func (a *componentAdapter[P]) call(ctx Ctx, props any) h.Node {
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
	frame    *hookFrame

	children map[string]*component
	rendered map[string]int
	nextIdx  int

	renderEpoch int

	providers map[contextID]any

	sess *ComponentSession
	mu   sync.Mutex
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
	}
	c.id = buildComponentID(parent, callable, key)
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
	if parent == nil {
		return fmt.Sprintf("root:%s#%s", base, key)
	}
	return fmt.Sprintf("%s/%s#%s", parent.id, base, key)
}

func (c *component) beginRender() {
	if c.frame == nil {
		c.frame = &hookFrame{}
	}
	c.frame.idx = 0
	c.nextIdx = 0
	c.renderEpoch++
	if c.rendered == nil {
		c.rendered = make(map[string]int, len(c.children))
	}
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
		existing.props = props
		existing.callable = callable
		existing.parent = c
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
	}
	c.children[childID] = child
	return child
}

func (c *component) render() h.Node {
	if c.sess != nil {
		c.sess.pushComponent(c)
		defer c.sess.popComponent()
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.beginRender()
	ctx := Ctx{sess: c.sess, comp: c, frame: c.frame}
	node := c.callable.call(ctx, c.props)
	if carrier, ok := node.(componentMetaResult); ok {
		if sess := c.sess; sess != nil {
			current := sess.Metadata()
			sess.SetMetadata(MergeMeta(current, carrier.Metadata()))
		}
		node = carrier.BodyNode()
	}
	c.endRender()
	return node
}

func (c *component) unmount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sess != nil {
		c.sess.releaseUploadSlots(c)
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

// hookFrame tracks per-component hook state.
type hookFrame struct {
	cells []any
	idx   int
}
