package runtime

import (
	"fmt"
	"reflect"
	"time"

	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

type Source[T any] interface {
	newDef(row func(reactive.Readable[T]) work.Node, exit time.Duration) regionDef
}

func For[T any](src Source[T], row func(item reactive.Readable[T]) work.Node, opts ...ForOpt) work.Node {
	var exit time.Duration
	for _, o := range opts {
		o.applyFor(&exit)
	}
	return &work.RegionNode{Impl: src.newDef(row, exit)}
}

type DuplicateKeyError struct {
	Key string
}

func (e *DuplicateKeyError) Error() string {
	return "runtime: For received duplicate key " + e.Key
}

func itemEquality[T any]() func(a, b T) bool {
	if reflect.TypeFor[T]().Comparable() {
		return func(a, b T) (equal bool) {
			defer func() {
				if recover() != nil {
					equal = reflect.DeepEqual(a, b)
				}
			}()
			return any(a) == any(b)
		}
	}
	return func(a, b T) bool { return reflect.DeepEqual(a, b) }
}

type itemsSource[T any, K comparable] struct {
	items reactive.Readable[[]T]
	key   func(T) K
}

func Items[T any, K comparable](items reactive.Readable[[]T], key func(T) K) Source[T] {
	return &itemsSource[T, K]{items: items, key: key}
}

type itemsDef[T any, K comparable] struct {
	src  *itemsSource[T, K]
	row  func(reactive.Readable[T]) work.Node
	exit time.Duration
}

func (src *itemsSource[T, K]) newDef(row func(reactive.Readable[T]) work.Node, exit time.Duration) regionDef {
	return &itemsDef[T, K]{src: src, row: row, exit: exit}
}

func (d *itemsDef[T, K]) kind() regionKind { return regionFor }

func (d *itemsDef[T, K]) mount(e *emitter, sc *scope, id uint32, force bool) {
	s := e.s
	us := newUnitSet[K](s, sc.o, regionFor, id, d.exit)
	eq := itemEquality[T]()
	us.run = func(ctx *Ctx, r *unitRow[K]) work.Node {
		return inlineComponent(ctx, d.row(r.val.(*reactive.Signal[T])))
	}
	us.open(e, force)
	us.initial = e.w
	sc.o.Regions = append(sc.o.Regions, us)
	us.spawn(func() {
		list := d.src.items.Get()
		keys := make([]K, 0, len(list))
		pos := make(map[K]int, len(list))
		for i, item := range list {
			k := d.src.key(item)
			if _, dup := pos[k]; dup {
				if s.devMode {
					panic(&DuplicateKeyError{Key: fmt.Sprint(k)})
				}
				s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "for:duplicate-key",
					Message: "duplicate key " + fmt.Sprint(k) + " ignored: the first occurrence wins",
				})
				continue
			}
			pos[k] = i
			keys = append(keys, k)
		}
		us.newVal = func(k K) any {
			return reactive.NewSignal(s.graph, list[pos[k]], reactive.WithEqual(eq))
		}
		us.reconcile(keys)
		us.newVal = nil
		for _, r := range us.rows {
			if i, ok := pos[r.key]; ok && !r.exiting {
				if sig, ok := r.val.(*reactive.Signal[T]); ok {
					sig.Set(list[i])
				}
			}
		}
	})
	us.initial = nil
	us.closeRegion(e)
}
