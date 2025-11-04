package protocol

// UploadClient represents upload lifecycle messages sent from the client.
type UploadClient struct {
	T      string      `json:"t"`
	SID    string      `json:"sid"`
	ID     string      `json:"id"`
	Op     string      `json:"op"`
	Meta   *UploadMeta `json:"meta,omitempty"`
	Loaded int64       `json:"loaded,omitempty"`
	Total  int64       `json:"total,omitempty"`
	Error  string      `json:"error,omitempty"`
}

// UploadMeta describes the file selected on the client.
type UploadMeta struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Type string `json:"type"`
}

// UploadControl is sent from the server to the client to control in-flight uploads.
type UploadControl struct {
	T   string `json:"t"`
	SID string `json:"sid"`
	ID  string `json:"id"`
	Op  string `json:"op"`
}
