package html

// Op is a diff operation applied to a dynamic slot.
type Op interface{ isOp() }

// SetText updates a text slot.
type SetText struct {
	Slot int
	Text string
}

func (SetText) isOp() {}

// SetAttrs updates attributes for a slot.
type SetAttrs struct {
	Slot   int
	Upsert map[string]string
	Remove []string
}

func (SetAttrs) isOp() {}

// ListOp describes list mutations for a dynamic list slot.
type ListOp struct {
	Slot int
	Ops  []ListChildOp
}

func (ListOp) isOp() {}

// ListChildOp is an operation affecting list children.
type ListChildOp interface{ isListChildOp() }

// Ins inserts a row at a position.
type Ins struct {
	Pos int
	Row Row
}

func (Ins) isListChildOp() {}

// Del removes a row by key.
type Del struct{ Key string }

func (Del) isListChildOp() {}

// Mov reorders a row from index From to To.
type Mov struct {
	From int
	To   int
}

func (Mov) isListChildOp() {}

// Set updates a nested slot value within a list row.
type Set struct {
	Key     string
	SubSlot int
	Value   any
}

func (Set) isListChildOp() {}
