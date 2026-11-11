package protocol

import (
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/eleven-am/pondlive/internal/route"
)

const (
	FrameBoot     = "boot"
	FrameJoin     = "join"
	FrameJoined   = "joined"
	FrameDeclined = "declined"
	FrameTok      = "tok"
	FrameEvack    = "evack"
	FrameAck      = "ack"
	FrameResync   = "resync"
)

var FrameTypes = []string{FrameBoot, FrameJoin, FrameJoined, FrameDeclined, FrameTok, FrameEvack, FrameAck, FrameResync}

const ScriptTopicPrefix = "script:"

func ScriptTopic(scriptID string) Topic { return Topic(ScriptTopicPrefix + scriptID) }

func IsScriptTopic(topic Topic) bool { return strings.HasPrefix(string(topic), ScriptTopicPrefix) }

type ActionSet struct {
	Name    string
	Topic   Topic
	Server  []string
	Client  []string
	Payload map[string]string
}

var ActionSets = []ActionSet{
	{
		Name: "Frame", Topic: TopicFrame,
		Server:  []string{string(FramePatchAction), string(FrameResetAction)},
		Payload: map[string]string{"patch": "Edit[]", "reset": "ResetPayload"},
	},
	{
		Name: "Router", Topic: RouteHandler,
		Server: []string{
			string(RouterPushAction), string(RouterReplaceAction), string(RouterBackAction),
			string(RouterForwardAction), string(RouterRedirectAction),
		},
		Client: []string{string(RouterPopstateAction)},
		Payload: map[string]string{
			"push": "RouterNavPayload", "replace": "RouterNavPayload", "back": "undefined",
			"forward": "undefined", "redirect": "RouterRedirectPayload", "popstate": "RouterPopstatePayload",
		},
	},
	{
		Name: "DOM", Topic: DOMHandler,
		Server: []string{
			string(DOMCallAction), string(DOMSetAction), string(DOMQueryAction),
			string(DOMAsyncAction), string(DOMObserveAction), string(DOMUnobserveAction),
		},
		Client: []string{string(DOMResponseAction), string(DOMValueAction)},
		Payload: map[string]string{
			"call": "DOMCallPayload", "set": "DOMSetPayload", "query": "DOMQueryPayload",
			"async": "DOMAsyncPayload", "observe": "DOMObservePayload", "unobserve": "DOMUnobservePayload",
			"response": "DOMResponsePayload", "value": "DOMValuePayload",
		},
	},
}

var ScriptActions = ActionSet{
	Name: "Script", Server: []string{string(ScriptSendAction)}, Client: []string{string(ScriptMessageAction)},
}

type EditSignature struct {
	Op  Op
	Sig string
}

var EditSignatures = []EditSignature{
	{OpText, "is"}, {OpAttr, "iss"}, {OpRemoveAttr, "is"}, {OpStyle, "iss"}, {OpHTML, "is"},
	{OpInsert, "irs"}, {OpMove, "iir"}, {OpRemove, "i"}, {OpClear, "i"},
}

const (
	SpecEvent          = "e"
	SpecPrevent        = "pv"
	SpecStop           = "sp"
	SpecPassive        = "ps"
	SpecOnce           = "o"
	SpecCapture        = "c"
	SpecCapturePointer = "cp"
	SpecDebounce       = "db"
	SpecThrottle       = "th"
	SpecKeys           = "k"
	SpecFields         = "p"

	ScriptSpecID     = "i"
	ScriptSpecSource = "s"
)

type SpecKey struct {
	Key  string
	TS   string
	Need bool
}

var HandlerSpecKeys = []SpecKey{
	{SpecEvent, "string", true},
	{SpecPrevent, "number", false},
	{SpecStop, "number", false},
	{SpecPassive, "number", false},
	{SpecOnce, "number", false},
	{SpecCapture, "number", false},
	{SpecDebounce, "number", false},
	{SpecThrottle, "number", false},
	{SpecKeys, "string[]", false},
	{SpecCapturePointer, "number", false},
	{SpecFields, "string", false},
}

