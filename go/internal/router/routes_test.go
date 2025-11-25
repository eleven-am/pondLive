package router

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// mockComponent is a simple component for testing
func mockComponent(ctx *runtime.Ctx, match Match, children []work.Node) work.Node {
	return &work.Element{Tag: "div"}
}

func mockComponent2(ctx *runtime.Ctx, match Match, children []work.Node) work.Node {
	return &work.Element{Tag: "span"}
}

func TestCollectRouteEntries_Empty(t *testing.T) {
	entries := collectRouteEntries(nil)
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for nil, got %d", len(entries))
	}

	entries = collectRouteEntries([]work.Node{})
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty slice, got %d", len(entries))
	}
}

func TestCollectRouteEntries_SingleRoute(t *testing.T) {
	nodes := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/about",
					component: mockComponent,
				},
			},
		},
	}

	entries := collectRouteEntries(nodes)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].pattern != "/about" {
		t.Errorf("expected pattern '/about', got %q", entries[0].pattern)
	}
}

func TestCollectRouteEntries_MultipleRoutes(t *testing.T) {
	nodes := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/",
					component: mockComponent,
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/about",
					component: mockComponent,
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/users/:id",
					component: mockComponent,
				},
			},
		},
	}

	entries := collectRouteEntries(nodes)

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	expectedPatterns := []string{"/", "/about", "/users/:id"}
	for i, expected := range expectedPatterns {
		if entries[i].pattern != expected {
			t.Errorf("entry %d: expected pattern %q, got %q", i, expected, entries[i].pattern)
		}
	}
}

func TestCollectRouteEntries_NestedFragments(t *testing.T) {

	nodes := []work.Node{
		&work.Fragment{
			Children: []work.Node{
				&work.Fragment{
					Metadata: map[string]any{
						routeMetadataKey: routeEntry{
							pattern:   "/nested",
							component: mockComponent,
						},
					},
				},
			},
		},
	}

	entries := collectRouteEntries(nodes)

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry from nested fragment, got %d", len(entries))
	}
	if entries[0].pattern != "/nested" {
		t.Errorf("expected pattern '/nested', got %q", entries[0].pattern)
	}
}

func TestCollectRouteEntries_MixedNodes(t *testing.T) {

	nodes := []work.Node{
		&work.Element{Tag: "div"},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/home",
					component: mockComponent,
				},
			},
		},
		&work.Text{Value: "text"},
		nil,
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/contact",
					component: mockComponent,
				},
			},
		},
	}

	entries := collectRouteEntries(nodes)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries (ignoring non-route nodes), got %d", len(entries))
	}
	if entries[0].pattern != "/home" {
		t.Errorf("first entry expected '/home', got %q", entries[0].pattern)
	}
	if entries[1].pattern != "/contact" {
		t.Errorf("second entry expected '/contact', got %q", entries[1].pattern)
	}
}

func TestCollectRouteEntries_FragmentWithoutMetadata(t *testing.T) {

	nodes := []work.Node{
		&work.Fragment{},
		&work.Fragment{
			Metadata: map[string]any{
				"other-key": "value",
			},
		},
	}

	entries := collectRouteEntries(nodes)

	if len(entries) != 0 {
		t.Errorf("expected 0 entries for fragments without route metadata, got %d", len(entries))
	}
}

func TestFingerprintChildren_Empty(t *testing.T) {
	fp := fingerprintChildren(nil)
	if fp != "" {
		t.Errorf("expected empty fingerprint for nil, got %q", fp)
	}

	fp = fingerprintChildren([]work.Node{})
	if fp != "" {
		t.Errorf("expected empty fingerprint for empty slice, got %q", fp)
	}
}

func TestFingerprintChildren_SingleRoute(t *testing.T) {
	nodes := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/about",
					component: mockComponent,
				},
			},
		},
	}

	fp := fingerprintChildren(nodes)
	if fp != "/about" {
		t.Errorf("expected fingerprint '/about', got %q", fp)
	}
}

func TestFingerprintChildren_MultipleRoutes(t *testing.T) {
	nodes := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/home",
					component: mockComponent,
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/about",
					component: mockComponent,
				},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{
					pattern:   "/users/:id",
					component: mockComponent,
				},
			},
		},
	}

	fp := fingerprintChildren(nodes)
	expected := "/home|/about|/users/:id"
	if fp != expected {
		t.Errorf("expected fingerprint %q, got %q", expected, fp)
	}
}

func TestFingerprintChildren_Stability(t *testing.T) {

	nodes1 := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{pattern: "/a", component: mockComponent},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{pattern: "/b", component: mockComponent2},
			},
		},
	}

	nodes2 := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{pattern: "/a", component: mockComponent2},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{pattern: "/b", component: mockComponent},
			},
		},
	}

	fp1 := fingerprintChildren(nodes1)
	fp2 := fingerprintChildren(nodes2)

	if fp1 != fp2 {
		t.Errorf("fingerprints should match when patterns are same: %q vs %q", fp1, fp2)
	}
}

func TestFingerprintChildren_OrderMatters(t *testing.T) {
	nodes1 := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{pattern: "/a", component: mockComponent},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{pattern: "/b", component: mockComponent},
			},
		},
	}

	nodes2 := []work.Node{
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{pattern: "/b", component: mockComponent},
			},
		},
		&work.Fragment{
			Metadata: map[string]any{
				routeMetadataKey: routeEntry{pattern: "/a", component: mockComponent},
			},
		},
	}

	fp1 := fingerprintChildren(nodes1)
	fp2 := fingerprintChildren(nodes2)

	if fp1 == fp2 {
		t.Errorf("fingerprints should differ when order changes: %q == %q", fp1, fp2)
	}
}
