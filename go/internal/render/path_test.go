package render

import "testing"

func TestCombineTypedPathDefaultsRangeSegment(t *testing.T) {
	combined := combineTypedPath(nil, []int{2, 3})
	if len(combined) != 3 {
		t.Fatalf("expected 3 segments, got %v", combined)
	}
	if combined[0].Kind != PathRangeOffset || combined[0].Index != 0 {
		t.Fatalf("expected implicit range offset, got %v", combined[0])
	}
	if combined[1].Kind != PathDomChild || combined[1].Index != 2 {
		t.Fatalf("expected dom child index 2, got %v", combined[1])
	}
	if combined[2].Index != 3 {
		t.Fatalf("expected dom child index 3, got %v", combined[2])
	}

	combined[0].Index = 99
	second := combineTypedPath(nil, []int{2, 3})
	if second[0].Index != 0 {
		t.Fatalf("expected clone on combine, got %v", second)
	}
}

func TestCombineTypedPathKeepsExistingRange(t *testing.T) {
	combined := combineTypedPath([]int{4}, []int{1})
	if len(combined) != 2 {
		t.Fatalf("expected 2 segments, got %v", combined)
	}
	if combined[0] != (PathSegment{Kind: PathRangeOffset, Index: 4}) {
		t.Fatalf("unexpected range segment %v", combined[0])
	}
	if combined[1] != (PathSegment{Kind: PathDomChild, Index: 1}) {
		t.Fatalf("unexpected dom child segment %v", combined[1])
	}
}

func TestValidatePathRejectsRangeAfterDomChild(t *testing.T) {
	path := []PathSegment{
		{Kind: PathRangeOffset, Index: 0},
		{Kind: PathDomChild, Index: 1},
		{Kind: PathRangeOffset, Index: 2},
	}
	if err := validatePath(path); err == nil {
		t.Fatalf("expected validatePath to reject range after dom child")
	}
}
