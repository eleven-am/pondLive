package dom

import (
	"encoding/json"
	"fmt"
)

// ToJSON serializes the StructuredNode to JSON.
// Validates the node structure before serialization.
func (n *StructuredNode) ToJSON() ([]byte, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(n, "", "  ")
}

// AssignHandlerKeys walks the tree and assigns handler Keys to all EventBindings
// using a component-local counter. Format: "{componentId}:h{counter}"
func AssignHandlerKeys(node *StructuredNode, componentID string) {
	if node == nil {
		return
	}

	counter := 0
	walkAndAssignKeys(node, componentID, &counter)
}

// walkAndAssignKeys recursively walks the tree and assigns Keys
func walkAndAssignKeys(node *StructuredNode, componentID string, counter *int) {
	if node == nil {
		return
	}

	if len(node.Events) > 0 {
		for event, binding := range node.Events {
			if binding.Key == "" {
				binding.Key = fmt.Sprintf("%s:h%d", componentID, *counter)
				*counter++
				node.Events[event] = binding
			}
		}
	}

	for _, child := range node.Children {
		walkAndAssignKeys(child, componentID, counter)
	}
}

// FromJSON deserializes a StructuredNode from JSON.
// Returns the deserialized node without validation (client data may be partial).
func FromJSON(data []byte) (*StructuredNode, error) {
	var node StructuredNode
	if err := json.Unmarshal(data, &node); err != nil {
		return nil, err
	}
	return &node, nil
}

// MarshalJSON implements custom JSON marshaling for StructuredNode.
// This converts the Events map to Handlers array during serialization.
func (n *StructuredNode) MarshalJSON() ([]byte, error) {
	if n == nil {
		return []byte("null"), nil
	}

	if n.Text == "" && n.Tag == "" && n.Comment == "" && n.ComponentID == "" && !n.Fragment &&
		len(n.Children) == 0 && len(n.Events) == 0 && len(n.Attrs) == 0 {
		return []byte(`{"text":""}`), nil
	}

	temp := *n

	if len(n.Events) > 0 {
		handlers := make([]HandlerMeta, 0, len(n.Events))
		for event, binding := range n.Events {
			meta := HandlerMeta{
				Event:   event,
				Handler: binding.Key,
				Listen:  binding.Listen,
				Props:   binding.Props,
			}
			handlers = append(handlers, meta)
		}
		temp.Handlers = handlers
	}

	type Alias StructuredNode
	return json.Marshal((*Alias)(&temp))
}