type HandlerSpec struct {
	Event          string   `json:"e"`
	Prevent        int      `json:"pv,omitempty"`
	Stop           int      `json:"sp,omitempty"`
	Passive        int      `json:"ps,omitempty"`
	Once           int      `json:"o,omitempty"`
	Capture        int      `json:"c,omitempty"`
	Debounce       int      `json:"db,omitempty"`
	Throttle       *int     `json:"th,omitempty"`
	Keys           []string `json:"k,omitempty"`
	CapturePointer int      `json:"cp,omitempty"`
	Fields         string   `json:"p,omitempty"`
}

var ScriptSpecKeys = []SpecKey{
	{ScriptSpecID, "string", true},
	{ScriptSpecSource, "string", true},
}

const (
	AttrID        = "data-pi"
	AttrHandlers  = "data-pon"
	AttrScript    = "data-pscript"
	AttrRaw       = "data-praw"
	AttrExiting   = "data-exiting"
	AttrSelection = "data-psel"
	AttrResyncing = "data-pond-resyncing"
)

var ReservedAttrs = []string{AttrID, AttrHandlers, AttrScript, AttrRaw, AttrExiting}

func IsReservedAttr(name string) bool {
	name = strings.ToLower(name)
	for _, r := range ReservedAttrs {
		if r == name {
			return true
		}
	}
	return false
}

const (
	MarkerText      = "t"
	MarkerRegion    = "r"
	MarkerRegionEnd = "/r"
)

var MarkerKinds = []string{MarkerText, MarkerRegion, MarkerRegionEnd}

func Marker(kind string, id uint32) string {
	return kind + ":" + strconv.FormatUint(uint64(id), 10)
}

func ParseMarker(s string) (kind string, id uint32, ok bool) {
	kind, digits, found := strings.Cut(s, ":")
	if !found || digits == "" {
		return "", 0, false
	}
	known := false
	for _, k := range MarkerKinds {
		if k == kind {
			known = true
		}
	}
	if !known {
		return "", 0, false
	}
	for i := 0; i < len(digits); i++ {
		if digits[i] < '0' || digits[i] > '9' {
			return "", 0, false
		}
	}
	n, err := strconv.ParseUint(digits, 10, 32)
	if err != nil {
		return kind, 0, true
	}
	return kind, uint32(n), true
}

func IsMarker(s string) bool {
	_, _, ok := ParseMarker(s)
	return ok
}

const (
	FileName     = "n"
	FileSize     = "s"
	FileType     = "t"
	FileModified = "m"
)

type ObservedProp struct {
	Prop   string
	Events []string
	Frame  bool
}

var ObservedProps = []ObservedProp{
	{"currentTime", []string{"timeupdate", "seeked"}, true},
	{"duration", []string{"durationchange", "loadedmetadata"}, false},
	{"paused", []string{"play", "pause", "ended"}, false},
	{"ended", []string{"play", "pause", "ended", "seeked"}, false},
	{"volume", []string{"volumechange"}, false},
	{"muted", []string{"volumechange"}, false},
	{"focused", []string{"focus", "blur"}, false},
	{"scroll", []string{"scroll"}, true},
	{"rect", nil, true},
}

const (
	ObserveImmediate = -1
	ObserveFrame     = 0
)

func ObservedPerFrame(prop string) bool {
	for _, o := range ObservedProps {
		if o.Prop == prop {
			return o.Frame
		}
	}
	return false
}

type unsafeRange struct{ lo, hi rune }

var unsafeURLRanges = []unsafeRange{
	{0x0000, 0x0020}, {0x005c, 0x005c}, {0x007f, 0x00a0}, {0x1680, 0x1680}, {0x2000, 0x200a},
	{0x2028, 0x2029}, {0x202f, 0x202f}, {0x205f, 0x205f}, {0x3000, 0x3000}, {0xfeff, 0xfeff},
}

func UnsafeURLRune(r rune) bool {
	for _, u := range unsafeURLRanges {
		if r >= u.lo && r <= u.hi {
			return true
		}
	}
	return false
}

var RedirectSchemes = []string{"http", "https"}

func SafeRedirectScheme(scheme string) bool {
	if scheme == "" {
		return true
	}
	scheme = strings.ToLower(scheme)
	for _, s := range RedirectSchemes {
		if s == scheme {
			return true
		}
	}
	return false
}

