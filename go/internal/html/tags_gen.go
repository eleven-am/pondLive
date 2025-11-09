//go:build ignore

package main

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type tagSpec struct {
	Name string
	Tag  string
	Doc  string
	Kind string
	Ref  *refSpec
}

type stateFieldSpec struct {
	Name     string
	Type     string
	Selector string
}

type refEventSpec struct {
	Method string
	Event  string
	Listen []string
	Props  []string
	Doc    string
}

type refSpec struct {
	StateName   string
	StateMethod string
	Fields      []stateFieldSpec
	Events      []refEventSpec
	Mixins      []string
}

// mixinSpec left intentionally empty to preserve compatibility with existing tag descriptors.
type mixinSpec struct{}

type handlerSpec struct {
	Type        string
	Constructor string
	FieldName   string
}

var handlerSpecs = []handlerSpec{
	{"AnimationHandler", "NewAnimationHandler", "AnimationHandler"},
	{"ClickHandler", "NewClickHandler", "ClickHandler"},
	{"ClipboardHandler", "NewClipboardHandler", "ClipboardHandler"},
	{"CompositionHandler", "NewCompositionHandler", "CompositionHandler"},
	{"DialogHandler", "NewDialogHandler", "DialogHandler"},
	{"DragHandler", "NewDragHandler", "DragHandler"},
	{"FocusHandler", "NewFocusHandler", "FocusHandler"},
	{"FormHandler", "NewFormHandler", "FormHandler"},
	{"FullscreenHandler", "NewFullscreenHandler", "FullscreenHandler"},
	{"HashChangeHandler", "NewHashChangeHandler", "HashChangeHandler"},
	{"InputHandler", "NewInputHandler", "InputHandler"},
	{"KeyboardHandler", "NewKeyboardHandler", "KeyboardHandler"},
	{"LifecycleHandler", "NewLifecycleHandler", "LifecycleHandler"},
	{"LoadHandler", "NewLoadHandler", "LoadHandler"},
	{"MediaHandler", "NewMediaHandler", "MediaHandler"},
	{"MouseHandler", "NewMouseHandler", "MouseHandler"},
	{"PointerHandler", "NewPointerHandler", "PointerHandler"},
	{"PrintHandler", "NewPrintHandler", "PrintHandler"},
	{"ResizeHandler", "NewResizeHandler", "ResizeHandler"},
	{"ScrollHandler", "NewScrollHandler", "ScrollHandler"},
	{"SelectionHandler", "NewSelectionHandler", "SelectionHandler"},
	{"StorageHandler", "NewStorageHandler", "StorageHandler"},
	{"ToggleHandler", "NewToggleHandler", "ToggleHandler"},
	{"TouchHandler", "NewTouchHandler", "TouchHandler"},
	{"TransitionHandler", "NewTransitionHandler", "TransitionHandler"},
	{"VisibilityHandler", "NewVisibilityHandler", "VisibilityHandler"},
	{"WheelHandler", "NewWheelHandler", "WheelHandler"},
}

