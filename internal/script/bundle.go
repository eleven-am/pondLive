package script

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

const (
	nsEntry    = "pl-entry"
	nsModule   = "pl-mod"
	entryName  = "pl-entry"
	scriptPref = "pl-script:"
	inlineFile = "(script)"
)

type Options struct {
	Dev      bool
	Features []Feature
}

type Bundle struct {
	Code []byte
	Keys []string
}

type BuildError struct {
	Declaration string
	File        string
	Line        int
	Col         int
	Text        string
}

func (e *BuildError) Error() string {
	var b strings.Builder
	if e.Declaration != "" {
		b.WriteString(e.Declaration)
		b.WriteByte(' ')
	}
	if e.File != "" {
		b.WriteString(e.File)
		b.WriteByte(' ')
	}
	if e.Line > 0 {
		fmt.Fprintf(&b, "%d:%d: ", e.Line, e.Col)
	} else if e.Declaration != "" || e.File != "" {
		b.WriteString(": ")
	}
	b.WriteString(e.Text)
	return b.String()
}

func (o Options) enabled(f Feature) bool {
	if f == "" {
		return true
	}
	for _, x := range o.Features {
		if x == f {
			return true
		}
	}
	return false
}

func (r *Registry) Bundle(opts Options) (*Bundle, error) {
	if err := r.Seal(); err != nil {
		return nil, err
	}
	var entries []Entry
	for _, e := range r.Entries() {
		if opts.enabled(e.Feature()) {
			entries = append(entries, e)
		}
	}
	keys := make([]string, len(entries))
	for i, e := range entries {
		keys[i] = e.Key()
	}
	sort.Strings(keys)
	if len(entries) == 0 {
		return &Bundle{}, nil
	}
	byKey := make(map[string]Entry, len(entries))
	for _, e := range entries {
		byKey[e.Key()] = e
	}
	entrySrc := generatedEntry(keys)

	res := api.Build(api.BuildOptions{
		EntryPoints:       []string{entryName},
		Bundle:            true,
		Write:             false,
		Format:            api.FormatIIFE,
		Platform:          api.PlatformBrowser,
		Target:            api.ES2020,
		MinifyWhitespace:  !opts.Dev,
		MinifyIdentifiers: !opts.Dev,
		MinifySyntax:      !opts.Dev,
		Charset:           api.CharsetUTF8,
		LegalComments:     api.LegalCommentsNone,
		LogLevel:          api.LogLevelSilent,
		Sourcemap:         api.SourceMapNone,
		Plugins:           []api.Plugin{{Name: "pondlive-scripts", Setup: setup(byKey, entrySrc)}},
	})
	if len(res.Errors) > 0 {
		return nil, joinErrors(res.Errors, byKey, keys)
	}
	if len(res.OutputFiles) != 1 {
		return nil, fmt.Errorf("script bundle: esbuild produced %d outputs, want 1", len(res.OutputFiles))
	}
	return &Bundle{Code: res.OutputFiles[0].Contents, Keys: keys}, nil
}

func Build(opts Options) (*Bundle, error) { return Default.Bundle(opts) }

func generatedEntry(keys []string) string {
	var b strings.Builder
	for i, k := range keys {
		fmt.Fprintf(&b, "import d%d from %q;\ndefine(%q, d%d);\n", i, scriptPref+k, k, i)
	}
	return b.String()
}

func modulePath(key, name string) string { return key + ":" + name }

func splitModulePath(p string) (key, name string) {
	i := strings.IndexByte(p, ':')
	if i < 0 {
		return "", p
	}
	return p[:i], p[i+1:]
}

func setup(byKey map[string]Entry, entrySrc string) func(api.PluginBuild) {
	return func(pb api.PluginBuild) {
		pb.OnResolve(api.OnResolveOptions{Filter: ".*"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
			return resolve(byKey, args)
		})
		pb.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: nsEntry}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			src := entrySrc
			return api.OnLoadResult{Contents: &src, Loader: api.LoaderJS}, nil
		})
		pb.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: nsModule}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
			return load(byKey, args)
		})
	}
}

