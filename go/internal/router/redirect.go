package router

import (
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

var Redirect = runtime.PropsComponent(func(ctx *runtime.Ctx, props RedirectProps, _ []work.Node) work.Node {
	requestState := headers.UseRequestState(ctx)
	isLive := requestState != nil && requestState.IsLive()

	if !isLive {
		if props.Replace {
			Replace(ctx, props.To)
		} else {
			Navigate(ctx, props.To)
		}
		return &work.Fragment{}
	}

	runtime.UseEffect(ctx, func() func() {
		if props.Replace {
			Replace(ctx, props.To)
		} else {
			Navigate(ctx, props.To)
		}
		return nil
	}, props.To, props.Replace)

	return &work.Fragment{}
})
