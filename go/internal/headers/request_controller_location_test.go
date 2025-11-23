package headers

import (
	"net/url"
	"sync"
	"testing"
)

// TestGetCurrentLocation_BeforeMutation tests that GetCurrentLocation returns initial location
// when no navigation has occurred (location not mutated)
func TestGetCurrentLocation_BeforeMutation(t *testing.T) {
	ctrl := NewRequestController()

	initialPath := "/home"
	initialQuery := url.Values{"foo": []string{"bar"}}
	initialHash := "section"

	ctrl.SetInitialLocation(initialPath, initialQuery, initialHash)

	path, query, hash := ctrl.GetCurrentLocation()

	if path != initialPath {
		t.Errorf("expected path %q, got %q", initialPath, path)
	}
	if hash != initialHash {
		t.Errorf("expected hash %q, got %q", initialHash, hash)
	}
	if query.Get("foo") != "bar" {
		t.Errorf("expected query foo=bar, got foo=%q", query.Get("foo"))
	}
}

// TestGetCurrentLocation_AfterMutation tests that GetCurrentLocation returns current location
// after SetCurrentLocation has been called
func TestGetCurrentLocation_AfterMutation(t *testing.T) {
	ctrl := NewRequestController()

	initialPath := "/home"
	initialQuery := url.Values{"foo": []string{"bar"}}
	initialHash := "section"
	ctrl.SetInitialLocation(initialPath, initialQuery, initialHash)

	newPath := "/about"
	newQuery := url.Values{"baz": []string{"qux"}}
	newHash := "details"
	ctrl.SetCurrentLocation(newPath, newQuery, newHash)

	path, query, hash := ctrl.GetCurrentLocation()

	if path != newPath {
		t.Errorf("expected path %q, got %q", newPath, path)
	}
	if hash != newHash {
		t.Errorf("expected hash %q, got %q", newHash, hash)
	}
	if query.Get("baz") != "qux" {
		t.Errorf("expected query baz=qux, got baz=%q", query.Get("baz"))
	}

	if query.Get("foo") != "" {
		t.Errorf("expected old query to be gone, but foo=%q", query.Get("foo"))
	}
}

// TestSetCurrentLocation_ClonesQuery tests that query values are deep cloned
// to prevent external mutation
func TestSetCurrentLocation_ClonesQuery(t *testing.T) {
	ctrl := NewRequestController()

	query := url.Values{"key": []string{"value"}}
	ctrl.SetCurrentLocation("/test", query, "")

	query.Set("key", "modified")
	query.Add("new", "param")

	_, retrievedQuery, _ := ctrl.GetCurrentLocation()
	if retrievedQuery.Get("key") != "value" {
		t.Errorf("expected key=value, got key=%q (external mutation leaked)", retrievedQuery.Get("key"))
	}
	if retrievedQuery.Get("new") != "" {
		t.Errorf("expected new param not to exist, got new=%q", retrievedQuery.Get("new"))
	}
}

// TestGetCurrentLocation_ClonesQuery tests that returned query values are cloned
// so external code can't mutate internal state
func TestGetCurrentLocation_ClonesQuery(t *testing.T) {
	ctrl := NewRequestController()

	ctrl.SetCurrentLocation("/test", url.Values{"key": []string{"value"}}, "")

	_, query1, _ := ctrl.GetCurrentLocation()
	query1.Set("key", "modified")

	_, query2, _ := ctrl.GetCurrentLocation()
	if query2.Get("key") != "value" {
		t.Errorf("expected key=value, got key=%q (external mutation leaked)", query2.Get("key"))
	}
}

// TestGetCurrentLocation_NilController tests nil safety
func TestGetCurrentLocation_NilController(t *testing.T) {
	var ctrl *RequestController

	path, query, hash := ctrl.GetCurrentLocation()

	if path != "" || query != nil || hash != "" {
		t.Error("expected empty values from nil controller")
	}
}

// TestSetCurrentLocation_NilController tests nil safety
func TestSetCurrentLocation_NilController(t *testing.T) {
	var ctrl *RequestController

	ctrl.SetCurrentLocation("/test", url.Values{}, "")
}

