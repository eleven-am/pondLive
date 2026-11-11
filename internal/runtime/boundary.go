package runtime

import (
	"slices"

	"github.com/eleven-am/pondlive/internal/reactive"
)

type Boundary struct {
	s     *Session
	errs  *reactive.Signal[*ErrorBatch]
	warns *reactive.Signal[[]*Warning]
}

func UseErrorBoundary(ctx *Ctx) *Boundary {
	return reactive.Cell(ctx.o, func() *Boundary {
		b := &Boundary{
			s:     ctx.s,
			errs:  reactive.NewSignal(ctx.s.graph, NewErrorBatch()),
			warns: reactive.NewSignal(ctx.s.graph, []*Warning(nil)),
		}
		ctx.o.SetContext(boundaryKey{}, b)
		if ctx.s.rootBoundary == nil {
			ctx.s.rootBoundary = b
			ctx.o.OnCleanup(func() {
				if ctx.s.rootBoundary == b {
					ctx.s.rootBoundary = nil
				}
			})
		}
		return b
	})
}

func (b *Boundary) Errors() ReadSignal[*ErrorBatch] { return b.errs }

func (b *Boundary) Warnings() ReadSignal[[]*Warning] { return b.warns }

func (b *Boundary) Reset() {
	if b == nil {
		return
	}
	b.errs.Set(NewErrorBatch())
}

func (b *Boundary) Dismiss(id uint64) {
	if b == nil {
		return
	}
	current := b.warns.Peek()
	i := slices.IndexFunc(current, func(w *Warning) bool { return w.ID == id })
	if i < 0 {
		return
	}
	b.warns.Set(slices.Delete(slices.Clone(current), i, i+1))
}

func (b *Boundary) DismissAll() {
	if b == nil || len(b.warns.Peek()) == 0 {
		return
	}
	b.warns.Set(nil)
}

func (b *Boundary) push(err *Error) {
	current := b.errs.Peek()
	all := append(append([]*Error(nil), current.All()...), err)
	b.errs.Set(NewErrorBatch(all...))
}

func (b *Boundary) pushWarning(w *Warning) {
	b.warns.Set(append(slices.Clone(b.warns.Peek()), w))
}
