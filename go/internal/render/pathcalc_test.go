package render

import "testing"

func TestPathCalculatorComponentPaths(t *testing.T) {
	pc := NewPathCalculator()
	pc.PushComponent("child")

	pc.AppendToPath(0)
	pc.RecordComponentTraversal()
	pc.TrimPath(1)

	pc.AppendToPath(2)
	pc.RecordComponentTraversal()
	pc.TrimPath(1)

	pc.PopComponent()

	paths := pc.ComponentPaths()
	if len(paths) != 1 {
		t.Fatalf("expected one component path, got %d", len(paths))
	}
	child := paths[0]
	if child.ComponentID != "child" {
		t.Fatalf("unexpected component id %s", child.ComponentID)
	}
	if len(child.FirstChild) != 1 || child.FirstChild[0].Kind != PathDomChild || child.FirstChild[0].Index != 0 {
		t.Fatalf("expected first child dom path [0], got %v", child.FirstChild)
	}
	if len(child.LastChild) != 1 || child.LastChild[0].Index != 2 {
		t.Fatalf("expected last child dom path [2], got %v", child.LastChild)
	}
}

func TestPathCalculatorRecordListPathWithFrame(t *testing.T) {
	pc := NewPathCalculator()
	frame := &elementFrame{
		componentID:   "cmp",
		basePath:      []int{5},
		componentPath: []int{1, 3},
	}

	pc.RecordListPath(9, frame)
	paths := pc.ListPaths()
	if len(paths) != 1 {
		t.Fatalf("expected single list path, got %d", len(paths))
	}
	entry := paths[0]
	if entry.ComponentID != "cmp" || entry.Slot != 9 {
		t.Fatalf("unexpected list entry %+v", entry)
	}
	if len(entry.Path) != 3 {
		t.Fatalf("expected combined typed path segments, got %v", entry.Path)
	}
	if entry.Path[0].Kind != PathRangeOffset || entry.Path[0].Index != 5 {
		t.Fatalf("expected first segment r:5, got %v", entry.Path[0])
	}
	if entry.Path[1].Kind != PathDomChild || entry.Path[1].Index != 1 {
		t.Fatalf("expected dom child index 1, got %v", entry.Path[1])
	}
	if entry.Path[2].Index != 3 {
		t.Fatalf("expected dom child index 3, got %v", entry.Path[2])
	}
}

func TestPathCalculatorRecordListPathAtRoot(t *testing.T) {
	pc := NewPathCalculator()
	pc.PushComponent("root")
	pc.RecordListPath(4, nil)
	listPaths := pc.ListPaths()
	if len(listPaths) != 1 {
		t.Fatalf("expected one list path, got %d", len(listPaths))
	}
	entry := listPaths[0]
	if !entry.AtRoot || entry.ComponentID != "root" {
		t.Fatalf("expected root list path, got %+v", entry)
	}
}

func TestPathCalculatorMultipleComponentsAndLists(t *testing.T) {
	pc := NewPathCalculator()

	pc.PushComponent("parent")

	pc.AppendToPath(0)
	pc.RecordComponentTraversal()
	pc.PushComponent("child")

	pc.AppendToPath(0)
	pc.RecordComponentTraversal()
	pc.TrimPath(1)
	pc.AppendToPath(2)
	pc.RecordComponentTraversal()
	pc.TrimPath(1)

	pc.PopComponent()
	pc.TrimPath(1)

	pc.AppendToPath(1)
	pc.RecordComponentTraversal()
	pc.TrimPath(1)

	frame := &elementFrame{
		componentID:   "parent",
		basePath:      []int{2},
		componentPath: []int{1},
	}
	pc.RecordListPath(7, frame)

	pc.PopComponent()

	paths := pc.ComponentPaths()
	if len(paths) != 2 {
		t.Fatalf("expected parent and child component paths, got %d", len(paths))
	}
	var parent, child ComponentPath
	for _, cp := range paths {
		if cp.ComponentID == "parent" {
			parent = cp
		} else if cp.ComponentID == "child" {
			child = cp
		}
	}
	if parent.ComponentID == "" || child.ComponentID == "" {
		t.Fatalf("expected both parent and child paths, got %+v", paths)
	}
	if len(parent.FirstChild) == 0 || parent.FirstChild[0].Index != 0 {
		t.Fatalf("parent first child path incorrect: %+v", parent.FirstChild)
	}
	if len(parent.LastChild) == 0 || parent.LastChild[0].Index != 1 {
		t.Fatalf("parent last child path incorrect: %+v", parent.LastChild)
	}
	if len(child.FirstChild) == 0 || child.FirstChild[0].Index != 0 {
		t.Fatalf("child first child path incorrect: %+v", child.FirstChild)
	}
	if len(child.LastChild) == 0 || child.LastChild[0].Index != 2 {
		t.Fatalf("child last child path incorrect: %+v", child.LastChild)
	}

	listPaths := pc.ListPaths()
	if len(listPaths) != 1 {
		t.Fatalf("expected 1 list path, got %d", len(listPaths))
	}
	last := listPaths[0]
	if last.ComponentID != "parent" || last.AtRoot {
		t.Fatalf("expected parent-scoped list path, got %+v", last)
	}
	if len(last.Path) != 2 || last.Path[0].Kind != PathRangeOffset || last.Path[0].Index != 2 {
		t.Fatalf("unexpected typed path for list: %+v", last.Path)
	}
	if last.Path[1].Kind != PathDomChild || last.Path[1].Index != 1 {
		t.Fatalf("unexpected dom child segments: %+v", last.Path)
	}
}
