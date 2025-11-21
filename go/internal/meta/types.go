package meta

import "github.com/eleven-am/pondlive/go/internal/html"

// Meta holds document metadata including title, description, and various head elements.
type Meta struct {
	Title       string
	Description string
	Meta        []html.MetaTag
	Links       []html.LinkTag
	Scripts     []html.ScriptTag
}

// Controller provides get/set access to meta state.
type Controller struct {
	get func() *Meta
	set func(*Meta)
}

// Get returns the current meta state.
func (c *Controller) Get() *Meta {
	if c == nil || c.get == nil {
		return defaultMeta
	}
	return c.get()
}

// Set updates the meta state.
func (c *Controller) Set(meta *Meta) {
	if c != nil && c.set != nil {
		c.set(meta)
	}
}

var defaultMeta = &Meta{
	Title:       "PondLive Application",
	Description: "A PondLive application",
}
