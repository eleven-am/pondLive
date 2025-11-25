package work

func SlotMarker(name string, children ...Node) Node {
	return &Fragment{
		Children: children,
		Metadata: map[string]any{
			"slot:name": name,
		},
	}
}

func ScopedSlotMarker[T any](name string, fn func(T) Node) Node {
	return &Fragment{
		Children: nil,
		Metadata: map[string]any{
			"scoped-slot:name": name,
			"scoped-slot:fn":   fn,
		},
	}
}
