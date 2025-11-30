package pkg

import "github.com/eleven-am/pondlive/go/internal/work"

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

func Fragment(children ...Node) Node {
	return work.NewFragment(children...)
}
