package runtime2

import (
	"github.com/eleven-am/pondlive/go/internal/css"
	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// stylesCell stores the parsed and scoped stylesheet for a component.
type stylesCell struct {
	styles *Styles
}

// Styles provides scoped CSS class names and a style tag for injection.
type Styles struct {
	hash       string
	stylesheet *metadata.Stylesheet
}

// Class returns the scoped class name by appending the component hash.
// If no hash exists (empty CSS), returns the original name unchanged.
func (s *Styles) Class(name string) string {
	if s.hash == "" {
		return name
	}
	return name + "-" + s.hash
}

// StyleTag returns a work.Element representing a <style> tag with the scoped CSS.
// Use this in your component's render output to inject the styles.
func (s *Styles) StyleTag() work.Node {
	return &work.Element{
		Tag:        "style",
		Stylesheet: s.stylesheet,
	}
}

// UseStyles parses and scopes CSS to the current component.
// Returns a Styles handle for getting scoped class names and the style tag.
// The CSS is parsed once on first render and cached for subsequent renders.
func UseStyles(ctx *Ctx, rawCSS string) *Styles {
	if ctx == nil || ctx.instance == nil {
		panic("runtime: UseStyles called outside component render")
	}

	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		compID := ""
		if ctx.instance != nil {
			compID = ctx.instance.ID
		}

		parsed := css.ParseAndScope(rawCSS, compID)
		stylesheet := convertStylesheet(parsed)

		styles := &Styles{
			hash:       parsed.SelectorHash,
			stylesheet: stylesheet,
		}

		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeStyles,
			Value: &stylesCell{styles: styles},
		})
	}

	cell, ok := ctx.instance.HookFrame[idx].Value.(*stylesCell)
	if !ok {
		panic("runtime: UseStyles hook mismatch")
	}

	return cell.styles
}

// convertStylesheet converts css.Stylesheet to metadata.Stylesheet.
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
