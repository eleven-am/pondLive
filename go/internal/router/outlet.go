package router

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

func Outlet(ctx *runtime.Ctx, name ...string) work.Node {
	slotName := defaultSlotName
	if len(name) > 0 && name[0] != "" {
		slotName = name[0]
	}
	return outlet(ctx, slotName)
}

var outlet = runtime.PropsComponent(func(ctx *runtime.Ctx, slotName string, _ []work.Node) work.Node {
	slots := slotsCtx.UseContextValue(ctx)
	if slots == nil {
		return &work.Fragment{}
	}

	if render, ok := slots[slotName]; ok && render != nil {
		return render(ctx)
	}

	return &work.Fragment{}
})
