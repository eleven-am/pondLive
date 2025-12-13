package runtime

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"runtime/debug"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/work"
)

func (inst *Instance) Render(sess *Session) work.Node {
	if inst == nil || inst.Fn == nil {
		return nil
	}

	inst.RenderedThisFlush = true
	inst.Dirty = false

	inst.mu.Lock()
	inst.ChildRenderIndex = 0
	inst.ProviderSeq = 0
	inst.ReferencedChildren = make(map[string]bool)
	inst.ContextDeps = nil
	inst.CombinedContextEpochs = inst.buildCombinedContextEpochs()
	if inst.Parent != nil {
		inst.CombinedContextEpoch = inst.ContextEpoch + inst.Parent.CombinedContextEpoch
		inst.ParentContextEpoch = inst.Parent.CombinedContextEpoch
	} else {
		inst.CombinedContextEpoch = inst.ContextEpoch
	}
	inst.mu.Unlock()

	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	var node work.Node
	var renderErr *Error

	func() {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())

				var parentID string
				if inst.Parent != nil {
					parentID = inst.Parent.ID
				}

				var sessionID string
				var devMode bool
				if sess != nil {
					sessionID = sess.SessionID
					devMode = sess.devMode
				}

				ectx := ErrorContext{
					SessionID:         sessionID,
					ComponentID:       inst.ID,
					ComponentName:     inst.ComponentName(),
					ParentID:          parentID,
					ComponentPath:     inst.BuildComponentPath(),
					ComponentNamePath: inst.BuildComponentNamePath(),
					Phase:             "render",
					HookIndex:         -1,
					HookCount:         len(inst.HookFrame),
					Props:             inst.Props,
					ProviderKeys:      inst.GetProviderKeys(),
					DevMode:           devMode,
				}
				renderErr = NewComponentErrorWithContext(ErrCodeRender, fmt.Sprintf("%v", r), stack, ectx)
				renderErr.Meta["panic_value"] = r

				if sess != nil && sess.devMode && sess.Bus != nil {
					sess.Bus.ReportDiagnostic(protocol.Diagnostic{
						Phase:      fmt.Sprintf("render:%s", inst.ID),
						Message:    fmt.Sprintf("panic: %v", r),
						StackTrace: stack,
						Metadata: map[string]any{
							"component_id": inst.ID,
							"panic_value":  r,
						},
					})
				}
			}
		}()

		combinedChildren := make([]work.Item, 0, len(inst.InputChildren)+len(inst.InputAttrs))
		for _, child := range inst.InputChildren {
			combinedChildren = append(combinedChildren, child)
		}
		combinedChildren = append(combinedChildren, inst.InputAttrs...)

		node = callComponent(inst.Fn, ctx, inst.Props, combinedChildren)
	}()

	if renderErr != nil {
		inst.setRenderError(renderErr)
		inst.WorkTree = nil
		return nil
	}

	inst.mu.Lock()
	if inst.RenderError != nil && inst.RenderError.Phase == "render" {
		inst.RenderError = nil
	}
	inst.mu.Unlock()

	inst.WorkTree = node

	return node
}

