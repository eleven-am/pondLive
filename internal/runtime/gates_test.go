package runtime

import (
	"fmt"
	goruntime "runtime"
	"slices"
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
)

const (
	gateSamples       = 5
	p3MountAllocs5000 = 225357
	p3MountBytes5000  = 21_900_000
)

func measureAllocs(iterations int, fn func()) (allocs, bytes int64) {
	allocSamples := make([]int64, gateSamples)
	byteSamples := make([]int64, gateSamples)
	for i := range allocSamples {
		goruntime.GC()
		var m0, m1 goruntime.MemStats
		goruntime.ReadMemStats(&m0)
		for j := 0; j < iterations; j++ {
			fn()
		}
		goruntime.ReadMemStats(&m1)
		allocSamples[i] = int64(m1.Mallocs-m0.Mallocs) / int64(iterations)
		byteSamples[i] = int64(m1.TotalAlloc-m0.TotalAlloc) / int64(iterations)
	}
	slices.Sort(allocSamples)
	slices.Sort(byteSamples)
	return allocSamples[gateSamples/2], byteSamples[gateSamples/2]
}

func TestGateBoundUpdateAllocsAreFlat(t *testing.T) {
	skipUnderRace(t)
	var counts []int64
	for _, n := range []int{100, 1000, 5000} {
		s, sigs := buildBindingPage(n)
		i := 0
		allocs, _ := measureAllocs(2000, func() {
			i++
			sigs[i%n].Set(i)
			if err := s.graph.RunQueue(reactive.QueueRender); err != nil {
				t.Fatal(err)
			}
			s.edits = s.edits[:0]
		})
		t.Logf("bound update at %d bindings: %d allocs/op", n, allocs)
		if allocs > 3 {
			t.Errorf("bound update at %d bindings costs %d allocs, gate 3", n, allocs)
		}
		counts = append(counts, allocs)
	}
	if slices.Max(counts)-slices.Min(counts) > 1 {
		t.Errorf("bound update allocs are not flat across sizes: %v", counts)
	}
}

func TestGateUpdateFrameAllocsAndBytes(t *testing.T) {
	skipUnderRace(t)
	var counts []int64
	for _, n := range []int{100, 1000, 5000} {
		s, sigs := buildBindingPage(n)
		var sink int
		s.Bus.SubscribeToFrameEdits(func(edits []protocol.Edit) {
			for _, e := range edits {
				raw, _ := e.MarshalJSON()
				sink += len(raw)
			}
		})
		i := 0
		allocs, bytes := measureAllocs(2000, func() {
			i++
			sigs[i%n].Set(i)
			s.flush()
		})
		t.Logf("update to frame at %d bindings: %d allocs/op, %d B/op", n, allocs, bytes)
		if allocs > 50 || bytes > 4096 {
			t.Errorf("update frame at %d bindings costs %d allocs and %d B, gate 50 and 4096", n, allocs, bytes)
		}
		counts = append(counts, allocs)
		_ = sink
	}
	if counts[2]-counts[0] > 2 {
		t.Errorf("update frame allocs at 5000 are more than 2 above the 100 case: %v", counts)
	}
}

func TestGateResetAndMount(t *testing.T) {
	allocLimit := map[int]int64{100: 500, 1000: 5000, 5000: 30000}
	for _, n := range []int{100, 1000, 5000} {
		for _, feed := range []bool{false, true} {
			p := buildBenchPage(n, feed)
			resetAllocs, resetBytes := measureAllocs(5, func() {
				if _, err := p.s.DocumentHTML(); err != nil {
					t.Fatal(err)
				}
			})
			mountAllocs, mountBytes := measureAllocs(3, func() { buildBenchPage(n, feed) })
			t.Logf("%d nodes feed=%v: reset %d allocs %d B, mount %d allocs %d B", n, feed, resetAllocs, resetBytes, mountAllocs, mountBytes)
			if resetAllocs > allocLimit[n] {
				t.Errorf("reset at %d nodes feed=%v costs %d allocs, gate %d", n, feed, resetAllocs, allocLimit[n])
			}
			if n == 5000 && resetBytes > 2<<20 {
				t.Errorf("reset at 5000 nodes feed=%v allocates %d B, gate %d", feed, resetBytes, 2<<20)
			}
			if resetAllocs > mountAllocs || resetBytes > mountBytes {
				t.Errorf("reset at %d nodes feed=%v (%d allocs, %d B) costs more than the mount (%d allocs, %d B)", n, feed, resetAllocs, resetBytes, mountAllocs, mountBytes)
			}
			if n == 5000 && (mountAllocs > p3MountAllocs5000 || mountBytes > p3MountBytes5000) {
				t.Errorf("mount at 5000 nodes costs %d allocs and %d B, p3-before was %d and %d", mountAllocs, mountBytes, p3MountAllocs5000, p3MountBytes5000)
			}
		}
	}
}

func lisLength(seq []int) int {
	var tails []int
	for _, v := range seq {
		i, _ := slices.BinarySearch(tails, v)
		if i == len(tails) {
			tails = append(tails, v)
		} else {
			tails[i] = v
		}
	}
	return len(tails)
}

func TestGateListOperationsAtFiveThousandRows(t *testing.T) {
	const n = 5000
	p := buildBenchPage(n*domNodesPerRow, false)
	apply := func(next []int) []protocol.Edit {
		p.list.Set(next)
		if err := p.s.graph.RunQueue(reactive.QueueRender); err != nil {
			t.Fatal(err)
		}
		edits := p.s.edits
		p.s.edits = nil
		return edits
	}
	editBytes := func(edits []protocol.Edit) int {
		total := 0
		for _, e := range edits {
			raw, _ := e.MarshalJSON()
			total += len(raw)
		}
		return total
	}

	inserted := apply(append([]int{1_000_000}, p.items...))
	t.Logf("insert at front: %d edits, %d B", len(inserted), editBytes(inserted))
	if len(inserted) != 1 || editBytes(inserted) > 400 {
		t.Errorf("insert at front is %d edits and %d B, gate 1 edit and 400 B", len(inserted), editBytes(inserted))
	}
	apply(p.items)

	swapped := slices.Clone(p.items)
	swapped[0], swapped[n-1] = swapped[n-1], swapped[0]
	edits := apply(swapped)
	t.Logf("swap: %d edits", len(edits))
	if len(edits) > 2 {
		t.Errorf("swap of two rows is %d edits, gate 2", len(edits))
	}
	apply(p.items)

	shuffled := slices.Clone(p.items)
	for j := range shuffled {
		k := (j*7919 + 3) % len(shuffled)
		shuffled[j], shuffled[k] = shuffled[k], shuffled[j]
	}
	position := map[int]int{}
	for i, v := range p.items {
		position[v] = i
	}
	order := make([]int, len(shuffled))
	for i, v := range shuffled {
		order[i] = position[v]
	}
	want := n - lisLength(order)
	edits = apply(shuffled)
	t.Logf("shuffle: %d edits, n - LIS = %d", len(edits), want)
	if len(edits) > want {
		t.Errorf("shuffle is %d edits, gate n - LIS = %d", len(edits), want)
	}
}

func TestGateTextEditWireSize(t *testing.T) {
	raw, err := protocol.TextEdit(7, "1").MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > 32 {
		t.Errorf("a t edit is %d B on the wire, gate 32: %s", len(raw), fmt.Sprint(string(raw)))
	}
}

func skipUnderRace(t *testing.T) {
	t.Helper()
	if raceEnabled {
		t.Skip("allocation gates are measured without -race, whose instrumentation inflates allocated bytes")
	}
}
