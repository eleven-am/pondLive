package runtime

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/dom2"
)

// Component is a function that renders a node tree.
type Component[P any] func(Ctx, P) *dom2.StructuredNode

type component struct {
	id       string
	sess     *ComponentSession
	parent   *component
	children map[string]*component

	callable           componentCallable
	props              any
	prevProps          any
	parentContextEpoch int

	frame                *hookFrame
	node                 *dom2.StructuredNode
	wrapper              *dom2.StructuredNode
	dirty                bool
	providers            map[contextID]any
	contextEpoch         int
	providerSeq          int
	combinedContextEpoch int

	renderedThisFlush bool

	mu sync.Mutex
}

type componentCallable interface {
	call(Ctx, any) *dom2.StructuredNode
	name() string
	hash() string
}

type componentAdapter[P any] struct {
	fn      Component[P]
	hashVal string
}

func newComponentAdapter[P any](fn Component[P]) *componentAdapter[P] {
	return &componentAdapter[P]{
		fn:      fn,
		hashVal: computeHash(fn),
	}
}

func (a *componentAdapter[P]) call(ctx Ctx, props any) *dom2.StructuredNode {
	p, ok := props.(P)
	if !ok {
		var zero P
		p = zero
	}
	return a.fn(ctx, p)
}

func (a *componentAdapter[P]) name() string {
	fnType := reflect.TypeOf(a.fn)
	if fnType.Kind() == reflect.Func {
		return fnType.String()
	}
	return "Component"
}

func (a *componentAdapter[P]) hash() string {
	return a.hashVal
}

func computeHash(fn any) string {
	ptr := reflect.ValueOf(fn).Pointer()
	h := sha256.Sum256([]byte(fmt.Sprintf("%d", ptr)))
	return fmt.Sprintf("%x", h[:8])
}

func newComponent[P any](sess *ComponentSession, parent *component, key string, fn Component[P], props P) *component {
	adapter := newComponentAdapter(fn)
	id := adapter.hash()
	if key != "" {
		id = fmt.Sprintf("%s:%s", id, key)
	}

	comp := &component{
		id:       id,
		sess:     sess,
		parent:   parent,
		callable: adapter,
		props:    props,
		children: make(map[string]*component),
		frame:    &hookFrame{},
	}

	if sess != nil {
		sess.mu.Lock()
		if sess.components == nil {
			sess.components = make(map[string]*component)
		}
		sess.components[id] = comp
		sess.mu.Unlock()
	}

	return comp
}

func (c *component) render() *dom2.StructuredNode {
	if c == nil || c.callable == nil {
		return nil
	}

	c.renderedThisFlush = true

	c.beginRender()
	ctx := Ctx{sess: c.sess, comp: c, frame: c.frame}
	node := c.callable.call(ctx, c.props)
	c.endRender()

	if node == nil {
		return c.node
	}

	if c.node == nil {
		c.node = node
	} else if c.node != node {
		cloneInto(c.node, node)
	}

	if c.wrapper != nil {
		c.wrapper.Children = []*dom2.StructuredNode{c.node}
	}

	return c.node
}

func (c *component) beginRender() {
	if c == nil || c.frame == nil {
		return
	}
	c.frame.idx = 0
	c.providerSeq = 0
	if c.parent != nil {
		c.combinedContextEpoch = c.contextEpoch + c.parent.combinedContextEpoch
	} else {
		c.combinedContextEpoch = c.contextEpoch
	}
}

func (c *component) endRender() {

}

func (c *component) notifyContextChange() {
	if c == nil {
		return
	}
	c.contextEpoch++
	if c.sess == nil {
		return
	}
	for _, child := range c.children {
		c.sess.markDirty(child)
	}
}

func (c *component) ensureChild(adapter componentCallable, key string, props any) *component {
	if c == nil {
		return nil
	}

	childID := adapter.hash()
	if key != "" {
		childID = fmt.Sprintf("%s:%s", childID, key)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if child, exists := c.children[childID]; exists {
		return child
	}

	child := &component{
		id:       childID,
		sess:     c.sess,
		parent:   c,
		callable: adapter,
		props:    props,
		children: make(map[string]*component),
		frame:    &hookFrame{},
	}

	c.children[childID] = child

	if c.sess != nil {
		c.sess.mu.Lock()
		if c.sess.components == nil {
			c.sess.components = make(map[string]*component)
		}
		c.sess.components[childID] = child
		c.sess.mu.Unlock()
	}

	return child
}

type hookFrame struct {
	cells []any
	idx   int
}

func cloneInto(dst, src *dom2.StructuredNode) {
	if dst == nil || src == nil {
		return
	}
	if dst == src {
		return
	}
	*dst = *src
}
