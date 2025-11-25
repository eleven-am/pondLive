package protocol

type LiveEndpoint interface {
	Path() string
	OnConnect() error
	OnClose() error
	OnEvent(evt string, payload any)
}
