package runtime

func RootKidCount(s *Session) int {
	n := 0
	for _, k := range s.root.Kids() {
		if k != nil {
			n++
		}
	}
	return n
}

func MountRuns(s *Session) int { return s.mountBodyRuns }

type GraphCount struct {
	Handlers, Failed int
	Nodes            uint64
}

func GraphCounts(s *Session) GraphCount {
	return GraphCount{Handlers: len(s.handlers), Failed: len(s.failed), Nodes: s.graph.NodeCount()}
}
