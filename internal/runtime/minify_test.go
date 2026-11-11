package runtime

import "testing"

func TestMinifyJS(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes unnecessary spaces",
			input:    "function   test()   {  return  42;  }",
			expected: "function test(){return 42;}",
		},
		{
			name:     "preserves string literals",
			input:    `console.log("hello world")`,
			expected: `console.log("hello world")`,
		},
		{
			name:     "removes newlines",
			input:    "function test() {\n  return 42;\n}",
			expected: "function test(){return 42;}",
		},
		{
			name:     "preserves spaces between identifiers",
			input:    "return value",
			expected: "return value",
		},
		{
			name:     "handles multiple string types",
			input:    "var a = \"double\"; var b = 'single'; var c = `template`;",
			expected: "var a=\"double\";var b='single';var c=`template`;",
		},
		{
			name:     "reduces consecutive whitespace to single space",
			input:    "return     value",
			expected: "return value",
		},
		{
			name:     "handles mixed whitespace types",
			input:    "function\t\t\ntest() {\n\treturn\t42;\n}",
			expected: "function test(){return 42;}",
		},
		{
			name:     "removes single-line comments",
			input:    "var x = 1; // this is a comment\nvar y = 2;",
			expected: "var x=1;var y=2;",
		},
		{
			name:     "removes multi-line comments",
			input:    "var x = 1; /* comment */ var y = 2;",
			expected: "var x=1;var y=2;",
		},
		{
			name:     "removes multi-line comment spanning lines",
			input:    "var x = 1;\n/* this\nis\na\ncomment */\nvar y = 2;",
			expected: "var x=1;var y=2;",
		},
		{
			name:     "preserves regex literals",
			input:    "var re = /test/gi;",
			expected: "var re=/test/gi;",
		},
		{
			name:     "handles regex with escapes",
			input:    "var re = /\\/path\\//;",
			expected: "var re=/\\/path\\//;",
		},
		{
			name:     "handles regex with character class",
			input:    "var re = /[a-z]/;",
			expected: "var re=/[a-z]/;",
		},
		{
			name:     "distinguishes division from regex",
			input:    "var x = a / b;",
			expected: "var x=a/b;",
		},
		{
			name:     "regex after assignment",
			input:    "x = /pattern/;",
			expected: "x=/pattern/;",
		},
		{
			name:     "division after closing paren",
			input:    "(a + b) / c",
			expected: "(a+b)/c",
		},
		{
			name:     "preserves slash in string",
			input:    `var s = "a/b";`,
			expected: `var s="a/b";`,
		},
		{
			name:     "comment inside string preserved",
			input:    `var s = "// not a comment";`,
			expected: `var s="// not a comment";`,
		},
		{
			name:     "template literal with expression",
			input:    "var s = `hello ${name}`;",
			expected: "var s=`hello ${name}`;",
		},
		{
			name:     "regex with all flags",
			input:    "var re = /test/gimsuy;",
			expected: "var re=/test/gimsuy;",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := minifyJS(tt.input)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}

func TestIsIdentifierChar(t *testing.T) {
	tests := []struct {
		ch       byte
		expected bool
	}{
		{'a', true},
		{'Z', true},
		{'5', true},
		{'_', true},
		{'$', true},
		{' ', false},
		{'+', false},
		{'(', false},
	}

	for _, tt := range tests {
		result := isIdentifierChar(tt.ch)
		if result != tt.expected {
			t.Errorf("isIdentifierChar(%c) = %v, expected %v", tt.ch, result, tt.expected)
		}
	}
}
