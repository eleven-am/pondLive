package protocol

import (
	"encoding/json"
	"testing"
)

func TestEditWireShapes(t *testing.T) {
	cases := []struct {
		edit Edit
		want string
	}{
		{TextEdit(7, "1"), `["t",7,"1"]`},
		{TextEdit(7, ""), `["t",7,""]`},
		{AttrEdit(2, "class", `a"b`), `["a",2,"class","a\"b"]`},
		{RemoveAttrEdit(2, "disabled"), `["ra",2,"disabled"]`},
		{StyleEdit(4, "color", "red"), `["st",4,"color","red"]`},
		{HTMLEdit(9, "<b>x</b>"), `["h",9,"<b>x</b>"]`},
		{InsertEdit(3, 0, "<li data-pi=\"5\">x</li>"), `["ins",3,0,"<li data-pi=\"5\">x</li>"]`},
		{MoveEdit(5, 3, 8), `["mv",5,3,8]`},
		{RemoveEdit(5), `["rm",5]`},
		{ClearEdit(3), `["clr",3]`},
	}
	for _, c := range cases {
		got, err := json.Marshal(c.edit)
		if err != nil {
			t.Fatalf("%v: %v", c.edit, err)
		}
		var compact, want any
		if err := json.Unmarshal(got, &compact); err != nil {
			t.Fatalf("%s is not JSON: %v", got, err)
		}
		_ = json.Unmarshal([]byte(c.want), &want)
		if gotJSON, _ := json.Marshal(compact); string(gotJSON) != mustJSON(t, want) {
			t.Errorf("edit %v: got %s want %s", c.edit, got, c.want)
		}
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestEditStringsRoundTrip(t *testing.T) {
	inputs := []string{"", "plain", "quote\"back\\slash", "line\nbreak\ttab\r", "ctl\x01\x1f", "unicode   é \U0001F600", "<script>&</script>"}
	for _, in := range inputs {
		raw, err := json.Marshal(TextEdit(1, in))
		if err != nil {
			t.Fatal(err)
		}
		var arr []any
		if err := json.Unmarshal(raw, &arr); err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if arr[2] != in {
			t.Errorf("round trip %q -> %q", in, arr[2])
		}
	}
}

func TestEditInvalidUTF8IsReplaced(t *testing.T) {
	raw, err := json.Marshal(TextEdit(1, "a\xffb"))
	if err != nil {
		t.Fatal(err)
	}
	var arr []any
	if err := json.Unmarshal(raw, &arr); err != nil {
		t.Fatal(err)
	}
	if arr[2] != "a�b" {
		t.Errorf("got %q", arr[2])
	}
}
