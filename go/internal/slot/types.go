package slot

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// SlotContent holds the nodes for a single slot.
type SlotContent struct {
	nodes []*dom.StructuredNode
}

// SlotMap organizes slot content by name.
// Holds both provided slots and fallback content.
// Preserves declaration order via slotOrder slice.
type SlotMap struct {
	slots     map[string]*SlotContent
	fallbacks map[string]*SlotContent
	slotOrder []string // Preserves declaration order of slot names
}

// scopedSlotMap organizes scoped slot functions by name.
// Supports multiple functions per slot name (appending behavior like regular slots).
type scopedSlotMap[T any] struct {
	slots map[string][]func(T) *dom.StructuredNode
}

// Metadata keys used to mark slot nodes
const (
	slotNameKey       = "slot:name"
	scopedSlotNameKey = "scoped-slot:name"
	scopedSlotFuncKey = "scoped-slot:fn"
)
