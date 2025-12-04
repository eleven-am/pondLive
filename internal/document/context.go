package document

import (
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type Ctx = runtime.Ctx

type documentState struct {
	entries    map[string]documentEntry
	setEntries func(map[string]documentEntry)
}

var noopState = &documentState{
	entries:    make(map[string]documentEntry),
	setEntries: func(map[string]documentEntry) {},
}

var documentCtx = runtime.CreateContext[*documentState](noopState)

var Provider = runtime.Component(func(ctx *Ctx, children []work.Item) work.Node {
	initialState := &documentState{
		entries:    make(map[string]documentEntry),
		setEntries: func(map[string]documentEntry) {},
	}

	state, setState := documentCtx.UseProvider(ctx, initialState)

	state.setEntries = func(newEntries map[string]documentEntry) {
		next := &documentState{
			entries:    newEntries,
			setEntries: state.setEntries,
		}
		setState(next)
	}

	nodes := work.ItemsToNodes(children)
	return &work.Fragment{Children: nodes}
})
