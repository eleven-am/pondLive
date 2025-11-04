package test

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/diff"
)

// AssertHTMLContains checks that HTML() contains substr (simple containment).
// Must return (ok, gotFragment, reason) with no panics.
func AssertHTMLContains(h Harness, substr string) (bool, string, string) {
	if h == nil {
		return false, "", "harness is nil"
	}
	html := h.HTML()
	if substr == "" {
		return true, html, ""
	}
	if strings.Contains(html, substr) {
		return true, html, ""
	}
	reason := fmt.Sprintf("expected HTML to contain %q", substr)
	return false, html, reason
}

// AssertOpsEqual serializes Ops() into a stable textual form and compares to want.
// Return (ok, gotText, diffText).
func AssertOpsEqual(h Harness, want string) (bool, string, string) {
	if h == nil {
		return false, "", "harness is nil"
	}
	gotText := formatOps(h.Ops())
	want = strings.TrimSpace(want)
	got := strings.TrimSpace(gotText)
	if got == want {
		return true, gotText, ""
	}
	diff := fmt.Sprintf("want %q got %q", want, got)
	return false, gotText, diff
}

// AssertNoOps ensures the last Flush() produced zero ops.
func AssertNoOps(h Harness) (bool, string) {
	if h == nil {
		return false, "harness is nil"
	}
	ops := h.Ops()
	if len(ops) == 0 {
		return true, ""
	}
	return false, formatOps(ops)
}

func formatOps(ops []diff.Op) string {
	if len(ops) == 0 {
		return ""
	}
	lines := make([]string, 0, len(ops))
	for _, op := range ops {
		switch v := op.(type) {
		case diff.SetText:
			lines = append(lines, fmt.Sprintf("setText slot=%d %s", v.Slot, quoteString(v.Text)))
		case diff.SetAttrs:
			lines = append(lines, formatSetAttrs(v))
		case diff.List:
			lines = append(lines, formatList(v))
		default:
			lines = append(lines, fmt.Sprintf("unknown %T", v))
		}
	}
	return strings.Join(lines, "\n")
}

func formatSetAttrs(op diff.SetAttrs) string {
	parts := []string{fmt.Sprintf("setAttrs slot=%d", op.Slot)}
	if len(op.Upsert) > 0 {
		keys := make([]string, 0, len(op.Upsert))
		for k := range op.Upsert {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts = append(parts, fmt.Sprintf("+%s:%s", k, quoteString(op.Upsert[k])))
		}
	}
	if len(op.Remove) > 0 {
		removed := append([]string(nil), op.Remove...)
		sort.Strings(removed)
		for _, k := range removed {
			parts = append(parts, fmt.Sprintf("-%s", k))
		}
	}
	return strings.Join(parts, " ")
}

func formatList(op diff.List) string {
	parts := []string{fmt.Sprintf("list slot=%d", op.Slot)}
	for _, child := range op.Ops {
		parts = append(parts, formatListChild(child))
	}
	return strings.Join(parts, " ")
}

func formatListChild(op diff.ListChildOp) string {
	switch v := op.(type) {
	case diff.Ins:
		fragment := fmt.Sprintf("ins pos=%d key=%s", v.Pos, v.Row.Key)
		if len(v.Row.Slots) > 0 {
			fragment = fmt.Sprintf("%s slots=%v", fragment, v.Row.Slots)
		}
		return fragment
	case diff.Del:
		return fmt.Sprintf("del key=%s", v.Key)
	case diff.Mov:
		return fmt.Sprintf("mov %d→%d", v.From, v.To)
	case diff.Set:
		return fmt.Sprintf("set key=%s sub=%d %s", v.Key, v.SubSlot, formatValue(v.Value))
	default:
		return fmt.Sprintf("unknown %T", v)
	}
}

func quoteString(s string) string {
	return strconv.Quote(s)
}

func formatValue(v any) string {
	switch val := v.(type) {
	case string:
		return quoteString(val)
	case fmt.Stringer:
		return quoteString(val.String())
	default:
		return quoteString(fmt.Sprintf("%v", val))
	}
}
