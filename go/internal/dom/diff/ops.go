package diff

type OpKind string

const (
	OpSetText      OpKind = "setText"
	OpSetComment   OpKind = "setComment"
	OpSetAttr      OpKind = "setAttr"
	OpDelAttr      OpKind = "delAttr"
	OpSetStyle     OpKind = "setStyle"
	OpDelStyle     OpKind = "delStyle"
	OpSetStyleDecl OpKind = "setStyleDecl"
	OpDelStyleDecl OpKind = "delStyleDecl"
	OpSetHandlers  OpKind = "setHandlers"
	OpSetRouter    OpKind = "setRouter"
	OpDelRouter    OpKind = "delRouter"
	OpSetUpload    OpKind = "setUpload"
	OpDelUpload    OpKind = "delUpload"
	OpSetScript    OpKind = "setScript"
	OpDelScript    OpKind = "delScript"
	OpSetRef       OpKind = "setRef"
	OpDelRef       OpKind = "delRef"
	OpSetComponent OpKind = "setComponent"
	OpReplaceNode  OpKind = "replaceNode"
	OpAddChild     OpKind = "addChild"
	OpDelChild     OpKind = "delChild"
	OpMoveChild    OpKind = "moveChild"
)

type Patch struct {
	Seq      int         `json:"seq"`
	Path     []int       `json:"path"`
	Op       OpKind      `json:"op"`
	Value    interface{} `json:"value,omitempty"`
	Name     string      `json:"name,omitempty"`
	Selector string      `json:"selector,omitempty"`
	Index    *int        `json:"index,omitempty"`
}