var tags = []tagSpec{
	{"A", "a", "A creates an <a> element.", "html", nil},
	{"Abbr", "abbr", "Abbr creates an <abbr> element.", "html", nil},
	{"Address", "address", "Address creates an <address> element.", "html", nil},
	{"Area", "area", "Area creates an <area> element.", "html", nil},
	{"Article", "article", "Article creates an <article> element.", "html", nil},
	{"Aside", "aside", "Aside creates an <aside> element.", "html", nil},
	{"Audio", "audio", "Audio creates an <audio> element.", "html", &refSpec{
		StateMethod: "AudioState",
		Mixins:      []string{"media"},
		Events: []refEventSpec{
			{Method: "OnAudioTimeUpdate", Event: "timeupdate", Listen: []string{"durationchange", "play", "pause", "seeking", "seeked"}},
			{Method: "OnAudioPlay", Event: "play", Listen: []string{"playing"}},
			{Method: "OnAudioPause", Event: "pause"},
			{Method: "OnAudioEnded", Event: "ended"},
			{Method: "OnAudioVolumeChange", Event: "volumechange"},
			{Method: "OnAudioSeeking", Event: "seeking"},
			{Method: "OnAudioSeeked", Event: "seeked"},
			{Method: "OnAudioRateChange", Event: "ratechange"},
		},
	}},
	{"B", "b", "B creates a <b> element.", "html", nil},
	{"Base", "base", "Base creates a <base> element.", "html", nil},
	{"Bdi", "bdi", "Bdi creates a <bdi> element.", "html", nil},
	{"Bdo", "bdo", "Bdo creates a <bdo> element.", "html", nil},
	{"Blockquote", "blockquote", "Blockquote creates a <blockquote> element.", "html", nil},
	{"Body", "body", "Body creates a <body> element.", "html", nil},
	{"Br", "br", "Br creates a <br> element.", "html", nil},
	{"Button", "button", "Button creates a <button> element.", "html", &refSpec{
		StateMethod: "ButtonState",
		Fields: []stateFieldSpec{
			{Name: "Disabled", Type: "bool", Selector: "target.disabled"},
			{Name: "Type", Type: "string", Selector: "target.type"},
			{Name: "Value", Type: "string", Selector: "target.value"},
			{Name: "Name", Type: "string", Selector: "target.name"},
			{Name: "FormAction", Type: "string", Selector: "target.formAction"},
			{Name: "FormMethod", Type: "string", Selector: "target.formMethod"},
		},
		Events: []refEventSpec{
			{Method: "OnButtonClick", Event: "click", Props: []string{"event.detail"}},
			{Method: "OnButtonFocus", Event: "focus"},
			{Method: "OnButtonBlur", Event: "blur"},
		},
	}},
	{"Canvas", "canvas", "Canvas creates a <canvas> element.", "html", &refSpec{
		StateMethod: "CanvasState",
		Fields: []stateFieldSpec{
			{Name: "Pressure", Type: "float64", Selector: "event.pressure"},
			{Name: "TangentialPressure", Type: "float64", Selector: "event.tangentialPressure"},
			{Name: "TiltX", Type: "float64", Selector: "event.tiltX"},
			{Name: "TiltY", Type: "float64", Selector: "event.tiltY"},
		},
		Events: []refEventSpec{
			{Method: "OnCanvasPointerMove", Event: "pointermove"},
			{Method: "OnCanvasPointerDown", Event: "pointerdown"},
			{Method: "OnCanvasPointerUp", Event: "pointerup"},
			{Method: "OnCanvasWheel", Event: "wheel"},
		},
	}},
	{"Caption", "caption", "Caption creates a <caption> element.", "html", nil},
	{"Cite", "cite", "Cite creates a <cite> element.", "html", nil},
	{"Code", "code", "Code creates a <code> element.", "html", nil},
	{"Col", "col", "Col creates a <col> element.", "html", nil},
	{"Colgroup", "colgroup", "Colgroup creates a <colgroup> element.", "html", nil},
	{"DataEl", "data", "DataEl creates a <data> element.", "html", nil},
	{"Datalist", "datalist", "Datalist creates a <datalist> element.", "html", nil},
	{"Dd", "dd", "Dd creates a <dd> element.", "html", nil},
	{"DelEl", "del", "DelEl creates a <del> element.", "html", nil},
	{"Details", "details", "Details creates a <details> element.", "html", &refSpec{
		StateMethod: "DetailsState",
		Fields: []stateFieldSpec{
			{Name: "Open", Type: "bool", Selector: "target.open"},
		},
		Events: []refEventSpec{
			{Method: "OnDetailsToggle", Event: "toggle"},
		},
	}},
	{"Dfn", "dfn", "Dfn creates a <dfn> element.", "html", nil},
	{"Dialog", "dialog", "Dialog creates a <dialog> element.", "html", &refSpec{
		StateMethod: "DialogState",
		Fields: []stateFieldSpec{
			{Name: "Open", Type: "bool", Selector: "target.open"},
			{Name: "ReturnValue", Type: "string", Selector: "target.returnValue"},
		},
		Events: []refEventSpec{
			{Method: "OnDialogClose", Event: "close"},
			{Method: "OnDialogCancel", Event: "cancel"},
		},
	}},
	{"Div", "div", "Div creates a <div> element.", "html", &refSpec{
		StateMethod: "DivState",
		Fields: []stateFieldSpec{
			{Name: "TargetID", Type: "string", Selector: "target.id"},
		},
		Events: []refEventSpec{
			{Method: "OnDivClick", Event: "click", Props: []string{"event.detail"}},
			{Method: "OnDivDrag", Event: "drag", Listen: []string{"dragstart", "dragend"}},
		},
	}},
	{"Dl", "dl", "Dl creates a <dl> element.", "html", nil},
	{"Dt", "dt", "Dt creates a <dt> element.", "html", nil},
	{"Em", "em", "Em creates an <em> element.", "html", nil},
	{"Embed", "embed", "Embed creates an <embed> element.", "html", nil},
	{"Fieldset", "fieldset", "Fieldset creates a <fieldset> element.", "html", nil},
	{"Figcaption", "figcaption", "Figcaption creates a <figcaption> element.", "html", nil},
	{"Figure", "figure", "Figure creates a <figure> element.", "html", nil},
	{"Footer", "footer", "Footer creates a <footer> element.", "html", nil},
	{"Form", "form", "Form creates a <form> element.", "html", &refSpec{
		StateMethod: "FormState",
		Fields: []stateFieldSpec{
			{Name: "Action", Type: "string", Selector: "target.action"},
			{Name: "Method", Type: "string", Selector: "target.method"},
			{Name: "NoValidate", Type: "bool", Selector: "target.noValidate"},
			{Name: "Target", Type: "string", Selector: "target.target"},
			{Name: "Encoding", Type: "string", Selector: "target.enctype"},
			{Name: "Name", Type: "string", Selector: "target.name"},
		},
		Events: []refEventSpec{
			{Method: "OnFormSubmit", Event: "submit", Listen: []string{"formdata"}},
			{Method: "OnFormReset", Event: "reset"},
			{Method: "OnFormInvalid", Event: "invalid"},
		},
	}},
	{"H1", "h1", "H1 creates an <h1> element.", "html", nil},
	{"H2", "h2", "H2 creates an <h2> element.", "html", nil},
	{"H3", "h3", "H3 creates an <h3> element.", "html", nil},
	{"H4", "h4", "H4 creates an <h4> element.", "html", nil},
	{"H5", "h5", "H5 creates an <h5> element.", "html", nil},
	{"H6", "h6", "H6 creates an <h6> element.", "html", nil},
	{"Head", "head", "Head creates a <head> element.", "html", nil},
	{"Header", "header", "Header creates a <header> element.", "html", nil},
	{"Hgroup", "hgroup", "Hgroup creates an <hgroup> element.", "html", nil},
	{"Hr", "hr", "Hr creates an <hr> element.", "html", nil},
	{"Html", "html", "Html creates an <html> element.", "html", nil},
	{"I", "i", "I creates an <i> element.", "html", nil},
	{"Iframe", "iframe", "Iframe creates an <iframe> element.", "html", nil},
	{"Img", "img", "Img creates an <img> element.", "html", nil},
	{"Input", "input", "Input creates an <input> element.", "html", &refSpec{
		StateMethod: "InputState",
		Fields: []stateFieldSpec{
			{Name: "Value", Type: "string", Selector: "target.value"},
			{Name: "Checked", Type: "bool", Selector: "target.checked"},
			{Name: "Type", Type: "string", Selector: "target.type"},
			{Name: "Name", Type: "string", Selector: "target.name"},
			{Name: "Disabled", Type: "bool", Selector: "target.disabled"},
			{Name: "Required", Type: "bool", Selector: "target.required"},
			{Name: "SelectionStart", Type: "int", Selector: "target.selectionStart"},
			{Name: "SelectionEnd", Type: "int", Selector: "target.selectionEnd"},
		},
		Events: []refEventSpec{
			{Method: "OnInput", Event: "input", Props: []string{"event.inputType", "event.isComposing"}},
			{Method: "OnChange", Event: "change"},
		},
	}},
	{"InsEl", "ins", "InsEl creates an <ins> element.", "html", nil},
	{"Kbd", "kbd", "Kbd creates a <kbd> element.", "html", nil},
	{"Label", "label", "Label creates a <label> element.", "html", nil},
	{"Legend", "legend", "Legend creates a <legend> element.", "html", nil},
	{"Li", "li", "Li creates an <li> element.", "html", nil},
	{"Link", "link", "Link creates a <link> element.", "html", nil},
	{"Main", "main", "Main creates a <main> element.", "html", nil},
	{"MapEl", "map", "MapEl creates a <map> element.", "html", nil},
	{"Mark", "mark", "Mark creates a <mark> element.", "html", nil},
	{"Menu", "menu", "Menu creates a <menu> element.", "html", nil},
	{"MenuItem", "menuitem", "MenuItem creates a <menuitem> element.", "html", nil},
	{"Meta", "meta", "Meta creates a <meta> element.", "html", nil},
	{"Meter", "meter", "Meter creates a <meter> element.", "html", &refSpec{
		StateMethod: "MeterState",
		Fields: []stateFieldSpec{
			{Name: "Value", Type: "float64", Selector: "target.value"},
			{Name: "Min", Type: "float64", Selector: "target.min"},
			{Name: "Max", Type: "float64", Selector: "target.max"},
			{Name: "Low", Type: "float64", Selector: "target.low"},
			{Name: "High", Type: "float64", Selector: "target.high"},
			{Name: "Optimum", Type: "float64", Selector: "target.optimum"},
		},
		Events: []refEventSpec{
			{Method: "OnMeterChange", Event: "change"},
			{Method: "OnMeterInput", Event: "input"},
		},
	}},
	{"Nav", "nav", "Nav creates a <nav> element.", "html", nil},
	{"Noscript", "noscript", "Noscript creates a <noscript> element.", "html", nil},
	{"Object", "object", "Object creates an <object> element.", "html", nil},
	{"Ol", "ol", "Ol creates an <ol> element.", "html", nil},
	{"Optgroup", "optgroup", "Optgroup creates an <optgroup> element.", "html", nil},
	{"Option", "option", "Option creates an <option> element.", "html", nil},
	{"Output", "output", "Output creates an <output> element.", "html", nil},
	{"P", "p", "P creates a <p> element.", "html", nil},
	{"Param", "param", "Param creates a <param> element.", "html", nil},
	{"Picture", "picture", "Picture creates a <picture> element.", "html", nil},
	{"Portal", "portal", "Portal creates a <portal> element.", "html", nil},
	{"Pre", "pre", "Pre creates a <pre> element.", "html", nil},
	{"Progress", "progress", "Progress creates a <progress> element.", "html", &refSpec{
		StateMethod: "ProgressState",
		Fields: []stateFieldSpec{
			{Name: "Value", Type: "float64", Selector: "target.value"},
			{Name: "Max", Type: "float64", Selector: "target.max"},
			{Name: "Position", Type: "float64", Selector: "target.position"},
		},
		Events: []refEventSpec{
			{Method: "OnProgressChange", Event: "change"},
		},
	}},
	{"Q", "q", "Q creates a <q> element.", "html", nil},
	{"Rb", "rb", "Rb creates an <rb> element.", "html", nil},
	{"Rp", "rp", "Rp creates an <rp> element.", "html", nil},
	{"Rt", "rt", "Rt creates an <rt> element.", "html", nil},
	{"Rtc", "rtc", "Rtc creates a <rtc> element.", "html", nil},
	{"Ruby", "ruby", "Ruby creates a <ruby> element.", "html", nil},
	{"S", "s", "S creates an <s> element.", "html", nil},
	{"Samp", "samp", "Samp creates a <samp> element.", "html", nil},
	{"Script", "script", "Script creates a <script> element.", "html", nil},
	{"Section", "section", "Section creates a <section> element.", "html", nil},
	{"Select", "select", "Select creates a <select> element.", "html", &refSpec{
		StateMethod: "SelectState",
		Fields: []stateFieldSpec{
			{Name: "Value", Type: "string", Selector: "target.value"},
			{Name: "SelectedIndex", Type: "int", Selector: "target.selectedIndex"},
			{Name: "Length", Type: "int", Selector: "target.length"},
			{Name: "Multiple", Type: "bool", Selector: "target.multiple"},
		},
		Events: []refEventSpec{
			{Method: "OnSelectChange", Event: "change"},
			{Method: "OnSelectInput", Event: "input"},
		},
	}},
	{"Slot", "slot", "Slot creates a <slot> element.", "html", nil},
	{"Small", "small", "Small creates a <small> element.", "html", nil},
	{"Source", "source", "Source creates a <source> element.", "html", nil},
	{"Span", "span", "Span creates a <span> element.", "html", nil},
	{"Strong", "strong", "Strong creates a <strong> element.", "html", nil},
	{"StyleEl", "style", "StyleEl creates a <style> element.", "html", nil},
	{"Sub", "sub", "Sub creates a <sub> element.", "html", nil},
	{"Summary", "summary", "Summary creates a <summary> element.", "html", &refSpec{
		StateMethod: "SummaryState",
		Fields: []stateFieldSpec{
			{Name: "Detail", Type: "int", Selector: "event.detail"},
		},
		Events: []refEventSpec{
			{Method: "OnSummaryClick", Event: "click"},
			{Method: "OnSummaryKeyDown", Event: "keydown"},
		},
	}},
	{"Sup", "sup", "Sup creates a <sup> element.", "html", nil},
	{"Table", "table", "Table creates a <table> element.", "html", nil},
	{"Tbody", "tbody", "Tbody creates a <tbody> element.", "html", nil},
	{"Td", "td", "Td creates a <td> element.", "html", nil},
	{"Template", "template", "Template creates a <template> element.", "html", nil},
	{"Textarea", "textarea", "Textarea creates a <textarea> element.", "html", &refSpec{
		StateMethod: "TextareaState",
		Fields: []stateFieldSpec{
			{Name: "Value", Type: "string", Selector: "target.value"},
			{Name: "SelectionStart", Type: "int", Selector: "target.selectionStart"},
			{Name: "SelectionEnd", Type: "int", Selector: "target.selectionEnd"},
			{Name: "Rows", Type: "int", Selector: "target.rows"},
			{Name: "Cols", Type: "int", Selector: "target.cols"},
			{Name: "Disabled", Type: "bool", Selector: "target.disabled"},
		},
		Events: []refEventSpec{
			{Method: "OnTextareaInput", Event: "input", Props: []string{"event.inputType", "event.isComposing"}},
			{Method: "OnTextareaChange", Event: "change"},
			{Method: "OnTextareaSelect", Event: "select"},
		},
	}},
	{"Tfoot", "tfoot", "Tfoot creates a <tfoot> element.", "html", nil},
	{"Th", "th", "Th creates a <th> element.", "html", nil},
	{"Thead", "thead", "Thead creates a <thead> element.", "html", nil},
	{"Time", "time", "Time creates a <time> element.", "html", nil},
	{"TitleEl", "title", "TitleEl creates a <title> element.", "html", nil},
	{"Tr", "tr", "Tr creates a <tr> element.", "html", nil},
	{"Track", "track", "Track creates a <track> element.", "html", nil},
	{"U", "u", "U creates a <u> element.", "html", nil},
	{"Ul", "ul", "Ul creates a <ul> element.", "html", nil},
	{"Var", "var", "Var creates a <var> element.", "html", nil},
	{"Video", "video", "Video creates a <video> element.", "html", &refSpec{
		Mixins: []string{"media"},
		Events: []refEventSpec{
			{Method: "OnTimeUpdate", Event: "timeupdate", Listen: []string{"durationchange", "play", "pause", "seeking", "seeked"}},
			{Method: "OnPlay", Event: "play", Listen: []string{"playing"}},
			{Method: "OnPause", Event: "pause"},
			{Method: "OnEnded", Event: "ended"},
			{Method: "OnVolumeChange", Event: "volumechange"},
			{Method: "OnSeeking", Event: "seeking"},
			{Method: "OnSeeked", Event: "seeked"},
			{Method: "OnRateChange", Event: "ratechange"},
		},
	}},
	{"Wbr", "wbr", "Wbr creates a <wbr> element.", "html", nil},
	{"Svg", "svg", "Svg creates an <svg> element.", "svg", nil},
	{"Circle", "circle", "Circle creates a <circle> element.", "svg", nil},
	{"ClipPath", "clipPath", "ClipPath creates a <clipPath> element.", "svg", nil},
	{"Defs", "defs", "Defs creates a <defs> element.", "svg", nil},
	{"Ellipse", "ellipse", "Ellipse creates an <ellipse> element.", "svg", nil},
	{"ForeignObject", "foreignObject", "ForeignObject creates a <foreignObject> element.", "svg", nil},
	{"G", "g", "G creates a <g> element.", "svg", nil},
	{"Image", "image", "Image creates an <image> element.", "svg", nil},
	{"Line", "line", "Line creates a <line> element.", "svg", nil},
	{"LinearGradient", "linearGradient", "LinearGradient creates a <linearGradient> element.", "svg", nil},
	{"Marker", "marker", "Marker creates a <marker> element.", "svg", nil},
	{"Mask", "mask", "Mask creates a <mask> element.", "svg", nil},
	{"Path", "path", "Path creates a <path> element.", "svg", nil},
	{"Pattern", "pattern", "Pattern creates a <pattern> element.", "svg", nil},
	{"Polygon", "polygon", "Polygon creates a <polygon> element.", "svg", nil},
	{"Polyline", "polyline", "Polyline creates a <polyline> element.", "svg", nil},
	{"RadialGradient", "radialGradient", "RadialGradient creates a <radialGradient> element.", "svg", nil},
	{"Rect", "rect", "Rect creates a <rect> element.", "svg", nil},
	{"Stop", "stop", "Stop creates a <stop> element.", "svg", nil},
	{"SvgText", "text", "SvgText creates an SVG <text> element.", "svg", nil},
	{"TSpan", "tspan", "TSpan creates a <tspan> element.", "svg", nil},
	{"Use", "use", "Use creates a <use> element.", "svg", nil},
}

