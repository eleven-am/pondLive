package runtime

import (
	"testing"
	"time"
)

func TestUsePresenceBooleanToggle(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Modal struct{}

	result := UsePresence(ctx, PresentIf(true, Modal{}, 100*time.Millisecond))
	if !result.Visible() {
		t.Error("expected visible when true")
	}
	if result.IsExiting() {
		t.Error("expected not exiting when active")
	}
	if len(result.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(result.Items))
	}

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(false, Modal{}, 100*time.Millisecond))
	if !result.Visible() {
		t.Error("expected still visible during exit animation")
	}
	if !result.IsExiting() {
		t.Error("expected exiting after toggle off")
	}
}

func TestUsePresenceZeroDuration(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Modal struct{}

	result := UsePresence(ctx, PresentIf(true, Modal{}, 0))
	if !result.Visible() {
		t.Error("expected visible when true")
	}

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(false, Modal{}, 0))
	if result.Visible() {
		t.Error("expected not visible with zero duration (immediate removal)")
	}
}

func TestUsePresenceListRemoval(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID   string
		Name string
	}

	items := []Item{
		{ID: "1", Name: "First"},
		{ID: "2", Name: "Second"},
		{ID: "3", Name: "Third"},
	}
	keyFn := func(i Item) string { return i.ID }

	result := UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}
	for _, item := range result.Items {
		if item.IsExiting() {
			t.Errorf("expected item %s not exiting", item.Key())
		}
	}

	ctx.hookIndex = 0
	items = []Item{
		{ID: "1", Name: "First"},
		{ID: "3", Name: "Third"},
	}
	result = UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))
	if len(result.Items) != 3 {
		t.Errorf("expected 3 items (including exiting), got %d", len(result.Items))
	}

	exitingCount := 0
	for _, item := range result.Items {
		if item.IsExiting() {
			exitingCount++
			if item.Key() != "2" {
				t.Errorf("expected item 2 to be exiting, got %s", item.Key())
			}
		}
	}
	if exitingCount != 1 {
		t.Errorf("expected 1 exiting item, got %d", exitingCount)
	}
}

func TestUsePresenceReentry(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID string
	}
	keyFn := func(i Item) string { return i.ID }

	items := []Item{{ID: "1"}, {ID: "2"}}
	result := UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(result.Items))
	}

	ctx.hookIndex = 0
	items = []Item{{ID: "1"}}
	result = UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items (1 active, 1 exiting), got %d", len(result.Items))
	}

	var exitingItem *PresenceItem[Item]
	for i := range result.Items {
		if result.Items[i].IsExiting() {
			exitingItem = &result.Items[i]
			break
		}
	}
	if exitingItem == nil || exitingItem.Key() != "2" {
		t.Error("expected item 2 to be exiting")
	}

	ctx.hookIndex = 0
	items = []Item{{ID: "1"}, {ID: "2"}}
	result = UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items after re-entry, got %d", len(result.Items))
	}
	for _, item := range result.Items {
		if item.IsExiting() {
			t.Errorf("expected no exiting items after re-entry, but %s is exiting", item.Key())
		}
	}
}

func TestUsePresenceIndexFallback(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	items := []string{"a", "b", "c"}
	result := UsePresence(ctx, PresenceInput[string]{
		Items:    items,
		Duration: 100 * time.Millisecond,
	})

	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}

	if result.Items[0].Key() != "0" {
		t.Errorf("expected key '0', got %s", result.Items[0].Key())
	}
	if result.Items[1].Key() != "1" {
		t.Errorf("expected key '1', got %s", result.Items[1].Key())
	}
	if result.Items[2].Key() != "2" {
		t.Errorf("expected key '2', got %s", result.Items[2].Key())
	}
}

func TestUsePresenceOrderingStability(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID string
	}
	keyFn := func(i Item) string { return i.ID }

	items := []Item{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}
	result := UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))
	if len(result.Items) != 4 {
		t.Errorf("expected 4 items, got %d", len(result.Items))
	}

	ctx.hookIndex = 0
	items = []Item{{ID: "a"}, {ID: "c"}, {ID: "d"}}
	result = UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	expectedOrder := []string{"a", "b", "c", "d"}
	for i, expected := range expectedOrder {
		if result.Items[i].Key() != expected {
			t.Errorf("expected item %d to be %s, got %s", i, expected, result.Items[i].Key())
		}
	}

	if !result.Items[1].IsExiting() {
		t.Error("expected 'b' to be exiting")
	}
}

