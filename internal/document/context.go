package document

import (
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type Ctx = runtime.Ctx

type documentState struct {
	entriesRef *runtime.Ref[map[string]documentEntry]
	entries    map[string]documentEntry
	setEntries func(map[string]documentEntry)
}

var noopEntriesRef = &runtime.Ref[map[string]documentEntry]{Current: make(map[string]documentEntry)}

var noopState = &documentState{
	entriesRef: noopEntriesRef,
	entries:    make(map[string]documentEntry),
	setEntries: func(map[string]documentEntry) {},
}

var documentCtx = runtime.CreateContext[*documentState](noopState)

var Provider = runtime.Component(func(ctx *Ctx, children []work.Item) work.Node {
	entriesRef := runtime.UseRef(ctx, make(map[string]documentEntry))

	initialState := &documentState{
		entriesRef: entriesRef,
		entries:    entriesRef.Current,
		setEntries: func(map[string]documentEntry) {},
	}

	state, setState := documentCtx.UseProvider(ctx, initialState)

	state.entriesRef = entriesRef
	state.entries = entriesRef.Current

	state.setEntries = func(newEntries map[string]documentEntry) {
		entriesRef.Current = newEntries
		next := &documentState{
			entriesRef: entriesRef,
			entries:    newEntries,
			setEntries: state.setEntries,
		}
		setState(next)
	}

	nodes := work.ItemsToNodes(children)
	return &work.Fragment{Children: nodes}
})
