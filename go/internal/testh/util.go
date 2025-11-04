package testh

import (
	"sort"

	"github.com/eleven-am/go/pondlive/internal/diff"
	render "github.com/eleven-am/go/pondlive/internal/render"
)

func cloneOps(ops []diff.Op) []diff.Op {
	if len(ops) == 0 {
		return nil
	}
	out := make([]diff.Op, len(ops))
	copy(out, ops)
	return out
}

func cloneStructured(in render.Structured) render.Structured {
	out := render.Structured{}
	if len(in.S) > 0 {
		out.S = append([]string(nil), in.S...)
	}
	if len(in.D) > 0 {
		out.D = make([]render.Dyn, len(in.D))
		for i, dyn := range in.D {
			out.D[i] = cloneDyn(dyn)
		}
	}
	return out
}

func cloneDyn(d render.Dyn) render.Dyn {
	out := render.Dyn{Kind: d.Kind, Text: d.Text}
	if d.Attrs != nil {
		out.Attrs = make(map[string]string, len(d.Attrs))
		keys := make([]string, 0, len(d.Attrs))
		for k := range d.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			out.Attrs[k] = d.Attrs[k]
		}
	}
	if len(d.List) > 0 {
		out.List = make([]render.Row, len(d.List))
		for i, row := range d.List {
			copyRow := render.Row{Key: row.Key}
			if len(row.Slots) > 0 {
				copyRow.Slots = append([]int(nil), row.Slots...)
			}
			out.List[i] = copyRow
		}
	}
	return out
}