func TestUsePresenceTimerScheduling(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Modal struct{}

	result := UsePresence(ctx, PresentIf(true, Modal{}, 50*time.Millisecond))
	if !result.Visible() {
		t.Error("expected visible")
	}

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(false, Modal{}, 50*time.Millisecond))
	if !result.Visible() {
		t.Error("expected still visible during exit")
	}
	if !result.IsExiting() {
		t.Error("expected exiting")
	}

	time.Sleep(100 * time.Millisecond)

	if len(sess.DirtyQueue) == 0 {
		t.Error("expected component to be marked dirty after timer")
	}

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(false, Modal{}, 50*time.Millisecond))
	if result.Visible() {
		t.Error("expected not visible after duration elapsed")
	}
}

func TestUsePresenceCleanup(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Modal struct{}

	UsePresence(ctx, PresentIf(true, Modal{}, 100*time.Millisecond))

	ctx.hookIndex = 0
	UsePresence(ctx, PresentIf(false, Modal{}, 100*time.Millisecond))

	if len(inst.cleanups) != 1 {
		t.Errorf("expected 1 cleanup registered, got %d", len(inst.cleanups))
	}

	inst.cleanups[0]()
}

func TestPresenceItemMethods(t *testing.T) {
	item := PresenceItem[string]{
		Value:   "test",
		key:     "key1",
		exiting: false,
	}

	if !item.Visible() {
		t.Error("expected Visible() to return true")
	}
	if item.IsExiting() {
		t.Error("expected IsExiting() to return false")
	}
	if item.Key() != "key1" {
		t.Errorf("expected Key() to return 'key1', got %s", item.Key())
	}

	exitingItem := PresenceItem[string]{
		Value:   "test",
		key:     "key2",
		exiting: true,
	}
	if !exitingItem.IsExiting() {
		t.Error("expected IsExiting() to return true for exiting item")
	}
}

func TestPresenceResultMethods(t *testing.T) {
	emptyResult := PresenceResult[string]{Items: []PresenceItem[string]{}}
	if emptyResult.Visible() {
		t.Error("expected Visible() to return false for empty result")
	}
	if emptyResult.IsExiting() {
		t.Error("expected IsExiting() to return false for empty result")
	}

	activeResult := PresenceResult[string]{
		Items: []PresenceItem[string]{
			{Value: "test", exiting: false},
		},
	}
	if !activeResult.Visible() {
		t.Error("expected Visible() to return true")
	}
	if activeResult.IsExiting() {
		t.Error("expected IsExiting() to return false for active item")
	}

	exitingResult := PresenceResult[string]{
		Items: []PresenceItem[string]{
			{Value: "test", exiting: true},
		},
	}
	if !exitingResult.Visible() {
		t.Error("expected Visible() to return true for exiting item")
	}
	if !exitingResult.IsExiting() {
		t.Error("expected IsExiting() to return true")
	}
}

func TestPresentHelper(t *testing.T) {
	input := Present("hello", 200*time.Millisecond)

	if len(input.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(input.Items))
	}
	if input.Items[0] != "hello" {
		t.Errorf("expected 'hello', got %s", input.Items[0])
	}
	if input.Duration != 200*time.Millisecond {
		t.Errorf("expected 200ms duration, got %v", input.Duration)
	}
	if input.KeyFunc == nil {
		t.Error("expected KeyFunc to be set for Present (uses _single key)")
	}
	if input.KeyFunc("anything") != "_single" {
		t.Errorf("expected key '_single', got %s", input.KeyFunc("anything"))
	}
}

func TestPresentListHelper(t *testing.T) {
	items := []int{1, 2, 3}
	keyFn := func(i int) string { return string(rune('a' + i)) }
	input := PresentList(items, keyFn, 300*time.Millisecond)

	if len(input.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(input.Items))
	}
	if input.Duration != 300*time.Millisecond {
		t.Errorf("expected 300ms duration, got %v", input.Duration)
	}
	if input.KeyFunc == nil {
		t.Error("expected KeyFunc to be set")
	}
	if input.KeyFunc(1) != "b" {
		t.Errorf("expected key 'b' for item 1, got %s", input.KeyFunc(1))
	}
}

