package css

import (
	"testing"
)

func TestCN_BasicMerge(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "single class",
			input:  []string{"px-4"},
			expect: "px-4",
		},
		{
			name:   "multiple non-conflicting classes",
			input:  []string{"px-4 py-2", "bg-blue-500"},
			expect: "px-4 py-2 bg-blue-500",
		},
		{
			name:   "empty strings",
			input:  []string{"", "px-4", ""},
			expect: "px-4",
		},
		{
			name:   "whitespace handling",
			input:  []string{"  px-4  py-2  ", "  bg-blue-500  "},
			expect: "px-4 py-2 bg-blue-500",
		},
		{
			name:   "no input",
			input:  []string{},
			expect: "",
		},
		{
			name:   "only empty strings",
			input:  []string{"", " ", "  "},
			expect: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_PaddingConflicts(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "padding-x conflict",
			input:  []string{"px-4", "px-2"},
			expect: "px-2",
		},
		{
			name:   "padding-y conflict",
			input:  []string{"py-4", "py-2"},
			expect: "py-2",
		},
		{
			name:   "padding conflict",
			input:  []string{"p-4", "p-2"},
			expect: "p-2",
		},
		{
			name:   "different padding sides no conflict",
			input:  []string{"px-4", "py-2"},
			expect: "px-4 py-2",
		},
		{
			name:   "specific sides no conflict",
			input:  []string{"pt-4", "pb-2", "pl-3", "pr-1"},
			expect: "pt-4 pb-2 pl-3 pr-1",
		},
		{
			name:   "same side conflict",
			input:  []string{"pt-4", "pt-2"},
			expect: "pt-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_MarginConflicts(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "margin-x conflict",
			input:  []string{"mx-4", "mx-2"},
			expect: "mx-2",
		},
		{
			name:   "margin-y conflict",
			input:  []string{"my-4", "my-2"},
			expect: "my-2",
		},
		{
			name:   "margin conflict",
			input:  []string{"m-4", "m-2"},
			expect: "m-2",
		},
		{
			name:   "different margin sides no conflict",
			input:  []string{"mx-4", "my-2"},
			expect: "mx-4 my-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_SizingConflicts(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "width conflict",
			input:  []string{"w-full", "w-1/2"},
			expect: "w-1/2",
		},
		{
			name:   "height conflict",
			input:  []string{"h-screen", "h-full"},
			expect: "h-full",
		},
		{
			name:   "width and height no conflict",
			input:  []string{"w-full", "h-screen"},
			expect: "w-full h-screen",
		},
		{
			name:   "min-width conflict",
			input:  []string{"min-w-0", "min-w-full"},
			expect: "min-w-full",
		},
		{
			name:   "max-width conflict",
			input:  []string{"max-w-sm", "max-w-lg"},
			expect: "max-w-lg",
		},
		{
			name:   "arbitrary width conflict",
			input:  []string{"w-[200px]", "w-[300px]"},
			expect: "w-[300px]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_TypographyConflicts(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "font size conflict",
			input:  []string{"text-sm", "text-base"},
			expect: "text-base",
		},
		{
			name:   "font weight conflict",
			input:  []string{"font-normal", "font-bold"},
			expect: "font-bold",
		},
		{
			name:   "text alignment conflict",
			input:  []string{"text-left", "text-center"},
			expect: "text-center",
		},
		{
			name:   "text decoration conflict",
			input:  []string{"underline", "no-underline"},
			expect: "no-underline",
		},
		{
			name:   "text transform conflict",
			input:  []string{"uppercase", "lowercase"},
			expect: "lowercase",
		},
		{
			name:   "font size and weight no conflict",
			input:  []string{"text-sm", "font-bold"},
			expect: "text-sm font-bold",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_ColorConflicts(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "text color conflict",
			input:  []string{"text-blue-500", "text-red-500"},
			expect: "text-red-500",
		},
		{
			name:   "background color conflict",
			input:  []string{"bg-blue-500", "bg-red-500"},
			expect: "bg-red-500",
		},
		{
			name:   "border color conflict",
			input:  []string{"border-blue-500", "border-red-500"},
			expect: "border-red-500",
		},
		{
			name:   "text and background no conflict",
			input:  []string{"text-blue-500", "bg-red-500"},
			expect: "text-blue-500 bg-red-500",
		},
		{
			name:   "arbitrary background color conflict",
			input:  []string{"bg-[#ff0000]", "bg-[#00ff00]"},
			expect: "bg-[#00ff00]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_LayoutConflicts(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "display conflict",
			input:  []string{"block", "flex"},
			expect: "flex",
		},
		{
			name:   "position conflict",
			input:  []string{"static", "absolute"},
			expect: "absolute",
		},
		{
			name:   "overflow conflict",
			input:  []string{"overflow-auto", "overflow-hidden"},
			expect: "overflow-hidden",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_FlexboxConflicts(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "flex direction conflict",
			input:  []string{"flex-row", "flex-col"},
			expect: "flex-col",
		},
		{
			name:   "justify content conflict",
			input:  []string{"justify-start", "justify-center"},
			expect: "justify-center",
		},
		{
			name:   "align items conflict",
			input:  []string{"items-start", "items-center"},
			expect: "items-center",
		},
		{
			name:   "flex and justify no conflict",
			input:  []string{"flex-row", "justify-center"},
			expect: "flex-row justify-center",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_BorderConflicts(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "border radius conflict",
			input:  []string{"rounded-md", "rounded-lg"},
			expect: "rounded-lg",
		},
		{
			name:   "border width conflict",
			input:  []string{"border-2", "border-4"},
			expect: "border-4",
		},
		{
			name:   "border style conflict",
			input:  []string{"border-solid", "border-dashed"},
			expect: "border-dashed",
		},
		{
			name:   "border width and radius no conflict",
			input:  []string{"border-2", "rounded-lg"},
			expect: "border-2 rounded-lg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_EffectsConflicts(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "opacity conflict",
			input:  []string{"opacity-50", "opacity-100"},
			expect: "opacity-100",
		},
		{
			name:   "shadow conflict",
			input:  []string{"shadow-sm", "shadow-xl"},
			expect: "shadow-xl",
		},
		{
			name:   "blur conflict",
			input:  []string{"blur-sm", "blur-lg"},
			expect: "blur-lg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_VariantHandling(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "hover variant conflict",
			input:  []string{"hover:bg-blue-500", "hover:bg-red-500"},
			expect: "hover:bg-red-500",
		},
		{
			name:   "focus variant conflict",
			input:  []string{"focus:ring-2", "focus:ring-4"},
			expect: "focus:ring-4",
		},
		{
			name:   "responsive variant conflict",
			input:  []string{"sm:text-sm", "sm:text-base"},
			expect: "sm:text-base",
		},
		{
			name:   "different variants no conflict",
			input:  []string{"hover:bg-blue-500", "focus:bg-red-500"},
			expect: "hover:bg-blue-500 focus:bg-red-500",
		},
		{
			name:   "variant and non-variant same class no conflict",
			input:  []string{"bg-blue-500", "hover:bg-red-500"},
			expect: "bg-blue-500 hover:bg-red-500",
		},
		{
			name:   "multiple variants",
			input:  []string{"sm:hover:bg-blue-500", "sm:hover:bg-red-500"},
			expect: "sm:hover:bg-red-500",
		},
		{
			name:   "complex variant combinations",
			input:  []string{"hover:px-4", "px-2", "hover:px-6"},
			expect: "px-2 hover:px-6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_ComplexScenarios(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "multiple conflicts in one call",
			input:  []string{"px-4 py-2 text-sm", "px-2 text-base"},
			expect: "py-2 px-2 text-base",
		},
		{
			name:   "shadcn button pattern",
			input:  []string{"rounded-md px-3 py-2 text-sm", "bg-blue-500 hover:bg-blue-600", "px-4 text-base"},
			expect: "rounded-md py-2 bg-blue-500 hover:bg-blue-600 px-4 text-base",
		},
		{
			name:   "preserves non-conflicting order",
			input:  []string{"flex items-center justify-center", "gap-4 rounded-lg"},
			expect: "flex items-center justify-center gap-4 rounded-lg",
		},
		{
			name:   "removes exact duplicates",
			input:  []string{"px-4 py-2", "px-4 rounded-md"},
			expect: "py-2 px-4 rounded-md",
		},
		{
			name:   "real component pattern",
			input:  []string{"flex min-h-screen items-center justify-center bg-slate-900 p-4", "text-slate-100"},
			expect: "flex min-h-screen items-center justify-center bg-slate-900 p-4 text-slate-100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}

func TestCN_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		input  []string
		expect string
	}{
		{
			name:   "arbitrary values with spaces",
			input:  []string{"w-[calc(100%-2rem)]", "w-[200px]"},
			expect: "w-[200px]",
		},
		{
			name:   "classes with numbers",
			input:  []string{"text-4xl", "text-6xl"},
			expect: "text-6xl",
		},
		{
			name:   "grid template columns",
			input:  []string{"grid-cols-3", "grid-cols-4"},
			expect: "grid-cols-4",
		},
		{
			name:   "important modifier (passes through)",
			input:  []string{"!px-4", "px-2"},
			expect: "!px-4 px-2",
		},
		{
			name:   "important beats later non-important",
			input:  []string{"!px-4", "px-2"},
			expect: "!px-4 px-2",
		},
		{
			name:   "later important overrides earlier important",
			input:  []string{"!px-4", "!px-2"},
			expect: "!px-2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CN(tt.input...)
			if result != tt.expect {
				t.Errorf("CN(%v) = %q, expected %q", tt.input, result, tt.expect)
			}
		})
	}
}
