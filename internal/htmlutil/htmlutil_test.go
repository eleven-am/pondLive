package htmlutil

import "testing"

func TestEscapeClose(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"key": "value"}`, `{"key": "value"}`},
		{`<script></script>`, `<script><\/script>`},
		{`</body>`, `<\/body>`},
		{`test</test`, `test<\/test`},
		{`no slash tags`, `no slash tags`},
		{`multiple </a></b></c>`, `multiple <\/a><\/b><\/c>`},
	}

	for _, tc := range tests {
		result := EscapeClose(tc.input)
		if result != tc.expected {
			t.Errorf("EscapeClose(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestLastIndexFold(t *testing.T) {
	tests := []struct {
		haystack string
		needle   string
		expected int
	}{
		{"hello</body>world", "</body>", 5},
		{"hello</BODY>world", "</body>", 5},
		{"hello</Body>world", "</body>", 5},
		{"<body></body></BODY>", "</body>", 13},
		{"no match here", "</body>", -1},
		{"short", "</body>", -1},
		{"", "</body>", -1},
		{"test", "", -1},
		{"</body>", "</body>", 0},
	}

	for _, tc := range tests {
		result := LastIndexFold(tc.haystack, tc.needle)
		if result != tc.expected {
			t.Errorf("LastIndexFold(%q, %q) = %d, want %d", tc.haystack, tc.needle, result, tc.expected)
		}
	}
}

func TestIndexFoldMatchesAnyCase(t *testing.T) {
	if got := IndexFold("a</STYLE>b</style>", "</style"); got != 1 {
		t.Fatalf("index = %d", got)
	}
	if IndexFold("abc", "x") != -1 {
		t.Fatal("a missing needle must be -1")
	}
}

func TestUniqueClassesKeepsTheFirstOfEachInOrder(t *testing.T) {
	for in, want := range map[string]string{
		"a b a":     "a b",
		"a  b":      "a  b",
		"a a a":     "a",
		"x y z":     "x y z",
		"":          "",
		" b a  b c": "b a c",
	} {
		if got := UniqueClasses(in); got != want {
			t.Errorf("UniqueClasses(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsVoidKnowsEverySelfClosingTag(t *testing.T) {
	for _, tag := range []string{"br", "img", "input", "keygen", "command", "wbr"} {
		if !IsVoid(tag) {
			t.Errorf("%s must be void", tag)
		}
	}
	for _, tag := range []string{"div", "p", "style", "script"} {
		if IsVoid(tag) {
			t.Errorf("%s must not be void", tag)
		}
	}
}

func TestAppendTextEscapesMarkupButNotQuotes(t *testing.T) {
	if got := string(AppendText([]byte("x:"), `a & b < c > "d" 'e'`)); got != `x:a &amp; b &lt; c &gt; "d" 'e'` {
		t.Fatalf("got %q", got)
	}
}

func TestAppendAttrAlsoEscapesDoubleQuotes(t *testing.T) {
	if got := string(AppendAttr(nil, `a & "b" < c > 'd'`)); got != `a &amp; &quot;b&quot; &lt; c &gt; 'd'` {
		t.Fatalf("got %q", got)
	}
}

func TestAppendRawTextBreaksOnlyTheClosingTagOfItsOwnElement(t *testing.T) {
	cases := []struct{ tag, in, want string }{
		{"script", `a</script>b</SCRIPT >c</style>`, `a<\/script>b<\/SCRIPT >c</style>`},
		{"style", `p{}</style>`, `p{}<\/style>`},
		{"script", "no closer", "no closer"},
	}
	for _, c := range cases {
		if got := string(AppendRawText(nil, c.tag, c.in)); got != c.want {
			t.Errorf("AppendRawText(%q, %q) = %q, want %q", c.tag, c.in, got, c.want)
		}
	}
}

func TestEscapeCloseBreaksEveryClosingTagStart(t *testing.T) {
	if got := EscapeClose(`{"a":"</script></b>"}`); got != `{"a":"<\/script><\/b>"}` {
		t.Fatalf("got %q", got)
	}
}
