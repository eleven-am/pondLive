package document

import "github.com/eleven-am/pondlive/internal/work"

type Document struct {
	HtmlClass string
	HtmlLang  string
	HtmlDir   string
	BodyClass string
}

type documentEntry struct {
	doc          *Document
	depth        int
	componentID  string
	bodyHandlers map[string][]work.Handler
}