func SafeRedirect(target string, scheme string) bool {
	if target == "" {
		return false
	}
	for _, r := range target {
		if UnsafeURLRune(r) {
			return false
		}
	}
	return SafeRedirectScheme(scheme)
}

func quote(items []string) string {
	out := make([]string, len(items))
	for i, s := range items {
		out[i] = "'" + s + "'"
	}
	return strings.Join(out, " | ")
}

func tsUnion(items []string) string {
	if len(items) == 0 {
		return "never"
	}
	return quote(items)
}

func escapeClass(r rune) string { return fmt.Sprintf("\\u%04x", r) }

func WireTS() string {
	var b strings.Builder
	b.WriteString("// Code generated by internal/protocol/wire.go; DO NOT EDIT.\n\n")
	writeRefKinds(&b)
	writeTopics(&b)
	writeFrames(&b)
	writeUnions(&b)
	writeEdits(&b)
	writeSpecs(&b)
	writeAttrs(&b)
	writeMarkers(&b)
	writeFields(&b)
	writeObserved(&b)
	writeShapes(&b)
	writeURLSafety(&b)
	writeStructs(&b)
	return b.String()
}

func writeTopics(b *strings.Builder) {
	var statics []string
	for _, s := range ActionSets {
		statics = append(statics, string(s.Topic))
	}
	b.WriteString("\nexport type StaticTopic = " + quote(statics) + ";\n")
	b.WriteString("export type ScriptTopic = `" + ScriptTopicPrefix + "${string}`;\n")
	b.WriteString("export const STATIC_TOPICS: StaticTopic[] = [" + strings.ReplaceAll(quote(statics), " | ", ", ") + "];\n")
	b.WriteString("export const SCRIPT_TOPIC_PREFIX = '" + ScriptTopicPrefix + "';\n")
	b.WriteString("export type HandlerTopic = '" + string(HandlerTopic) + "';\n")
	b.WriteString("export type Topic = StaticTopic | ScriptTopic;\n")
	b.WriteString("export type EventTopic = Topic | HandlerTopic;\n")
	all := append(append([]ActionSet(nil), ActionSets...), ScriptActions)
	for _, s := range all {
		b.WriteString("export type " + s.Name + "ServerAction = " + tsUnion(s.Server) + ";\n")
		b.WriteString("export type " + s.Name + "ClientAction = " + tsUnion(s.Client) + ";\n")
	}
	b.WriteString("export interface StaticTopicActionMap {\n")
	for _, s := range ActionSets {
		b.WriteString("    " + string(s.Topic) + ": {\n")
		for _, a := range append(append([]string(nil), s.Server...), s.Client...) {
			b.WriteString("        " + a + ": " + s.Payload[a] + ";\n")
		}
		b.WriteString("    };\n")
	}
	b.WriteString("}\n")
}

func writeFrames(b *strings.Builder) {
	b.WriteString("\nexport type FrameType = " + quote(FrameTypes) + ";\n")
	for _, f := range FrameTypes {
		b.WriteString("export type " + strings.ToUpper(f[:1]) + f[1:] + "Type = '" + f + "';\n")
	}
}

func writeEdits(b *strings.Builder) {
	b.WriteString("\nexport type Edit =\n")
	kinds := map[byte]string{'i': "number", 'r': "number", 's': "string"}
	for _, e := range EditSignatures {
		parts := []string{"'" + string(e.Op) + "'"}
		for i := 0; i < len(e.Sig); i++ {
			parts = append(parts, kinds[e.Sig[i]])
		}
		b.WriteString("    | [" + strings.Join(parts, ", ") + "]\n")
	}
	b.WriteString(";\n\nexport type EditOp = Edit[0];\n\n")
	b.WriteString("export const SIGNATURES: Record<EditOp, string> = {\n")
	for _, e := range EditSignatures {
		b.WriteString("    " + tsKey(string(e.Op)) + ": '" + e.Sig + "',\n")
	}
	b.WriteString("};\n")
}

func tsKey(k string) string {
	if k == "" || strings.ContainsAny(k, "-/: ") {
		return "'" + k + "'"
	}
	return k
}

