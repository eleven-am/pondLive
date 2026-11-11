package reactive

import "fmt"

type Kind uint8

const (
	KindRender Kind = iota
	KindWatch
	KindSink
)

type Violation uint8

const (
	WriteInComputed Violation = iota + 1
	WriteInBody
	WriteInSerialize
	GetInBody
	OffActorRead
	OffActorWrite
	CycleDetected
	NoLiveOwner
	CreateInSerialize
	SealedOwner
)

var violationText = [...]string{
	WriteInComputed:   "signal written inside a computed",
	WriteInBody:       "signal written inside a component body",
	WriteInSerialize:  "signal written during a serialize pass",
	GetInBody:         "untracked Get inside a component body",
	OffActorRead:      "signal read off the actor goroutine",
	OffActorWrite:     "signal written off the actor goroutine with no poster",
	CycleDetected:     "cycle detected in the dependency graph",
	NoLiveOwner:       "computation created without a live owner",
	CreateInSerialize: "node created during a serialize pass",
	SealedOwner:       "hook created on a sealed owner",
}

type DevError struct {
	Violation Violation
	Component string
}

func (e *DevError) Error() string {
	if e.Violation == SealedOwner && e.Component != "" {
		return fmt.Sprintf("reactive: a hook was called with the ctx of %q after its body finished; an outer ctx was used inside a Show/Switch arm, a For row or a handler. Hooks may only be called at the top of a component body with that component's own ctx: declare the hook there, or make the row a PropsComponent and use its ctx", e.Component)
	}
	return "reactive: " + violationText[e.Violation]
}

type DeterminismError struct {
	Component string
	Cell      int
	Detail    string
}

func (e *DeterminismError) Error() string {
	return fmt.Sprintf("reactive: determinism violation in %s at cell %d: %s", e.Component, e.Cell, e.Detail)
}

type BudgetKind uint8

const (
	BudgetPasses BudgetKind = iota + 1
	BudgetRounds
)

type BudgetError struct {
	Kind  BudgetKind
	Owner *Owner
}

func (e *BudgetError) Error() string {
	if e.Kind == BudgetPasses {
		return fmt.Sprintf("reactive: cascade budget of %d passes exceeded", CascadeBudget)
	}
	name := ""
	if e.Owner != nil {
		name = e.Owner.Info.Name
	}
	return fmt.Sprintf("reactive: render loop limit of %d rounds exceeded in %q", MaxRenderIterations, name)
}
