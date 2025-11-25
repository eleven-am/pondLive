package metadata

// StyleRule represents a single CSS rule with selector and properties.
type StyleRule struct {
	Selector string            `json:"selector"`
	Props    map[string]string `json:"props"`
}

// MediaBlock represents a @media query block containing CSS rules.
type MediaBlock struct {
	Query string      `json:"query"`
	Rules []StyleRule `json:"rules"`
}

// Stylesheet represents a structured CSS stylesheet for a component.
type Stylesheet struct {
	Rules       []StyleRule  `json:"rules,omitempty"`
	MediaBlocks []MediaBlock `json:"mediaBlocks,omitempty"`
	Hash        string       `json:"hash,omitempty"` // Component hash for class scoping
}
