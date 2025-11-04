package html

// DynKind enumerates dynamic slot kinds.
type DynKind uint8

const (
	DynText DynKind = iota
	DynAttrs
	DynList
)

// Dyn represents a dynamic slot value.
type Dyn struct {
	Kind DynKind

	Text  string
	Attrs map[string]string
	List  []Row
}

// Row represents a keyed row inside a dynamic list slot.
type Row struct {
	Key   string
	Slots []int
}

// Statics represents immutable HTML segments between dynamic slots.
type Statics []string

// Structured represents the structured render output of a node tree.
type Structured struct {
	S Statics
	D []Dyn
}
