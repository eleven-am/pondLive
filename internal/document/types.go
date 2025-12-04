package document

type Document struct {
	HtmlClass string
	HtmlLang  string
	HtmlDir   string
	BodyClass string
}

type documentEntry struct {
	doc         *Document
	depth       int
	componentID string
}
