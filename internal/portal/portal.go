package portal

import "github.com/eleven-am/pondlive/internal/work"

func Portal(children ...work.Item) *work.PortalNode {
	return work.NewPortal(children...)
}

func Target() *work.PortalTarget {
	return work.NewPortalTarget()
}
