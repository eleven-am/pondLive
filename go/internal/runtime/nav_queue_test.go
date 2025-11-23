package runtime

import (
	"testing"
)

// TestNavigationQueue_PreservesOrder tests that navigation queue preserves order
func TestNavigationQueue_PreservesOrder(t *testing.T) {
	sess := &ComponentSession{}

	sess.EnqueueNavigation("/first", false)
	sess.EnqueueNavigation("/second", true)
	sess.EnqueueNavigation("/third", false)

	if len(sess.pendingNavs) != 3 {
		t.Fatalf("expected 3 pending navigations, got %d", len(sess.pendingNavs))
	}

	if sess.pendingNavs[0].Push != "/first" {
		t.Errorf("expected first nav Push='/first', got %q", sess.pendingNavs[0].Push)
	}
	if sess.pendingNavs[1].Replace != "/second" {
		t.Errorf("expected second nav Replace='/second', got %q", sess.pendingNavs[1].Replace)
	}
	if sess.pendingNavs[2].Push != "/third" {
		t.Errorf("expected third nav Push='/third', got %q", sess.pendingNavs[2].Push)
	}
}

// TestTakeNavDeltas_ReturnsAllInOrder tests that TakeNavDeltas returns all navigations
func TestTakeNavDeltas_ReturnsAllInOrder(t *testing.T) {
	sess := &ComponentSession{}

	sess.EnqueueNavigation("/a", false)
	sess.EnqueueNavigation("/b", true)
	sess.EnqueueNavigation("/c", false)

	navs := sess.TakeNavDeltas()

	if len(navs) != 3 {
		t.Fatalf("expected 3 navigations, got %d", len(navs))
	}

	if navs[0].Push != "/a" {
		t.Errorf("expected navs[0].Push='/a', got %q", navs[0].Push)
	}
	if navs[1].Replace != "/b" {
		t.Errorf("expected navs[1].Replace='/b', got %q", navs[1].Replace)
	}
	if navs[2].Push != "/c" {
		t.Errorf("expected navs[2].Push='/c', got %q", navs[2].Push)
	}

	navs2 := sess.TakeNavDeltas()
	if len(navs2) != 0 {
		t.Errorf("expected empty queue after take, got %d items", len(navs2))
	}
}

// TestTakeNavDelta_FIFO tests that TakeNavDelta returns navigations in FIFO order
func TestTakeNavDelta_FIFO(t *testing.T) {
	sess := &ComponentSession{}

	sess.EnqueueNavigation("/first", false)
	sess.EnqueueNavigation("/second", true)
	sess.EnqueueNavigation("/third", false)

	nav1 := sess.TakeNavDelta()
	if nav1 == nil {
		t.Fatal("expected first navigation to be returned")
	}
	if nav1.Push != "/first" {
		t.Errorf("expected first navigation Push='/first', got %q", nav1.Push)
	}

	nav2 := sess.TakeNavDelta()
	if nav2 == nil {
		t.Fatal("expected second navigation to be returned")
	}
	if nav2.Replace != "/second" {
		t.Errorf("expected second navigation Replace='/second', got %q", nav2.Replace)
	}

	nav3 := sess.TakeNavDelta()
	if nav3 == nil {
		t.Fatal("expected third navigation to be returned")
	}
	if nav3.Push != "/third" {
		t.Errorf("expected third navigation Push='/third', got %q", nav3.Push)
	}

	nav4 := sess.TakeNavDelta()
	if nav4 != nil {
		t.Error("expected nil after taking all navigations")
	}
}

// TestNavigationQueue_EmptyQueue tests handling of empty queue
func TestNavigationQueue_EmptyQueue(t *testing.T) {
	sess := &ComponentSession{}

	navs := sess.TakeNavDeltas()
	if navs != nil {
		t.Errorf("expected nil for empty queue, got %d items", len(navs))
	}

	nav := sess.TakeNavDelta()
	if nav != nil {
		t.Error("expected nil from TakeNavDelta on empty queue")
	}
}

