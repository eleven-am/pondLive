package effectguard

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/petermattis/goid"
)

type ForbiddenCallError struct {
	Op        string
	Component string
}

func (e *ForbiddenCallError) Error() string {
	return fmt.Sprintf("%s was called inside UseRenderEffect of %q; a render effect may only do bookkeeping, so move this call to UseEffect (runs after the socket connects) or to a handler", e.Op, e.Component)
}

type Scope struct {
	Component string
	Dev       bool
	Dropped   func(op, component string)
}

var (
	active atomic.Int64
	scopes sync.Map
)

func Enter(sc *Scope) (exit func()) {
	gid := goid.Get()
	prev, had := scopes.Swap(gid, sc)
	if !had {
		active.Add(1)
	}
	return func() {
		if had {
			scopes.Store(gid, prev)
			return
		}
		scopes.Delete(gid)
		active.Add(-1)
	}
}

func Current() *Scope {
	if active.Load() == 0 {
		return nil
	}
	v, ok := scopes.Load(goid.Get())
	if !ok {
		return nil
	}
	return v.(*Scope)
}

func Refuse(op string) bool {
	sc := Current()
	if sc == nil {
		return false
	}
	if sc.Dev {
		panic(&ForbiddenCallError{Op: op, Component: sc.Component})
	}
	if sc.Dropped != nil {
		sc.Dropped(op, sc.Component)
	}
	return true
}

func Active() bool { return active.Load() > 0 }
