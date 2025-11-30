package css

type Declaration struct {
	Property string
	Value    string
}

type PropertyMap map[string]string

type SelectorBlock struct {
	Selector string
	Props    PropertyMap
	Decls    []Declaration
}

type Stylesheet struct {
	Rules        []SelectorBlock
	MediaRules   []MediaRule
	OtherBlocks  []string
	SelectorHash string
}

type MediaRule struct {
	Query string
	Rules []SelectorBlock
}