func TestPresentIfHelper(t *testing.T) {
	inputTrue := PresentIf(true, "hello", 200*time.Millisecond)
	if len(inputTrue.Items) != 1 {
		t.Errorf("expected 1 item when true, got %d", len(inputTrue.Items))
	}
	if inputTrue.Items[0] != "hello" {
		t.Errorf("expected 'hello', got %s", inputTrue.Items[0])
	}
	if inputTrue.Duration != 200*time.Millisecond {
		t.Errorf("expected 200ms duration, got %v", inputTrue.Duration)
	}

	inputFalse := PresentIf(false, "hello", 200*time.Millisecond)
	if len(inputFalse.Items) != 0 {
		t.Errorf("expected 0 items when false, got %d", len(inputFalse.Items))
	}
	if inputFalse.Duration != 200*time.Millisecond {
		t.Errorf("expected 200ms duration, got %v", inputFalse.Duration)
	}
}

func TestUsePresenceHookMismatch(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	UseState[int](ctx, 0)

	ctx.hookIndex = 0
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for hook mismatch")
		}
	}()
	UsePresence(ctx, Present(true, 100*time.Millisecond))
}

func TestUsePresenceMultipleExits(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID string
	}
	keyFn := func(i Item) string { return i.ID }

	items := []Item{{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"}, {ID: "5"}}
	UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	ctx.hookIndex = 0
	items = []Item{{ID: "1"}, {ID: "3"}, {ID: "5"}}
	result := UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	if len(result.Items) != 5 {
		t.Errorf("expected 5 items, got %d", len(result.Items))
	}

	exitingCount := 0
	exitingKeys := make(map[string]bool)
	for _, item := range result.Items {
		if item.IsExiting() {
			exitingCount++
			exitingKeys[item.Key()] = true
		}
	}

	if exitingCount != 2 {
		t.Errorf("expected 2 exiting items, got %d", exitingCount)
	}
	if !exitingKeys["2"] || !exitingKeys["4"] {
		t.Errorf("expected items 2 and 4 to be exiting, got %v", exitingKeys)
	}
}

func TestUsePresenceInitiallyFalse(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Modal struct{}

	result := UsePresence(ctx, PresentIf(false, Modal{}, 100*time.Millisecond))
	if result.Visible() {
		t.Error("expected not visible when starting with false")
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestUsePresenceEmptyListInitially(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID string
	}
	keyFn := func(i Item) string { return i.ID }

	result := UsePresence(ctx, PresentList([]Item{}, keyFn, 100*time.Millisecond))
	if result.Visible() {
		t.Error("expected not visible with empty list")
	}
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(result.Items))
	}
}

func TestUsePresenceAddItemsToEmptyList(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID string
	}
	keyFn := func(i Item) string { return i.ID }

	result := UsePresence(ctx, PresentList([]Item{}, keyFn, 100*time.Millisecond))
	if len(result.Items) != 0 {
		t.Errorf("expected 0 items initially, got %d", len(result.Items))
	}

	ctx.hookIndex = 0
	items := []Item{{ID: "a"}, {ID: "b"}}
	result = UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))
	if len(result.Items) != 2 {
		t.Errorf("expected 2 items after adding, got %d", len(result.Items))
	}
	for _, item := range result.Items {
		if item.IsExiting() {
			t.Errorf("expected no exiting items, but %s is exiting", item.Key())
		}
	}
}

func TestUsePresenceRemoveAllItems(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID string
	}
	keyFn := func(i Item) string { return i.ID }

	items := []Item{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	ctx.hookIndex = 0
	result := UsePresence(ctx, PresentList([]Item{}, keyFn, 100*time.Millisecond))

	if len(result.Items) != 3 {
		t.Errorf("expected 3 exiting items, got %d", len(result.Items))
	}
	for _, item := range result.Items {
		if !item.IsExiting() {
			t.Errorf("expected item %s to be exiting", item.Key())
		}
	}
}