func descriptorName(spec tagSpec) string {
	base := spec.Name
	if strings.HasSuffix(base, "El") {
		base = strings.TrimSuffix(base, "El")
	}
	prefix := "HTML"
	if spec.Kind == "svg" {
		prefix = "SVG"
	}
	return prefix + base + "Element"
}

func main() {
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })

	var descriptors strings.Builder
	descriptors.WriteString("// Code generated by tags_gen.go; DO NOT EDIT.\n")
	descriptors.WriteString("package html\n\n")

	for _, t := range tags {
		descriptor := descriptorName(t)
		descriptors.WriteString("type ")
		descriptors.WriteString(descriptor)
		descriptors.WriteString(" struct{}\n")
		descriptors.WriteString("func (")
		descriptors.WriteString(descriptor)
		descriptors.WriteString(") elementDescriptor() {}\n")
		descriptors.WriteString("func (")
		descriptors.WriteString(descriptor)
		descriptors.WriteString(") TagName() string { return \"")
		descriptors.WriteString(t.Tag)
		descriptors.WriteString("\" }\n\n")
	}

	for _, t := range tags {
		descriptor := descriptorName(t)
		descriptors.WriteString("// ")
		descriptors.WriteString(t.Doc)
		descriptors.WriteString("\n")
		descriptors.WriteString("func ")
		descriptors.WriteString(t.Name)
		descriptors.WriteString("(items ...Item) *Element { return el(")
		descriptors.WriteString(descriptor)
		descriptors.WriteString("{}, \"")
		descriptors.WriteString(t.Tag)
		descriptors.WriteString("\", items...) }\n\n")
	}

	descriptorTarget := filepath.Join("tags_generated.go")
	if err := os.WriteFile(descriptorTarget, []byte(descriptors.String()), 0o644); err != nil {
		panic(err)
	}

	generateElementRefs(tags)
	generateHTMLHooks(tags)
	generateHookProvider(tags)
	generatePublicFacade(tags)

	fmt.Println("generated tags_generated.go")
	fmt.Println("generated refs_elements_generated.go")
	fmt.Println("generated html/hooks_generated.go")
	fmt.Println("generated live/hooks_provider_generated.go")
	fmt.Println("generated ../../pkg/live/html/generated.go")
}

