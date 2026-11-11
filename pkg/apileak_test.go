package pkg

import (
	"go/importer"
	"go/token"
	"go/types"
	"sort"
	"strings"
	"sync"
	"testing"
)

const modulePath = "github.com/eleven-am/pondlive"

var (
	importerOnce sync.Once
	sharedFset   *token.FileSet
	sharedSource types.ImporterFrom
)

func sourceImporter() (*token.FileSet, types.ImporterFrom) {
	importerOnce.Do(func() {
		sharedFset = token.NewFileSet()
		sharedSource = importer.ForCompiler(sharedFset, "source", nil).(types.ImporterFrom)
	})
	return sharedFset, sharedSource
}

func isInternal(p *types.Package) bool {
	return p != nil && strings.HasPrefix(p.Path(), modulePath+"/internal/")
}

type leakWalker struct {
	exported map[*types.TypeName]bool
	seen     map[types.Type]bool
	leaks    map[string]bool
}

func (w *leakWalker) visit(path string, t types.Type) {
	t = types.Unalias(t)
	if w.seen[t] {
		return
	}
	w.seen[t] = true
	switch x := t.(type) {
	case *types.Pointer:
		w.visit(path, x.Elem())
	case *types.Slice:
		w.visit(path, x.Elem())
	case *types.Array:
		w.visit(path, x.Elem())
	case *types.Chan:
		w.visit(path, x.Elem())
	case *types.Map:
		w.visit(path, x.Key())
		w.visit(path, x.Elem())
	case *types.Signature:
		for i := 0; i < x.Params().Len(); i++ {
			w.visit(path, x.Params().At(i).Type())
		}
		for i := 0; i < x.Results().Len(); i++ {
			w.visit(path, x.Results().At(i).Type())
		}
	case *types.Struct:
		for i := 0; i < x.NumFields(); i++ {
			if f := x.Field(i); f.Exported() {
				w.visit(path+"."+f.Name(), f.Type())
			}
		}
	case *types.Interface:
		for i := 0; i < x.NumMethods(); i++ {
			if m := x.Method(i); m.Exported() {
				w.visit(path+"."+m.Name(), m.Type())
			}
		}
	case *types.Named:
		origin := x.Origin()
		for i := 0; i < x.TypeArgs().Len(); i++ {
			w.visit(path, x.TypeArgs().At(i))
		}
		obj := origin.Obj()
		if isInternal(obj.Pkg()) && obj.Exported() && !w.exported[obj] {
			w.leaks[path+" -> "+obj.Pkg().Name()+"."+obj.Name()] = true
			return
		}
		if obj.Pkg() != nil && !strings.HasPrefix(obj.Pkg().Path(), modulePath) {
			return
		}
		w.visit(path, origin.Underlying())
		for i := 0; i < origin.NumMethods(); i++ {
			if m := origin.Method(i); m.Exported() {
				w.visit(path+"."+m.Name(), m.Type())
			}
		}
	}
}

func TestNoInternalTypeIsReachableFromThePublicAPIOrLivetest(t *testing.T) {
	_, imp := sourceImporter()
	pk, err := imp.ImportFrom(modulePath+"/pkg", ".", 0)
	if err != nil {
		t.Fatal(err)
	}
	live, err := imp.ImportFrom(modulePath+"/pkg/livetest", ".", 0)
	if err != nil {
		t.Fatal(err)
	}
	w := &leakWalker{exported: map[*types.TypeName]bool{}, seen: map[types.Type]bool{}, leaks: map[string]bool{}}
	for _, name := range pk.Scope().Names() {
		obj := pk.Scope().Lookup(name)
		if tn, ok := obj.(*types.TypeName); ok && obj.Exported() {
			if n, ok := types.Unalias(tn.Type()).(*types.Named); ok {
				w.exported[n.Origin().Obj()] = true
			}
		}
	}
	for _, name := range pk.Scope().Names() {
		obj := pk.Scope().Lookup(name)
		if obj.Exported() {
			w.visit("pkg."+name, obj.Type())
		}
	}
	for _, name := range live.Scope().Names() {
		if obj := live.Scope().Lookup(name); obj.Exported() {
			w.visit("livetest."+name, obj.Type())
		}
	}
	var leaks []string
	for l := range w.leaks {
		leaks = append(leaks, l)
	}
	sort.Strings(leaks)
	if len(leaks) > 0 {
		t.Fatalf("public API reaches types that pkg does not re-export:\n%s", strings.Join(leaks, "\n"))
	}
}
