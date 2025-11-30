package metadata

type StyleRule struct {
	Selector string            `json:"selector"`
	Props    map[string]string `json:"props"`
}

type MediaBlock struct {
	Query string      `json:"query"`
	Rules []StyleRule `json:"rules"`
}

type Stylesheet struct {
	Rules       []StyleRule  `json:"rules,omitempty"`
	MediaBlocks []MediaBlock `json:"mediaBlocks,omitempty"`
	Hash        string       `json:"hash,omitempty"`
}
