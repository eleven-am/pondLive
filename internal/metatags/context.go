package metatags

import (
	"net/http"
	"sync"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type Ctx = runtime.Ctx

type entriesStore struct {
	mu      sync.RWMutex
	entries map[string]metaEntry
}

func (s *entriesStore) set(componentID string, entry metaEntry) {
	s.mu.Lock()
	s.entries[componentID] = entry
	s.mu.Unlock()
}

func (s *entriesStore) delete(componentID string) {
	s.mu.Lock()
	delete(s.entries, componentID)
	s.mu.Unlock()
}

func (s *entriesStore) snapshot() map[string]metaEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cp := make(map[string]metaEntry, len(s.entries))
	for k, v := range s.entries {
		cp[k] = v
	}
	return cp
}

type metaState struct {
	store    *entriesStore
	revision int
}

type faviconState struct {
	png32URL    string
	png180URL   string
	initialized bool
}

var noopStore = &entriesStore{entries: make(map[string]metaEntry)}

var noopState = &metaState{
	store:    noopStore,
	revision: 0,
}

var metaCtx = runtime.CreateContext[*metaState](noopState)
var faviconCtx = runtime.CreateContext[*faviconState](nil)

var Provider = runtime.Component(func(ctx *Ctx, children []work.Item) work.Node {
	storeRef := runtime.UseRef(ctx, &entriesStore{entries: make(map[string]metaEntry)})

	state, setState := metaCtx.UseProvider(ctx, &metaState{
		store:    storeRef.Current,
		revision: 0,
	})

	state.store = storeRef.Current

	providerCtx.UseProvider(ctx, &providerFuncs{
		update: func(componentID string, entry metaEntry) {
			storeRef.Current.set(componentID, entry)
			setState(&metaState{
				store:    storeRef.Current,
				revision: state.revision + 1,
			})
		},
		remove: func(componentID string) {
			storeRef.Current.delete(componentID)
			setState(&metaState{
				store:    storeRef.Current,
				revision: state.revision + 1,
			})
		},
	})

	snapshot := storeRef.Current.snapshot()
	icon := getWinningIcon(snapshot)

	svgData := runtime.UseMemo(ctx, func() []byte {
		return renderIconToSVG(icon)
	}, icon)

	var iconBg, iconColor string
	if icon != nil {
		iconBg = icon.background
		iconColor = icon.color
	}

	png32Data := runtime.UseMemo(ctx, func() []byte {
		return renderIconToPNG(svgData, 32, iconBg, iconColor)
	}, svgData, iconBg, iconColor)

	png180Data := runtime.UseMemo(ctx, func() []byte {
		return renderIconToPNG(svgData, 180, iconBg, iconColor)
	}, svgData, iconBg, iconColor)

	png32Ref := runtime.UseRef(ctx, png32Data)
	png32Ref.Current = png32Data

	png180Ref := runtime.UseRef(ctx, png180Data)
	png180Ref.Current = png180Data

	png32Handler := runtime.UseHandler(ctx, "GET", func(w http.ResponseWriter, r *http.Request) error {
		data := png32Ref.Current
		if len(data) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return nil
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Write(data)
		return nil
	})

	png180Handler := runtime.UseHandler(ctx, "GET", func(w http.ResponseWriter, r *http.Request) error {
		data := png180Ref.Current
		if len(data) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return nil
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Write(data)
		return nil
	})

	faviconCtx.UseProvider(ctx, &faviconState{
		png32URL:    png32Handler.URL(),
		png180URL:   png180Handler.URL(),
		initialized: true,
	})

	nodes := itemsToNodes(children)
	return &work.Fragment{Children: nodes}
})

type providerFuncs struct {
	update func(componentID string, entry metaEntry)
	remove func(componentID string)
}

var providerCtx = runtime.CreateContext[*providerFuncs](nil)

func itemsToNodes(items []work.Item) []work.Node {
	nodes := make([]work.Node, 0, len(items))
	for _, item := range items {
		if node, ok := item.(work.Node); ok {
			nodes = append(nodes, node)
		}
	}
	return nodes
}
