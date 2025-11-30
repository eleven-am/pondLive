//go:build ignore

package main

import (
	"fmt"
	"go/format"
	"os"
	"sort"
	"strings"
)

type tagSpec struct {
	Name   string
	Tag    string
	Kind   string
	Mixins []string
}

type actionSpec struct {
	Type        string
	Constructor string
	Field       string
}

var baseActions = []actionSpec{
	{"ElementActions", "NewElementActions", "ElementActions"},
	{"ScrollActions", "NewScrollActions", "ScrollActions"},
}

var mixinActions = map[string][]actionSpec{
	"media":       {{"MediaActions", "NewMediaActions", "MediaActions"}},
	"form":        {{"FormActions", "NewFormActions", "FormActions"}},
	"details":     {{"DetailsActions", "NewDetailsActions", "DetailsActions"}},
	"value":       {{"ValueActions", "NewValueActions", "ValueActions"}},
	"dialog":      {{"DialogActions", "NewDialogActions", "DialogActions"}},
	"canvas":      {{"CanvasActions", "NewCanvasActions", "CanvasActions"}},
	"disableable": {{"DisableableActions", "NewDisableableActions", "DisableableActions"}},
	"selection":   {{"SelectionActions", "NewSelectionActions", "SelectionActions"}},
}

func actionsForTag(spec tagSpec) []actionSpec {
	result := append([]actionSpec{}, baseActions...)
	seen := map[string]bool{"ElementActions": true, "ScrollActions": true}

	for _, mixin := range spec.Mixins {
		if actions, ok := mixinActions[mixin]; ok {
			for _, action := range actions {
				if !seen[action.Type] {
					result = append(result, action)
					seen[action.Type] = true
				}
			}
		}
	}
	return result
}

var tags = []tagSpec{

	{"A", "a", "html", nil},
	{"Abbr", "abbr", "html", nil},
	{"Address", "address", "html", nil},
	{"Area", "area", "html", nil},
	{"Article", "article", "html", nil},
	{"Aside", "aside", "html", nil},
	{"Audio", "audio", "html", []string{"media"}},
	{"B", "b", "html", nil},
	{"Base", "base", "html", nil},
	{"Bdi", "bdi", "html", nil},
	{"Bdo", "bdo", "html", nil},
	{"Blockquote", "blockquote", "html", nil},
	{"Body", "body", "html", nil},
	{"Br", "br", "html", nil},
	{"Button", "button", "html", []string{"disableable"}},
	{"Canvas", "canvas", "html", []string{"canvas"}},
	{"Caption", "caption", "html", nil},
	{"Cite", "cite", "html", nil},
	{"Code", "code", "html", nil},
	{"Col", "col", "html", nil},
	{"Colgroup", "colgroup", "html", nil},
	{"DataEl", "data", "html", nil},
	{"Datalist", "datalist", "html", nil},
	{"Dd", "dd", "html", nil},
	{"DelEl", "del", "html", nil},
	{"Details", "details", "html", []string{"details"}},
	{"Dfn", "dfn", "html", nil},
	{"Dialog", "dialog", "html", []string{"dialog"}},
	{"Div", "div", "html", nil},
	{"Dl", "dl", "html", nil},
	{"Dt", "dt", "html", nil},
	{"Em", "em", "html", nil},
	{"Embed", "embed", "html", nil},
	{"Fieldset", "fieldset", "html", []string{"disableable"}},
	{"Figcaption", "figcaption", "html", nil},
	{"Figure", "figure", "html", nil},
	{"Footer", "footer", "html", nil},
	{"Form", "form", "html", []string{"form"}},
	{"H1", "h1", "html", nil},
	{"H2", "h2", "html", nil},
	{"H3", "h3", "html", nil},
	{"H4", "h4", "html", nil},
	{"H5", "h5", "html", nil},
	{"H6", "h6", "html", nil},
	{"Head", "head", "html", nil},
	{"Header", "header", "html", nil},
	{"Hgroup", "hgroup", "html", nil},
	{"Hr", "hr", "html", nil},
	{"Html", "html", "html", nil},
	{"I", "i", "html", nil},
	{"Iframe", "iframe", "html", nil},
	{"Img", "img", "html", nil},
	{"Input", "input", "html", []string{"value", "disableable", "selection"}},
	{"InsEl", "ins", "html", nil},
	{"Kbd", "kbd", "html", nil},
	{"Label", "label", "html", nil},
	{"Legend", "legend", "html", nil},
	{"Li", "li", "html", nil},
	{"Link", "link", "html", nil},
	{"Main", "main", "html", nil},
	{"MapEl", "map", "html", nil},
	{"Mark", "mark", "html", nil},
	{"Menu", "menu", "html", nil},
	{"MenuItem", "menuitem", "html", nil},
	{"Meta", "meta", "html", nil},
	{"Meter", "meter", "html", []string{"value"}},
	{"Nav", "nav", "html", nil},
	{"Noscript", "noscript", "html", nil},
	{"Object", "object", "html", nil},
	{"Ol", "ol", "html", nil},
	{"Optgroup", "optgroup", "html", nil},
	{"Option", "option", "html", nil},
	{"Output", "output", "html", nil},
	{"P", "p", "html", nil},
	{"Param", "param", "html", nil},
	{"Picture", "picture", "html", nil},
	{"Portal", "portal", "html", nil},
	{"Pre", "pre", "html", nil},
	{"Progress", "progress", "html", []string{"value"}},
	{"Q", "q", "html", nil},
	{"Rb", "rb", "html", nil},
	{"Rp", "rp", "html", nil},
	{"Rt", "rt", "html", nil},
	{"Rtc", "rtc", "html", nil},
	{"Ruby", "ruby", "html", nil},
	{"S", "s", "html", nil},
	{"Samp", "samp", "html", nil},
	{"ScriptEl", "script", "html", nil},
	{"Section", "section", "html", nil},
	{"Select", "select", "html", []string{"value", "disableable"}},
	{"Slot", "slot", "html", nil},
	{"Small", "small", "html", nil},
	{"Source", "source", "html", nil},
	{"Span", "span", "html", nil},
	{"Strong", "strong", "html", nil},
	{"StyleEl", "style", "html", nil},
	{"Sub", "sub", "html", nil},
	{"Summary", "summary", "html", nil},
	{"Sup", "sup", "html", nil},
	{"Table", "table", "html", nil},
	{"Tbody", "tbody", "html", nil},
	{"Td", "td", "html", nil},
	{"Template", "template", "html", nil},
	{"Textarea", "textarea", "html", []string{"value", "disableable", "selection"}},
	{"Tfoot", "tfoot", "html", nil},
	{"Th", "th", "html", nil},
	{"Thead", "thead", "html", nil},
	{"Time", "time", "html", nil},
	{"TitleEl", "title", "html", nil},
	{"Tr", "tr", "html", nil},
	{"Track", "track", "html", nil},
	{"U", "u", "html", nil},
	{"Ul", "ul", "html", nil},
	{"Var", "var", "html", nil},
	{"Video", "video", "html", []string{"media"}},
	{"Wbr", "wbr", "html", nil},

	{"Svg", "svg", "svg", nil},
	{"Circle", "circle", "svg", nil},
	{"ClipPath", "clipPath", "svg", nil},
	{"Defs", "defs", "svg", nil},
	{"Ellipse", "ellipse", "svg", nil},
	{"ForeignObject", "foreignObject", "svg", nil},
	{"G", "g", "svg", nil},
	{"Image", "image", "svg", nil},
	{"Line", "line", "svg", nil},
	{"LinearGradient", "linearGradient", "svg", nil},
	{"Marker", "marker", "svg", nil},
	{"Mask", "mask", "svg", nil},
	{"Path", "path", "svg", nil},
	{"Pattern", "pattern", "svg", nil},
	{"Polygon", "polygon", "svg", nil},
	{"Polyline", "polyline", "svg", nil},
	{"RadialGradient", "radialGradient", "svg", nil},
	{"Rect", "rect", "svg", nil},
	{"Stop", "stop", "svg", nil},
	{"SvgText", "text", "svg", nil},
	{"TSpan", "tspan", "svg", nil},
	{"SvgUse", "use", "svg", nil},
}

