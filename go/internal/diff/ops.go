package diff

import (
	"encoding/json"

	render "github.com/eleven-am/go/pondlive/internal/render"
)

// Op represents a diff operation applied to a dynamic slot.
type Op interface{ isOp() }

// SetText updates a text slot.
type SetText struct {
	Slot int
	Text string
}

func (SetText) isOp() {}

func (op SetText) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{"setText", op.Slot, op.Text})
}

// SetAttrs updates attributes for a slot.
type SetAttrs struct {
	Slot   int
	Upsert map[string]string
	Remove []string
}

func (SetAttrs) isOp() {}

func (op SetAttrs) MarshalJSON() ([]byte, error) {
	upsert := op.Upsert
	if upsert == nil {
		upsert = map[string]string{}
	}
	remove := op.Remove
	if remove == nil {
		remove = []string{}
	}
	return json.Marshal([]any{"setAttrs", op.Slot, upsert, remove})
}

// ListChildOp is an operation affecting list children.
type ListChildOp interface{ isListChildOp() }

// Ins inserts a row at a position.
type Ins struct {
	Pos int
	Row render.Row
}

func (Ins) isListChildOp() {}

func (op Ins) MarshalJSON() ([]byte, error) {
	payload := map[string]any{
		"key":  op.Row.Key,
		"html": op.Row.HTML,
	}
	if len(op.Row.Slots) > 0 {
		payload["slots"] = op.Row.Slots
	}
	return json.Marshal([]any{"ins", op.Pos, payload})
}

// Del removes a row by key.
type Del struct{ Key string }

func (Del) isListChildOp() {}

func (op Del) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{"del", op.Key})
}

// Mov reorders a row from index From to To.
type Mov struct {
	From int
	To   int
}

func (Mov) isListChildOp() {}

func (op Mov) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{"mov", op.From, op.To})
}

// Set updates a nested slot value within a list row.
type Set struct {
	Key     string
	SubSlot int
	Value   any
}

func (Set) isListChildOp() {}

func (op Set) MarshalJSON() ([]byte, error) {
	return json.Marshal([]any{"set", op.Key, op.SubSlot, op.Value})
}

// ListOp describes list mutations for a dynamic list slot.
type List struct {
	Slot int
	Ops  []ListChildOp
}

func (List) isOp() {}

func (op List) MarshalJSON() ([]byte, error) {
	arr := make([]any, 2+len(op.Ops))
	arr[0] = "list"
	arr[1] = op.Slot
	for i, child := range op.Ops {
		arr[2+i] = child
	}
	return json.Marshal(arr)
}
