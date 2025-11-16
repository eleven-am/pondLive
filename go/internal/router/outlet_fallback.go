package router

import (
	"sync"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

var contextlessOutlets struct {
	mu    sync.Mutex
	stack []func() h.Node
}

var contextlessBases struct {
	mu    sync.Mutex
	stack []string
}

func pushContextlessOutlet(fn func() h.Node) func() {
	if fn == nil {
		return func() {}
	}
	contextlessOutlets.mu.Lock()
	contextlessOutlets.stack = append(contextlessOutlets.stack, fn)
	contextlessOutlets.mu.Unlock()
	return func() {
		contextlessOutlets.mu.Lock()
		if n := len(contextlessOutlets.stack); n > 0 {
			contextlessOutlets.stack = contextlessOutlets.stack[:n-1]
		}
		contextlessOutlets.mu.Unlock()
	}
}

func peekContextlessOutlet() func() h.Node {
	contextlessOutlets.mu.Lock()
	defer contextlessOutlets.mu.Unlock()
	if len(contextlessOutlets.stack) == 0 {
		return nil
	}
	return contextlessOutlets.stack[len(contextlessOutlets.stack)-1]
}

func pushContextlessBase(base string) func() {
	contextlessBases.mu.Lock()
	contextlessBases.stack = append(contextlessBases.stack, base)
	contextlessBases.mu.Unlock()
	return func() {
		contextlessBases.mu.Lock()
		if n := len(contextlessBases.stack); n > 0 {
			contextlessBases.stack = contextlessBases.stack[:n-1]
		}
		contextlessBases.mu.Unlock()
	}
}

func peekContextlessBase() string {
	contextlessBases.mu.Lock()
	defer contextlessBases.mu.Unlock()
	if len(contextlessBases.stack) == 0 {
		return "/"
	}
	return contextlessBases.stack[len(contextlessBases.stack)-1]
}
