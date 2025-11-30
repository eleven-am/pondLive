package router

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type RedirectProps struct {
	To      string
	Replace bool
}

func Redirect(ctx *runtime.Ctx, props RedirectProps) work.Node {

	runtime.UseEffect(ctx, func() func() {
		if props.Replace {
			Replace(ctx, props.To)
		} else {
			Navigate(ctx, props.To)
		}
		return nil
	}, props.To, props.Replace)

	return &work.Fragment{}
}

func RedirectIf(ctx *runtime.Ctx, condition bool, to string, otherwise work.Node) work.Node {
	if condition {
		return Redirect(ctx, RedirectProps{To: to})
	}
	return otherwise
}

func RedirectIfNot(ctx *runtime.Ctx, condition bool, to string, otherwise work.Node) work.Node {
	return RedirectIf(ctx, !condition, to, otherwise)
}
