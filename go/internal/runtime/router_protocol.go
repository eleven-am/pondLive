package runtime

type navMessage struct {
	T    string `json:"t"`
	Path string `json:"path"`
	Q    string `json:"q,omitempty"`
	Hash string `json:"hash,omitempty"`
}

// NavMsg represents a navigation-related message exchanged with the router.
type NavMsg = navMessage

// PopMsg aliases NavMsg to avoid duplicating the message structure for popstate events.
type PopMsg = navMessage
