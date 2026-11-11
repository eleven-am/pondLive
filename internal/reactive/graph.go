package reactive

const (
	CascadeBudget       = 50
	MaxRenderIterations = 100
)

const unused = ^uint64(0)

type Mode uint8

const (
	ModeMount Mode = iota
	ModeSerialize
)

type Queue uint8

const (
	QueueRender Queue = iota
	QueueWatch
	queueCount
)

type flag uint8

const (
	fRunning flag = 1 << iota
	fNotified
	fOutdated
	fTracking
	fDisposed
	fInHeap
)

type link struct {
	src          *source
	tgt          *node
	version      uint64
	prevS, nextS *link
	prevT, nextT *link
	rollback     *link
	subscribed   bool
}

type source struct {
	g       *Graph
	version uint64
	targets *link
	cur     *link
	comp    *node
}

type node struct {
	source
	flags    flag
	kind     Kind
	computed bool
	gver     uint64
	srcHead  *link
	srcTail  *link
	fn       func()
	compute  func() bool
	owner    *Owner
	ownerIdx int
	depth    uint32
	seq      uint64
	id       uint64
	passID   uint64
	passRuns int
}

type Graph struct {
	version     uint64
	observer    *node
	mode        Mode
	computing   int
	inBody      bool
	pass        uint64
	nodeSeq     uint64
	ownerSeq    uint64
	heaps       [queueCount]heap
	live        [queueCount]int
	actorGID    int64
	post        func(func())
	onViolation func(*DevError)
	onCleanup   func(*Owner, any)
}

func NewGraph() *Graph {
	return &Graph{version: 1}
}

func (g *Graph) SetViolationHandler(fn func(*DevError)) { g.onViolation = fn }

func (g *Graph) SetCleanupPanicHandler(fn func(*Owner, any)) { g.onCleanup = fn }

func (g *Graph) SetPoster(post func(func())) { g.post = post }

func (g *Graph) BindActor() { g.actorGID = currentGoroutine() }

func (g *Graph) OnActor() bool {
	return g.actorGID == 0 || currentGoroutine() == g.actorGID
}

func (g *Graph) Mode() Mode { return g.mode }

func (g *Graph) Serialize(fn func()) {
	prev := g.mode
	g.mode = ModeSerialize
	defer func() { g.mode = prev }()
	fn()
}

func (g *Graph) RunBody(fn func()) {
	prev := g.inBody
	g.inBody = true
	defer func() { g.inBody = prev }()
	fn()
}

func (g *Graph) Untracked(fn func()) {
	prev := g.observer
	g.observer = nil
	defer func() { g.observer = prev }()
	fn()
}

func (g *Graph) fatal(v Violation) {
	panic(&DevError{Violation: v})
}

func (g *Graph) soft(v Violation) bool {
	if g.onViolation == nil {
		g.fatal(v)
	}
	g.onViolation(&DevError{Violation: v})
	return false
}

func (g *Graph) writeAllowed() bool {
	switch {
	case g.mode == ModeSerialize:
		return g.soft(WriteInSerialize)
	case g.computing > 0:
		return g.soft(WriteInComputed)
	case g.inBody:
		return g.soft(WriteInBody)
	}
	return true
}

func (g *Graph) readAllowed() {
	if g.actorGID != 0 && currentGoroutine() != g.actorGID {
		g.soft(OffActorRead)
		return
	}
	if g.observer == nil && g.inBody {
		g.soft(GetInBody)
	}
}

func (g *Graph) offActor() bool {
	return g.actorGID != 0 && currentGoroutine() != g.actorGID
}

func (g *Graph) postWrite(fn func()) {
	if g.post == nil {
		g.soft(OffActorWrite)
		return
	}
	g.post(fn)
}

func (g *Graph) track(s *source) *link {
	t := g.observer
	if t == nil {
		return nil
	}
	l := s.cur
	if l == nil || l.tgt != t {
		l = &link{src: s, tgt: t, rollback: s.cur}
		l.prevT = t.srcTail
		if t.srcTail != nil {
			t.srcTail.nextT = l
		} else {
			t.srcHead = l
		}
		t.srcTail = l
		s.cur = l
		if t.flags&fTracking != 0 {
			s.subscribe(l)
		}
		return l
	}
	if l.version == unused {
		l.version = 0
		if l.nextT != nil {
			if l.prevT != nil {
				l.prevT.nextT = l.nextT
			} else {
				t.srcHead = l.nextT
			}
			l.nextT.prevT = l.prevT
			l.prevT = t.srcTail
			l.nextT = nil
			t.srcTail.nextT = l
			t.srcTail = l
		}
	}
	return l
}

