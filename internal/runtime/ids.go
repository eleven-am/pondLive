package runtime

import (
	"fmt"

	"github.com/eleven-am/pondlive/internal/work"
)

func forEachChild(children []work.Node, fn func(work.Node)) {
	for _, c := range children {
		if f, ok := c.(*work.Fragment); ok {
			forEachChild(f.Children, fn)
			continue
		}
		if c != nil {
			fn(c)
		}
	}
}

func soleBoundText(el *work.Element) *work.BoundText {
	if el.Raw != nil {
		return nil
	}
	var found *work.BoundText
	total := 0
	forEachChild(el.Children, func(n work.Node) {
		total++
		if bt, ok := n.(*work.BoundText); ok {
			found = bt
		}
	})
	if total == 1 {
		return found
	}
	return nil
}

func isElement(n work.Node) bool {
	_, ok := n.(*work.Element)
	return ok
}

func elementDynamic(el *work.Element, force bool) bool {
	return force || el.Dynamic() || soleBoundText(el) != nil
}

func count(n work.Node, force bool) uint32 {
	switch x := n.(type) {
	case *work.Element:
		var total uint32
		if elementDynamic(x, force) {
			total = 1
		}
		if x.Raw != nil || isVoid(x.Tag) {
			return total
		}
		sole := soleBoundText(x)
		forEachChild(x.Children, func(c work.Node) {
			if sole != nil {
				return
			}
			total += count(c, false)
		})
		return total
	case *work.BoundText:
		return 1
	case *work.RegionNode:
		return 1
	case *work.PortalTarget:
		return 1
	case *work.Fragment:
		var total uint32
		forEachChild(x.Children, func(c work.Node) { total += count(c, false) })
		return total
	case *work.UnitRegion:
		total := uint32(1)
		forEachChild(x.Children, func(c work.Node) { total += count(c, isElement(c)) })
		return total
	}
	return 0
}

func normalizeRoot(n work.Node) work.Node {
	for {
		switch x := n.(type) {
		case *work.Element, *work.ComponentNode, *work.RegionNode, *work.PortalTarget:
			return x
		case *work.Fragment:
			var only work.Node
			total := 0
			forEachChild(x.Children, func(c work.Node) {
				total++
				only = c
			})
			if total != 1 {
				return &work.UnitRegion{Children: x.Children}
			}
			n = only
		case *work.UnitRegion:
			return x
		case nil:
			return &work.UnitRegion{}
		default:
			return &work.UnitRegion{Children: []work.Node{x}}
		}
	}
}

func validate(n work.Node) error {
	switch x := n.(type) {
	case *work.Element:
		if !validName(x.Tag) {
			return &InvalidNameError{Kind: "tag", Name: x.Tag}
		}
		for _, c := range x.Children {
			if err := validate(c); err != nil {
				return err
			}
		}
	case *work.Fragment:
		for _, c := range x.Children {
			if err := validate(c); err != nil {
				return err
			}
		}
	case *work.UnitRegion:
		for _, c := range x.Children {
			if err := validate(c); err != nil {
				return err
			}
		}
	case *work.PortalNode:
		for _, c := range x.Children {
			if err := validate(c); err != nil {
				return err
			}
		}
	case *work.ComponentNode:
		if _, ok := x.Fn.(componentFn); !ok {
			return fmt.Errorf("runtime: component %q has an unsupported function type %T", x.Name, x.Fn)
		}
	case *work.RegionNode:
		if _, ok := x.Impl.(regionDef); !ok {
			return fmt.Errorf("runtime: unsupported region implementation %T", x.Impl)
		}
	}
	return nil
}
