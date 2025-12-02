package runtime

import (
	"sync"
	"time"

	"github.com/eleven-am/pondlive/go/internal/work"
)

type ComponentNode[P any] func(*Ctx, P, []work.Node) work.Node

type ComponentError struct {
	Message     string
	StackTrace  string
	ComponentID string
	Phase       string
	HookIndex   int
	Timestamp   time.Time
}

type Instance struct {
	ID  string
	Fn  any
	Key string

	Props         any
	PrevProps     any
	InputChildren []work.Node

	HookFrame []HookSlot
	Parent    *Instance
	Children  []*Instance

	WorkTree work.Node
	ViewNode any
	Wrapper  any

	Dirty             bool
	RenderedThisFlush bool

	Providers            map[any]any
	ContextEpoch         int
	ParentContextEpoch   int
	CombinedContextEpoch int
	ProviderSeq          int

	ContextEpochs         map[contextID]int
	CombinedContextEpochs map[contextID]int
	ContextDeps           map[contextID]struct{}
	SeenContextEpochs     map[contextID]int

	ChildRenderIndex   int
	ReferencedChildren map[string]bool
	NextHandlerIndex   int

	RenderError *ComponentError

	cleanups   []func()
	cleanupsMu sync.Mutex

	mu sync.Mutex
}

type HookType int

const (
	HookTypeState HookType = iota
	HookTypeRef
	HookTypeMemo
	HookTypeEffect
	HookTypeElement
	HookTypeHandler
	HookTypeScript
	HookTypeStyles
	HookTypeErrorBoundary
)

type HookSlot struct {
	Type  HookType
	Value any
	Deps  []any
}

func (inst *Instance) RegisterCleanup(fn func()) {
	if inst == nil || fn == nil {
		return
	}

	inst.cleanupsMu.Lock()
	inst.cleanups = append(inst.cleanups, fn)
	inst.cleanupsMu.Unlock()
}

func (inst *Instance) findChildError() *ComponentError {
	if inst == nil {
		return nil
	}

	inst.mu.Lock()
	if inst.RenderError != nil {
		err := inst.RenderError
		inst.mu.Unlock()
		return err
	}
	children := make([]*Instance, len(inst.Children))
	copy(children, inst.Children)
	inst.mu.Unlock()

	for _, child := range children {
		if err := child.findChildError(); err != nil {
			return err
		}
	}

	return nil
}

func (inst *Instance) clearChildErrors() {
	if inst == nil {
		return
	}

	inst.mu.Lock()
	inst.RenderError = nil
	children := make([]*Instance, len(inst.Children))
	copy(children, inst.Children)
	inst.mu.Unlock()

	for _, child := range children {
		child.clearChildErrors()
	}
}

func (inst *Instance) ensureContextEpochEntry(id contextID) {
	if inst == nil {
		return
	}
	if inst.ContextEpochs == nil {
		inst.ContextEpochs = make(map[contextID]int)
	}
	if _, ok := inst.ContextEpochs[id]; !ok {
		inst.ContextEpochs[id] = 0
	}
}

func (inst *Instance) buildCombinedContextEpochs() map[contextID]int {
	combined := make(map[contextID]int)

	if inst != nil && inst.Parent != nil {
		for id, epoch := range inst.Parent.CombinedContextEpochs {
			combined[id] = epoch
		}
	}

	if inst != nil && inst.ContextEpochs != nil {
		for id, epoch := range inst.ContextEpochs {
			combined[id] = epoch
		}
	}

	return combined
}