// TestGetCurrentLocation_EmptyQuery tests handling of empty/nil query values
func TestGetCurrentLocation_EmptyQuery(t *testing.T) {
	ctrl := NewRequestController()

	ctrl.SetCurrentLocation("/test", nil, "")
	path, query, hash := ctrl.GetCurrentLocation()
	if path != "/test" {
		t.Errorf("expected path /test, got %q", path)
	}
	if query == nil {
		t.Error("expected non-nil query, got nil")
	}
	if len(query) != 0 {
		t.Errorf("expected empty query, got %d items", len(query))
	}
	if hash != "" {
		t.Errorf("expected empty hash, got %q", hash)
	}

	ctrl.SetCurrentLocation("/test2", url.Values{}, "hash")
	path, query, hash = ctrl.GetCurrentLocation()
	if path != "/test2" {
		t.Errorf("expected path /test2, got %q", path)
	}
	if len(query) != 0 {
		t.Errorf("expected empty query, got %d items", len(query))
	}
	if hash != "hash" {
		t.Errorf("expected hash 'hash', got %q", hash)
	}
}

// TestLocationMutation_MultipleUpdates tests that multiple calls to SetCurrentLocation work correctly
func TestLocationMutation_MultipleUpdates(t *testing.T) {
	ctrl := NewRequestController()

	ctrl.SetInitialLocation("/initial", url.Values{"a": []string{"1"}}, "init")

	ctrl.SetCurrentLocation("/first", url.Values{"b": []string{"2"}}, "first")
	path, query, hash := ctrl.GetCurrentLocation()
	if path != "/first" || query.Get("b") != "2" || hash != "first" {
		t.Errorf("first mutation failed: got %q, %q, %q", path, query.Get("b"), hash)
	}

	ctrl.SetCurrentLocation("/second", url.Values{"c": []string{"3"}}, "second")
	path, query, hash = ctrl.GetCurrentLocation()
	if path != "/second" || query.Get("c") != "3" || hash != "second" {
		t.Errorf("second mutation failed: got %q, %q, %q", path, query.Get("c"), hash)
	}

	ctrl.SetCurrentLocation("/third", url.Values{"d": []string{"4"}}, "third")
	path, query, hash = ctrl.GetCurrentLocation()
	if path != "/third" || query.Get("d") != "4" || hash != "third" {
		t.Errorf("third mutation failed: got %q, %q, %q", path, query.Get("d"), hash)
	}

	initPath, initQuery, initHash := ctrl.GetInitialLocation()
	if initPath != "/initial" || initQuery.Get("a") != "1" || initHash != "init" {
		t.Errorf("initial location was modified: got %q, %q, %q", initPath, initQuery.Get("a"), initHash)
	}
}

// TestLocationMutation_ConcurrentAccess tests thread safety with concurrent reads and writes
func TestLocationMutation_ConcurrentAccess(t *testing.T) {
	ctrl := NewRequestController()
	ctrl.SetInitialLocation("/initial", url.Values{}, "")

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				query := url.Values{"id": []string{string(rune(id + '0'))}}
				ctrl.SetCurrentLocation("/path", query, "")
			}
		}(i)
	}

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_, query, _ := ctrl.GetCurrentLocation()

				_ = query.Get("id")
			}
		}()
	}

	wg.Wait()

	path, query, hash := ctrl.GetCurrentLocation()
	if path == "" && query == nil && hash == "" {
		t.Error("expected non-empty location after concurrent access")
	}
}

// TestGetCurrentLocation_InitialLocationUnchanged tests that GetInitialLocation
// always returns the original initial location, even after mutations
func TestGetCurrentLocation_InitialLocationUnchanged(t *testing.T) {
	ctrl := NewRequestController()

	initialPath := "/start"
	initialQuery := url.Values{"start": []string{"true"}}
	initialHash := "top"
	ctrl.SetInitialLocation(initialPath, initialQuery, initialHash)

	ctrl.SetCurrentLocation("/page1", url.Values{"page": []string{"1"}}, "")
	ctrl.SetCurrentLocation("/page2", url.Values{"page": []string{"2"}}, "")
	ctrl.SetCurrentLocation("/page3", url.Values{"page": []string{"3"}}, "")

	path, query, hash := ctrl.GetInitialLocation()
	if path != initialPath {
		t.Errorf("initial path changed: expected %q, got %q", initialPath, path)
	}
	if query.Get("start") != "true" {
		t.Errorf("initial query changed: expected start=true, got start=%q", query.Get("start"))
	}
	if hash != initialHash {
		t.Errorf("initial hash changed: expected %q, got %q", initialHash, hash)
	}
}
