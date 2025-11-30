package css

// Declaration stores a single CSS declaration in order.
type Declaration struct {
	Property string
	Value    string
}

// PropertyMap stores CSS property declarations for a selector.
// Keys are property names (e.g., "color"), values are raw values (e.g., "#fff").
// This map reflects the last value for a property; see Decls for original order.
type PropertyMap map[string]string

// SelectorBlock represents a selector (optionally scoped) and its declarations.
type SelectorBlock struct {
	Selector string
	Props    PropertyMap
	Decls    []Declaration
}

// Stylesheet is a structured representation of CSS rules scoped to a component.
// Selectors include scoped hashes (e.g., `.btn-abc123`).
type Stylesheet struct {
	Rules        []SelectorBlock
	MediaRules   []MediaRule
	SelectorHash string
}

// MediaRule captures scoped selectors inside an @media block.
type MediaRule struct {
	Query string
	Rules []SelectorBlock
}
