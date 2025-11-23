package runtime

import (
	"testing"
)

func TestEnqueueNavigation_Push(t *testing.T) {
	sess := &ComponentSession{}

	sess.EnqueueNavigation("/new-path?foo=bar", false)

	if len(sess.pendingNavs) != 1 {
		t.Fatalf("expected 1 pending navigation, got %d", len(sess.pendingNavs))
	}
	if sess.pendingNavs[0].Push != "/new-path?foo=bar" {
		t.Errorf("expected Push to be '/new-path?foo=bar', got %q", sess.pendingNavs[0].Push)
	}
	if sess.pendingNavs[0].Replace != "" {
		t.Errorf("expected Replace to be empty, got %q", sess.pendingNavs[0].Replace)
	}
}

func TestEnqueueNavigation_Replace(t *testing.T) {
	sess := &ComponentSession{}

	sess.EnqueueNavigation("/replaced-path", true)

	if len(sess.pendingNavs) != 1 {
		t.Fatalf("expected 1 pending navigation, got %d", len(sess.pendingNavs))
	}
	if sess.pendingNavs[0].Replace != "/replaced-path" {
		t.Errorf("expected Replace to be '/replaced-path', got %q", sess.pendingNavs[0].Replace)
	}
	if sess.pendingNavs[0].Push != "" {
		t.Errorf("expected Push to be empty, got %q", sess.pendingNavs[0].Push)
	}
}

func TestEnqueueNavigation_QueuesMultiple(t *testing.T) {
	sess := &ComponentSession{}

	sess.EnqueueNavigation("/first", false)
	sess.EnqueueNavigation("/second", true)

	if len(sess.pendingNavs) != 2 {
		t.Fatalf("expected 2 pending navigations, got %d", len(sess.pendingNavs))
	}

	if sess.pendingNavs[0].Push != "/first" {
		t.Errorf("expected first Push to be '/first', got %q", sess.pendingNavs[0].Push)
	}

	if sess.pendingNavs[1].Replace != "/second" {
		t.Errorf("expected second Replace to be '/second', got %q", sess.pendingNavs[1].Replace)
	}

	// Test FIFO behavior - take one at a time
	nav1 := sess.TakeNavDelta()
	if nav1 == nil || nav1.Push != "/first" {
		t.Errorf("expected first navigation Push='/first'")
	}

	nav2 := sess.TakeNavDelta()
	if nav2 == nil || nav2.Replace != "/second" {
		t.Errorf("expected second navigation Replace='/second'")
	}

	// Queue should be empty now
	nav3 := sess.TakeNavDelta()
	if nav3 != nil {
		t.Error("expected empty queue after taking all navigations")
	}
}

func TestEnqueueNavigation_NilSession(t *testing.T) {
	var sess *ComponentSession

	sess.EnqueueNavigation("/test", false)
}

func TestTakeNavDelta_ReturnsAndClears(t *testing.T) {
	sess := &ComponentSession{}

	sess.EnqueueNavigation("/path?query=value#hash", false)

	nav := sess.TakeNavDelta()
	if nav == nil {
		t.Fatal("expected TakeNavDelta to return navigation")
	}
	if nav.Push != "/path?query=value#hash" {
		t.Errorf("expected Push to be '/path?query=value#hash', got %q", nav.Push)
	}

	nav2 := sess.TakeNavDelta()
	if nav2 != nil {
		t.Error("expected TakeNavDelta to return nil after being taken")
	}
}

func TestTakeNavDelta_NilSession(t *testing.T) {
	var sess *ComponentSession
	nav := sess.TakeNavDelta()
	if nav != nil {
		t.Error("expected TakeNavDelta on nil session to return nil")
	}
}

func TestTakeNavDelta_NoPending(t *testing.T) {
	sess := &ComponentSession{}

	nav := sess.TakeNavDelta()
	if nav != nil {
		t.Error("expected TakeNavDelta to return nil when no pending navigation")
	}
}

func TestNavDelta_QueryParams(t *testing.T) {
	tests := []struct {
		name    string
		href    string
		replace bool
	}{
		{
			name:    "path with query params",
			href:    "/search?q=test&page=1",
			replace: false,
		},
		{
			name:    "path with hash",
			href:    "/docs#section-1",
			replace: true,
		},
		{
			name:    "path with query and hash",
			href:    "/page?foo=bar#anchor",
			replace: false,
		},
		{
			name:    "query only update",
			href:    "/current?updated=true",
			replace: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sess := &ComponentSession{}
			sess.EnqueueNavigation(tt.href, tt.replace)

			nav := sess.TakeNavDelta()
			if nav == nil {
				t.Fatal("expected navigation to be set")
			}

			if tt.replace {
				if nav.Replace != tt.href {
					t.Errorf("expected Replace to be %q, got %q", tt.href, nav.Replace)
				}
				if nav.Push != "" {
					t.Error("expected Push to be empty for replace navigation")
				}
			} else {
				if nav.Push != tt.href {
					t.Errorf("expected Push to be %q, got %q", tt.href, nav.Push)
				}
				if nav.Replace != "" {
					t.Error("expected Replace to be empty for push navigation")
				}
			}
		})
	}
}

func TestNavDeltaStruct(t *testing.T) {

	nav := NavDelta{
		Push:    "/push-path",
		Replace: "",
	}
	if nav.Push != "/push-path" {
		t.Errorf("expected Push field to work")
	}

	nav2 := NavDelta{
		Push:    "",
		Replace: "/replace-path",
	}
	if nav2.Replace != "/replace-path" {
		t.Errorf("expected Replace field to work")
	}
}
