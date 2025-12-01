package router

import (
	"github.com/eleven-am/pondlive/go/internal/route"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

var locationCtx = runtime.CreateContext[Location](Location{Path: "/"}).WithEqual(locationEqual)

var matchCtx = runtime.CreateContext[*MatchState](nil).WithEqual(matchStateEqual)

const defaultSlotName = "__default__"

var slotsCtx = runtime.CreateContext[map[string]outletRenderer](nil)

var routeBaseCtx = runtime.CreateContext[string]("/")

func matchStateEqual(a, b *MatchState) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if a.Matched != b.Matched || a.Pattern != b.Pattern || a.Path != b.Path || a.Rest != b.Rest {
		return false
	}
	if len(a.Params) != len(b.Params) {
		return false
	}
	for k, v := range a.Params {
		if b.Params[k] != v {
			return false
		}
	}
	return true
}

func locationEqual(a, b Location) bool {
	return route.LocEqual(a, b)
}
