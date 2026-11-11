package errors

import (
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type Ctx = runtime.Ctx

type Props struct {
	DevMode    bool
	Children   work.Node
	OnBoundary func(*runtime.Boundary)
}

var Provider = runtime.PropsComponent(func(ctx *Ctx, props Props) work.Node {
	boundary := runtime.UseErrorBoundary(ctx)
	if props.OnBoundary != nil {
		props.OnBoundary(boundary)
	}
	failed := runtime.UseComputed(ctx, func() bool { return boundary.Errors().Get().HasErrors() })
	errs := runtime.UseComputed(ctx, func() []*runtime.Error { return boundary.Errors().Get().All() })

	warns := boundary.Warnings()
	warned := runtime.UseComputed(ctx, func() bool { return len(warns.Get()) > 0 })

	if props.DevMode {
		return work.NewFragment(
			props.Children,
			runtime.Show(failed, func() work.Node { return devOverlay(errs) }),
			runtime.Show(warned, func() work.Node { return warningPanel(boundary, warns) }),
		)
	}

	return runtime.Show(failed,
		crashPage,
		runtime.Else(func() work.Node { return props.Children }),
	)
})
