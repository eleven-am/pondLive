package runtime

import (
	"errors"
	"fmt"

	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

func (e *emitter) component(parent *scope, c *work.ComponentNode, force bool) {
	fn := c.Fn.(componentFn)
	if e.ser {
		e.serializeComponent(parent, c, fn, force)
		return
	}
	o := parent.o.NewChild()
	o.Info = reactive.ComponentInfo{Name: c.Name, Func: c.Ident}
	e.ownerBody(o, fn, force)
}

func (s *Session) runBody(ctx *Ctx, fn func(*Ctx) work.Node) (out work.Node, ok bool) {
	o := ctx.o
	s.mountBodyRuns++
	err := s.recoverUser(ErrCodeRender, "render", o, func() {
		s.graph.Untracked(func() { s.graph.RunBody(func() { out = fn(ctx) }) })
	})
	ctx.finish()
	if err != nil {
		s.deliver(o, err)
		return nil, false
	}
	return out, true
}

func (s *Session) runBodySerialize(ctx *Ctx, fn func(*Ctx) work.Node) (out work.Node) {
	s.graph.Untracked(func() { s.graph.RunBody(func() { out = fn(ctx) }) })
	return out
}

func (c *Ctx) finish() {
	c.o.Seal()
}

func (e *emitter) ownerBody(o *reactive.Owner, fn func(*Ctx) work.Node, force bool) {
	s := e.s
	node, ok := s.runBody(s.newCtx(o), fn)
	if ok && force {
		node = normalizeRoot(node)
	}
	if !ok {
		e.failedOwner(o, force)
		return
	}
	if ok {
		if err := validate(node); err != nil {
			s.deliver(o, s.buildError(ErrCodeRender, err.Error(), o, "render:invalid", "", err))
			e.failedOwner(o, force)
			return
		}
	}
	n := count(node, force)
	o.Base, o.N = s.allocIDs(n), n
	sc := &scope{o: o, next: o.Base}
	e.node(sc, node, force)
	if sc.next != o.Base+o.N {
		panic(fmt.Errorf("runtime: %s emitted %d dynamic nodes but reserved %d", o.Info.Name, sc.next-o.Base, o.N))
	}
	if o.Droppable() {
		o.Drop()
	}
}

func (e *emitter) failedOwner(o *reactive.Owner, force bool) {
	s := e.s
	if s.failed == nil {
		s.failed = make(map[*reactive.Owner]struct{})
	}
	s.failed[o] = struct{}{}
	o.OnCleanup(func() { delete(s.failed, o) })
	if force {
		o.Base, o.N = s.allocIDs(1), 1
		e.emptyRegion(o.Base)
	}
}

func (e *emitter) emptyRegion(id uint32) {
	e.lastBound = false
	e.noteRoot(id, true)
	e.w.regionStart(id)
	e.w.regionEnd(id)
}

var ErrNondeterministic = errors.New("runtime: component output is not deterministic")

type NondeterministicError struct {
	Component string
	Detail    string
}

func (e *NondeterministicError) Error() string {
	return fmt.Sprintf("runtime: nondeterministic output in %s: %s", e.Component, e.Detail)
}

func (e *NondeterministicError) Is(target error) bool { return target == ErrNondeterministic }

func determinism(o *reactive.Owner, detail string) *reactive.DeterminismError {
	name := ""
	if o != nil {
		name = o.Info.Name
	}
	return &reactive.DeterminismError{Component: name, Detail: detail}
}

func (e *emitter) serializeComponent(parent *scope, c *work.ComponentNode, fn componentFn, force bool) {
	kids := parent.o.Kids()
	i := parent.kid
	parent.kid++
	if i >= len(kids) {
		panic(determinism(parent.o, "more child components than at mount"))
	}
	o := kids[i]
	transient := o == nil
	if transient {
		o = e.s.graph.NewTransient(parent.o)
		o.Info = reactive.ComponentInfo{Name: c.Name, Func: c.Ident}
	} else {
		if o.Info.Func != c.Ident {
			panic(determinism(parent.o, fmt.Sprintf("child component %d is %q but was %q at mount", i, c.Name, o.Info.Name)))
		}
		o.ResetCursor()
	}
	e.ownerSerialize(o, fn, force, transient)
}

func (e *emitter) ownerSerialize(o *reactive.Owner, fn func(*Ctx) work.Node, force, transient bool) {
	s := e.s
	if _, bad := s.failed[o]; bad {
		if force {
			e.emptyRegion(o.Base)
		}
		return
	}
	node := s.runBodySerialize(s.newCtx(o), fn)
	if force {
		node = normalizeRoot(node)
	}
	n := count(node, force)
	if n != o.N {
		panic(determinism(o, fmt.Sprintf("dynamic node count %d differs from %d at mount", n, o.N)))
	}
	if err := o.EndBody(); err != nil {
		panic(err)
	}
	sc := &scope{o: o, next: o.Base}
	e.node(sc, node, force)
	if sc.reg != len(o.Regions) {
		panic(determinism(o, fmt.Sprintf("%d regions emitted but %d at mount", sc.reg, len(o.Regions))))
	}
	if !transient && sc.kid != len(o.Kids()) {
		panic(determinism(o, fmt.Sprintf("%d child components emitted but %d at mount", sc.kid, len(o.Kids()))))
	}
}
