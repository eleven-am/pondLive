package router2

import (
	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// RedirectProps configures the Redirect component.
type RedirectProps struct {
	To      string // Target href
	Replace bool   // Use replaceState instead of pushState
}

// Redirect triggers navigation when rendered.
// Use for programmatic redirects within the component tree.
//
// Usage:
//
//	if !isAuthenticated {
//	    return router.Redirect(ctx, router.RedirectProps{To: "/login"})
//	}
func Redirect(ctx *runtime2.Ctx, props RedirectProps) work.Node {

	runtime2.UseEffect(ctx, func() func() {
		if props.Replace {
			Replace(ctx, props.To)
		} else {
			Navigate(ctx, props.To)
		}
		return nil
	}, props.To, props.Replace)

	return &work.Fragment{}
}

// RedirectIf conditionally redirects based on a condition.
// If the condition is true, redirects to the target.
// Otherwise, renders the children.
//
// Usage:
//
//	return router.RedirectIf(ctx, !isAuthenticated, "/login", content)
func RedirectIf(ctx *runtime2.Ctx, condition bool, to string, otherwise work.Node) work.Node {
	if condition {
		return Redirect(ctx, RedirectProps{To: to})
	}
	return otherwise
}

// RedirectIfNot conditionally redirects based on a condition.
// If the condition is false, redirects to the target.
// Otherwise, renders the children.
//
// Usage:
//
//	return router.RedirectIfNot(ctx, isAuthenticated, "/login", content)
func RedirectIfNot(ctx *runtime2.Ctx, condition bool, to string, otherwise work.Node) work.Node {
	return RedirectIf(ctx, !condition, to, otherwise)
}