func generateElementRefs(specs []tagSpec) {
	var b strings.Builder
	b.WriteString("// Code generated by tags_gen.go; DO NOT EDIT.\n")
	b.WriteString("package html\n\n")
	for _, spec := range specs {
		refName := spec.Name + "Ref"
		descriptor := descriptorName(spec)
		fmt.Fprintf(&b, "type %s struct {\n", refName)
		fmt.Fprintf(&b, "\t*ElementRef[%s]\n", descriptor)
		for _, handler := range handlerSpecs {
			fmt.Fprintf(&b, "\t*%s\n", handler.Type)
		}
		b.WriteString("}\n\n")

		fmt.Fprintf(&b, "func (r *%s) Ref() *ElementRef[%s] {\n", refName, descriptor)
		b.WriteString("\tif r == nil {\n\t\treturn nil\n\t}\n")
		b.WriteString("\treturn r.ElementRef\n}\n\n")

		constructor := "New" + refName
		fmt.Fprintf(&b, "func %s(ref *ElementRef[%s]) *%s {\n", constructor, descriptor, refName)
		b.WriteString("\tif ref == nil {\n\t\treturn nil\n\t}\n")
		fmt.Fprintf(&b, "\treturn &%s{\n", refName)
		b.WriteString("\t\tElementRef: ref,\n")
		for _, handler := range handlerSpecs {
			fmt.Fprintf(&b, "\t\t%s: %s(ref),\n", handler.FieldName, handler.Constructor)
		}
		b.WriteString("\t}\n}\n\n")
	}

	writeFormatted("refs_elements_generated.go", b.String())
}

