package dom2

import "encoding/json"

// ToJSON serializes the StructuredNode to JSON.
// Validates the node structure before serialization.
func (n *StructuredNode) ToJSON() ([]byte, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(n, "", "  ")
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
