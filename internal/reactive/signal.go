package reactive

type Readable[T any] interface {
	Get() T
	Peek() T
}

type Signal[T any] struct {
	source
	v  T
	eq func(a, b T) bool
}

func NewSignal[T any](g *Graph, initial T, opts ...Option[T]) *Signal[T] {
	if g.mode == ModeSerialize {
		g.fatal(CreateInSerialize)
	}
	s := &Signal[T]{v: initial, eq: resolveEqual(opts)}
	s.g = g
	return s
}

func (s *Signal[T]) Get() T {
	g := s.g
	g.readAllowed()
	g.track(&s.source)
	return s.v
}

func (s *Signal[T]) Peek() T { return s.v }

func (s *Signal[T]) Set(v T) {
	g := s.g
	if g.offActor() {
		g.postWrite(func() { s.Set(v) })
		return
	}
	if !g.writeAllowed() {
		return
	}
	s.set(v)
}

func (s *Signal[T]) Update(fn func(T) T) {
	g := s.g
	if g.offActor() {
		g.postWrite(func() { s.Update(fn) })
		return
	}
	if !g.writeAllowed() {
		return
	}
	s.set(fn(s.v))
}

func (s *Signal[T]) set(v T) {
	if s.eq(s.v, v) {
		return
	}
	s.v = v
	s.version++
	s.g.version++
	for l := s.targets; l != nil; l = l.nextS {
		l.tgt.notify()
	}
}

type Computed[T any] struct {
	n  node
	v  T
	eq func(a, b T) bool
}

func NewComputed[T any](o *Owner, fn func() T, opts ...Option[T]) *Computed[T] {
	g := o.liveGraph()
	c := &Computed[T]{eq: resolveEqual(opts)}
	n := &c.n
	n.g, n.computed, n.comp = g, true, n
	n.gver = g.version - 1
	n.compute = func() bool {
		nv := fn()
		if n.version > 0 && c.eq(c.v, nv) {
			return false
		}
		c.v = nv
		return true
	}
	o.adopt(n, KindRender)
	return c
}

func (c *Computed[T]) Get() T {
	n := &c.n
	g := n.g
	g.readAllowed()
	l := g.track(&n.source)
	if !n.refreshComputed() {
		g.fatal(CycleDetected)
	}
	if l != nil {
		l.version = n.version
	}
	return c.v
}

func (c *Computed[T]) Peek() T {
	if !c.n.refreshComputed() {
		c.n.g.fatal(CycleDetected)
	}
	return c.v
}

type Effect struct {
	n node
}

func NewEffect(o *Owner, kind Kind, fn func()) *Effect {
	g := o.liveGraph()
	e := &Effect{}
	n := &e.n
	n.g, n.fn = g, fn
	n.flags = fTracking
	o.adopt(n, kind)
	n.runEffect()
	return e
}

func (e *Effect) Dispose() { e.n.dispose() }

func (e *Effect) Disposed() bool { return e.n.flags&fDisposed != 0 }