func writeSpecs(b *strings.Builder) {
	write := func(name string, keys []SpecKey) {
		b.WriteString("\nexport interface " + name + " {\n")
		for _, k := range keys {
			opt := "?"
			if k.Need {
				opt = ""
			}
			b.WriteString("    " + k.Key + opt + ": " + k.TS + ";\n")
		}
		b.WriteString("}\n")
	}
	write("HandlerSpec", HandlerSpecKeys)
	write("ScriptSpec", ScriptSpecKeys)
}

func writeAttrs(b *strings.Builder) {
	b.WriteString("\nexport const ATTR_ID = '" + AttrID + "';\n")
	b.WriteString("export const ATTR_HANDLERS = '" + AttrHandlers + "';\n")
	b.WriteString("export const ATTR_SCRIPT = '" + AttrScript + "';\n")
	b.WriteString("export const ATTR_RAW = '" + AttrRaw + "';\n")
	b.WriteString("export const ATTR_EXITING = '" + AttrExiting + "';\n")
	b.WriteString("export const ATTR_SELECTION = '" + AttrSelection + "';\n")
	b.WriteString("export const ATTR_RESYNCING = '" + AttrResyncing + "';\n")
}

func writeMarkers(b *strings.Builder) {
	alts := make([]string, len(MarkerKinds))
	for i, k := range MarkerKinds {
		alts[i] = strings.ReplaceAll(k, "/", "\\/")
	}
	b.WriteString("\nexport const MARKER = /^(" + strings.Join(alts, "|") + "):(\\d+)$/;\n")
}

func writeFields(b *strings.Builder) {
	b.WriteString("\nexport type ComputedField = " + quote(ComputedFields) + ";\n")
	b.WriteString("export interface FileWire {\n")
	b.WriteString("    " + FileName + ": string;\n    " + FileSize + ": number;\n    " + FileType + ": string;\n    " + FileModified + ": number;\n}\n")
	b.WriteString("export type TouchWire = [number, number, number];\n")
}

func writeObserved(b *strings.Builder) {
	props := make([]string, len(ObservedProps))
	for i, o := range ObservedProps {
		props[i] = o.Prop
	}
	b.WriteString("\nexport type ObservedProp = " + quote(props) + ";\n")
	b.WriteString("export const OBSERVED_EVENTS: Record<ObservedProp, string[]> = {\n")
	for _, o := range ObservedProps {
		events := make([]string, len(o.Events))
		for i, e := range o.Events {
			events[i] = "'" + e + "'"
		}
		b.WriteString("    " + o.Prop + ": [" + strings.Join(events, ", ") + "],\n")
	}
	b.WriteString("};\n")
}

func writeShapes(b *strings.Builder) {
	shape := func(name string, keys []string) {
		b.WriteString("export interface " + name + " {\n")
		for _, k := range keys {
			b.WriteString("    " + k + ": number;\n")
		}
		b.WriteString("}\n")
	}
	b.WriteString("\n")
	shape("RectWire", RectKeys)
	shape("ScrollWire", ScrollKeys)
}

func writeURLSafety(b *strings.Builder) {
	var class strings.Builder
	for _, u := range unsafeURLRanges {
		switch {
		case u.lo == 0x5c:
			class.WriteString("\\\\")
		case u.lo == u.hi:
			class.WriteString(escapeClass(u.lo))
		default:
			class.WriteString(escapeClass(u.lo) + "-" + escapeClass(u.hi))
		}
	}
	b.WriteString("\nexport const UNSAFE_URL = /[" + class.String() + "]/;\n")
	b.WriteString("export const SAFE_PROTOCOL = /^(" + strings.Join(RedirectSchemes, "|") + "):$/;\n")
}

