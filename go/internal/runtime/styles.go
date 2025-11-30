package runtime

import (
	"github.com/eleven-am/pondlive/go/internal/css"
	"github.com/eleven-am/pondlive/go/internal/metadata"
)

type stylesCell struct {
	hash       string
	stylesheet *metadata.Stylesheet
}

type Styles struct {
	hash string
}

func (s *Styles) Class(name string) string {
	if s.hash == "" {
		return name
	}
	return name + "-" + s.hash
}

func UseStyles(ctx *Ctx, rawCSS string, appendFn ...func(*Ctx, *metadata.Stylesheet)) *Styles {
	if ctx == nil || ctx.instance == nil {
		panic("runtime: UseStyles called outside component render")
	}

	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		compID := ctx.instance.ID

		parsed := css.ParseAndScope(rawCSS, compID)
		stylesheet := convertStylesheet(parsed)

		cell := &stylesCell{
			hash:       parsed.SelectorHash,
			stylesheet: stylesheet,
		}

		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeStyles,
			Value: cell,
		})

		if len(appendFn) > 0 && appendFn[0] != nil {
			appendFn[0](ctx, stylesheet)
		}
	}

	cell, ok := ctx.instance.HookFrame[idx].Value.(*stylesCell)
	if !ok {
		panic("runtime: UseStyles hook mismatch")
	}

	return &Styles{hash: cell.hash}
}

func convertStylesheet(parsed *css.Stylesheet) *metadata.Stylesheet {
	if parsed == nil {
		return nil
	}

	stylesheet := &metadata.Stylesheet{
		Hash: parsed.SelectorHash,
	}

	for _, rule := range parsed.Rules {
		stylesheet.Rules = append(stylesheet.Rules, metadata.StyleRule{
			Selector: rule.Selector,
			Props:    rule.Props,
		})
	}

	for _, media := range parsed.MediaRules {
		block := metadata.MediaBlock{
			Query: media.Query,
		}
		for _, rule := range media.Rules {
			block.Rules = append(block.Rules, metadata.StyleRule{
				Selector: rule.Selector,
				Props:    rule.Props,
			})
		}
		stylesheet.MediaBlocks = append(stylesheet.MediaBlocks, block)
	}

	return stylesheet
}
