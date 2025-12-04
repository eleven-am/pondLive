package metatags

import (
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type Ctx = runtime.Ctx

type metaState struct {
	entries    map[string]metaEntry
	setEntries func(map[string]metaEntry)
}

var noopState = &metaState{
	entries:    make(map[string]metaEntry),
	setEntries: func(map[string]metaEntry) {},
}

var metaCtx = runtime.CreateContext[*metaState](noopState)

var Provider = runtime.Component(func(ctx *Ctx, children []work.Item) work.Node {
	initialState := &metaState{
		entries:    make(map[string]metaEntry),
		setEntries: func(map[string]metaEntry) {},
	}

	state, setState := metaCtx.UseProvider(ctx, initialState)

	state.setEntries = func(newEntries map[string]metaEntry) {
		next := &metaState{
			entries:    newEntries,
			setEntries: state.setEntries,
		}
		setState(next)
	}

	nodes := itemsToNodes(children)
	return &work.Fragment{Children: nodes}
})

func itemsToNodes(items []work.Item) []work.Node {
	nodes := make([]work.Node, 0, len(items))
	for _, item := range items {
		if node, ok := item.(work.Node); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}