func callComponent(fn any, ctx *Ctx, props any, children []work.Item) work.Node {
	if f, ok := fn.(func(*Ctx, any, []work.Item) work.Node); ok {
		return f(ctx, props, children)
	}

	fnVal := reflect.ValueOf(fn)
	fnType := fnVal.Type()

	if fnType.Kind() != reflect.Func {
		panic(fmt.Sprintf("runtime: component must be a function, got %T", fn))
	}

	if fnType.NumOut() != 1 {
		panic(fmt.Sprintf("runtime: component must return exactly one value, got %d", fnType.NumOut()))
	}

	numIn := fnType.NumIn()
	args := make([]reflect.Value, numIn)

	for i := 0; i < numIn; i++ {
		paramType := fnType.In(i)

		switch i {
		case 0:
			if paramType == reflect.TypeOf((*Ctx)(nil)) {
				args[i] = reflect.ValueOf(ctx)
			} else {
				panic(fmt.Sprintf("runtime: first parameter must be *Ctx, got %v", paramType))
			}
		case 1:
			if props == nil {
				args[i] = reflect.Zero(paramType)
			} else {
				propsVal := reflect.ValueOf(props)
				if propsVal.Type().AssignableTo(paramType) {
					args[i] = propsVal
				} else if propsVal.Type().ConvertibleTo(paramType) {
					args[i] = propsVal.Convert(paramType)
				} else {
					panic(fmt.Sprintf("runtime: props type %T not assignable to parameter type %v", props, paramType))
				}
			}
		case 2:
			if paramType == reflect.TypeOf([]work.Item{}) {
				args[i] = reflect.ValueOf(children)
			} else {
				panic(fmt.Sprintf("runtime: third parameter must be []work.Item, got %v", paramType))
			}
		default:
			panic(fmt.Sprintf("runtime: component has too many parameters (%d)", numIn))
		}
	}

	results := fnVal.Call(args)
	if results[0].IsNil() {
		return nil
	}
	return results[0].Interface().(work.Node)
}

func (inst *Instance) BeginRender() {
	if inst == nil {
		return
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	inst.ChildRenderIndex = 0
	inst.ProviderSeq = 0
	inst.ReferencedChildren = make(map[string]bool)
}

func (inst *Instance) EndRender() {

}

func (inst *Instance) SetDirty(dirty bool) {
	if inst == nil {
		return
	}
	inst.Dirty = dirty
}

func (inst *Instance) NotifyContextChange(sess *Session, id contextID) {
	if inst == nil {
		return
	}

	inst.mu.Lock()
	if inst.ContextEpochs == nil {
		inst.ContextEpochs = make(map[contextID]int)
	}
	inst.ContextEpochs[id]++
	inst.CombinedContextEpochs = inst.buildCombinedContextEpochs()
	inst.ContextEpoch++
	if inst.Parent != nil {
		inst.CombinedContextEpoch = inst.ContextEpoch + inst.Parent.CombinedContextEpoch
	} else {
		inst.CombinedContextEpoch = inst.ContextEpoch
	}
	inst.mu.Unlock()

	if sess == nil {
		return
	}

	sess.MarkDirty(inst)
}

func (inst *Instance) EnsureChild(sess *Session, fn any, key string, props any, children []work.Node) *Instance {
	if inst == nil {
		return nil
	}

	childID := buildComponentID(inst, fn, key)
	inst.mu.Lock()

	if inst.ReferencedChildren != nil {
		inst.ReferencedChildren[childID] = true
	}

	var child *Instance
	for _, c := range inst.Children {
		if c.ID == childID {
			child = c
			break
		}
	}

	if child == nil {
		child = &Instance{
			ID:                 childID,
			Fn:                 fn,
			Key:                key,
			Parent:             inst,
			ParentContextEpoch: inst.CombinedContextEpoch,
			HookFrame:          []HookSlot{},
			Children:           []*Instance{},
		}
		inst.Children = append(inst.Children, child)

		if sess != nil && sess.Components != nil {
			sess.Components[childID] = child
		}
	}

	inst.mu.Unlock()

	child.PrevProps = child.Props
	child.Props = props
	child.PrevInputChildren = child.InputChildren
	child.InputChildren = children

	return child
}

func buildComponentID(parent *Instance, fn any, key string) string {
	if parent == nil {
		return "root"
	}

	fnPtr := reflect.ValueOf(fn).Pointer()

	componentKey := key
	if componentKey == "" {
		parent.mu.Lock()
		componentKey = fmt.Sprintf("_%d", parent.ChildRenderIndex)
		parent.ChildRenderIndex++
		parent.mu.Unlock()
	}

	hasher := fnv.New64a()
	hasher.Write([]byte(parent.ID))
	hasher.Write([]byte{0})

	hasher.Write([]byte(fmt.Sprintf("%x", fnPtr)))
	hasher.Write([]byte{0})
	hasher.Write([]byte(componentKey))

	return fmt.Sprintf("c%016x", hasher.Sum64())
}
