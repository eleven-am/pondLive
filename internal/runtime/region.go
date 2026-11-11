package runtime

import (
	"fmt"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

type regionKind uint8

const (
	regionShow regionKind = iota + 1
	regionSwitch
	regionFor
	regionFeed
	regionPortal
)

func (k regionKind) String() string {
	switch k {
	case regionShow:
		return "Show"
	case regionSwitch:
		return "Switch"
	case regionFor:
		return "For"
	case regionFeed:
		return "For(Feed)"
	case regionPortal:
		return "Portal"
	}
	return "region"
}

type regionDef interface {
	kind() regionKind
	mount(e *emitter, sc *scope, id uint32, force bool)
}

type earlyRegionError interface {
	error
	earlyRegionError()
}

type emptyRegion struct {
	kd regionKind
	id uint32
}

func (r *emptyRegion) kind() regionKind { return r.kd }

func (r *emptyRegion) regionID() uint32 { return r.id }

func (r *emptyRegion) serialize(e *emitter, force bool) { e.emptyRegion(r.id) }

type liveRegion interface {
	kind() regionKind
	regionID() uint32
	serialize(e *emitter, force bool)
}

func (e *emitter) region(sc *scope, rn *work.RegionNode, force bool) {
	def, ok := rn.Impl.(regionDef)
	if !ok {
		panic(fmt.Errorf("runtime: unsupported region implementation %T", rn.Impl))
	}
	if e.ser {
		if sc.reg >= len(sc.o.Regions) {
			panic(determinism(sc.o, "more regions than at mount"))
		}
		lr, ok := sc.o.Regions[sc.reg].(liveRegion)
		sc.reg++
		if !ok || lr.kind() != def.kind() {
			panic(determinism(sc.o, fmt.Sprintf("region %d is %s but was a different kind at mount", sc.reg-1, def.kind())))
		}
		if id := sc.take(); id != lr.regionID() {
			panic(determinism(sc.o, fmt.Sprintf("region %d has id %d but was %d at mount", sc.reg-1, id, lr.regionID())))
		}
		lr.serialize(e, force)
		return
	}
	e.lastBound = false
	id := sc.take()
	defer func() {
		if r := recover(); r != nil {
			err, ok := r.(earlyRegionError)
			if !ok {
				panic(r)
			}
			e.s.deliver(sc.o, e.s.buildError(ErrCodeRender, err.Error(), sc.o, "render:region", "", err))
			sc.o.Regions = append(sc.o.Regions, &emptyRegion{kd: def.kind(), id: id})
			e.emptyRegion(id)
			if force {
				e.noteRoot(id, true)
			}
		}
	}()
	def.mount(e, sc, id, force)
}

type unitRow[K comparable] struct {
	key     K
	o       *reactive.Owner
	val     any
	base    uint32
	n       uint32
	id      uint32
	exiting bool
	rootReg bool
	fresh   bool
}

type unitSet[K comparable] struct {
	s         *Session
	owner     *reactive.Owner
	kd        regionKind
	id        uint32
	rows      []*unitRow[K]
	byKey     map[K]*unitRow[K]
	initial   *writer
	exit      time.Duration
	drops     map[*unitRow[K]]int64
	tops      map[*unitRow[K]][]uint32
	timer     *Timer
	dead      bool
	run       func(ctx *Ctx, row *unitRow[K]) work.Node
	newVal    func(k K) any
	refresh   func()
	tag       string
	dropper   bool
	forget    func(k K)
	parentFor func(r *unitRow[K]) *reactive.Owner
}

func newUnitSet[K comparable](s *Session, owner *reactive.Owner, kd regionKind, id uint32, exit time.Duration) *unitSet[K] {
	us := &unitSet[K]{s: s, owner: owner, kd: kd, id: id, exit: exit, byKey: make(map[K]*unitRow[K])}
	owner.OnCleanup(func() {
		us.dead = true
		if us.timer != nil {
			us.timer.Stop()
		}
		for _, r := range us.rows {
			delete(s.exiting, r.o)
		}
	})
	return us
}

func (us *unitSet[K]) kind() regionKind { return us.kd }

func (us *unitSet[K]) regionID() uint32 { return us.id }

func (us *unitSet[K]) open(e *emitter, force bool) {
	e.lastBound = false
	if force {
		e.noteRoot(us.id, true)
	}
	us.tag = e.w.parentTag()
	e.w.regionStart(us.id)
}

func (us *unitSet[K]) closeRegion(e *emitter) {
	e.lastBound = false
	e.w.regionEnd(us.id)
}

func (us *unitSet[K]) serialize(e *emitter, force bool) {
	us.open(e, force)
	for _, r := range us.rows {
		us.serializeRow(e.w, r)
	}
	us.closeRegion(e)
}

func (us *unitSet[K]) serializeRow(w *writer, r *unitRow[K]) {
	s := us.s
	ue := s.newEmitter(w, true)
	ue.exiting = r.exiting
	o := r.o
	transient := o == nil
	if transient {
		o = s.graph.NewTransient(us.owner)
		o.Base, o.N = r.base, r.n
		o.Info = reactive.ComponentInfo{Name: us.kd.String()}
	} else {
		o.ResetCursor()
	}
	ue.ownerSerialize(o, func(ctx *Ctx) work.Node { return us.run(ctx, r) }, true, transient)
}

func (us *unitSet[K]) mountRow(r *unitRow[K], w *writer) {
	s := us.s
	parent := us.owner
	if us.parentFor != nil {
		parent = us.parentFor(r)
	}
	u := parent.NewUnit()
	u.Info = reactive.ComponentInfo{Name: us.kd.String()}
	r.o = u
	if us.newVal != nil && r.val == nil {
		r.val = us.newVal(r.key)
	}
	ue := s.newEmitter(w, false)
	ue.ownerBody(u, func(ctx *Ctx) work.Node { return us.run(ctx, r) }, true)
	r.id, r.rootReg = ue.rootID, ue.rootRegion
	us.setTops(r, ue.tops)
	r.base, r.n = u.Base, u.N
	if !s.rootMounting {
		s.runArmed()
	}
	if us.dropper && !s.failedOwner(u) && u.DropUnit() {
		r.o = nil
	}
}

func (s *Session) failedOwner(o *reactive.Owner) bool {
	_, bad := s.failed[o]
	return bad
}

func (us *unitSet[K]) mountToString(r *unitRow[K]) string {
	w := newWriter(us.s.devMode)
	w.tags = append(w.tags, us.tag)
	us.mountRow(r, w)
	html := w.String()
	if us.s.devMode {
		us.checkUnit(w, html)
		us.selfCheck(r, html)
	}
	return html
}

func (us *unitSet[K]) insert(r *unitRow[K], ref uint32) {
	html := us.mountToString(r)
	us.s.edit(protocol.InsertEdit(us.id, ref, html))
}

func (us *unitSet[K]) revive(r *unitRow[K]) {
	r.exiting = false
	delete(us.drops, r)
	delete(us.s.exiting, r.o)
	for _, id := range us.exitTargets(r) {
		us.s.edit(protocol.RemoveAttrEdit(id, protocol.AttrExiting))
		us.s.edit(protocol.RemoveAttrEdit(id, "inert"))
	}
}

func (us *unitSet[K]) setTops(r *unitRow[K], tops []uint32) {
	if len(tops) == 0 {
		delete(us.tops, r)
		return
	}
	if us.tops == nil {
		us.tops = make(map[*unitRow[K]][]uint32)
	}
	us.tops[r] = tops
}

func (us *unitSet[K]) exitTargets(r *unitRow[K]) []uint32 {
	if !r.rootReg {
		return []uint32{r.id}
	}
	return us.tops[r]
}

func (us *unitSet[K]) beginExit(r *unitRow[K]) {
	s := us.s
	r.exiting = true
	if us.drops == nil {
		us.drops = make(map[*unitRow[K]]int64)
	}
	us.drops[r] = s.clock.Now().Add(us.exit).UnixNano()
	if r.o != nil {
		if s.exiting == nil {
			s.exiting = make(map[*reactive.Owner]struct{})
		}
		s.exiting[r.o] = struct{}{}
	}
	targets := us.exitTargets(r)
	if len(targets) == 0 {
		s.Report(us.owner, Report{
			Severity: SeverityWarning,
			Code:     string(WarnExitNoElement),
			Message:  "a unit with no top-level element cannot show an exit effect",
			Meta:     map[string]any{"region": us.kd.String()},
		})
		return
	}
	for _, id := range targets {
		s.edit(protocol.AttrEdit(id, protocol.AttrExiting, ""))
		s.edit(protocol.AttrEdit(id, "inert", ""))
	}
}

func (us *unitSet[K]) dispose(r *unitRow[K]) {
	delete(us.drops, r)
	delete(us.tops, r)
	delete(us.s.exiting, r.o)
	if us.byKey[r.key] == r {
		delete(us.byKey, r.key)
		if us.forget != nil {
			us.forget(r.key)
		}
	}
	if r.o != nil {
		r.o.Dispose()
		r.o = nil
	}
}

func (us *unitSet[K]) remove(r *unitRow[K]) {
	us.s.edit(protocol.RemoveEdit(r.id))
	us.dispose(r)
}

func (us *unitSet[K]) clearAll() {
	us.s.edit(protocol.ClearEdit(us.id))
	rows := us.rows
	us.rows = nil
	for _, r := range rows {
		us.dispose(r)
	}
}

func (us *unitSet[K]) reconcile(desired []K) {
	if us.dead {
		return
	}
	if len(desired) == 0 && us.exit == 0 {
		if len(us.rows) > 0 {
			us.clearAll()
		}
		return
	}
	want := make(map[K]struct{}, len(desired))
	for _, k := range desired {
		want[k] = struct{}{}
	}
	old := make([]*unitRow[K], 0, len(us.rows))
	lastIdx := make([]int, 0, len(us.rows))
	for i, r := range us.rows {
		if _, ok := want[r.key]; ok {
			if r.exiting {
				us.revive(r)
			}
			old = append(old, r)
			lastIdx = append(lastIdx, i)
			continue
		}
		switch {
		case r.exiting && us.exit > 0:
			old = append(old, r)
			lastIdx = append(lastIdx, i)
		case us.exit > 0:
			us.beginExit(r)
			old = append(old, r)
			lastIdx = append(lastIdx, i)
		default:
			us.remove(r)
		}
	}
	oldPos := make(map[*unitRow[K]]int, len(old))
	for i, r := range old {
		oldPos[r] = i
	}

	active := make([]*unitRow[K], len(desired))
	for i, k := range desired {
		r := us.byKey[k]
		if r == nil {
			r = &unitRow[K]{key: k, fresh: true}
			us.byKey[k] = r
		}
		active[i] = r
	}
	merged := mergeStable(old, lastIdx, active, want)

	if us.initial != nil {
		for _, r := range merged {
			if r.fresh {
				us.mountRow(r, us.initial)
				r.fresh = false
			}
		}
		us.rows = merged
		us.schedule()
		return
	}

	seq := make([]int, 0, len(merged))
	at := make([]int, 0, len(merged))
	for i, r := range merged {
		if !r.fresh {
			seq = append(seq, oldPos[r])
			at = append(at, i)
		}
	}
	stay := make([]bool, len(merged))
	for j, in := range lis(seq) {
		if in {
			stay[at[j]] = true
		}
	}
	ref := uint32(0)
	for i := len(merged) - 1; i >= 0; i-- {
		r := merged[i]
		switch {
		case r.fresh:
			us.insert(r, ref)
			r.fresh = false
		case !stay[i]:
			us.s.edit(protocol.MoveEdit(r.id, us.id, ref))
		}
		ref = r.id
	}
	us.rows = merged
	us.schedule()
}

func mergeStable[K comparable](old []*unitRow[K], lastIdx []int, active []*unitRow[K], want map[K]struct{}) []*unitRow[K] {
	var exiting []*unitRow[K]
	var exitingIdx []int
	for i, r := range old {
		if _, ok := want[r.key]; !ok {
			exiting = append(exiting, r)
			exitingIdx = append(exitingIdx, lastIdx[i])
		}
	}
	merged := make([]*unitRow[K], 0, len(exiting)+len(active))
	ai, ei := 0, 0
	for ai < len(active) || ei < len(exiting) {
		switch {
		case ai >= len(active):
			merged = append(merged, exiting[ei])
			ei++
		case ei >= len(exiting):
			merged = append(merged, active[ai])
			ai++
		case exitingIdx[ei] <= ai:
			merged = append(merged, exiting[ei])
			ei++
		default:
			merged = append(merged, active[ai])
			ai++
		}
	}
	return merged
}

func lis(seq []int) []bool {
	n := len(seq)
	in := make([]bool, n)
	if n == 0 {
		return in
	}
	tails := make([]int, 0, n)
	prev := make([]int, n)
	for i, v := range seq {
		lo, hi := 0, len(tails)
		for lo < hi {
			mid := (lo + hi) / 2
			if seq[tails[mid]] < v {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
		if lo > 0 {
			prev[i] = tails[lo-1]
		} else {
			prev[i] = -1
		}
		if lo == len(tails) {
			tails = append(tails, i)
		} else {
			tails[lo] = i
		}
	}
	for i := tails[len(tails)-1]; i >= 0; i = prev[i] {
		in[i] = true
	}
	return in
}

func (us *unitSet[K]) schedule() {
	if us.timer != nil {
		us.timer.Stop()
		us.timer = nil
	}
	if us.dead {
		return
	}
	var earliest int64
	for _, at := range us.drops {
		if earliest == 0 || at < earliest {
			earliest = at
		}
	}
	if earliest == 0 {
		return
	}
	delay := time.Duration(earliest - us.s.clock.Now().UnixNano())
	if delay < 0 {
		delay = 0
	}
	us.timer = us.s.After(delay, us.expire)
}

func (us *unitSet[K]) expire() {
	if us.dead {
		return
	}
	now := us.s.clock.Now().UnixNano()
	kept := us.rows[:0]
	for _, r := range us.rows {
		if at, ok := us.drops[r]; ok && at <= now {
			us.remove(r)
			continue
		}
		kept = append(kept, r)
	}
	clear(us.rows[len(kept):])
	us.rows = kept
	us.schedule()
	if us.refresh != nil {
		us.refresh()
	}
}

func (us *unitSet[K]) spawn(fn func()) {
	var eff *reactive.Effect
	failed := false
	s, owner := us.s, us.owner
	eff = reactive.NewEffect(owner, reactive.KindRender, func() {
		defer func() {
			if r := recover(); r != nil {
				s.bindingPanic(owner, r)
				failed = true
				if eff != nil {
					eff.Dispose()
				}
			}
		}()
		fn()
	})
	if failed {
		eff.Dispose()
	}
}
