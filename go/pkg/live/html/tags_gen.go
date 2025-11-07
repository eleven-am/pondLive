//go:build ignore

package main

import (
	"fmt"
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
		Fields: []stateFieldSpec{
			{Name: "CurrentTime", Type: "float64", Selector: "target.currentTime"},
			{Name: "Duration", Type: "float64", Selector: "target.duration"},
			{Name: "Paused", Type: "bool", Selector: "target.paused"},
			{Name: "Muted", Type: "bool", Selector: "target.muted"},
			{Name: "Volume", Type: "float64", Selector: "target.volume"},
			{Name: "Seeking", Type: "bool", Selector: "target.seeking"},
			{Name: "Ended", Type: "bool", Selector: "target.ended"},
			{Name: "PlaybackRate", Type: "float64", Selector: "target.playbackRate"},
			{Name: "ReadyState", Type: "int", Selector: "target.readyState"},
		},
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
			{Name: "PointerType", Type: "string", Selector: "event.pointerType"},
			{Name: "Buttons", Type: "int", Selector: "event.buttons"},
			{Name: "ClientX", Type: "float64", Selector: "event.clientX"},
			{Name: "ClientY", Type: "float64", Selector: "event.clientY"},
			{Name: "OffsetX", Type: "float64", Selector: "event.offsetX"},
			{Name: "OffsetY", Type: "float64", Selector: "event.offsetY"},
			{Name: "Pressure", Type: "float64", Selector: "event.pressure"},
			{Name: "TangentialPressure", Type: "float64", Selector: "event.tangentialPressure"},
			{Name: "TiltX", Type: "float64", Selector: "event.tiltX"},
			{Name: "TiltY", Type: "float64", Selector: "event.tiltY"},
			{Name: "WheelDeltaX", Type: "float64", Selector: "event.deltaX"},
			{Name: "WheelDeltaY", Type: "float64", Selector: "event.deltaY"},
			{Name: "WheelDeltaZ", Type: "float64", Selector: "event.deltaZ"},
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
			{Name: "AltKey", Type: "bool", Selector: "event.altKey"},
			{Name: "CtrlKey", Type: "bool", Selector: "event.ctrlKey"},
			{Name: "ShiftKey", Type: "bool", Selector: "event.shiftKey"},
			{Name: "MetaKey", Type: "bool", Selector: "event.metaKey"},
			{Name: "Button", Type: "int", Selector: "event.button"},
			{Name: "ClientX", Type: "float64", Selector: "event.clientX"},
			{Name: "ClientY", Type: "float64", Selector: "event.clientY"},
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
			{Method: "OnFocus", Event: "focus"},
			{Method: "OnBlur", Event: "blur"},
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
			{Name: "AltKey", Type: "bool", Selector: "event.altKey"},
			{Name: "CtrlKey", Type: "bool", Selector: "event.ctrlKey"},
			{Name: "ShiftKey", Type: "bool", Selector: "event.shiftKey"},
			{Name: "MetaKey", Type: "bool", Selector: "event.metaKey"},
			{Name: "Button", Type: "int", Selector: "event.button"},
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
		Fields: []stateFieldSpec{
			{Name: "CurrentTime", Type: "float64", Selector: "target.currentTime"},
			{Name: "Duration", Type: "float64", Selector: "target.duration"},
			{Name: "Paused", Type: "bool", Selector: "target.paused"},
			{Name: "Muted", Type: "bool", Selector: "target.muted"},
			{Name: "Volume", Type: "float64", Selector: "target.volume"},
			{Name: "Seeking", Type: "bool", Selector: "target.seeking"},
			{Name: "Ended", Type: "bool", Selector: "target.ended"},
			{Name: "PlaybackRate", Type: "float64", Selector: "target.playbackRate"},
			{Name: "ReadyState", Type: "int", Selector: "target.readyState"},
		},
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

	var refSpecs []tagSpec
	for _, t := range tags {
		if t.Ref != nil {
			refSpecs = append(refSpecs, t)
		}
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

	descriptorTarget := filepath.Join("pkg", "live", "html", "tags_generated.go")
	if err := os.WriteFile(descriptorTarget, []byte(descriptors.String()), 0o644); err != nil {
		panic(err)
	}

	var refs strings.Builder
	refs.WriteString("// Code generated by tags_gen.go; DO NOT EDIT.\n")
	refs.WriteString("package html\n\n")
	if len(refSpecs) > 0 {
		for _, spec := range refSpecs {
			writeRefSpec(&refs, spec)
		}
		writeRefInit(&refs, refSpecs)
	} else {
		refs.WriteString("func init() {}\n")
	}

	refsTarget := filepath.Join("pkg", "live", "html", "refs_generated.go")
	if err := os.WriteFile(refsTarget, []byte(refs.String()), 0o644); err != nil {
		panic(err)
	}

	fmt.Println("generated tags_generated.go")
	fmt.Println("generated refs_generated.go")
}

func writeRefSpec(b *strings.Builder, spec tagSpec) {
	if spec.Ref == nil {
		return
	}
	descriptor := descriptorName(spec)
	stateName := stateStructName(spec)
	buildName := buildFuncName(spec)
	dispatchName := dispatchFuncName(spec)

	fmt.Fprintf(b, "type %s struct {\n", stateName)
	if len(spec.Ref.Fields) == 0 {
		fmt.Fprintf(b, "}\n\n")
	} else {
		for _, field := range spec.Ref.Fields {
			fmt.Fprintf(b, "\t%s %s\n", field.Name, field.Type)
		}
		fmt.Fprintf(b, "}\n\n")
	}

	stateMethod := spec.Ref.StateMethod
	if stateMethod == "" {
		stateMethod = "State"
	}
	fmt.Fprintf(b, "// %s returns the cached snapshot for the <%s> element ref.\n", stateMethod, spec.Tag)
	fmt.Fprintf(b, "func (ref *ElementRef[%s]) %s() %s {\n", descriptor, stateMethod, stateName)
	fmt.Fprintf(b, "\tif ref == nil {\n\t\treturn %s{}\n\t}\n", stateName)
	fmt.Fprintf(b, "\tif raw, ok := ref.CachedState().(%s); ok {\n\t\treturn raw\n\t}\n", stateName)
	fmt.Fprintf(b, "\treturn %s{}\n}\n\n", stateName)

	fmt.Fprintf(b, "func (ref *ElementRef[%s]) %s(event string, snapshot %s, evt Event) Updates {\n", descriptor, dispatchName, stateName)
	fmt.Fprintf(b, "\tvar result Updates\n")
	fmt.Fprintf(b, "\tfor _, raw := range ref.listenersFor(event) {\n")
	fmt.Fprintf(b, "\t\tcb, ok := raw.(func(%s, Event) Updates)\n", stateName)
	fmt.Fprintf(b, "\t\tif !ok {\n\t\t\tcontinue\n\t\t}\n")
	fmt.Fprintf(b, "\t\tif out := cb(snapshot, evt); out != nil {\n\t\t\tresult = out\n\t\t}\n")
	fmt.Fprintf(b, "\t}\n\treturn result\n}\n\n")

	fmt.Fprintf(b, "func %s(prev %s, payload map[string]any) %s {\n", buildName, stateName, stateName)
	fmt.Fprintf(b, "\tnext := prev\n")
	for _, field := range spec.Ref.Fields {
		helper := payloadHelper(field.Type)
		fmt.Fprintf(b, "\tnext.%s = %s(payload, %q, prev.%s)\n", field.Name, helper, field.Selector, field.Name)
	}
	fmt.Fprintf(b, "\treturn next\n}\n\n")

	applyName := applyFuncName(spec)
	fmt.Fprintf(b, "func %s(ref *ElementRef[%s]) {\n", applyName, descriptor)
	fmt.Fprintf(b, "\tif ref == nil {\n\t\treturn\n\t}\n")
	for _, event := range spec.Ref.Events {
		props := collectEventProps(event, spec.Ref.Fields)
		fmt.Fprintf(b, "\t{\n")
		fmt.Fprintf(b, "\t\thandler := func(evt Event) Updates {\n")
		fmt.Fprintf(b, "\t\t\tprev := ref.%s()\n", stateMethod)
		fmt.Fprintf(b, "\t\t\tnext := %s(prev, evt.Payload)\n", buildName)
		fmt.Fprintf(b, "\t\t\tref.updateState(next)\n")
		fmt.Fprintf(b, "\t\t\treturn ref.%s(%q, next, evt)\n", dispatchName, event.Event)
		fmt.Fprintf(b, "\t\t}\n")
		if len(event.Listen) == 0 && len(props) == 0 {
			fmt.Fprintf(b, "\t\topts := defaultEventOptions(%q)\n", event.Event)
		} else {
			fmt.Fprintf(b, "\t\topts := mergeEventOptions(defaultEventOptions(%q), EventOptions{\n", event.Event)
			if len(event.Listen) > 0 {
				fmt.Fprintf(b, "\t\t\tListen: []string{%s},\n", formatStringSlice(event.Listen))
			}
			if len(props) > 0 {
				fmt.Fprintf(b, "\t\t\tProps: []string{%s},\n", formatStringSlice(props))
			}
			fmt.Fprintf(b, "\t\t})\n")
		}
		fmt.Fprintf(b, "\t\tbinding := (EventBinding{Handler: handler}).withOptions(opts, %q)\n", event.Event)
		fmt.Fprintf(b, "\t\tref.Bind(%q, binding)\n", event.Event)
		fmt.Fprintf(b, "\t}\n")
	}
	fmt.Fprintf(b, "}\n\n")

	for _, event := range spec.Ref.Events {
		doc := strings.TrimSpace(event.Doc)
		if doc == "" {
			doc = fmt.Sprintf("%s registers a handler for the %q event.", event.Method, event.Event)
		}
		fmt.Fprintf(b, "// %s\n", doc)
		fmt.Fprintf(b, "func (ref *ElementRef[%s]) %s(handler func(%s, Event) Updates) {\n", descriptor, event.Method, stateName)
		fmt.Fprintf(b, "\tif ref == nil || handler == nil {\n\t\treturn\n\t}\n")
		fmt.Fprintf(b, "\tref.addListener(%q, handler)\n", event.Event)
		fmt.Fprintf(b, "}\n\n")
	}
}

func writeRefInit(b *strings.Builder, specs []tagSpec) {
	fmt.Fprintf(b, "func init() {\n")
	fmt.Fprintf(b, "\tprev := applyRefDefaultsFunc\n")
	fmt.Fprintf(b, "\tapplyRefDefaultsFunc = func(ref any) {\n")
	fmt.Fprintf(b, "\t\tprev(ref)\n")
	fmt.Fprintf(b, "\t\tswitch typed := ref.(type) {\n")
	for _, spec := range specs {
		descriptor := descriptorName(spec)
		fmt.Fprintf(b, "\t\tcase *ElementRef[%s]:\n", descriptor)
		fmt.Fprintf(b, "\t\t\t%s(typed)\n", applyFuncName(spec))
	}
	fmt.Fprintf(b, "\t\t}\n\t}\n}\n")
}

func stateStructName(spec tagSpec) string {
	if spec.Ref != nil && spec.Ref.StateName != "" {
		return spec.Ref.StateName
	}
	return descriptorName(spec) + "State"
}

func buildFuncName(spec tagSpec) string {
	return "build" + descriptorName(spec) + "State"
}

func dispatchFuncName(spec tagSpec) string {
	return "dispatch" + descriptorName(spec) + "Event"
}

func applyFuncName(spec tagSpec) string {
	return "apply" + descriptorName(spec) + "Defaults"
}

func formatStringSlice(values []string) string {
	quoted := make([]string, 0, len(values))
	for _, v := range values {
		quoted = append(quoted, fmt.Sprintf("%q", v))
	}
	return strings.Join(quoted, ", ")
}

func collectEventProps(event refEventSpec, fields []stateFieldSpec) []string {
	seen := map[string]struct{}{}
	props := make([]string, 0, len(event.Props)+len(fields))
	for _, p := range event.Props {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		props = append(props, p)
	}
	for _, field := range fields {
		sel := strings.TrimSpace(field.Selector)
		if sel == "" {
			continue
		}
		if _, ok := seen[sel]; ok {
			continue
		}
		seen[sel] = struct{}{}
		props = append(props, sel)
	}
	return props
}

func payloadHelper(fieldType string) string {
	switch fieldType {
	case "float64":
		return "payloadFloat"
	case "bool":
		return "payloadBool"
	case "string":
		return "payloadString"
	case "int":
		return "payloadInt"
	default:
		panic(fmt.Sprintf("unsupported state field type %q", fieldType))
	}
}
