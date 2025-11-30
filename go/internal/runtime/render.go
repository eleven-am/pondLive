package runtime

import (
	"fmt"
	"hash/fnv"
	"reflect"
	"runtime/debug"
	"time"

	"github.com/eleven-am/pondlive/go/internal/work"
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
	inst.mu.Unlock()

	if inst.Parent != nil {
		inst.CombinedContextEpoch = inst.ContextEpoch + inst.Parent.CombinedContextEpoch

		inst.ParentContextEpoch = inst.Parent.CombinedContextEpoch
	} else {
		inst.CombinedContextEpoch = inst.ContextEpoch
	}

	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	var node work.Node
	var renderErr *ComponentError

	func() {
		defer func() {
			if r := recover(); r != nil {
				stack := string(debug.Stack())
				renderErr = &ComponentError{
					Message:     fmt.Sprintf("%v", r),
					StackTrace:  stack,
					ComponentID: inst.ID,
					Phase:       "render",
					HookIndex:   -1,
					Timestamp:   time.Now(),
				}

				if sess != nil && sess.devMode && sess.reporter != nil {
					sess.reporter.ReportDiagnostic(Diagnostic{
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

		node = callComponent(inst.Fn, ctx, inst.Props, inst.InputChildren)
	}()

	if renderErr != nil {
		inst.mu.Lock()
		inst.RenderError = renderErr
		inst.mu.Unlock()

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

func callComponent(fn any, ctx *Ctx, props any, children []work.Node) work.Node {
	f, ok := fn.(func(*Ctx, any, []work.Node) work.Node)
	if !ok {
		panic(fmt.Sprintf("runtime: component signature mismatch, expected func(*Ctx, any, []work.Node) work.Node, got %T", fn))
	}
	return f(ctx, props, children)
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

func (inst *Instance) NotifyContextChange(sess *Session) {
	if inst == nil {
		return
	}

	inst.ContextEpoch++

	if sess == nil {
		return
	}

	for _, child := range inst.Children {
		sess.MarkDirty(child)
	}
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
