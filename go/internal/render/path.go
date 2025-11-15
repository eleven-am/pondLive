package render

import (
	"fmt"
	"strconv"
)

type PathKind uint8

const (
	PathRangeOffset PathKind = iota
	PathDomChild
)

type PathSegment struct {
	Kind  PathKind
	Index int
}

func (p PathSegment) Clone() PathSegment {
	return PathSegment{Kind: p.Kind, Index: p.Index}
}

func (p PathSegment) String() string {
	prefix := "c:"
	if p.Kind == PathRangeOffset {
		prefix = "r:"
	}
	return prefix + strconv.Itoa(p.Index)
}

func validatePath(segments []PathSegment) error {
	if len(segments) == 0 {
		return fmt.Errorf("path cannot be empty")
	}

	if segments[0].Kind != PathRangeOffset {
		return fmt.Errorf("path must start with PathRangeOffset, got %v", segments[0].Kind)
	}

	for i := 1; i < len(segments); i++ {
		if segments[i].Kind == PathRangeOffset && segments[i-1].Kind == PathDomChild {
			return fmt.Errorf("invalid path: PathRangeOffset cannot follow PathDomChild at index %d", i)
		}
	}

	return nil
}

func rangeSegments(path []int) []PathSegment {
	return toSegments(path, PathRangeOffset)
}

func domSegments(path []int) []PathSegment {
	return toSegments(path, PathDomChild)
}

func toSegments(path []int, kind PathKind) []PathSegment {
	if len(path) == 0 {
		return nil
	}
	segs := make([]PathSegment, len(path))
	for i, idx := range path {
		segs[i] = PathSegment{Kind: kind, Index: idx}
	}
	return segs
}

func combineTypedPath(rangePath, domPath []int) []PathSegment {
	rangeSeg := rangeSegments(rangePath)
	domSeg := domSegments(domPath)
	if len(rangeSeg) == 0 && len(domSeg) == 0 {
		return []PathSegment{{Kind: PathRangeOffset, Index: 0}}
	}
	if len(rangeSeg) == 0 {
		rangeSeg = []PathSegment{{Kind: PathRangeOffset, Index: 0}}
	}
	if len(domSeg) == 0 {
		return clonePath(rangeSeg)
	}
	combined := make([]PathSegment, 0, len(rangeSeg)+len(domSeg))
	combined = append(combined, rangeSeg...)
	combined = append(combined, domSeg...)

	if err := validatePath(combined); err != nil {
		panic(fmt.Sprintf("combineTypedPath produced invalid path: %v (rangePath=%v, domPath=%v)", err, rangePath, domPath))
	}

	return combined
}

func clonePath(path []PathSegment) []PathSegment {
	if len(path) == 0 {
		return nil
	}
	out := make([]PathSegment, len(path))
	copy(out, path)
	return out
}
