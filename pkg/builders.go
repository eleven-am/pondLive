package pkg

import (
	"github.com/eleven-am/pondlive/internal/portal"
	"github.com/eleven-am/pondlive/internal/work"
)

func El(tag string, items ...Item) Node {
	return work.BuildElement(tag, items...)
}

func Text(s string) Node {
	return work.NewText(s)
}

func Textf(format string, args ...any) Node {
	return work.NewTextf(format, args...)
}

func Comment(value string) Node {
	return work.NewComment(value)
}

func Fragment(children ...Item) Node {
	return work.NewFragment(children...)
}

func Portal(children ...Item) Node {
	return portal.Portal(children...)
}
