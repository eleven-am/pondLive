package render

import (
	"testing"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// buildLargeTree creates a tree with many siblings to benchmark
func buildLargeTree(siblingCount int) h.Node {
	items := make([]h.Item, siblingCount)
	for i := 0; i < siblingCount; i++ {
		items[i] = h.Div(
			h.Text("static text"),
			h.Textf("%d", i),
			h.Button(h.Text("click")),
		)
	}

	return h.WrapComponent("root", h.Div(items...))
}

// buildNestedComponentTree creates a tree with nested components
func buildNestedComponentTree(componentCount int) h.Node {
	items := make([]h.Item, componentCount)
	for i := 0; i < componentCount; i++ {
		items[i] = h.WrapComponent("comp"+string(rune('A'+i)),
			h.Div(
				h.Text("text"),
				h.Textf("%d", i),
				h.Button(h.Text("action")),
				h.Div(
					h.Text("nested"),
					h.Textf("val%d", i),
				),
			),
		)
	}

	return h.WrapComponent("root", h.Div(items...))
}

func BenchmarkSequential10Siblings(b *testing.B) {
	root := buildLargeTree(10)
	analyzer := NewComponentAnalyzer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(root)
	}
}

func BenchmarkConcurrent10Siblings(b *testing.B) {
	root := buildLargeTree(10)
	analyzer := NewComponentAnalyzerWithOptions(AnalysisOptions{
		Concurrent:           true,
		ConcurrencyThreshold: 4,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(root)
	}
}

func BenchmarkSequential50Siblings(b *testing.B) {
	root := buildLargeTree(50)
	analyzer := NewComponentAnalyzer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(root)
	}
}

func BenchmarkConcurrent50Siblings(b *testing.B) {
	root := buildLargeTree(50)
	analyzer := NewComponentAnalyzerWithOptions(AnalysisOptions{
		Concurrent:           true,
		ConcurrencyThreshold: 4,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(root)
	}
}

func BenchmarkSequential100Siblings(b *testing.B) {
	root := buildLargeTree(100)
	analyzer := NewComponentAnalyzer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(root)
	}
}

func BenchmarkConcurrent100Siblings(b *testing.B) {
	root := buildLargeTree(100)
	analyzer := NewComponentAnalyzerWithOptions(AnalysisOptions{
		Concurrent:           true,
		ConcurrencyThreshold: 4,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(root)
	}
}

func BenchmarkSequentialNestedComponents20(b *testing.B) {
	root := buildNestedComponentTree(20)
	analyzer := NewComponentAnalyzer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(root)
	}
}

func BenchmarkConcurrentNestedComponents20(b *testing.B) {
	root := buildNestedComponentTree(20)
	analyzer := NewComponentAnalyzerWithOptions(AnalysisOptions{
		Concurrent:           true,
		ConcurrencyThreshold: 4,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(root)
	}
}

func BenchmarkSequentialNestedComponents50(b *testing.B) {
	root := buildNestedComponentTree(50)
	analyzer := NewComponentAnalyzer()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(root)
	}
}

func BenchmarkConcurrentNestedComponents50(b *testing.B) {
	root := buildNestedComponentTree(50)
	analyzer := NewComponentAnalyzerWithOptions(AnalysisOptions{
		Concurrent:           true,
		ConcurrencyThreshold: 4,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.Analyze(root)
	}
}

func buildKeyedList(count int) h.Node {
	items := make([]h.Item, count)
	for i := 0; i < count; i++ {
		elem := h.Div(
			h.Text("Item: "),
			h.Textf("%d", i),
			h.Button(h.Text("Action")),
			h.Div(
				h.Text("Nested content"),
				h.Textf("Value %d", i*10),
			),
		)
		elem.Key = "item-" + string(rune('0'+i%10))
		items[i] = elem
	}
	return h.WrapComponent("root", h.Div(items...))
}

func buildHeavyKeyedList(count int) h.Node {
	items := make([]h.Item, count)
	for i := 0; i < count; i++ {
		subItems := make([]h.Item, 20)
		for j := 0; j < 20; j++ {
			subItems[j] = h.Div(
				h.Text("Sub"),
				h.Textf("%d-%d", i, j),
				h.Button(h.Text("Click")),
			)
		}

		elem := h.Div(
			h.Text("Header "),
			h.Textf("%d", i),
			h.Div(subItems...),
			h.Button(h.Text("Main Action")),
			h.Div(
				h.Text("Footer"),
				h.Textf("Total: %d", i*100),
			),
		)
		elem.Key = "heavy-" + string(rune('A'+i%26))
		items[i] = elem
	}
	return h.WrapComponent("root", h.Div(items...))
}

func BenchmarkSequentialKeyedRows20(b *testing.B) {
	root := buildKeyedList(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructured(root)
	}
}

func BenchmarkConcurrentKeyedRows20(b *testing.B) {
	root := buildKeyedList(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructuredWithHandlers(root, StructuredOptions{
			ConcurrentRows:          true,
			RowConcurrencyThreshold: 5,
		})
	}
}

func BenchmarkSequentialKeyedRows100(b *testing.B) {
	root := buildKeyedList(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructured(root)
	}
}

func BenchmarkConcurrentKeyedRows100(b *testing.B) {
	root := buildKeyedList(100)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructuredWithHandlers(root, StructuredOptions{
			ConcurrentRows:          true,
			RowConcurrencyThreshold: 5,
		})
	}
}

func BenchmarkSequentialHeavyKeyedRows20(b *testing.B) {
	root := buildHeavyKeyedList(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructured(root)
	}
}

func BenchmarkConcurrentHeavyKeyedRows20(b *testing.B) {
	root := buildHeavyKeyedList(20)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructuredWithHandlers(root, StructuredOptions{
			ConcurrentRows:          true,
			RowConcurrencyThreshold: 5,
		})
	}
}

func BenchmarkSequentialHeavyKeyedRows50(b *testing.B) {
	root := buildHeavyKeyedList(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructured(root)
	}
}

func BenchmarkConcurrentHeavyKeyedRows50(b *testing.B) {
	root := buildHeavyKeyedList(50)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructuredWithHandlers(root, StructuredOptions{
			ConcurrentRows:          true,
			RowConcurrencyThreshold: 5,
		})
	}
}

func buildHeavySiblings(count int) h.Node {
	items := make([]h.Item, count)
	for i := 0; i < count; i++ {
		subItems := make([]h.Item, 15)
		for j := 0; j < 15; j++ {
			subItems[j] = h.Div(
				h.Text("Content "),
				h.Textf("%d-%d", i, j),
				h.Span(h.Text("nested")),
			)
		}
		items[i] = h.Div(
			h.Text("Parent "),
			h.Textf("%d", i),
			h.Div(subItems...),
			h.Button(h.Text("Action")),
		)
	}
	return h.WrapComponent("root", h.Div(items...))
}

func BenchmarkSequentialChildren10(b *testing.B) {
	root := buildHeavySiblings(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructured(root)
	}
}

func BenchmarkConcurrentChildren10(b *testing.B) {
	root := buildHeavySiblings(10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructuredWithHandlers(root, StructuredOptions{
			ConcurrentChildren:        true,
			ChildConcurrencyThreshold: 5,
		})
	}
}

func BenchmarkSequentialChildren30(b *testing.B) {
	root := buildHeavySiblings(30)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructured(root)
	}
}

func BenchmarkConcurrentChildren30(b *testing.B) {
	root := buildHeavySiblings(30)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ToStructuredWithHandlers(root, StructuredOptions{
			ConcurrentChildren:        true,
			ChildConcurrencyThreshold: 5,
		})
	}
}
