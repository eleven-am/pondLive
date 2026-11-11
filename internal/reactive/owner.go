package reactive

import "context"

type ComponentInfo struct {
	Name string
	Func uintptr
}

type ctxEntry struct {
	key any
	val any
}

type Owner struct {
	g        *Graph
	parent   *Owner
	depth    uint32
	seq      uint64
	Info     ComponentInfo
	Base     uint32
	N        uint32
	cells    []any
	cursor   int
	kids     []*Owner
	Regions  []any
	comps    []*node
	cleanups []func()
	ctxs     []ctxEntry
	units    *Owner
	prevU    *Owner
	nextU    *Owner
	slot     int
	ctx      context.Context
	cancel   context.CancelFunc
	disposed bool
	sealed   bool
}

var cancelledContext = func() context.Context {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx
}()

func (g *Graph) NewRoot() *Owner {
	g.createAllowed()
	g.ownerSeq++
	return &Owner{g: g, seq: g.ownerSeq, slot: -1}
}

func (g *Graph) createAllowed() {
	if g.mode == ModeSerialize {
		g.fatal(CreateInSerialize)
	}
}

func (g *Graph) NewTransient(parent *Owner) *Owner {
	if g.mode != ModeSerialize {
		panic("reactive: transient owners exist only during a serialize pass")
	}
	g.ownerSeq++
	return &Owner{g: g, parent: parent, depth: parent.depth + 1, seq: g.ownerSeq, slot: -1}
}

func (o *Owner) liveGraph() *Graph {
	if o == nil {
		panic(&DevError{Violation: NoLiveOwner})
	}
	if o.disposed {
		o.g.fatal(NoLiveOwner)
	}
	o.g.createAllowed()
	return o.g
}

func (o *Owner) Graph() *Graph { return o.g }

func (o *Owner) Parent() *Owner { return o.parent }

func (o *Owner) Depth() uint32 { return o.depth }

func (o *Owner) Disposed() bool { return o.disposed }

func (o *Owner) NewChild() *Owner {
	g := o.liveGraph()
	c := o.newOwner(g)
	c.slot = len(o.kids)
	o.kids = append(o.kids, c)
	return c
}

func (o *Owner) NewUnit() *Owner {
	g := o.liveGraph()
	u := o.newOwner(g)
	u.slot = -1
	u.nextU = o.units
	if o.units != nil {
		o.units.prevU = u
	}
	o.units = u
	return u
}

func (o *Owner) newOwner(g *Graph) *Owner {
	g.ownerSeq++
	return &Owner{g: g, parent: o, depth: o.depth + 1, seq: g.ownerSeq}
}

func (o *Owner) Kids() []*Owner { return o.kids }

func (o *Owner) adopt(n *node, kind Kind) {
	g := o.g
	g.nodeSeq++
	n.id = g.nodeSeq
	n.kind = kind
	n.owner = o
	n.depth, n.seq = o.depth, o.seq
	n.ownerIdx = len(o.comps)
	o.comps = append(o.comps, n)
}

func (o *Owner) OnCleanup(fn func()) {
	o.liveGraph()
	o.cleanups = append(o.cleanups, fn)
}

func (o *Owner) Seal() { o.sealed = true }

func (o *Owner) ResetCursor() { o.cursor = 0 }

func (o *Owner) EndBody() error {
	if o.g.mode == ModeSerialize && o.cursor != len(o.cells) {
		return &DeterminismError{Component: o.Info.Name, Cell: o.cursor, Detail: "body resolved fewer cells than the mount created"}
	}
	return nil
}

func (o *Owner) CellCount() int { return len(o.cells) }

func Cell[T any](o *Owner, mk func() *T) *T {
	g := o.g
	if g.mode == ModeSerialize {
		if o.cursor >= len(o.cells) {
			panic(&DeterminismError{Component: o.Info.Name, Cell: o.cursor, Detail: "more cells requested than the mount created"})
		}
		c, ok := o.cells[o.cursor].(*T)
		if !ok {
			panic(&DeterminismError{Component: o.Info.Name, Cell: o.cursor, Detail: "cell type differs from the mount"})
		}
		o.cursor++
		return c
	}
	o.liveGraph()
	if o.sealed {
		panic(&DevError{Violation: SealedOwner, Component: o.Info.Name})
	}
	c := mk()
	o.cells = append(o.cells, c)
	return c
}

func (o *Owner) SetContext(key, val any) {
	o.liveGraph()
	for i := range o.ctxs {
		if o.ctxs[i].key == key {
			o.ctxs[i].val = val
			return
		}
	}
	o.ctxs = append(o.ctxs, ctxEntry{key, val})
}

func (o *Owner) Lookup(key any) (any, bool) {
	for p := o; p != nil; p = p.parent {
		for i := range p.ctxs {
			if p.ctxs[i].key == key {
				return p.ctxs[i].val, true
			}
		}
	}
	return nil, false
}

func (o *Owner) Context() context.Context {
	if o.disposed {
		return cancelledContext
	}
	if o.ctx == nil {
		o.ctx, o.cancel = context.WithCancel(context.Background())
	}
	return o.ctx
}

func (o *Owner) Droppable() bool {
	if len(o.cells) != 0 || o.N != 0 || len(o.Regions) != 0 || len(o.comps) != 0 ||
		len(o.cleanups) != 0 || len(o.ctxs) != 0 || o.units != nil || o.ctx != nil {
		return false
	}
	for _, k := range o.kids {
		if k != nil {
			return false
		}
	}
	return true
}

func (o *Owner) Drop() bool {
	if o.disposed || o.slot < 0 || !o.Droppable() {
		return false
	}
	o.parent.kids[o.slot] = nil
	o.disposed = true
	o.parent, o.kids = nil, nil
	return true
}

func (o *Owner) Dispose() {
	if o.disposed {
		return
	}
	o.disposed = true
	var first any
	panicked := false
	record := func(v any) {
		if o.g.onCleanup != nil {
			o.g.onCleanup(o, v)
			return
		}
		if !panicked {
			first, panicked = v, true
		}
	}

	kids := o.kids
	o.kids = nil
	for _, k := range kids {
		if k != nil {
			runCleanup(k.Dispose, record)
		}
	}
	for u := o.units; u != nil; u = o.units {
		runCleanup(u.Dispose, record)
	}
	comps := o.comps
	o.comps = nil
	for _, n := range comps {
		n.owner = nil
		n.dispose()
	}
	cleanups := o.cleanups
	o.cleanups = nil
	for i := len(cleanups) - 1; i >= 0; i-- {
		runCleanup(cleanups[i], record)
	}
	if o.cancel != nil {
		o.cancel()
	}
	o.detach()
	o.cells, o.ctxs, o.Regions = nil, nil, nil
	if panicked {
		panic(first)
	}
}

func runCleanup(fn func(), record func(any)) {
	defer func() {
		if v := recover(); v != nil {
			record(v)
		}
	}()
	fn()
}

func (o *Owner) detach() {
	p := o.parent
	if p == nil {
		return
	}
	if o.slot >= 0 {
		if o.slot < len(p.kids) && p.kids[o.slot] == o {
			p.kids[o.slot] = nil
		}
		return
	}
	if o.prevU != nil {
		o.prevU.nextU = o.nextU
	} else if p.units == o {
		p.units = o.nextU
	}
	if o.nextU != nil {
		o.nextU.prevU = o.prevU
	}
	o.prevU, o.nextU = nil, nil
}
