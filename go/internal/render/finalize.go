package render

import (
	"sort"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/handlers"
)

// Finalize normalizes a node tree by folding metadata into attributes.
func Finalize(n dom.Node) dom.Node {
	finalizeNode(n, nil)
	return n
}

// FinalizeWithHandlers normalizes the tree and attaches handler IDs using the
// provided registry.
func FinalizeWithHandlers(n dom.Node, reg handlers.Registry) dom.Node {
	finalizeNode(n, reg)
	return n
}

func finalizeNode(n dom.Node, reg handlers.Registry) {
	switch v := n.(type) {
	case *dom.Element:
		finalizeElement(v, reg)
		for _, child := range v.Children {
			if child == nil {
				continue
			}
			finalizeNode(child, reg)
		}
	case *dom.FragmentNode:
		for _, child := range v.Children {
			if child == nil {
				continue
			}
			finalizeNode(child, reg)
		}
	case *dom.ComponentNode:
		if v.Child != nil {
			finalizeNode(v.Child, reg)
		}
	}
}

func finalizeElement(e *dom.Element, reg handlers.Registry) {
	if e == nil {
		return
	}
	dom.FinalizeElement(e)

	attachHandlers(e, reg)

}

func attachHandlers(e *dom.Element, reg handlers.Registry) {
	if e == nil || len(e.Events) == 0 || reg == nil {
		return
	}
	if e.Attrs == nil {
		e.Attrs = map[string]string{}
	}
	if e.HandlerAssignments == nil {
		e.HandlerAssignments = map[string]dom.EventAssignment{}
	}
	keys := make([]string, 0, len(e.Events))
	for name := range e.Events {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	for _, name := range keys {
		binding := e.Events[name]
		id := reg.Ensure(binding.Handler)
		if id == "" {
			continue
		}
		attrName := "data-on" + name
		e.Attrs[attrName] = string(id)
		if listens := binding.Listen; len(listens) > 0 {
			e.Attrs[attrName+"-listen"] = strings.Join(listens, " ")
		}
		if props := binding.Props; len(props) > 0 {
			e.Attrs[attrName+"-props"] = strings.Join(props, " ")
		}
		e.HandlerAssignments[name] = dom.EventAssignment{
			ID:     string(id),
			Listen: append([]string(nil), binding.Listen...),
			Props:  append([]string(nil), binding.Props...),
		}
	}
}
