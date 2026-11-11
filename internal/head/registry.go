package head

import (
	"bytes"
	"net/http"
	"slices"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type Registry struct {
	entries *runtime.Signal[[]*entry]
	seq     uint64

	merged      runtime.ReadSignal[*merged]
	title       runtime.ReadSignal[string]
	description runtime.ReadSignal[string]
	lang        runtime.ReadSignal[string]
	dir         runtime.ReadSignal[string]
	htmlClass   runtime.ReadSignal[string]
	bodyClass   runtime.ReadSignal[string]
	meta        runtime.ReadSignal[[]MetaTag]
	links       runtime.ReadSignal[[]LinkTag]
	scripts     runtime.ReadSignal[[]ScriptTag]
	styles      runtime.ReadSignal[[]StyleTag]
	hasIcon     runtime.ReadSignal[bool]

	png32URL  string
	png180URL string
	cache     iconCache
}

type iconCache struct {
	icon   *iconInfo
	png32  []byte
	png180 []byte
}

var (
	headCtx  = runtime.NewContext[*Registry]()
	depthCtx = runtime.NewContext[int]()
)

func (r *Registry) add(e *entry) {
	r.entries.Set(append(slices.Clone(r.entries.Peek()), e))
}

func (r *Registry) remove(e *entry) {
	current := r.entries.Peek()
	i := slices.Index(current, e)
	if i < 0 {
		return
	}
	r.entries.Set(slices.Delete(slices.Clone(current), i, i+1))
}

func (r *Registry) EntryCount() int { return len(r.entries.Peek()) }

func (r *Registry) png(size int) []byte {
	icon := mergeEntries(r.entries.Peek()).icon
	if !iconEqual(r.cache.icon, icon) || (r.cache.png32 == nil && r.cache.png180 == nil) {
		r.cache = iconCache{icon: icon}
		if icon != nil {
			svg := renderIconToSVG(icon)
			r.cache.png32 = renderIconToPNG(svg, 32, icon.background, icon.color)
			r.cache.png180 = renderIconToPNG(svg, 180, icon.background, icon.color)
		}
	}
	if size == 180 {
		return r.cache.png180
	}
	return r.cache.png32
}

func (r *Registry) serveIcon(size int) runtime.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) error {
		data := r.png(size)
		if len(data) == 0 {
			w.WriteHeader(http.StatusNotFound)
			return nil
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		_, _ = bytes.NewReader(data).WriteTo(w)
		return nil
	}
}

type ProviderProps struct {
	Children work.Node
}

var Provider = runtime.PropsComponent(func(ctx *runtime.Ctx, props ProviderProps) work.Node {
	reg := &Registry{entries: runtime.UseSignal(ctx, []*entry(nil))}

	m := runtime.UseComputed(ctx, func() *merged {
		return mergeEntries(reg.entries.Get())
	})
	reg.merged = m
	reg.title = runtime.UseComputed(ctx, func() string { return m.Get().title })
	reg.description = runtime.UseComputed(ctx, func() string { return m.Get().description })
	reg.lang = runtime.UseComputed(ctx, func() string { return m.Get().lang })
	reg.dir = runtime.UseComputed(ctx, func() string { return m.Get().dir })
	reg.htmlClass = runtime.UseComputed(ctx, func() string { return m.Get().htmlClass })
	reg.bodyClass = runtime.UseComputed(ctx, func() string { return m.Get().bodyClass })
	reg.meta = runtime.UseComputed(ctx, func() []MetaTag { return m.Get().meta }, runtime.WithEqual(equalBy(MetaTag.signature)))
	reg.links = runtime.UseComputed(ctx, func() []LinkTag { return m.Get().links }, runtime.WithEqual(equalBy(LinkTag.signature)))
	reg.scripts = runtime.UseComputed(ctx, func() []ScriptTag { return m.Get().scripts }, runtime.WithEqual(equalBy(ScriptTag.signature)))
	reg.styles = runtime.UseComputed(ctx, func() []StyleTag { return m.Get().styles }, runtime.WithEqual(equalBy(StyleTag.signature)))
	reg.hasIcon = runtime.UseComputed(ctx, func() bool { return m.Get().icon != nil })

	reg.png32URL = runtime.UseHandler(ctx, "GET", reg.serveIcon(32)).URL()
	reg.png180URL = runtime.UseHandler(ctx, "GET", reg.serveIcon(180)).URL()

	runtime.Provide(ctx, headCtx, runtime.UseSignal(ctx, reg))
	runtime.Provide(ctx, depthCtx, runtime.UseSignal(ctx, 0))

	if props.Children == nil {
		return work.NewFragment()
	}
	return props.Children
})

func UseHead(ctx *runtime.Ctx, h Head) {
	reg := runtime.UseContext(ctx, headCtx).Peek()
	depth := runtime.UseContext(ctx, depthCtx).Peek()
	runtime.Provide(ctx, depthCtx, runtime.UseSignal(ctx, depth+1))
	runtime.UseRenderEffect(ctx, func() func() {
		reg.seq++
		e := &entry{depth: depth, seq: reg.seq, h: h}
		reg.add(e)
		return func() { reg.remove(e) }
	})
}

func UseRegistry(ctx *runtime.Ctx) *Registry {
	return runtime.UseContext(ctx, headCtx).Peek()
}