// TestNavigationQueue_AlternatingTypes tests alternating push/replace
func TestNavigationQueue_AlternatingTypes(t *testing.T) {
	sess := &ComponentSession{}

	sess.EnqueueNavigation("/a", false)
	sess.EnqueueNavigation("/b", true)
	sess.EnqueueNavigation("/c", false)
	sess.EnqueueNavigation("/d", true)

	navs := sess.TakeNavDeltas()

	if len(navs) != 4 {
		t.Fatalf("expected 4 navigations, got %d", len(navs))
	}

	if navs[0].Push != "/a" || navs[0].Replace != "" {
		t.Error("expected first to be push")
	}
	if navs[1].Replace != "/b" || navs[1].Push != "" {
		t.Error("expected second to be replace")
	}
	if navs[2].Push != "/c" || navs[2].Replace != "" {
		t.Error("expected third to be push")
	}
	if navs[3].Replace != "/d" || navs[3].Push != "" {
		t.Error("expected fourth to be replace")
	}
}

// TestNavigationQueue_ComplexPaths tests queue with complex URLs
func TestNavigationQueue_ComplexPaths(t *testing.T) {
	sess := &ComponentSession{}

	complexPaths := []struct {
		href    string
		replace bool
	}{
		{"/search?q=test&page=1&sort=date", false},
		{"/user/123?tab=profile#settings", true},
		{"/docs/api#methods", false},
		{"/?welcome=true", true},
	}

	for _, p := range complexPaths {
		sess.EnqueueNavigation(p.href, p.replace)
	}

	navs := sess.TakeNavDeltas()

	if len(navs) != len(complexPaths) {
		t.Fatalf("expected %d navigations, got %d", len(complexPaths), len(navs))
	}

	for i, expected := range complexPaths {
		if expected.replace {
			if navs[i].Replace != expected.href {
				t.Errorf("nav[%d]: expected Replace=%q, got %q", i, expected.href, navs[i].Replace)
			}
			if navs[i].Push != "" {
				t.Errorf("nav[%d]: expected empty Push, got %q", i, navs[i].Push)
			}
		} else {
			if navs[i].Push != expected.href {
				t.Errorf("nav[%d]: expected Push=%q, got %q", i, expected.href, navs[i].Push)
			}
			if navs[i].Replace != "" {
				t.Errorf("nav[%d]: expected empty Replace, got %q", i, navs[i].Replace)
			}
		}
	}
}

// TestNavigationQueue_ConcurrentSafety tests mutex protection
func TestNavigationQueue_ConcurrentSafety(t *testing.T) {
	sess := &ComponentSession{}

	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 100; i++ {
			sess.EnqueueNavigation("/path", false)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			sess.TakeNavDeltas()
		}
		done <- true
	}()

	<-done
	<-done

}

// TestNavigationQueue_NilSession tests nil safety
func TestNavigationQueue_NilSession(t *testing.T) {
	var sess *ComponentSession

	sess.EnqueueNavigation("/test", false)
	navs := sess.TakeNavDeltas()
	if navs != nil {
		t.Error("expected nil from TakeNavDeltas on nil session")
	}

	nav := sess.TakeNavDelta()
	if nav != nil {
		t.Error("expected nil from TakeNavDelta on nil session")
	}
}

// TestNavigationQueue_LargeQueue tests that queue is capped to prevent memory leaks.
// The implementation limits the queue to 100 items and drops oldest entries when full.
func TestNavigationQueue_LargeQueue(t *testing.T) {
	sess := &ComponentSession{}

	for i := 0; i < 1000; i++ {
		sess.EnqueueNavigation("/path", i%2 == 0)
	}

	navs := sess.TakeNavDeltas()

	if len(navs) != 100 {
		t.Errorf("expected queue to be capped at 100, got %d", len(navs))
	}

	if len(sess.pendingNavs) != 0 {
		t.Errorf("expected queue to be cleared after TakeNavDeltas, still has %d items", len(sess.pendingNavs))
	}

}
