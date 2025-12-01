package metatags

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
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

var Provider = runtime.Component(func(ctx *Ctx, children []work.Node) work.Node {
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

	return &work.Fragment{Children: children}
})
