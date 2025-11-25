package html

import "github.com/eleven-am/pondlive/go/internal/work"

func SlotMarker(name string, children ...work.Node) work.Node {
	return work.SlotMarker(name, children...)
}

func ScopedSlotMarker[T any](name string, fn func(T) work.Node) work.Node {
	return work.ScopedSlotMarker(name, fn)
}
