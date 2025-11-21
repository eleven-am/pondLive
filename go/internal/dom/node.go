package dom

import "fmt"

// StructuredNode is the unified representation for SSR, hydration, and diffing.
// Exactly one of ComponentID, Tag, or Text must be set.
type StructuredNode struct {
	// Node type discriminators (exactly one must be set)
	ComponentID string `json:"componentId,omitempty"` // Component boundary
	Tag         string `json:"tag,omitempty"`         // Element node
	Text        string `json:"text,omitempty"`        // Text node
	Comment     string `json:"comment,omitempty"`     // Comment node
	Fragment    bool   `json:"fragment,omitempty"`    // Fragment node (no wrapper)

	// Optional fields
	Key string `json:"key,omitempty"` // For keyed diffing

	// Element content (mutually exclusive)
	Children   []*StructuredNode `json:"children,omitempty"`   // Structured children (shared references)
	UnsafeHTML string            `json:"unsafeHtml,omitempty"` // Raw HTML (innerHTML)

	// Element attributes and styling
	Attrs  map[string][]string          `json:"attrs,omitempty"` // All attributes as token arrays
	Style  map[string]string            `json:"style,omitempty"` // Inline CSS properties
	Styles map[string]map[string]string `json:"styles,omitempty"`

	// Element metadata
	RefID    string                  `json:"refId,omitempty"`
	Handlers []HandlerMeta           `json:"handlers,omitempty"`
	Router   *RouterMeta             `json:"router,omitempty"`
	Upload   *UploadMeta             `json:"upload,omitempty"`
	Events   map[string]EventBinding `json:"-"` // Builder-time event bindings

	// Type safety for refs (not serialized)
	Descriptor ElementDescriptor `json:"-"`

	// Metadata allows builders to attach arbitrary non-serialized data used during rendering.
	Metadata map[string]any `json:"-"`

	// Runtime metadata (non-serialized)
	HandlerAssignments map[string]EventAssignment `json:"-"`
	UploadBindings     []UploadBinding            `json:"-"`
}

// Node interface that StructuredNode implements
type Node interface {
	ToHTML() string
	ToJSON() ([]byte, error)
	Validate() error
}

// Item interface for builder pattern compatibility
type Item interface {
	ApplyTo(*StructuredNode)
}

// ElementDescriptor provides type-safe ref attachment
type ElementDescriptor interface {
	TagName() string
}

// HandlerMeta describes an event handler attachment
type HandlerMeta struct {
	Event   string   `json:"event"`            // "click", "input", etc.
	Handler string   `json:"handler"`          // Handler ID: "ref:0/click"
	Listen  []string `json:"listen,omitempty"` // Additional events to listen
	Props   []string `json:"props,omitempty"`  // Event properties to capture
}

// RouterMeta describes router navigation metadata
type RouterMeta struct {
	PathValue string `json:"path,omitempty"`
	Query     string `json:"query"` // No omitempty - empty string means "clear query params"
	Hash      string `json:"hash,omitempty"`
	Replace   string `json:"replace,omitempty"`
}

// UploadMeta describes file upload configuration
type UploadMeta struct {
	UploadID string   `json:"uploadId"`
	Accept   []string `json:"accept,omitempty"`
	Multiple bool     `json:"multiple,omitempty"`
	MaxSize  int64    `json:"maxSize,omitempty"`
}

// UploadBinding mirrors the legacy dom.UploadBinding for parity with runtime.
type UploadBinding struct {
	UploadID string
	Accept   []string
	Multiple bool
	MaxSize  int64
}

// Validate ensures the node structure is correct
func (n *StructuredNode) Validate() error {
	if n == nil {
		return fmt.Errorf("cannot validate nil node")
	}

	typeCount := 0
	if n.ComponentID != "" {
		typeCount++
	}
	if n.Tag != "" {
		typeCount++
	}
	if n.Text != "" {
		typeCount++
	}
	if n.Comment != "" {
		typeCount++
	}
	if n.Fragment {
		typeCount++
	}

	if typeCount != 1 {
		return fmt.Errorf("node must have exactly one type discriminator (component, tag, text, comment, fragment)")
	}

	if n.Tag != "" {
		if n.UnsafeHTML != "" && len(n.Children) > 0 {
			return fmt.Errorf("element cannot have both UnsafeHTML and Children")
		}

		if len(n.Styles) > 0 && n.Tag != "style" {
			return fmt.Errorf("styles map only valid on <style> elements, not <%s>", n.Tag)
		}
	} else {
		if len(n.Attrs) > 0 || len(n.Style) > 0 || len(n.Styles) > 0 {
			return fmt.Errorf("only elements can have attributes or styles")
		}
		if len(n.Handlers) > 0 {
			return fmt.Errorf("only elements can have handlers")
		}
		if len(n.HandlerAssignments) > 0 {
			return fmt.Errorf("only elements can have handler assignments")
		}
		if n.Router != nil {
			return fmt.Errorf("only elements can have router metadata")
		}
		if n.Upload != nil {
			return fmt.Errorf("only elements can have upload metadata")
		}
		if len(n.UploadBindings) > 0 {
			return fmt.Errorf("only elements can have upload bindings")
		}
		if n.RefID != "" {
			return fmt.Errorf("only elements can have ref IDs")
		}
	}

	return nil
}

// ApplyTo implements the Item interface so nodes can be used as items
func (n *StructuredNode) ApplyTo(parent *StructuredNode) {
	parent.Children = append(parent.Children, n)
}
