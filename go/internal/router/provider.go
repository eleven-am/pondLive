package router

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

type Handle struct {
	controller *Controller
}

func (h *Handle) Controller() *Controller {
	return h.controller
}

func ProvideRouter(ctx runtime.Ctx, onInit func(*Handle), render func(runtime.Ctx) *dom.StructuredNode) *dom.StructuredNode {
	requestController := headers.UseRequestController(ctx)

	var initialLoc Location
	if requestController != nil {
		path, query, hash := requestController.GetInitialLocation()
		initialLoc = Location{
			Path:  path,
			Query: query,
			Hash:  hash,
		}
	} else {
		initialLoc = Location{Path: "/"}
	}

	initial := &State{
		Location: initialLoc,
		Matched:  false,
		Pattern:  "",
		Params:   make(map[string]string),
		Path:     "",
	}

	current, setCurrent := runtime.UseState(ctx, initial)
	controller := runtime.UseMemo(ctx, func() *Controller {
		return NewController(current, setCurrent)
	})

	handle := &Handle{
		controller: controller,
	}

	if onInit != nil {
		onInit(handle)
	}

	return ProvideRouterState(ctx, controller, render)
}
