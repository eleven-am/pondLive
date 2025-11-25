package runtime2

import (
	"sync"
	"time"

	"github.com/eleven-am/pondlive/go/internal/work"
)

// Component is a function that renders a node tree.
// P is the props type, children are passed from the parent component.
type Component[P any] func(*Ctx, P, []work.Node) work.Node

// ComponentError captures error details for error boundaries.
type ComponentError struct {
	Message     string    // Error message
	StackTrace  string    // Full stack trace
	ComponentID string    // Which component failed
	Phase       string    // "render", "effect", "memo", "handler"
	HookIndex   int       // Which hook failed (if applicable, -1 otherwise)
	Timestamp   time.Time // When the error occurred
}

// Instance represents a component instance with identity and state.
type Instance struct {
	ID  string // Unique identity for reconciliation
	Fn  any    // Component function
	Key string // Optional user-provided key

	Props         any         // Current props
	PrevProps     any         // Previous props for comparison
	InputChildren []work.Node // Children passed from parent

	HookFrame []HookSlot  // Hook state storage (indexed)
	Parent    *Instance   // Parent component instance
	Children  []*Instance // Child component instances

	WorkTree work.Node // Last rendered work output
	ViewNode any       // Last rendered view node (view.Element)
	Wrapper  any       // Wrapper node for components (view.Element)

	Dirty             bool // Needs re-render
	RenderedThisFlush bool // Rendered during current flush cycle

	// Context management
	Providers            map[any]any // Context providers at this level
	ContextEpoch         int         // Incremented when context changes
	ParentContextEpoch   int         // Parent's context epoch when mounted
	CombinedContextEpoch int         // Sum of context epochs up the tree
	ProviderSeq          int         // Provider sequence counter

	// Child tracking during render
	ChildRenderIndex   int             // Current child position for auto-keys
	ReferencedChildren map[string]bool // Children referenced this render

	// Error boundary support
	RenderError *ComponentError // Error from this component's render

	// Cleanup management
	cleanups   []func() // Generic cleanup functions registered by hooks
	cleanupsMu sync.Mutex

	mu sync.Mutex // Protects concurrent access
}

// HookType identifies the type of hook.
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

// HookSlot stores state for a single hook call.
type HookSlot struct {
	Type  HookType // UseState, UseRef, UseMemo, UseEffect, UseElement, UseScript
	Value any      // The hook's stored value
	Deps  []any    // Dependencies for memo/effect
}

// RegisterCleanup registers a cleanup function to be called when the instance unmounts.
// Hooks should use this to register their cleanup logic (e.g., unsubscribe, remove from registries).
func (inst *Instance) RegisterCleanup(fn func()) {
	if inst == nil || fn == nil {
		return
	}

	inst.cleanupsMu.Lock()
	inst.cleanups = append(inst.cleanups, fn)
	inst.cleanupsMu.Unlock()
}

// findChildError recursively checks this instance and all children for errors.
// Returns the first error found in the tree.
func (inst *Instance) findChildError() *ComponentError {
	if inst == nil {
		return nil
	}

	inst.mu.Lock()
	defer inst.mu.Unlock()

	if inst.RenderError != nil {
		return inst.RenderError
	}

	for _, child := range inst.Children {
		if err := child.findChildError(); err != nil {
			return err
		}
	}

	return nil
}

// clearChildErrors recursively clears all errors from this instance and its children.
// Used by error boundaries to "consume" errors after catching them.
func (inst *Instance) clearChildErrors() {
	if inst == nil {
		return
	}

	inst.mu.Lock()
	inst.RenderError = nil
	inst.mu.Unlock()

	for _, child := range inst.Children {
		child.clearChildErrors()
	}
}
