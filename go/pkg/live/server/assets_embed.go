package server

import (
	"bytes"
	"embed"
	"net/http"
	"sync"
	"time"
)

const clientScriptPathValue = "/pondlive.js"

var (
	//go:embed static/pondlive.js
	clientAssetFS embed.FS

	clientAssetHandler     http.Handler
	clientAssetHandlerOnce sync.Once
)

func clientScriptHandler() http.Handler {
	clientAssetHandlerOnce.Do(func() {
		data, err := clientAssetFS.ReadFile("static/pondlive.js")
		if err != nil {
			panic("live: failed to load client assets: " + err.Error())
		}
		modTime := time.Time{}
		clientAssetHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
			http.ServeContent(w, r, "pondlive.js", modTime, bytes.NewReader(data))
		})
	})
	return clientAssetHandler
}

func clientScriptPath() string {
	return clientScriptPathValue
}
