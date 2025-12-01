package router

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

func Slot(ctx *runtime.Ctx, props SlotProps, children ...work.Node) work.Node {
	routes := collectRouteEntries(children, "/")

	return &work.Fragment{
		Metadata: map[string]any{
			slotMetadataKey: slotEntry{
				name:     props.Name,
				fallback: props.Fallback,
				routes:   routes,
			},
		},
	}
}