func (s *source) subscribe(l *link) {
	if l.subscribed {
		return
	}
	l.subscribed = true
	l.prevS = nil
	l.nextS = s.targets
	if s.targets != nil {
		s.targets.prevS = l
	}
	s.targets = l
	if s.comp != nil && s.targets.nextS == nil {
		s.comp.startTracking()
	}
}

func (s *source) unsubscribe(l *link) {
	if !l.subscribed {
		return
	}
	l.subscribed = false
	if l.prevS != nil {
		l.prevS.nextS = l.nextS
	} else {
		s.targets = l.nextS
	}
	if l.nextS != nil {
		l.nextS.prevS = l.prevS
	}
	l.prevS, l.nextS = nil, nil
	if s.comp != nil && s.targets == nil {
		s.comp.stopTracking()
	}
}

func (n *node) startTracking() {
	n.flags |= fOutdated | fTracking
	for l := n.srcHead; l != nil; l = l.nextT {
		l.src.subscribe(l)
	}
}

func (n *node) stopTracking() {
	n.flags &^= fTracking
	for l := n.srcHead; l != nil; l = l.nextT {
		l.src.unsubscribe(l)
	}
}

func (n *node) prepare() {
	for l := n.srcHead; l != nil; l = l.nextT {
		l.rollback = l.src.cur
		l.src.cur = l
		l.version = unused
	}
}

func (n *node) cleanup() {
	for l := n.srcTail; l != nil; {
		prev := l.prevT
		l.src.cur = l.rollback
		l.rollback = nil
		if l.version == unused {
			l.src.unsubscribe(l)
			if prev != nil {
				prev.nextT = l.nextT
			} else {
				n.srcHead = l.nextT
			}
			if l.nextT != nil {
				l.nextT.prevT = prev
			} else {
				n.srcTail = prev
			}
			l.src, l.tgt, l.prevT, l.nextT = nil, nil, nil, nil
		}
		l = prev
	}
}

func (n *node) releaseSources() {
	for l := n.srcHead; l != nil; {
		next := l.nextT
		l.src.unsubscribe(l)
		l.src, l.tgt, l.prevT, l.nextT, l.rollback = nil, nil, nil, nil, nil
		l = next
	}
	n.srcHead, n.srcTail = nil, nil
}

func (n *node) needsRecompute() bool {
	for l := n.srcHead; l != nil; l = l.nextT {
		if l.src.version != l.version || !l.src.refresh() || l.src.version != l.version {
			return true
		}
	}
	return false
}

func (s *source) refresh() bool {
	if s.comp == nil {
		return true
	}
	return s.comp.refreshComputed()
}

func (n *node) notify() {
	if n.flags&fNotified != 0 {
		return
	}
	n.flags |= fNotified
	if n.computed {
		n.flags |= fOutdated
		for l := n.targets; l != nil; l = l.nextS {
			l.tgt.notify()
		}
		return
	}
	n.g.enqueue(n)
}

func (n *node) refreshComputed() bool {
	g := n.g
	n.flags &^= fNotified
	if n.flags&fRunning != 0 {
		return false
	}
	if n.flags&(fOutdated|fTracking) == fTracking {
		return true
	}
	n.flags &^= fOutdated
	if n.gver == g.version {
		return true
	}
	n.gver = g.version
	n.flags |= fRunning
	if n.version > 0 && !n.needsRecompute() {
		n.flags &^= fRunning
		return true
	}
	n.evaluate()
	return true
}

func (n *node) evaluate() {
	g := n.g
	prev, prevBody := g.observer, g.inBody
	g.observer, g.inBody = n, false
	g.computing++
	n.prepare()
	ok := false
	defer func() {
		g.observer, g.inBody = prev, prevBody
		g.computing--
		n.cleanup()
		n.flags &^= fRunning
		if !ok {
			n.flags |= fOutdated
			n.gver = g.version - 1
		}
	}()
	if n.compute() {
		n.version++
	}
	ok = true
}

