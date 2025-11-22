package runtime

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

// Component is a function that renders a node tree.
type Component[P any] func(Ctx, P) *dom.StructuredNode

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
	node                 *dom.StructuredNode
	wrapper              *dom.StructuredNode
	dirty                bool
	providers            map[contextID]any
	contextEpoch         int
	providerSeq          int
	combinedContextEpoch int

	renderedThisFlush bool

	// childRenderIndex tracks the current child render position during this render cycle.
	// Used to generate positional auto-keys when no explicit key is provided.
	childRenderIndex int

	mu sync.Mutex
}

// depth calculates the depth of this component in the tree (root = 0).
func (c *component) depth() int {
	if c == nil {
		return 0
	}
	depth := 0
	current := c
	for current.parent != nil {
		depth++
		current = current.parent
	}
	return depth
}

type componentCallable interface {
	call(Ctx, any) *dom.StructuredNode
	name() string
	pointer() uintptr
}

type componentAdapter[P any] struct {
	fn Component[P]
}

func newComponentAdapter[P any](fn Component[P]) *componentAdapter[P] {
	return &componentAdapter[P]{
		fn: fn,
	}
}

func (a *componentAdapter[P]) call(ctx Ctx, props any) *dom.StructuredNode {
	p, ok := props.(P)
	if !ok {
		var zero P
		p = zero
	}
	return a.fn(ctx, p)
}

func (a *componentAdapter[P]) name() string {

	return ""
}

func (a *componentAdapter[P]) pointer() uintptr {
	return reflect.ValueOf(a.fn).Pointer()
}

// buildComponentID generates a hierarchical component ID based on parent context,
// component name, and key. This uses the dumpv1 approach for better tree-aware identity.
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

func newComponent[P any](sess *ComponentSession, parent *component, key string, fn Component[P], props P) *component {
	adapter := newComponentAdapter(fn)
	id := buildComponentID(parent, adapter, key)

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

func (c *component) render() *dom.StructuredNode {
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
		c.wrapper.Children = []*dom.StructuredNode{c.node}
	}

	return c.node
}

func (c *component) beginRender() {
	if c == nil || c.frame == nil {
		return
	}
	c.frame.idx = 0
	c.providerSeq = 0
	c.childRenderIndex = 0
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

	childID := buildComponentID(c, adapter, key)

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

func cloneInto(dst, src *dom.StructuredNode) {
	if dst == nil || src == nil {
		return
	}
	if dst == src {
		return
	}
	*dst = *src
}
