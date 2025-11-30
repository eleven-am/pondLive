package styles

import (
	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type Ctx = runtime.Ctx
type Styles = runtime.Styles

var slot = runtime.CreateSlotContext()

var Provider = runtime.Component(func(ctx *Ctx, children []work.Node) work.Node {
	slot.ProvideWithoutDefault(ctx, children)
	return &work.Fragment{Children: children}
})

var Render = runtime.Component(func(ctx *Ctx, children []work.Node) work.Node {
	return slot.Render(ctx, runtime.DefaultSlotName)
})

func UseStyles(ctx *Ctx, rawCSS string) *Styles {
	return runtime.UseStyles(ctx, rawCSS, appendStylesheet)
}

func appendStylesheet(ctx *Ctx, stylesheet *metadata.Stylesheet) {
	styleNode := &work.Element{
		Tag:        "style",
		Stylesheet: stylesheet,
	}

	slot.AppendSlot(ctx, runtime.DefaultSlotName, styleNode)
}