func (n *node) runEffect() {
	g := n.g
	n.flags &^= fNotified
	if n.flags&fRunning != 0 {
		g.fatal(CycleDetected)
	}
	n.flags |= fRunning
	prev, prevBody := g.observer, g.inBody
	g.observer, g.inBody = n, false
	n.prepare()
	defer func() {
		g.observer, g.inBody = prev, prevBody
		n.cleanup()
		n.flags &^= fRunning
		if n.flags&fDisposed != 0 {
			n.releaseSources()
		}
	}()
	n.fn()
}

func (n *node) dispose() {
	if n.flags&fDisposed != 0 {
		return
	}
	n.flags |= fDisposed
	if n.flags&fInHeap != 0 {
		n.g.live[queueOf(n.kind)]--
	}
	if n.flags&fRunning == 0 {
		n.releaseSources()
	}
	n.flags &^= fTracking
	n.fn, n.compute = nil, nil
	if o := n.owner; o != nil {
		last := len(o.comps) - 1
		if n.ownerIdx <= last && o.comps[n.ownerIdx] == n {
			o.comps[n.ownerIdx] = o.comps[last]
			o.comps[n.ownerIdx].ownerIdx = n.ownerIdx
			o.comps[last] = nil
			o.comps = o.comps[:last]
		}
		n.owner = nil
	}
}

func queueOf(k Kind) Queue {
	if k == KindRender {
		return QueueRender
	}
	return QueueWatch
}

func (g *Graph) enqueue(n *node) {
	q := queueOf(n.kind)
	n.flags |= fInHeap
	g.live[q]++
	g.heaps[q].push(n)
}

func (g *Graph) Pending(q Queue) bool { return g.live[q] > 0 }

func (g *Graph) RunQueue(q Queue) error {
	g.pass++
	h := &g.heaps[q]
	for len(h.items) > 0 {
		n := h.pop()
		n.flags &^= fInHeap
		if n.flags&fDisposed != 0 {
			continue
		}
		g.live[q]--
		if !n.needsRecompute() {
			n.flags &^= fNotified
			continue
		}
		if n.passID != g.pass {
			n.passID, n.passRuns = g.pass, 0
		}
		n.passRuns++
		if n.passRuns > MaxRenderIterations {
			n.flags &^= fNotified
			return &BudgetError{Kind: BudgetRounds, Owner: n.owner}
		}
		n.runEffect()
	}
	return nil
}

type FlushHooks struct {
	Drain    func()
	Publish  func()
	Deferred func() bool
}

func (g *Graph) Flush(h FlushHooks) error {
	for pass := 0; ; pass++ {
		if h.Drain != nil {
			h.Drain()
		}
		if pass == CascadeBudget {
			return &BudgetError{Kind: BudgetPasses}
		}
		if err := g.RunQueue(QueueRender); err != nil {
			return err
		}
		if h.Publish != nil {
			h.Publish()
		}
		if h.Deferred != nil && h.Deferred() {
			return nil
		}
		if err := g.RunQueue(QueueWatch); err != nil {
			return err
		}
		if !g.Pending(QueueRender) {
			return nil
		}
	}
}

type heap struct {
	items []*node
}

func less(a, b *node) bool {
	if a.depth != b.depth {
		return a.depth < b.depth
	}
	if a.seq != b.seq {
		return a.seq < b.seq
	}
	return a.id < b.id
}

func (h *heap) push(n *node) {
	h.items = append(h.items, n)
	i := len(h.items) - 1
	for i > 0 {
		p := (i - 1) / 2
		if !less(h.items[i], h.items[p]) {
			break
		}
		h.items[i], h.items[p] = h.items[p], h.items[i]
		i = p
	}
}

func (h *heap) pop() *node {
	top := h.items[0]
	last := len(h.items) - 1
	h.items[0] = h.items[last]
	h.items[last] = nil
	h.items = h.items[:last]
	i := 0
	for {
		l, r, m := 2*i+1, 2*i+2, i
		if l < last && less(h.items[l], h.items[m]) {
			m = l
		}
		if r < last && less(h.items[r], h.items[m]) {
			m = r
		}
		if m == i {
			break
		}
		h.items[i], h.items[m] = h.items[m], h.items[i]
		i = m
	}
	return top
}
