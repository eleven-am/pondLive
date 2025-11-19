package html

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
)

func TestTernary(t *testing.T) {
	trueNode := Text("true")
	falseNode := Text("false")

	if got := Ternary(true, trueNode, falseNode); got != trueNode {
		t.Fatalf("expected true branch, got %#v", got)
	}

	if got := Ternary(false, trueNode, falseNode); got != falseNode {
		t.Fatalf("expected false branch, got %#v", got)
	}

	if _, ok := Ternary(true, nil, falseNode).(noopNode); !ok {
		t.Fatalf("expected noop when true branch missing")
	}

	if _, ok := Ternary(false, trueNode, nil).(noopNode); !ok {
		t.Fatalf("expected noop when false branch missing")
	}
}

func TestTernaryFn(t *testing.T) {
	trueNode := Text("true")
	falseNode := Text("false")

	trueCalls := 0
	falseCalls := 0

	got := TernaryFn(true,
		func() dom2.Item {
			trueCalls++
			return trueNode
		},
		func() dom2.Item {
			falseCalls++
			return falseNode
		},
	)

	if got != trueNode {
		t.Fatalf("expected true branch, got %#v", got)
	}

	if trueCalls != 1 || falseCalls != 0 {
		t.Fatalf("unexpected call counts, true=%d false=%d", trueCalls, falseCalls)
	}

	got = TernaryFn(false,
		func() dom2.Item {
			trueCalls++
			return trueNode
		},
		func() dom2.Item {
			falseCalls++
			return falseNode
		},
	)

	if got != falseNode {
		t.Fatalf("expected false branch, got %#v", got)
	}

	if trueCalls != 1 || falseCalls != 1 {
		t.Fatalf("unexpected call counts, true=%d false=%d", trueCalls, falseCalls)
	}

	if _, ok := TernaryFn(true, nil, nil).(noopNode); !ok {
		t.Fatalf("expected noop when true function missing")
	}

	if _, ok := TernaryFn(false,
		func() dom2.Item { return trueNode },
		nil,
	).(noopNode); !ok {
		t.Fatalf("expected noop when false function missing")
	}
}