func generateHTMLHooks(specs []tagSpec) {
	var b strings.Builder
	b.WriteString("// Code generated by tags_gen.go; DO NOT EDIT.\n")
	b.WriteString("package html\n\n")
	for _, spec := range specs {
		useName := "Use" + spec.Name
		descriptor := descriptorName(spec)
		fmt.Fprintf(&b, "func %s(ctx ElementHookContext) *%sRef {\n", useName, spec.Name)
		fmt.Fprintf(&b, "\treturn New%[1]sRef(useElementRef[%s](ctx))\n", spec.Name, descriptor)
		b.WriteString("}\n\n")
	}

	writeFormatted("hooks_generated.go", b.String())
}

func generateHookProvider(specs []tagSpec) {
	var b strings.Builder
	b.WriteString("// Code generated by tags_gen.go; DO NOT EDIT.\n")
	b.WriteString("package live\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"github.com/eleven-am/pondlive/go/internal/dom\"\n")
	b.WriteString("\tinternalhtml \"github.com/eleven-am/pondlive/go/internal/html\"\n")
	b.WriteString(")\n\n")
	b.WriteString("func init() {\n")
	b.WriteString("\tdom.InstallElementRefFactory(func(ctx any, descriptor dom.ElementDescriptor) any {\n")
	b.WriteString("\t\tliveCtx, ok := ctx.(Ctx)\n")
	b.WriteString("\t\tif !ok {\n")
	b.WriteString("\t\t\tpanic(\"live: invalid element hook context\")\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t\tswitch descriptor.(type) {\n")
	for _, spec := range specs {
		descriptor := descriptorName(spec)
		fmt.Fprintf(&b, "\t\tcase internalhtml.%s:\n", descriptor)
		fmt.Fprintf(&b, "\t\t\treturn UseElement[internalhtml.%s](liveCtx)\n", descriptor)
	}
	b.WriteString("\t\tdefault:\n")
	b.WriteString("\t\t\tpanic(fmt.Sprintf(\"live: unsupported element descriptor %T\", descriptor))\n")
	b.WriteString("\t\t}\n")
	b.WriteString("\t})\n")
	b.WriteString("}\n")

	target := filepath.Join("..", "..", "pkg", "live", "hooks_provider_generated.go")
	writeFormatted(target, b.String())
}