func TestUsePresenceReorderItems(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID string
	}
	keyFn := func(i Item) string { return i.ID }

	items := []Item{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	ctx.hookIndex = 0
	items = []Item{{ID: "c"}, {ID: "a"}, {ID: "b"}}
	result := UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	if len(result.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(result.Items))
	}

	expectedOrder := []string{"c", "a", "b"}
	for i, expected := range expectedOrder {
		if result.Items[i].Key() != expected {
			t.Errorf("expected item %d to be %s, got %s", i, expected, result.Items[i].Key())
		}
		if result.Items[i].IsExiting() {
			t.Errorf("expected item %s not to be exiting", result.Items[i].Key())
		}
	}
}

func TestUsePresenceUpdateItemValue(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID   string
		Name string
	}
	keyFn := func(i Item) string { return i.ID }

	items := []Item{{ID: "1", Name: "First"}, {ID: "2", Name: "Second"}}
	UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	ctx.hookIndex = 0
	items = []Item{{ID: "1", Name: "Updated First"}, {ID: "2", Name: "Updated Second"}}
	result := UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	if result.Items[0].Value.Name != "Updated First" {
		t.Errorf("expected updated name, got %s", result.Items[0].Value.Name)
	}
	if result.Items[1].Value.Name != "Updated Second" {
		t.Errorf("expected updated name, got %s", result.Items[1].Value.Name)
	}
}

func TestUsePresenceMultipleReentries(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Modal struct{}

	UsePresence(ctx, PresentIf(true, Modal{}, 100*time.Millisecond))

	ctx.hookIndex = 0
	result := UsePresence(ctx, PresentIf(false, Modal{}, 100*time.Millisecond))
	if !result.IsExiting() {
		t.Error("expected exiting after first toggle off")
	}

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(true, Modal{}, 100*time.Millisecond))
	if result.IsExiting() {
		t.Error("expected not exiting after re-entry")
	}

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(false, Modal{}, 100*time.Millisecond))
	if !result.IsExiting() {
		t.Error("expected exiting after second toggle off")
	}

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(true, Modal{}, 100*time.Millisecond))
	if result.IsExiting() {
		t.Error("expected not exiting after second re-entry")
	}
}

func TestUsePresenceWithNilSession(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	ctx := &Ctx{
		instance:  inst,
		session:   nil,
		hookIndex: 0,
	}

	type Modal struct{}

	result := UsePresence(ctx, PresentIf(true, Modal{}, 100*time.Millisecond))
	if !result.Visible() {
		t.Error("expected visible even with nil session")
	}

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(false, Modal{}, 100*time.Millisecond))
	if !result.Visible() {
		t.Error("expected still visible during exit")
	}
	if !result.IsExiting() {
		t.Error("expected exiting")
	}
}

func TestUsePresenceConsecutiveRenders(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID string
	}
	keyFn := func(i Item) string { return i.ID }

	items := []Item{{ID: "a"}}
	for i := 0; i < 10; i++ {
		ctx.hookIndex = 0
		result := UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))
		if len(result.Items) != 1 {
			t.Errorf("render %d: expected 1 item, got %d", i, len(result.Items))
		}
		if result.Items[0].IsExiting() {
			t.Errorf("render %d: expected not exiting", i)
		}
	}
}

func TestUsePresenceExitingItemValuePreserved(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID   string
		Data string
	}
	keyFn := func(i Item) string { return i.ID }

	items := []Item{{ID: "a", Data: "original"}, {ID: "b", Data: "keep"}}
	UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	ctx.hookIndex = 0
	items = []Item{{ID: "b", Data: "keep"}}
	result := UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	var exitingItem *PresenceItem[Item]
	for i := range result.Items {
		if result.Items[i].Key() == "a" {
			exitingItem = &result.Items[i]
			break
		}
	}

	if exitingItem == nil {
		t.Fatal("expected to find exiting item 'a'")
	}
	if exitingItem.Value.Data != "original" {
		t.Errorf("expected preserved data 'original', got %s", exitingItem.Value.Data)
	}
}

func TestUsePresenceComplexScenario(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID string
	}
	keyFn := func(i Item) string { return i.ID }

	items := []Item{{ID: "a"}, {ID: "b"}, {ID: "c"}}
	UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	ctx.hookIndex = 0
	items = []Item{{ID: "a"}, {ID: "c"}}
	UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	ctx.hookIndex = 0
	items = []Item{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}
	result := UsePresence(ctx, PresentList(items, keyFn, 100*time.Millisecond))

	if len(result.Items) != 4 {
		t.Errorf("expected 4 items, got %d", len(result.Items))
	}

	for _, item := range result.Items {
		if item.IsExiting() {
			t.Errorf("expected no exiting items after re-adding all, but %s is exiting", item.Key())
		}
	}
}

