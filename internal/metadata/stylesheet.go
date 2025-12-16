package metadata

type Declaration struct {
	Property string `json:"property"`
	Value    string `json:"value"`
}

type StyleRule struct {
	Selector string        `json:"selector"`
	Decls    []Declaration `json:"decls"`
}

type MediaBlock struct {
	Query string      `json:"query"`
	Rules []StyleRule `json:"rules"`
}

type KeyframesStep struct {
	Selector string        `json:"selector"`
	Decls    []Declaration `json:"decls"`
}

type KeyframesBlock struct {
	Name  string          `json:"name"`
	Steps []KeyframesStep `json:"steps"`
}

type Stylesheet struct {
	Rules       []StyleRule      `json:"rules,omitempty"`
	MediaBlocks []MediaBlock     `json:"mediaBlocks,omitempty"`
	Keyframes   []KeyframesBlock `json:"keyframes,omitempty"`
	OtherBlocks []string         `json:"otherBlocks,omitempty"`
	Hash        string           `json:"hash,omitempty"`
}
