package diff

import (
	"fmt"

	"github.com/eleven-am/pondlive/go/internal/render"
)

// StrictMismatch controls whether template mismatches cause a panic. It should
// remain true in development so structural regressions are caught early.
var StrictMismatch = true

// SetStrictMismatch toggles the panic-on-mismatch behaviour. Consumers can
// disable strict mode during initialization if they prefer recovering from
// structural drift instead of panicking.
func SetStrictMismatch(enabled bool) {
	StrictMismatch = enabled
}

// Diff computes the diff operations required to update prev into next.
// It assumes both structured renders originate from the same template and only
// compares dynamic slot values for text and attribute slots.
func Diff(prev, next render.Structured) []Op {
	if mismatch := ensureTemplateCompatibility(prev, next); mismatch {
		return nil
	}

	n := len(next.D)
	ops := make([]Op, 0, n/4)
	for i := 0; i < n; i++ {
		p := prev.D[i]
		q := next.D[i]

		if p.Kind != q.Kind {
			if handleMismatch("slot %d kind changed from %v to %v", i, p.Kind, q.Kind) {
				return nil
			}
			continue
		}

		switch q.Kind {
		case render.DynText:
			if p.Text != q.Text {
				ops = append(ops, SetText{Slot: i, Text: q.Text})
			}
		case render.DynAttrs:
			op := diffAttrsSlot(i, p.Attrs, q.Attrs)
			if op != nil {
				ops = append(ops, op)
			}
		case render.DynList:
			listOps := diffListSlot(i, p.List, q.List)
			if len(listOps) > 0 {
				ops = append(ops, List{Slot: i, Ops: listOps})
			}
		}
	}

	return ops
}

func ensureTemplateCompatibility(prev, next render.Structured) bool {
	if len(prev.S) != len(next.S) {
		return handleMismatch("static segment count changed: prev=%d next=%d", len(prev.S), len(next.S))
	}
	for i := range prev.S {
		if prev.S[i] != next.S[i] {
			return handleMismatch("static segment %d mismatched", i)
		}
	}
	if len(prev.D) != len(next.D) {
		return handleMismatch("slot count changed: prev=%d next=%d", len(prev.D), len(next.D))
	}
	return false
}

func handleMismatch(format string, args ...any) bool {
	err := fmt.Errorf("diff: template mismatch: "+format, args...)
	callMismatchHandler(err)
	if StrictMismatch {
		panic(err)
	}
	return true
}

func diffAttrsSlot(slot int, prev, next map[string]string) Op {
	if len(prev) == 0 && len(next) == 0 {
		return nil
	}

	var upsert map[string]string
	for k, v := range next {
		if prev == nil {
			if upsert == nil {
				upsert = make(map[string]string)
			}
			upsert[k] = v
			continue
		}
		if old, ok := prev[k]; !ok || old != v {
			if upsert == nil {
				upsert = make(map[string]string)
			}
			upsert[k] = v
		}
	}

	var remove []string
	if len(prev) != 0 {
		for k := range prev {
			if next == nil {
				remove = append(remove, k)
				continue
			}
			if _, ok := next[k]; !ok {
				remove = append(remove, k)
			}
		}
	}

	if len(upsert) == 0 && len(remove) == 0 {
		return nil
	}

	return SetAttrs{Slot: slot, Upsert: upsert, Remove: remove}
}

func diffListSlot(slot int, prevRows, nextRows []render.Row) []ListChildOp {
	if len(prevRows) == 0 && len(nextRows) == 0 {
		return nil
	}

	prevIndex := make(map[string]int, len(prevRows))
	for i, row := range prevRows {
		prevIndex[row.Key] = i
	}

	nextIndex := make(map[string]int, len(nextRows))
	for i, row := range nextRows {
		nextIndex[row.Key] = i
	}

	ops := make([]ListChildOp, 0)

	for _, row := range prevRows {
		if _, ok := nextIndex[row.Key]; !ok {
			ops = append(ops, Del{Key: row.Key})
		}
	}

	current := make([]string, 0, len(prevRows))
	for _, row := range prevRows {
		if _, ok := nextIndex[row.Key]; ok {
			current = append(current, row.Key)
		}
	}

	findIndex := func(keys []string, key string) int {
		for idx, k := range keys {
			if k == key {
				return idx
			}
		}
		return -1
	}

	for targetIdx, row := range nextRows {
		if _, ok := prevIndex[row.Key]; !ok {
			ops = append(ops, Ins{Pos: targetIdx, Row: row})
			current = insertKey(current, targetIdx, row.Key)
			continue
		}

		fromIdx := findIndex(current, row.Key)
		if fromIdx == -1 {

			ops = append(ops, Ins{Pos: targetIdx, Row: row})
			current = insertKey(current, targetIdx, row.Key)
			continue
		}
		if fromIdx != targetIdx {
			ops = append(ops, Mov{From: fromIdx, To: targetIdx})
			key := current[fromIdx]
			current = removeKey(current, fromIdx)
			current = insertKey(current, targetIdx, key)
		}
	}

	return ops
}

func insertKey(keys []string, idx int, key string) []string {
	if idx >= len(keys) {
		return append(keys, key)
	}
	keys = append(keys[:idx+1], keys[idx:]...)
	keys[idx] = key
	return keys
}

func removeKey(keys []string, idx int) []string {
	if idx < 0 || idx >= len(keys) {
		return keys
	}
	return append(keys[:idx], keys[idx+1:]...)
}
