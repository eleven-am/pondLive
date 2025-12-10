package portal

import (
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type Ctx = runtime.Ctx

type portalState struct {
	nodes []work.Node
}

var portalCtx = runtime.CreateContext[*portalState](nil)

var Provider = runtime.Component(func(ctx *Ctx, children []work.Item) work.Node {
	state := &portalState{
		nodes: make([]work.Node, 0),
	}

	portalCtx.UseProvider(ctx, state)

	return work.NewFragment(children...)
})
