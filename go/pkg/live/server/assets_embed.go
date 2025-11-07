package server

import (
	"bytes"
	"embed"
	"errors"
	"io/fs"
	"net/http"
	"path"
	"sync"
	"time"
)

const (
	prodClientScriptPath = "/pondlive.js"
	devClientScriptPath  = "/pondlive-dev.js"
)

var (
	//go:embed static/pondlive.js static/pondlive.js.map static/pondlive-dev.js static/pondlive-dev.js.map
	clientAssetFS embed.FS

	clientAssetsOnce sync.Once
	clientScripts    map[string][]byte
	clientSourceMaps map[string][]byte
)

func ensureClientAssets() {
	clientAssetsOnce.Do(func() {
		clientScripts = make(map[string][]byte)
		clientSourceMaps = make(map[string][]byte)

		load := func(name string) []byte {
			data, err := clientAssetFS.ReadFile("static/" + name)
			if err != nil {
				panic("live: failed to load client assets: " + err.Error())
			}
			return data
		}
		loadOptional := func(name string) []byte {
			data, err := clientAssetFS.ReadFile("static/" + name)
			if err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				panic("live: failed to load client assets: " + err.Error())
			}
			return data
		}

		clientScripts["pondlive.js"] = load("pondlive.js")
		if devData := loadOptional("pondlive-dev.js"); len(devData) > 0 {
			clientScripts["pondlive-dev.js"] = devData
		}

		if mapData := loadOptional("pondlive.js.map"); len(mapData) > 0 {
			clientSourceMaps["pondlive.js.map"] = mapData
		}
		if mapData := loadOptional("pondlive-dev.js.map"); len(mapData) > 0 {
			clientSourceMaps["pondlive-dev.js.map"] = mapData
		}
	})
}

func clientScriptPath(dev bool) string {
	if dev {
		return devClientScriptPath
	}
	return prodClientScriptPath
}

func clientSourceMapPath(scriptPath string) string {
	if scriptPath == "" {
		return ""
	}
	return scriptPath + ".map"
}

func embeddedClientScriptHandler(scriptPath string) http.Handler {
	ensureClientAssets()
	name := path.Base(scriptPath)
	if name == "." || name == "/" || name == "" {
		name = "pondlive.js"
	}
	data, ok := clientScripts[name]
	if !ok {
		name = "pondlive.js"
		data = clientScripts[name]
	}
	modTime := time.Time{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		http.ServeContent(w, r, name, modTime, bytes.NewReader(data))
	})
}

func embeddedClientSourceMapHandler(mapPath string) http.Handler {
	ensureClientAssets()
	name := path.Base(mapPath)
	if name == "." || name == "/" || name == "" {
		name = "pondlive.js.map"
	}
	data, ok := clientSourceMaps[name]
	if !ok {
		return nil
	}
	modTime := time.Time{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		http.ServeContent(w, r, name, modTime, bytes.NewReader(data))
	})
}