func resolve(byKey map[string]Entry, args api.OnResolveArgs) (api.OnResolveResult, error) {
	if args.Kind == api.ResolveEntryPoint {
		if args.Path != entryName {
			return api.OnResolveResult{}, fmt.Errorf("unexpected entry point %q", args.Path)
		}
		return api.OnResolveResult{Path: entryName, Namespace: nsEntry}, nil
	}
	if args.Namespace == nsEntry {
		key, ok := strings.CutPrefix(args.Path, scriptPref)
		if !ok {
			return api.OnResolveResult{}, fmt.Errorf("unexpected import %q", args.Path)
		}
		e, ok := byKey[key]
		if !ok {
			return api.OnResolveResult{}, fmt.Errorf("unknown script %q", key)
		}
		name := inlineFile
		if e.FS() != nil {
			name = e.Entry()
		}
		return api.OnResolveResult{Path: modulePath(key, name), Namespace: nsModule}, nil
	}
	key, importer := splitModulePath(args.Importer)
	e, ok := byKey[key]
	if !ok {
		return api.OnResolveResult{}, fmt.Errorf("import %q from unknown module %q", args.Path, args.Importer)
	}
	if e.FS() == nil {
		return api.OnResolveResult{}, fmt.Errorf("inline scripts cannot import %q; use NewScriptFS and vendor the dependency", args.Path)
	}
	spec := args.Path
	switch {
	case strings.HasPrefix(spec, "/"):
		return api.OnResolveResult{}, fmt.Errorf("absolute import %q is not allowed", spec)
	case !strings.HasPrefix(spec, "./") && !strings.HasPrefix(spec, "../"):
		return api.OnResolveResult{}, fmt.Errorf("bare import %q is not allowed; vendor it into the script FS", spec)
	}
	resolved := path.Join(path.Dir(importer), spec)
	if !fs.ValidPath(resolved) {
		return api.OnResolveResult{}, fmt.Errorf("import %q escapes the script FS", spec)
	}
	found, err := findFile(e.FS(), resolved)
	if err != nil {
		return api.OnResolveResult{}, fmt.Errorf("import %q: %w", spec, err)
	}
	return api.OnResolveResult{Path: modulePath(key, found), Namespace: nsModule}, nil
}

func findFile(fsys fs.FS, name string) (string, error) {
	for _, cand := range []string{name, name + ".ts", name + ".js"} {
		info, err := fs.Stat(fsys, cand)
		if err == nil && !info.IsDir() {
			return cand, nil
		}
	}
	return "", fmt.Errorf("%q not found in the script FS", name)
}

func load(byKey map[string]Entry, args api.OnLoadArgs) (api.OnLoadResult, error) {
	key, name := splitModulePath(args.Path)
	e, ok := byKey[key]
	if !ok {
		return api.OnLoadResult{}, fmt.Errorf("unknown script %q", key)
	}
	var src string
	loader := api.LoaderTS
	isEntry := true
	if e.FS() == nil {
		src = e.Source()
	} else {
		data, err := fs.ReadFile(e.FS(), name)
		if err != nil {
			return api.OnLoadResult{}, err
		}
		src = string(data)
		isEntry = name == e.Entry()
		switch path.Ext(name) {
		case ".ts":
		case ".js", ".mjs":
			loader = api.LoaderJS
		default:
			return api.OnLoadResult{}, fmt.Errorf("unsupported file type %q", name)
		}
	}
	if isEntry {
		src += "\nexport {};\n"
	}
	return api.OnLoadResult{Contents: &src, Loader: loader}, nil
}

func joinErrors(msgs []api.Message, byKey map[string]Entry, keys []string) error {
	errs := make([]error, 0, len(msgs))
	for _, m := range msgs {
		errs = append(errs, convert(m, byKey, keys))
	}
	return errors.Join(errs...)
}

func convert(m api.Message, byKey map[string]Entry, keys []string) *BuildError {
	be := &BuildError{Text: m.Text}
	loc := m.Location
	if loc == nil {
		return be
	}
	switch {
	case strings.HasPrefix(loc.File, nsEntry+":"):
		idx := (loc.Line - 1) / 2
		if idx >= 0 && idx < len(keys) {
			e := byKey[keys[idx]]
			be.Declaration = e.Location()
			be.File = displayName(e, e.Entry())
			if strings.Contains(m.Text, "default") {
				be.Text = "script module has no default export"
			}
		}
	case strings.HasPrefix(loc.File, nsModule+":"):
		key, name := splitModulePath(strings.TrimPrefix(loc.File, nsModule+":"))
		if e, ok := byKey[key]; ok {
			be.Declaration = e.Location()
			be.File = displayName(e, name)
		}
		be.Line = loc.Line
		be.Col = loc.Column + 1
	default:
		be.File = loc.File
		be.Line = loc.Line
		be.Col = loc.Column + 1
	}
	return be
}

func displayName(e Entry, name string) string {
	if e.FS() == nil {
		return inlineFile
	}
	return name
}
