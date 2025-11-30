package server

type Event struct {
	Handler string      `json:"handler"`
	Target  string      `json:"target"`
	Data    interface{} `json:"data"`
}
