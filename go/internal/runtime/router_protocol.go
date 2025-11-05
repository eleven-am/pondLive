package runtime

type NavMsg struct {
	T    string `json:"t"`
	Path string `json:"path"`
	Q    string `json:"q,omitempty"`
	Hash string `json:"hash,omitempty"`
}

type PopMsg struct {
	T    string `json:"t"`
	Path string `json:"path"`
	Q    string `json:"q,omitempty"`
	Hash string `json:"hash,omitempty"`
}
