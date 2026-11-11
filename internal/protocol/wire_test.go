package protocol

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"unicode"
)

const clientSrc = "../../client/src"

func TestClientWireFileIsGeneratedFromTheGoTables(t *testing.T) {
	want := WireTS()
	if dir := os.Getenv("WIRE_OUT"); dir != "" {
		if err := os.WriteFile(filepath.Join(dir, "wire.ts"), []byte(want), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	got, err := os.ReadFile(filepath.Join(clientSrc, "wire.ts"))
	if err != nil {
		t.Fatalf("missing wire.ts: run `make client-wire`")
	}
	if string(got) != want {
		t.Fatalf("wire.ts is stale: run `make client-wire`")
	}
}

func TestEditSignaturesMatchTheEncodedArrays(t *testing.T) {
	samples := map[Op]Edit{
		OpText: TextEdit(1, "x"), OpAttr: AttrEdit(1, "a", "b"), OpRemoveAttr: RemoveAttrEdit(1, "a"),
		OpStyle: StyleEdit(1, "a", "b"), OpHTML: HTMLEdit(1, "x"), OpInsert: InsertEdit(1, 2, "x"),
		OpMove: MoveEdit(1, 2, 3), OpRemove: RemoveEdit(1), OpClear: ClearEdit(1),
	}
	if len(samples) != len(EditSignatures) {
		t.Fatalf("%d ops sampled for %d signatures", len(samples), len(EditSignatures))
	}
	for _, sig := range EditSignatures {
		raw, err := json.Marshal(samples[sig.Op])
		if err != nil {
			t.Fatal(err)
		}
		var arr []any
		if err := json.Unmarshal(raw, &arr); err != nil {
			t.Fatal(err)
		}
		if arr[0] != string(sig.Op) || len(arr) != len(sig.Sig)+1 {
			t.Fatalf("%s encodes as %s but the signature is %q", sig.Op, raw, sig.Sig)
		}
		for i := 0; i < len(sig.Sig); i++ {
			_, isString := arr[i+1].(string)
			if isString != (sig.Sig[i] == 's') {
				t.Fatalf("%s argument %d: encoded %s, signature %q", sig.Op, i+1, raw, sig.Sig)
			}
		}
	}
}

func TestUnsafeURLRunesCoverEveryWhitespaceAndControlCharacter(t *testing.T) {
	for r := rune(0); r <= unicode.MaxRune; r++ {
		must := unicode.IsSpace(r) || unicode.IsControl(r) || r == '\\' || r == 0xfeff
		if must && !UnsafeURLRune(r) {
			t.Fatalf("U+%04X must be unsafe", r)
		}
		if !must && UnsafeURLRune(r) {
			t.Fatalf("U+%04X must not be unsafe", r)
		}
	}
}

func TestSafeRedirectDecidesOnTheTargetAndTheScheme(t *testing.T) {
	cases := []struct {
		target, scheme string
		want           bool
	}{
		{"/a?b=1#c", "", true},
		{"https://x.test/a", "https", true},
		{"HTTP://x.test", "HTTP", true},
		{"javascript:alert(1)", "javascript", false},
		{"", "", false},
		{"/a b", "", false},
		{"/a\u007fb", "", false},
		{"/a\\b", "", false},
		{"/a b", "", false},
	}
	for _, c := range cases {
		if got := SafeRedirect(c.target, c.scheme); got != c.want {
			t.Errorf("SafeRedirect(%q, %q) = %v, want %v", c.target, c.scheme, got, c.want)
		}
	}
}

func TestMarkerGrammarRoundTrips(t *testing.T) {
	for _, kind := range MarkerKinds {
		got, id, ok := ParseMarker(Marker(kind, 42))
		if !ok || got != kind || id != 42 {
			t.Fatalf("%s: parsed %q %d %v", kind, got, id, ok)
		}
	}
	for _, bad := range []string{"", "t:", "x:1", "t:1a", "r:-1", "t1", "/t:1"} {
		if IsMarker(bad) {
			t.Fatalf("%q must not be a marker", bad)
		}
	}
}

func TestObservedPropsMatchTheObserveAllowlist(t *testing.T) {
	allowed := map[string]bool{}
	for _, k := range RefKinds {
		for _, p := range k.Observe {
			allowed[p] = true
		}
	}
	for _, o := range ObservedProps {
		if !allowed[o.Prop] {
			t.Errorf("%s is observed on the wire but not allowed by any ref kind", o.Prop)
		}
		delete(allowed, o.Prop)
	}
	for p := range allowed {
		t.Errorf("%s is allowed for observe but has no entry in ObservedProps", p)
	}
}

func TestEveryActionAndFrameTypeIsUniqueAndListedInTheGeneratedFile(t *testing.T) {
	ts := WireTS()
	seen := map[string]bool{}
	for _, f := range FrameTypes {
		if seen[f] {
			t.Fatalf("duplicate frame type %q", f)
		}
		seen[f] = true
		if !strings.Contains(ts, "'"+f+"'") {
			t.Fatalf("frame type %q is missing from wire.ts", f)
		}
	}
	for _, s := range append(append([]ActionSet(nil), ActionSets...), ScriptActions) {
		for _, a := range append(append([]string(nil), s.Server...), s.Client...) {
			if !strings.Contains(ts, "'"+a+"'") {
				t.Fatalf("action %q is missing from wire.ts", a)
			}
		}
	}
}

func TestHandlerSpecStructMatchesTheSpecKeyTable(t *testing.T) {
	typ := reflect.TypeFor[HandlerSpec]()
	fromStruct := map[string]bool{}
	for i := 0; i < typ.NumField(); i++ {
		name, _, _ := strings.Cut(typ.Field(i).Tag.Get("json"), ",")
		fromStruct[name] = true
	}
	for _, k := range HandlerSpecKeys {
		if !fromStruct[k.Key] {
			t.Errorf("HandlerSpec has no field for the wire key %q", k.Key)
		}
		delete(fromStruct, k.Key)
	}
	for extra := range fromStruct {
		t.Errorf("HandlerSpec has the field %q which is not in HandlerSpecKeys", extra)
	}
}

type fixtureBase struct {
	ID uint32 `json:"id"`
}

type fixtureNested struct {
	Name string `json:"name"`
}

type fixtureWire struct {
	fixtureBase
	Required string              `json:"required"`
	Optional string              `json:"optional,omitempty"`
	Tagged   string              `json:"tagged" ts:"SomeUnion"`
	Pointer  *bool               `json:"pointer,omitempty"`
	List     []any               `json:"list,omitempty"`
	Lookup   map[string][]string `json:"lookup"`
	Dynamic  any                 `json:"dynamic"`
	Count    uint64              `json:"count"`
	Skipped  string              `json:"-"`
	Untagged string
}

func TestStructTSFollowsTheGoJSONTagsExactly(t *testing.T) {
	got := structTS(reflect.TypeFor[fixtureWire]())
	want := "export interface fixtureWire extends fixtureBase {\n" +
		"    required: string;\n" +
		"    optional?: string;\n" +
		"    tagged: SomeUnion;\n" +
		"    pointer?: boolean;\n" +
		"    list?: unknown[];\n" +
		"    lookup: Record<string, string[]>;\n" +
		"    dynamic: unknown;\n" +
		"    count: number;\n" +
		"}\n"
	if got != want {
		t.Fatalf("got:\n%s\nwant:\n%s", got, want)
	}
}

func TestStructTSRefusesATypeItCannotMap(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("an unmappable field type was silently emitted")
		}
	}()
	structTS(reflect.TypeFor[struct {
		C chan int `json:"c"`
	}]())
}

func TestEveryFrameTypeHasOneGeneratedFrameStruct(t *testing.T) {
	typed := map[string]string{}
	for _, st := range WireStructs {
		if f, ok := st.FieldByName("T"); ok {
			typed[strings.TrimSuffix(f.Tag.Get("ts"), "Type")] = st.Name()
		}
	}
	for _, frame := range FrameTypes {
		name := strings.ToUpper(frame[:1]) + frame[1:]
		if typed[name] == "" {
			t.Errorf("frame type %q has no struct with ts:%q", frame, name+"Type")
		}
	}
	if len(typed) != len(FrameTypes) {
		t.Errorf("%d frame structs for %d frame types: %v", len(typed), len(FrameTypes), typed)
	}
}

func TestEveryActionPayloadTypeIsAGeneratedStruct(t *testing.T) {
	generated := map[string]bool{}
	for _, st := range WireStructs {
		generated[st.Name()] = true
	}
	for _, set := range ActionSets {
		for action, payload := range set.Payload {
			if payload != "undefined" && payload != "Edit[]" && !generated[payload] {
				t.Errorf("%s.%s payload %s is not in WireStructs", set.Name, action, payload)
			}
		}
	}
}
