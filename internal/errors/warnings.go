package errors

import (
	"strings"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

const warningStackLines = 8

func warningPanel(boundary *runtime.Boundary, warns runtime.ReadSignal[[]*runtime.Warning]) work.Node {
	return work.BuildElement("div",
		work.Attribute("data-pondlive-warnings", ""),
		warningPanelStyles(),
		work.BuildElement("div",
			warningHeaderStyles(),
			work.BuildElement("span", work.TextOf("Warnings")),
			work.BuildElement("span", warningBadgeStyles(), work.TextOf(runtime.Derive(warns, func(l []*runtime.Warning) int { return len(l) }))),
			work.BuildElement("button",
				warningButtonStyles(),
				work.Attribute("type", "button"),
				work.On("click", func(work.Event) { boundary.DismissAll() }),
				work.TextOf("Dismiss all"),
			),
		),
		work.BuildElement("div",
			warningListStyles(),
			runtime.For(runtime.Items(warns, func(w *runtime.Warning) uint64 { return w.ID }),
				runtime.PropsComponent(func(_ *runtime.Ctx, item runtime.ReadSignal[*runtime.Warning]) work.Node {
					return warningItem(boundary, item.Peek())
				})),
		),
	)
}

func warningItem(boundary *runtime.Boundary, w *runtime.Warning) work.Node {
	id := w.ID
	children := []work.Item{
		work.BuildElement("div",
			warningItemHeaderStyles(),
			work.BuildElement("span", warningCodeStyles(), work.TextOf(string(w.Code))),
			work.BuildElement("span", warningComponentStyles(), work.TextOf(w.Component)),
			work.BuildElement("button",
				warningButtonStyles(),
				work.Attribute("type", "button"),
				work.Attribute("aria-label", "Dismiss warning"),
				work.On("click", func(work.Event) { boundary.Dismiss(id) }),
				work.TextOf("Dismiss"),
			),
		),
		work.BuildElement("div", warningMessageStyles(), work.TextOf(w.Message)),
	}
	if stack := truncateLines(w.Stack, warningStackLines); stack != "" {
		children = append(children, work.BuildElement("pre", warningStackStyles(), work.TextOf(stack)))
	}
	return work.BuildElement("div", warningItemStyles(), work.NewFragment(children...))
}

func truncateLines(s string, n int) string {
	s = strings.TrimSpace(s)
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[:n], "\n") + "\n..."
}
