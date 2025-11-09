package html

import "testing"

func TestTernary(t *testing.T) {
	trueNode := Text("true")
	falseNode := Text("false")

	if got := Ternary(true, trueNode, falseNode); got != trueNode {
		t.Fatalf("expected true branch, got %#v", got)
	}

	if got := Ternary(false, trueNode, falseNode); got != falseNode {
		t.Fatalf("expected false branch, got %#v", got)
	}

	assertNoopNode(t, Ternary(true, nil, falseNode))
	assertNoopNode(t, Ternary(false, trueNode, nil))
}

func TestTernaryFn(t *testing.T) {
	trueNode := Text("true")
	falseNode := Text("false")

	trueCalls := 0
	falseCalls := 0

	got := TernaryFn(true,
		func() Node {
			trueCalls++
			return trueNode
		},
		func() Node {
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
		func() Node {
			trueCalls++
			return trueNode
		},
		func() Node {
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

	assertNoopNode(t, TernaryFn(true, nil, nil))
	assertNoopNode(t, TernaryFn(false,
		func() Node { return trueNode },
		nil,
	))
}

func assertNoopNode(t *testing.T, node Node) {
	t.Helper()
	if node == nil {
		t.Fatal("expected noop node, got nil")
	}
	host := &Element{}
	node.ApplyTo(host)
	if len(host.Children) != 0 {
		t.Fatalf("expected noop node to add no children, got %d", len(host.Children))
	}
}
