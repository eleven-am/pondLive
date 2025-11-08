package html

import "fmt"

type ElementHookContext interface{}

type ElementHookProvider func(ctx ElementHookContext, descriptor any) any

var elementHookProvider ElementHookProvider

func RegisterElementHookProvider(fn ElementHookProvider) {
	if fn == nil {
		panic("html: element hook provider cannot be nil")
	}
	if elementHookProvider != nil {
		panic("html: element hook provider already registered")
	}
	elementHookProvider = fn
}

func useElementRef[T ElementDescriptor](ctx ElementHookContext) *ElementRef[T] {
	if ctx == nil {
		return nil
	}
	if elementHookProvider == nil {
		panic("html: element hook provider not installed")
	}
	var descriptor T
	raw := elementHookProvider(ctx, descriptor)
	if raw == nil {
		return nil
	}
	ref, ok := raw.(*ElementRef[T])
	if !ok {
		panic(fmt.Sprintf("html: element hook provider returned %T for %T", raw, descriptor))
	}
	return ref
}