func generatePublicFacade(specs []tagSpec) {
	var b strings.Builder
	b.WriteString("// Code generated by tags_gen.go; DO NOT EDIT.\n")
	b.WriteString("// Package html provides public re-exports from internal/html.\n")
	b.WriteString("package html\n\n")
	b.WriteString("import internalhtml \"github.com/eleven-am/pondlive/go/internal/html\"\n\n")

	b.WriteString("// Element descriptor types\n")
	b.WriteString("type (\n")
	for _, spec := range specs {
		descriptor := descriptorName(spec)
		fmt.Fprintf(&b, "\t%s = internalhtml.%s\n", descriptor, descriptor)
	}
	b.WriteString(")\n\n")

	b.WriteString("// Element builder functions\n")
	b.WriteString("var (\n")
	for _, spec := range specs {
		fmt.Fprintf(&b, "\t%s = internalhtml.%s\n", spec.Name, spec.Name)
	}
	b.WriteString(")\n\n")

	b.WriteString("// Element ref types\n")
	b.WriteString("type (\n")
	for _, spec := range specs {
		refName := spec.Name + "Ref"
		fmt.Fprintf(&b, "\t%s = internalhtml.%s\n", refName, refName)
	}
	b.WriteString(")\n\n")

	for _, spec := range specs {
		useName := "Use" + spec.Name
		fmt.Fprintf(&b, "func %s(ctx any) *%sRef {\n", useName, spec.Name)
		fmt.Fprintf(&b, "\treturn internalhtml.%s(ctx)\n", useName)
		b.WriteString("}\n\n")
	}

	target := filepath.Join("..", "..", "pkg", "live", "html", "generated.go")
	writeFormatted(target, b.String())
}

func writeFormatted(target string, src string) {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(target, formatted, 0o644); err != nil {
		panic(err)
	}
}
