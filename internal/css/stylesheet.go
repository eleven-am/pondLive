package css

type Declaration struct {
	Property string
	Value    string
}

type SelectorBlock struct {
	Selector string
	Decls    []Declaration
}

type Stylesheet struct {
	Rules        []SelectorBlock
	MediaRules   []MediaRule
	OtherBlocks  []string
	Keyframes    []KeyframesBlock
	SelectorHash string
}

type MediaRule struct {
	Query string
	Rules []SelectorBlock
}

type KeyframesBlock struct {
	Name  string
	Steps []KeyframesStep
}

type KeyframesStep struct {
	Selector string
	Decls    []Declaration
}

type Scoped struct {
	Hash string
	Text string
}

func (s *Scoped) Class(name string) string {
	if s == nil || s.Hash == "" {
		return name
	}
	return name + "-" + s.Hash
}

func Scope(key, raw string) *Scoped {
	parsed := ParseAndScope(raw, key)
	return &Scoped{Hash: parsed.SelectorHash, Text: render(parsed)}
}
