package runtime

import (
	"fmt"
	"time"

	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

type ReadSignal[T any] = reactive.Readable[T]

type RegionOpt struct {
	exit time.Duration
}

type ShowOpt interface{ applyShow(*showDef) }

type ForOpt interface{ applyFor(*time.Duration) }

type SwitchArg interface{ applySwitch(*switchSpec) }

func Exit(d time.Duration) RegionOpt { return RegionOpt{exit: d} }

func (o RegionOpt) applyShow(d *showDef)         { d.exit = o.exit }
func (o RegionOpt) applyFor(exit *time.Duration) { *exit = o.exit }
func (o RegionOpt) applySwitch(sp *switchSpec)   { sp.exit = o.exit }

type elseOpt struct{ fn func() work.Node }

func Else(fn func() work.Node) ShowOpt { return elseOpt{fn: fn} }

func (o elseOpt) applyShow(d *showDef) { d.els = o.fn }

type showDef struct {
	els  func() work.Node
	exit time.Duration
}

func Show(cond reactive.Readable[bool], then func() work.Node, opts ...ShowOpt) work.Node {
	var d showDef
	for _, o := range opts {
		o.applyShow(&d)
	}
	sp := switchSpec{cases: []caseArg{{key: true, fn: then}}, def: d.els, exit: d.exit}
	return &work.RegionNode{Impl: &switchDef[bool]{kd: regionShow, sel: cond, sp: sp}}
}

type caseArg struct {
	key any
	fn  func() work.Node
}

type defaultArg struct{ fn func() work.Node }

type waitExitArg struct{}

type switchSpec struct {
	cases []caseArg
	def   func() work.Node
	wait  bool
	exit  time.Duration
}

func Case[K comparable](k K, fn func() work.Node) SwitchArg { return caseArg{key: k, fn: fn} }

func Default(fn func() work.Node) SwitchArg { return defaultArg{fn: fn} }

func WaitExit() SwitchArg { return waitExitArg{} }

func (a caseArg) applySwitch(sp *switchSpec)    { sp.cases = append(sp.cases, a) }
func (a defaultArg) applySwitch(sp *switchSpec) { sp.def = a.fn }
func (waitExitArg) applySwitch(sp *switchSpec)  { sp.wait = true }

type switchDef[K comparable] struct {
	kd  regionKind
	sel reactive.Readable[K]
	sp  switchSpec
}

func Switch[K comparable](sel reactive.Readable[K], args ...SwitchArg) work.Node {
	d := &switchDef[K]{kd: regionSwitch, sel: sel}
	for _, a := range args {
		a.applySwitch(&d.sp)
	}
	for _, c := range d.sp.cases {
		if _, ok := c.key.(K); !ok {
			panic(fmt.Errorf("runtime: Switch case key %T does not match selector type", c.key))
		}
	}
	return &work.RegionNode{Impl: d}
}

func (d *switchDef[K]) kind() regionKind { return d.kd }

func (d *switchDef[K]) arm(k K) int {
	for i, c := range d.sp.cases {
		if c.key.(K) == k {
			return i
		}
	}
	if d.sp.def != nil {
		return len(d.sp.cases)
	}
	return -1
}

func (d *switchDef[K]) desired(us *unitSet[int], k K) []int {
	target := d.arm(k)
	if target < 0 {
		return nil
	}
	if d.sp.wait && us.exit > 0 {
		for _, r := range us.rows {
			if r.key != target || r.exiting {
				return nil
			}
		}
	}
	return []int{target}
}

func (d *switchDef[K]) mount(e *emitter, sc *scope, id uint32, force bool) {
	us := newUnitSet[int](e.s, sc.o, d.kd, id, d.sp.exit)
	us.run = func(_ *Ctx, r *unitRow[int]) work.Node {
		if r.key < len(d.sp.cases) {
			return d.sp.cases[r.key].fn()
		}
		return d.sp.def()
	}
	us.refresh = func() {
		us.s.graph.Untracked(func() { us.reconcile(d.desired(us, d.sel.Peek())) })
	}
	us.open(e, force)
	us.initial = e.w
	sc.o.Regions = append(sc.o.Regions, us)
	us.spawn(func() { us.reconcile(d.desired(us, d.sel.Get())) })
	us.initial = nil
	us.closeRegion(e)
}
