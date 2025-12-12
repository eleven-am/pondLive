package errors

import (
	"strings"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func devOverlay(batch *runtime.ErrorBatch) work.Node {
	errs := batch.All()

	errorItems := make([]work.Node, 0, len(errs))
	for _, err := range errs {
		errorItems = append(errorItems, errorItem(err))
	}

	return work.BuildElement("div",
		overlayStyles(),
		work.BuildElement("div",
			containerStyles(),
			work.BuildElement("div",
				headerStyles(),
				work.NewText("Runtime Error"),
				work.BuildElement("span",
					countBadgeStyles(),
					work.NewTextf("%d", len(errs)),
				),
			),
			work.BuildElement("div",
				errorListStyles(),
				&work.Fragment{Children: errorItems},
			),
		),
	)
}

func errorItem(err *runtime.Error) work.Node {
	frames := err.UserFrames()

	stackItems := make([]work.Node, 0, len(frames))
	for _, frame := range frames {
		stackItems = append(stackItems, work.BuildElement("div",
			frameStyles(),
			work.BuildElement("span",
				funcNameStyles(),
				work.NewText(frame.Function),
			),
			work.BuildElement("span",
				fileStyles(),
				work.NewTextf("%s:%d", frame.File, frame.Line),
			),
		))
	}

	componentPath := buildComponentPath(err)

	var children []work.Node
	children = append(children,
		work.BuildElement("div",
			errorHeaderStyles(),
			work.BuildElement("span", codeStyles(), work.NewText(string(err.Code()))),
			work.BuildElement("span", phaseStyles(), work.NewText(err.Phase)),
		),
		work.BuildElement("div", messageStyles(), work.NewText(err.Message)),
	)

	if componentPath != "" {
		children = append(children,
			work.BuildElement("div", componentPathStyles(), work.NewText(componentPath)),
		)
	}

	if len(stackItems) > 0 {
		children = append(children,
			work.BuildElement("div", stackStyles(), &work.Fragment{Children: stackItems}),
		)
	}

	return work.BuildElement("div", errorItemStyles(), &work.Fragment{Children: children})
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

	return strings.Join(path, " > ")
}
