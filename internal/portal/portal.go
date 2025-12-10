package portal

import "github.com/eleven-am/pondlive/internal/work"

func Portal(children ...work.Item) *work.PortalNode {
	nodes := work.ItemsToNodes(children)
	return &work.PortalNode{Children: nodes}
}

func Target() *work.PortalTarget {
	return &work.PortalTarget{}
}
