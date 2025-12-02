package router

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type routeMountProps struct {
	match        Match
	matchState   *MatchState
	base         string
	childSlots   map[string]outletRenderer
	component    func(*runtime.Ctx, Match) work.Node
	componentKey string
}

var routeMount = runtime.PropsComponent(func(ctx *runtime.Ctx, props routeMountProps, _ []work.Node) work.Node {
	matchCtx.UseProvider(ctx, props.matchState)
	routeBaseCtx.UseProvider(ctx, props.base)
	slotsCtx.UseProvider(ctx, props.childSlots)

	return &work.ComponentNode{
		Fn:    props.component,
		Props: props.match,
		Key:   props.componentKey,
	}
})