func writeRefKinds(b *strings.Builder) {
	kinds := make([]string, 0, len(RefKinds))
	for k := range RefKinds {
		kinds = append(kinds, k)
	}
	sort.Strings(kinds)
	b.WriteString("export type RefOp = 'method' | 'set' | 'query' | 'observe';\n\n")
	b.WriteString("export const REF_KINDS: Record<string, Partial<Record<RefOp, string>>> = {\n")
	for _, name := range kinds {
		k := RefKinds[name]
		b.WriteString("    " + name + ": {")
		first := true
		for _, op := range []RefOp{RefMethod, RefSet, RefQuery, RefObserve} {
			names := refNames(k, op)
			if len(names) == 0 {
				continue
			}
			if !first {
				b.WriteString(",")
			}
			first = false
			b.WriteString(" " + string(op) + ": ' " + strings.Join(names, " ") + " '")
		}
		b.WriteString(" },\n")
	}
	b.WriteString("};\n\nexport function refAllows(kind: string | undefined, op: RefOp, name: string): boolean {\n")
	b.WriteString("    const key = ` ${name} `;\n")
	b.WriteString("    const own = REF_KINDS[kind ?? ''];\n    return !!own && [own, REF_KINDS.ElementRef].some((k) => !!k[op]?.includes(key));\n}\n")
}

type NamedUnion struct {
	Name   string
	Values []string
}

var WireUnions = []NamedUnion{
	{"JoinedMode", JoinedModes},
	{"ResetCause", ResetCauses},
	{"ResyncReason", ResyncReasons},
}

var WireStructs = []reflect.Type{
	reflect.TypeFor[route.Location](),
	reflect.TypeFor[Event](),
	reflect.TypeFor[ClientEvt](),
	reflect.TypeFor[ClientAck](),
	reflect.TypeFor[ClientResync](),
	reflect.TypeFor[ClientConfig](),
	reflect.TypeFor[Boot](),
	reflect.TypeFor[JoinFrame](),
	reflect.TypeFor[JoinedFrame](),
	reflect.TypeFor[EvackFrame](),
	reflect.TypeFor[DeclinedFrame](),
	reflect.TypeFor[TokFrame](),
	reflect.TypeFor[Message](),
	reflect.TypeFor[ResetPayload](),
	reflect.TypeFor[RouterNavPayload](),
	reflect.TypeFor[RouterPopstatePayload](),
	reflect.TypeFor[RouterRedirectPayload](),
	reflect.TypeFor[DOMCallPayload](),
	reflect.TypeFor[DOMSetPayload](),
	reflect.TypeFor[DOMQueryPayload](),
	reflect.TypeFor[DOMAsyncPayload](),
	reflect.TypeFor[DOMObservePayload](),
	reflect.TypeFor[DOMUnobservePayload](),
	reflect.TypeFor[DOMValuePayload](),
	reflect.TypeFor[DOMResponsePayload](),
	reflect.TypeFor[ScriptPayload](),
}

func writeUnions(b *strings.Builder) {
	b.WriteString("\n")
	for _, u := range WireUnions {
		b.WriteString("export type " + u.Name + " = " + quote(u.Values) + ";\n")
	}
}

func writeStructs(b *strings.Builder) {
	for _, t := range WireStructs {
		b.WriteString("\n" + structTS(t))
	}
}

func structTS(t reflect.Type) string {
	var b strings.Builder
	b.WriteString("export interface " + t.Name())
	var parents []string
	for i := 0; i < t.NumField(); i++ {
		if f := t.Field(i); f.Anonymous {
			parents = append(parents, f.Type.Name())
		}
	}
	if len(parents) > 0 {
		b.WriteString(" extends " + strings.Join(parents, ", "))
	}
	b.WriteString(" {\n")
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		name, opts, _ := strings.Cut(f.Tag.Get("json"), ",")
		if f.Anonymous || name == "-" || name == "" {
			continue
		}
		opt := ""
		if opts == "omitempty" {
			opt = "?"
		}
		b.WriteString("    " + tsKey(name) + opt + ": " + tsType(f.Type, f.Tag.Get("ts")) + ";\n")
	}
	b.WriteString("}\n")
	return b.String()
}

func tsType(t reflect.Type, override string) string {
	if override != "" {
		return override
	}
	for _, s := range WireStructs {
		if s == t {
			return t.Name()
		}
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Interface:
		return "unknown"
	case reflect.Pointer:
		return tsType(t.Elem(), "")
	case reflect.Slice, reflect.Array:
		return tsType(t.Elem(), "") + "[]"
	case reflect.Map:
		return "Record<string, " + tsType(t.Elem(), "") + ">"
	}
	panic("protocol: no TypeScript mapping for " + t.String() + "; register the struct in WireStructs or give the field a ts tag")
}
