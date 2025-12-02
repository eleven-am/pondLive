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

var routeMount = runtime.PropsComponent(func(ctx *runtime.Ctx, props routeMountProps, _ []work.Item) work.Node {
	_, setMatch := matchCtx.UseProvider(ctx, props.matchState)
	_, setBase := routeBaseCtx.UseProvider(ctx, props.base)
	_, setSlots := slotsCtx.UseProvider(ctx, props.childSlots)

	setMatch(props.matchState)
	setBase(props.base)
	setSlots(props.childSlots)

	return &work.ComponentNode{
		Fn:    props.component,
		Props: props.match,
		Key:   props.componentKey,
	}
})
