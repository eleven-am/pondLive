package reactive

func (g *Graph) Abandon() []*Owner {
	var owners []*Owner
	seen := make(map[*Owner]struct{})
	for q := range g.heaps {
		h := &g.heaps[q]
		for _, n := range h.items {
			n.flags &^= fInHeap | fNotified
			if n.flags&fDisposed != 0 || n.owner == nil {
				continue
			}
			if _, dup := seen[n.owner]; !dup {
				seen[n.owner] = struct{}{}
				owners = append(owners, n.owner)
			}
		}
		h.items = nil
		g.live[q] = 0
	}
	return owners
}

func (o *Owner) DropUnit() bool {
	if o.disposed || o.slot >= 0 || o.parent == nil {
		return false
	}
	saved := o.N
	o.N = 0
	ok := o.Droppable()
	o.N = saved
	if !ok {
		return false
	}
	o.detach()
	o.disposed = true
	o.parent, o.kids = nil, nil
	return true
}

func (g *Graph) NodeCount() uint64 { return g.nodeSeq }
