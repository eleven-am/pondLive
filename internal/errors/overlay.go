package errors

import (
	"strings"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func devOverlay(errs runtime.ReadSignal[[]*runtime.Error]) work.Node {
	return work.BuildElement("div",
		overlayStyles(),
		work.BuildElement("div",
			containerStyles(),
			work.BuildElement("div",
				headerStyles(),
				work.TextOf("Runtime Error"),
				work.BuildElement("span",
					countBadgeStyles(),
					work.TextOf(runtime.Derive(errs, func(l []*runtime.Error) int { return len(l) })),
				),
			),
			work.BuildElement("div",
				errorListStyles(),
				runtime.For(runtime.Items(errs, func(e *runtime.Error) *runtime.Error { return e }),
					runtime.PropsComponent(func(_ *runtime.Ctx, item runtime.ReadSignal[*runtime.Error]) work.Node {
						return errorItem(item.Peek())
					})),
			),
		),
	)
}

func errorItem(err *runtime.Error) work.Node {
	frames := err.UserFrames()
	maxFrames := 3
	if len(frames) > maxFrames {
		frames = frames[:maxFrames]
	}

	stackItems := make([]work.Item, 0, len(frames))
	for _, frame := range frames {
		stackItems = append(stackItems, work.BuildElement("div",
			frameStyles(),
			work.BuildElement("span",
				funcNameStyles(),
				work.TextOf(frame.Function),
			),
			work.BuildElement("span",
				fileStyles(),
				work.NewTextf("%s:%d", frame.File, frame.Line),
			),
		))
	}

	componentPath := buildComponentPath(err)

	var children []work.Item
	children = append(children,
		work.BuildElement("div",
			errorHeaderStyles(),
			work.BuildElement("span", codeStyles(), work.TextOf(string(err.Code()))),
			work.BuildElement("span", phaseStyles(), work.TextOf(err.Phase)),
		),
		work.BuildElement("div", messageStyles(), work.TextOf(err.Message)),
	)

	if componentPath != "" {
		children = append(children,
			work.BuildElement("div", componentPathStyles(), work.TextOf(componentPath)),
		)
	}

	if len(stackItems) > 0 {
		children = append(children,
			work.BuildElement("div", stackStyles(), work.NewFragment(stackItems...)),
		)
	}

	return work.BuildElement("div", errorItemStyles(), work.NewFragment(children...))
}

func buildComponentPath(err *runtime.Error) string {
	meta := err.Metadata()
	if meta == nil {
		return ""
	}

	path, ok := meta["component_name_path"].([]string)
	if !ok || len(path) == 0 {
		return ""
	}

	var filtered []string
	for _, name := range path {
		if !isAnonymousFunc(name) {
			filtered = append(filtered, name)
		}
	}

	if len(filtered) == 0 {
		return ""
	}

	return strings.Join(filtered, " → ")
}

func isAnonymousFunc(name string) bool {
	if name == "" {
		return true
	}
	if strings.HasPrefix(name, "func") {
		rest := strings.TrimPrefix(name, "func")
		if rest == "" {
			return true
		}
		for _, c := range rest {
			if c < '0' || c > '9' {
				return false
			}
		}
		return true
	}
	return false
}