func TestUsePresenceRapidToggle(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Modal struct{}

	for i := 0; i < 20; i++ {
		ctx.hookIndex = 0
		show := i%2 == 0
		result := UsePresence(ctx, PresentIf(show, Modal{}, 50*time.Millisecond))

		if show {
			if !result.Visible() {
				t.Errorf("iteration %d: expected visible when show=true", i)
			}
			if result.IsExiting() {
				t.Errorf("iteration %d: expected not exiting when show=true", i)
			}
		} else {
			if !result.Visible() {
				t.Errorf("iteration %d: expected visible during exit", i)
			}
			if !result.IsExiting() {
				t.Errorf("iteration %d: expected exiting when show=false", i)
			}
		}
	}
}

func TestUsePresenceDifferentDurations(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
		cleanups:  []func(){},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Modal struct{}

	UsePresence(ctx, PresentIf(true, Modal{}, 100*time.Millisecond))

	ctx.hookIndex = 0
	UsePresence(ctx, PresentIf(false, Modal{}, 50*time.Millisecond))

	time.Sleep(75 * time.Millisecond)

	ctx.hookIndex = 0
	result := UsePresence(ctx, PresentIf(false, Modal{}, 50*time.Millisecond))

	if result.Visible() {
		t.Error("expected not visible after shorter duration elapsed")
	}
}

func TestUsePresenceSingleKeyStability(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Modal struct {
		Title string
	}

	result := UsePresence(ctx, PresentIf(true, Modal{Title: "First"}, 100*time.Millisecond))
	if result.Items[0].Key() != "_single" {
		t.Errorf("expected key '_single', got %s", result.Items[0].Key())
	}

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(true, Modal{Title: "Second"}, 100*time.Millisecond))
	if result.Items[0].Key() != "_single" {
		t.Errorf("expected same key '_single', got %s", result.Items[0].Key())
	}
	if result.Items[0].Value.Title != "Second" {
		t.Errorf("expected updated title 'Second', got %s", result.Items[0].Value.Title)
	}
}

func TestUsePresenceDynamicDurationShortens(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Modal struct{}

	UsePresence(ctx, PresentIf(true, Modal{}, 500*time.Millisecond))

	ctx.hookIndex = 0
	result := UsePresence(ctx, PresentIf(false, Modal{}, 500*time.Millisecond))
	if !result.IsExiting() {
		t.Error("expected exiting")
	}

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(false, Modal{}, 50*time.Millisecond))
	if !result.Visible() {
		t.Error("expected still visible")
	}

	time.Sleep(60 * time.Millisecond)

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(false, Modal{}, 50*time.Millisecond))
	if result.Visible() {
		t.Error("expected removed after shortened duration elapsed")
	}
}

func TestUsePresenceDynamicDurationToZero(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Modal struct{}

	UsePresence(ctx, PresentIf(true, Modal{}, 500*time.Millisecond))

	ctx.hookIndex = 0
	result := UsePresence(ctx, PresentIf(false, Modal{}, 500*time.Millisecond))
	if !result.IsExiting() {
		t.Error("expected exiting")
	}

	ctx.hookIndex = 0
	result = UsePresence(ctx, PresentIf(false, Modal{}, 0))
	if result.Visible() {
		t.Error("expected immediate removal when duration changed to zero")
	}
}

func TestUsePresenceDuplicateKeyPanic(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	type Item struct {
		ID   string
		Name string
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Error("expected panic on duplicate key")
		}
		if msg, ok := r.(string); ok {
			if msg != "runtime: UsePresence duplicate key: item-1" {
				t.Errorf("unexpected panic message: %s", msg)
			}
		}
	}()

	items := []Item{
		{ID: "item-1", Name: "First"},
		{ID: "item-1", Name: "Duplicate"},
	}
	UsePresence(ctx, PresentList(items, func(i Item) string { return i.ID }, 100*time.Millisecond))
}
