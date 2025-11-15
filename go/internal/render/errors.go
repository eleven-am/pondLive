package render

import (
	"fmt"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// ValidationError represents an error during tree validation.
type ValidationError struct {
	Message string
	Node    h.Node
}

func (e *ValidationError) Error() string {
	if e.Node != nil {
		return fmt.Sprintf("validation error: %s (node type: %T)", e.Message, e.Node)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// TreeValidator validates a node tree before rendering.
type TreeValidator struct {
	componentIDs map[string]int
	errors       []error
}

// NewTreeValidator creates a new tree validator.
func NewTreeValidator() *TreeValidator {
	return &TreeValidator{
		componentIDs: make(map[string]int),
		errors:       make([]error, 0),
	}
}

// Validate validates the entire node tree.
func (tv *TreeValidator) Validate(root h.Node) error {
	if root == nil {
		return &ValidationError{Message: "root node cannot be nil"}
	}

	tv.validateNode(root)

	if len(tv.errors) > 0 {
		return tv.errors[0]
	}
	return nil
}

func (tv *TreeValidator) validateNode(n h.Node) {
	if n == nil {
		return
	}

	switch v := n.(type) {
	case *h.Element:
		tv.validateElement(v)
		for _, child := range v.Children {
			tv.validateNode(child)
		}
	case *h.FragmentNode:
		for _, child := range v.Children {
			tv.validateNode(child)
		}
	case *h.ComponentNode:
		tv.validateComponent(v)
		if v.Child != nil {
			tv.validateNode(v.Child)
		}
	case *h.TextNode, *h.CommentNode:

	}
}

func (tv *TreeValidator) validateElement(el *h.Element) {
	if el == nil {
		return
	}

	if el.Tag == "" {
		tv.addError(&ValidationError{
			Message: "element tag cannot be empty",
			Node:    el,
		})
	}

	if el.Attrs != nil {
		pathValue := el.Attrs["data-router-path"]
		if pathValue != "" && pathValue[0] != '/' {
			tv.addError(&ValidationError{
				Message: fmt.Sprintf("router path must start with /, got: %s", pathValue),
				Node:    el,
			})
		}
	}
}

func (tv *TreeValidator) validateComponent(comp *h.ComponentNode) {
	if comp == nil {
		return
	}

	if comp.ID != "" {
		tv.componentIDs[comp.ID]++

	}
}

func (tv *TreeValidator) addError(err error) {
	tv.errors = append(tv.errors, err)
}
