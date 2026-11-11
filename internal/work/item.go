package work

type Item interface {
	applyTo(*Element)
}

type ItemFunc func(*Element)

func (f ItemFunc) applyTo(el *Element) { f(el) }

func Apply(item Item, el *Element) { item.applyTo(el) }

type ElementDescriptor interface {
	TagName() string
}
