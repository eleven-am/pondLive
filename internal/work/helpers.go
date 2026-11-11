package work

type noopItem struct{}

func (noopItem) applyTo(*Element) {}

func Skip() Item { return noopItem{} }

func NodesToItems[T Node](nodes []T) []Item {
	items := make([]Item, 0, len(nodes))
	for _, n := range nodes {
		if any(n) != nil {
			items = append(items, n)
		}
	}
	return items
}

func ItemsToNodes(items []Item) []Node {
	nodes := make([]Node, 0, len(items))
	for _, item := range items {
		if node, ok := item.(Node); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}
