package session

import (
	"github.com/eleven-am/pondlive/internal/errors"
	"github.com/eleven-am/pondlive/internal/head"
	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/portal"
	"github.com/eleven-am/pondlive/internal/router"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type DocumentConfig struct {
	State       *headers.RequestState
	Body        work.Node
	ClientAsset string
	DevMode     bool
	OnBoundary  func(*runtime.Boundary)
}

func Document(cfg DocumentConfig) work.Node {
	return headers.Provider(headers.ProviderProps{
		State: cfg.State,
		Children: head.Provider(head.ProviderProps{
			Children: router.Provider(router.ProviderProps{
				Children: head.HTML(head.ElementProps{Children: []work.Item{
					work.BuildElement("head", head.Render(), headers.Render()),
					head.Body(head.ElementProps{Children: []work.Item{
						errors.Provider(errors.Props{DevMode: cfg.DevMode, Children: cfg.Body, OnBoundary: cfg.OnBoundary}),
						portal.Target(),
						work.BuildElement("script", work.Attribute("src", cfg.ClientAsset), work.Attribute("defer", true)),
					}}),
				}}),
			}),
		}),
	})
}

func bootRoot(liveSession *LiveSession, component Component, clientAsset string, devMode bool) func(*runtime.Ctx) work.Node {
	return func(*runtime.Ctx) work.Node {
		return Document(DocumentConfig{
			State:       liveSession.requestState,
			Body:        runtime.Component(component)(),
			ClientAsset: clientAsset,
			DevMode:     devMode,
			OnBoundary:  func(b *runtime.Boundary) { liveSession.rootBoundary = b },
		})
	}
}
