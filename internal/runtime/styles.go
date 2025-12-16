package runtime

import (
	"github.com/eleven-am/pondlive/internal/css"
	"github.com/eleven-am/pondlive/internal/metadata"
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
			Decls:    convertDecls(rule.Decls),
		})
	}

	for _, media := range parsed.MediaRules {
		block := metadata.MediaBlock{
			Query: media.Query,
		}
		for _, rule := range media.Rules {
			block.Rules = append(block.Rules, metadata.StyleRule{
				Selector: rule.Selector,
				Decls:    convertDecls(rule.Decls),
			})
		}
		stylesheet.MediaBlocks = append(stylesheet.MediaBlocks, block)
	}

	for _, kf := range parsed.Keyframes {
		block := metadata.KeyframesBlock{
			Name: kf.Name,
		}
		for _, step := range kf.Steps {
			block.Steps = append(block.Steps, metadata.KeyframesStep{
				Selector: step.Selector,
				Decls:    convertDecls(step.Decls),
			})
		}
		stylesheet.Keyframes = append(stylesheet.Keyframes, block)
	}

	stylesheet.OtherBlocks = append(stylesheet.OtherBlocks, parsed.OtherBlocks...)

	return stylesheet
}

func convertDecls(decls []css.Declaration) []metadata.Declaration {
	result := make([]metadata.Declaration, len(decls))
	for i, d := range decls {
		result[i] = metadata.Declaration{
			Property: d.Property,
			Value:    d.Value,
		}
	}
	return result
}
