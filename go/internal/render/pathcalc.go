package render

type PathCalculator struct {
	componentStack []componentFrame
	componentPath  []int
	listPaths      []ListPath
	componentPaths []ComponentPath
}

func NewPathCalculator() *PathCalculator {
	return &PathCalculator{
		componentStack: make([]componentFrame, 0),
		componentPath:  make([]int, 0),
		listPaths:      make([]ListPath, 0),
		componentPaths: make([]ComponentPath, 0),
	}
}

func (pc *PathCalculator) ListPaths() []ListPath {
	return append([]ListPath(nil), pc.listPaths...)
}

func (pc *PathCalculator) ComponentPaths() []ComponentPath {
	return append([]ComponentPath(nil), pc.componentPaths...)
}

func (pc *PathCalculator) RecordComponentTraversal() {
	if len(pc.componentStack) == 0 {
		return
	}
	frame := &pc.componentStack[len(pc.componentStack)-1]
	if frame.startPath == nil {
		frame.startPath = append([]int(nil), pc.componentPath...)
	}
	frame.endPath = append([]int(nil), pc.componentPath...)
}

func (pc *PathCalculator) PushComponent(id string) {
	parentID := ""
	if len(pc.componentStack) > 0 {
		parentID = pc.componentStack[len(pc.componentStack)-1].id
	}
	frame := componentFrame{
		id:       id,
		parentID: parentID,
		prevPath: append([]int(nil), pc.componentPath...),
		basePath: pc.currentAbsolutePath(),
	}
	pc.componentStack = append(pc.componentStack, frame)
	pc.componentPath = pc.componentPath[:0]
}

func (pc *PathCalculator) PopComponent() {
	if len(pc.componentStack) == 0 {
		return
	}
	last := len(pc.componentStack) - 1
	frame := pc.componentStack[last]
	pc.componentStack = pc.componentStack[:last]
	pc.componentPath = append([]int(nil), frame.prevPath...)
	if frame.id != "" {
		pc.componentPaths = append(pc.componentPaths, ComponentPath{
			ComponentID: frame.id,
			ParentID:    frame.parentID,
			ParentPath:  rangeSegments(frame.basePath),
			FirstChild:  domSegments(frame.startPath),
			LastChild:   domSegments(frame.endPath),
		})
	}
}

func (pc *PathCalculator) CurrentComponentID() string {
	if len(pc.componentStack) == 0 {
		return ""
	}
	return pc.componentStack[len(pc.componentStack)-1].id
}

func (pc *PathCalculator) CurrentComponentPath() []int {
	if len(pc.componentStack) == 0 {
		return nil
	}
	return append([]int(nil), pc.componentPath...)
}

func (pc *PathCalculator) CurrentComponentBasePath() []int {
	if len(pc.componentStack) == 0 {
		return nil
	}
	return append([]int(nil), pc.componentStack[len(pc.componentStack)-1].basePath...)
}

func (pc *PathCalculator) currentAbsolutePath() []int {
	base := pc.CurrentComponentBasePath()
	if len(pc.componentPath) == 0 {
		return base
	}
	combined := append([]int(nil), base...)
	combined = append(combined, pc.componentPath...)
	return combined
}

func (pc *PathCalculator) RecordListPath(listSlot int, frame *elementFrame) {
	if frame != nil && frame.componentID != "" {
		pc.listPaths = append(pc.listPaths, ListPath{
			Slot:        listSlot,
			ComponentID: frame.componentID,
			Path:        combineTypedPath(frame.basePath, frame.componentPath),
		})
		return
	}

	if componentID := pc.CurrentComponentID(); componentID != "" {
		pc.listPaths = append(pc.listPaths, ListPath{
			Slot:        listSlot,
			ComponentID: componentID,
			AtRoot:      true,
		})
	}
}

func (pc *PathCalculator) IncrementPath() {
	if len(pc.componentPath) > 0 {
		pc.componentPath[len(pc.componentPath)-1]++
	} else {
		pc.componentPath = append(pc.componentPath, 0)
	}
}

func (pc *PathCalculator) AppendToPath(index int) {
	pc.componentPath = append(pc.componentPath, index)
}

func (pc *PathCalculator) TrimPath(n int) {
	if n >= len(pc.componentPath) {
		pc.componentPath = pc.componentPath[:0]
	} else {
		pc.componentPath = pc.componentPath[:len(pc.componentPath)-n]
	}
}

func (pc *PathCalculator) ListPathsLen() int {
	return len(pc.listPaths)
}

func (pc *PathCalculator) ComponentPathsLen() int {
	return len(pc.componentPaths)
}

func (pc *PathCalculator) ExtractListPaths(startIdx int) []ListPath {
	endIdx := len(pc.listPaths)
	if endIdx <= startIdx {
		return nil
	}
	return append([]ListPath(nil), pc.listPaths[startIdx:endIdx]...)
}

func (pc *PathCalculator) ExtractComponentPaths(startIdx int) []ComponentPath {
	endIdx := len(pc.componentPaths)
	if endIdx <= startIdx {
		return nil
	}
	return append([]ComponentPath(nil), pc.componentPaths[startIdx:endIdx]...)
}

func (pc *PathCalculator) CloneState() *PathCalculator {
	clone := NewPathCalculator()
	if len(pc.componentStack) > 0 {
		clone.componentStack = make([]componentFrame, len(pc.componentStack))
		for i, frame := range pc.componentStack {
			clone.componentStack[i] = componentFrame{
				id:        frame.id,
				parentID:  frame.parentID,
				prevPath:  append([]int(nil), frame.prevPath...),
				basePath:  append([]int(nil), frame.basePath...),
				startPath: append([]int(nil), frame.startPath...),
				endPath:   append([]int(nil), frame.endPath...),
			}
		}
	}
	if len(pc.componentPath) > 0 {
		clone.componentPath = append([]int(nil), pc.componentPath...)
	}
	return clone
}

func (pc *PathCalculator) mergeFrom(other *PathCalculator, dynamicsOffset int) {
	for _, listPath := range other.listPaths {
		listPath.Slot += dynamicsOffset
		pc.listPaths = append(pc.listPaths, listPath)
	}
	pc.componentPaths = append(pc.componentPaths, other.componentPaths...)
}
