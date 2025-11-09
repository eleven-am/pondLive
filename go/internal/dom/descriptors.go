package dom

// ElementDescriptor describes the compile-time identity of a generated HTML
// element. Descriptor implementations are generated alongside element builders
// and allow typed helpers (such as Attach and UseElement) to ensure a ref is
// wired to the correct element instance.
type ElementDescriptor interface {
	TagName() string
}
