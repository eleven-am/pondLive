package runtime

import (
	"runtime"
	"sync"
	"sync/atomic"
)

// RenderScheduler coordinates concurrent subtree rendering with topological safety.
type RenderScheduler struct {
	workerCount int
	session     *ComponentSession

	// Rendering state
	pending      map[*component]struct{}   // Components needing render
	dependencies map[*component]*component // child -> parent mapping
	readyQueue   chan *component
	wg           sync.WaitGroup
	stopped      atomic.Bool

	mu sync.Mutex
}

// NewRenderScheduler creates a scheduler with the specified worker count.
// If workerCount <= 0, defaults to runtime.GOMAXPROCS(0).
func NewRenderScheduler(sess *ComponentSession, workerCount int) *RenderScheduler {
	if workerCount <= 0 {
		workerCount = runtime.GOMAXPROCS(0)
	}
	return &RenderScheduler{
		workerCount:  workerCount,
		session:      sess,
		pending:      make(map[*component]struct{}),
		dependencies: make(map[*component]*component),
		readyQueue:   make(chan *component, workerCount*2),
	}
}

// ScheduleComponents initiates concurrent rendering of the provided components.
// Returns after all components have been rendered.
func (sched *RenderScheduler) ScheduleComponents(comps []*component) {
	if len(comps) == 0 {
		return
	}

	sched.mu.Lock()

	sched.pending = make(map[*component]struct{}, len(comps))
	sched.dependencies = make(map[*component]*component, len(comps))
	for _, comp := range comps {
		if comp == nil {
			continue
		}
		sched.pending[comp] = struct{}{}
		if comp.parent != nil {
			sched.dependencies[comp] = comp.parent
		}
	}

	var ready []*component
	for comp := range sched.pending {
		if !sched.hasPendingDependency(comp) {
			ready = append(ready, comp)
		}
	}
	sched.mu.Unlock()

	sched.stopped.Store(false)
	for i := 0; i < sched.workerCount; i++ {
		sched.wg.Add(1)
		go sched.worker()
	}

	for _, comp := range ready {
		sched.readyQueue <- comp
	}

	sched.wg.Wait()

	sched.mu.Lock()
	sched.pending = make(map[*component]struct{})
	sched.dependencies = make(map[*component]*component)
	sched.mu.Unlock()
}

// hasPendingDependency checks if component's parent is still pending render.
// Must be called with sched.mu held.
func (sched *RenderScheduler) hasPendingDependency(comp *component) bool {
	parent, hasParent := sched.dependencies[comp]
	if !hasParent {
		return false
	}
	_, parentPending := sched.pending[parent]
	return parentPending
}

// worker processes components from the ready queue.
func (sched *RenderScheduler) worker() {
	defer sched.wg.Done()

	for comp := range sched.readyQueue {
		if comp == nil || sched.stopped.Load() {
			continue
		}

		comp.render()

		sched.completeComponent(comp)
	}
}

// completeComponent marks a component as rendered and schedules any children
// that are now ready (i.e., have no more pending dependencies).
func (sched *RenderScheduler) completeComponent(comp *component) {
	sched.mu.Lock()

	delete(sched.pending, comp)

	var newlyReady []*component
	for candidate := range sched.pending {
		parent, hasParent := sched.dependencies[candidate]
		if hasParent && parent == comp {

			if !sched.hasPendingDependency(candidate) {
				newlyReady = append(newlyReady, candidate)
			}
		}
	}

	if len(sched.pending) == 0 {
		sched.mu.Unlock()
		close(sched.readyQueue)
		return
	}

	sched.mu.Unlock()

	for _, ready := range newlyReady {
		sched.readyQueue <- ready
	}
}

// Stop halts the scheduler gracefully.
func (sched *RenderScheduler) Stop() {
	sched.stopped.Store(true)
	sched.wg.Wait()
}
