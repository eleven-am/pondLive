package errors

import "github.com/eleven-am/pondlive/internal/work"

func crashPage() work.Node {
	return work.BuildElement("div",
		crashContainerStyles(),
		work.BuildElement("h1",
			crashTitleStyles(),
			work.NewText("Something went wrong"),
		),
		work.BuildElement("p",
			crashMessageStyles(),
			work.NewText("The application encountered an unexpected error. Please try refreshing the page."),
		),
	)
}
