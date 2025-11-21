package runtime

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/dom/css"
)

type stylesCell struct {
	styles *Styles
}

type Styles struct {
	hash       string
	stylesheet *dom.Stylesheet
}

func (s *Styles) Class(name string) string {
	if s.hash == "" {
		return name
	}
	return name + "-" + s.hash
}

func (s *Styles) StyleTag() dom.Item {
	return &dom.StructuredNode{
		Tag:        "style",
		Stylesheet: s.stylesheet,
	}
}

func UseStyles(ctx Ctx, rawCSS string) *Styles {
	if ctx.frame == nil {
		panic("runtime: UseStyles called outside render")
	}
	idx := ctx.frame.idx
	ctx.frame.idx++

	if idx >= len(ctx.frame.cells) {
		compID := ""
		if ctx.comp != nil {
			compID = ctx.comp.id
		}
		parsed := css.ParseAndScope(rawCSS, compID)
		stylesheet := convertStylesheet(parsed)
		styles := &Styles{
			hash:       parsed.SelectorHash,
			stylesheet: stylesheet,
		}
		ctx.frame.cells = append(ctx.frame.cells, &stylesCell{styles: styles})
	}

	cell, ok := ctx.frame.cells[idx].(*stylesCell)
	if !ok {
		panicHookMismatch(ctx.comp, idx, "UseStyles", ctx.frame.cells[idx])
	}
	return cell.styles
}

func convertStylesheet(parsed *css.Stylesheet) *dom.Stylesheet {
	if parsed == nil {
		return nil
	}

	stylesheet := &dom.Stylesheet{
		Hash: parsed.SelectorHash,
	}

	for _, rule := range parsed.Rules {
		stylesheet.Rules = append(stylesheet.Rules, dom.StyleRule{
			Selector: rule.Selector,
			Props:    rule.Props,
		})
	}

	for _, media := range parsed.MediaRules {
		block := dom.MediaBlock{
			Query: media.Query,
		}
		for _, rule := range media.Rules {
			block.Rules = append(block.Rules, dom.StyleRule{
				Selector: rule.Selector,
				Props:    rule.Props,
			})
		}
		stylesheet.MediaBlocks = append(stylesheet.MediaBlocks, block)
	}

	return stylesheet
}
