package runtime

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/eleven-am/pondlive/internal/work"
)

type ComponentNode[P any] func(*Ctx, P, []work.Item) work.Node

type Instance struct {
	ID   string
	Fn   any
	Key  string
	Name string

	Props             any
	PrevProps         any
	InputChildren     []work.Node
	PrevInputChildren []work.Node
	InputAttrs        []work.Item

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

	RenderError        *Error
	EffectError        *Error
	hasDescendantError bool

	renderCtx    context.Context
	cancelRender context.CancelFunc

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
	HookTypeChannel
	HookTypeUpload
	HookTypePresence
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

func (inst *Instance) BuildComponentPath() []string {
	var path []string
	for current := inst; current != nil; current = current.Parent {
		path = append([]string{current.ID}, path...)
	}
	return path
}

func (inst *Instance) ComponentName() string {
	if inst == nil {
		return ""
	}
	if inst.Name != "" {
		return inst.Name
	}
	if inst.Fn == nil {
		return ""
	}

	fn := reflect.ValueOf(inst.Fn)
	if fn.Kind() != reflect.Func {
		return ""
	}

	ptr := fn.Pointer()
	funcInfo := runtime.FuncForPC(ptr)
	if funcInfo == nil {
		return ""
	}

	fullName := funcInfo.Name()
	if idx := strings.LastIndex(fullName, "."); idx >= 0 {
		return fullName[idx+1:]
	}
	return fullName
}

func (inst *Instance) BuildComponentNamePath() []string {
	var path []string
	for current := inst; current != nil; current = current.Parent {
		name := current.ComponentName()
		if name == "" {
			name = current.ID
		}
		path = append([]string{name}, path...)
	}
	return path
}

func (inst *Instance) GetProviderKeys() []string {
	if inst == nil {
		return nil
	}
	inst.mu.Lock()
	defer inst.mu.Unlock()
	var keys []string
	for key := range inst.Providers {
		keys = append(keys, fmt.Sprintf("%T", key))
	}
	return keys
}

func (inst *Instance) markAncestorsWithError() {
	for ancestor := inst.Parent; ancestor != nil; ancestor = ancestor.Parent {
		ancestor.mu.Lock()
		if ancestor.hasDescendantError {
			ancestor.mu.Unlock()
			break
		}
		ancestor.hasDescendantError = true
		ancestor.mu.Unlock()
	}
}

func (inst *Instance) setRenderError(err *Error) {
	if inst == nil {
		return
	}
	inst.mu.Lock()
	inst.RenderError = err
	inst.mu.Unlock()

	if err != nil {
		inst.markAncestorsWithError()
	}
}

func (inst *Instance) collectChildErrors() []*Error {
	if inst == nil {
		return nil
	}

	var errors []*Error

	inst.mu.Lock()
	hasError := inst.RenderError != nil
	hasDescendantError := inst.hasDescendantError

	if hasError {
		errors = append(errors, inst.RenderError)
	}

	if !hasDescendantError && !hasError {
		inst.mu.Unlock()
		return nil
	}

	children := make([]*Instance, len(inst.Children))
	copy(children, inst.Children)
	inst.mu.Unlock()

	for _, child := range children {
		errors = append(errors, child.collectChildErrors()...)
	}

	return errors
}

func (inst *Instance) setEffectError(err *Error) {
	if inst == nil {
		return
	}
	inst.mu.Lock()
	inst.EffectError = err
	inst.mu.Unlock()

	if err != nil {
		inst.markAncestorsWithError()
	}
}

func (inst *Instance) collectChildEffectErrors() []*Error {
	if inst == nil {
		return nil
	}

	var errors []*Error

	inst.mu.Lock()
	hasError := inst.EffectError != nil
	hasDescendantError := inst.hasDescendantError

	if hasError {
		errors = append(errors, inst.EffectError)
	}

	if !hasDescendantError && !hasError {
		inst.mu.Unlock()
		return nil
	}

	children := make([]*Instance, len(inst.Children))
	copy(children, inst.Children)
	inst.mu.Unlock()

	for _, child := range children {
		errors = append(errors, child.collectChildEffectErrors()...)
	}

	return errors
}

func (inst *Instance) clearChildEffectErrors() {
	if inst == nil {
		return
	}

	inst.mu.Lock()
	inst.EffectError = nil
	inst.hasDescendantError = false
	children := make([]*Instance, len(inst.Children))
	copy(children, inst.Children)
	inst.mu.Unlock()

	for _, child := range children {
		child.clearChildEffectErrors()
	}
}

func (inst *Instance) clearChildErrors() {
	if inst == nil {
		return
	}

	inst.mu.Lock()
	inst.RenderError = nil
	inst.hasDescendantError = false
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