func main() {
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })

	generateTagBuilders(tags)
	generateRefs(tags)

	fmt.Println("generated tags_generated.go")
	fmt.Println("generated refs_generated.go")
}

func generateTagBuilders(specs []tagSpec) {
	var b strings.Builder
	b.WriteString("// Code generated by tags_gen.go; DO NOT EDIT.\n\n")
	b.WriteString("package html\n\n")

	for _, spec := range specs {
		fmt.Fprintf(&b, "func %s(items ...Item) Node { return El(%q, items...) }\n", spec.Name, spec.Tag)
	}

	writeFormatted("tags_generated.go", b.String())
}

func generateRefs(specs []tagSpec) {
	var b strings.Builder
	b.WriteString("// Code generated by tags_gen.go; DO NOT EDIT.\n\n")
	b.WriteString("package html\n\n")

	for _, spec := range specs {
		refName := spec.Name + "Ref"
		actions := actionsForTag(spec)

		fmt.Fprintf(&b, "type %s struct {\n", refName)
		b.WriteString("\t*ElementRef\n")
		for _, action := range actions {
			fmt.Fprintf(&b, "\t*%s\n", action.Type)
		}
		b.WriteString("}\n\n")
	}

	for _, spec := range specs {
		refName := spec.Name + "Ref"
		hookName := "Use" + spec.Name
		actions := actionsForTag(spec)

		fmt.Fprintf(&b, "func %s(ctx *Ctx) *%s {\n", hookName, refName)
		b.WriteString("\tref := UseElement(ctx)\n")
		fmt.Fprintf(&b, "\treturn &%s{\n", refName)
		b.WriteString("\t\tElementRef: ref,\n")
		for _, action := range actions {
			fmt.Fprintf(&b, "\t\t%s: %s(ctx, ref),\n", action.Field, action.Constructor)
		}
		b.WriteString("\t}\n}\n\n")
	}

	writeFormatted("refs_generated.go", b.String())
}

func writeFormatted(target string, src string) {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		fmt.Printf("Error formatting %s: %v\n", target, err)
		fmt.Println("Source:")
		fmt.Println(src)
		panic(err)
	}
	if err := os.WriteFile(target, formatted, 0o644); err != nil {
		panic(err)
	}
}
