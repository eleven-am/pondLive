package errors

import (
	"github.com/eleven-am/pondlive/internal/portal"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type Ctx = runtime.Ctx

type Props struct {
	DevMode bool
}

var Provider = runtime.PropsComponent(func(ctx *Ctx, props Props, children []work.Item) work.Node {
	batch := runtime.UseErrorBoundary(ctx)
	childNodes := work.ItemsToNodes(children)

	if !batch.HasErrors() {
		return &work.Fragment{Children: childNodes}
	}

	if props.DevMode {
		return &work.Fragment{
			Children: []work.Node{
				&work.Fragment{Children: childNodes},
				portal.Portal(devOverlay(batch)),
			},
		}
	}

	return crashPage()
})
