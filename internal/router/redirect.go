package router

import (
	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

var Redirect = runtime.PropsComponent(func(ctx *runtime.Ctx, props RedirectProps, _ []work.Item) work.Node {
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
