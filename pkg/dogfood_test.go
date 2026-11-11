package pkg

import (
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

type symbol struct{ pkg, name string }

func (s symbol) String() string { return s.pkg + "." + s.name }

func typeCheck(t *testing.T, fset *token.FileSet, imp types.Importer, path, dir string) (*types.Package, []*ast.File, *types.Info) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var files []*ast.File
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
			continue
		}
		if ok, err := build.Default.MatchFile(dir, e.Name()); err != nil || !ok {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, e.Name()), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, f)
	}
	info := &types.Info{Uses: map[*ast.Ident]types.Object{}, Selections: map[*ast.SelectorExpr]*types.Selection{}}
	conf := types.Config{Importer: imp}
	pk, err := conf.Check(path, fset, files, info)
	if err != nil {
		t.Fatalf("%s does not type-check: %v", path, err)
	}
	return pk, files, info
}

func internalPath(p *types.Package) string {
	if p == nil || !strings.HasPrefix(p.Path(), modulePath+"/internal/") {
		return ""
	}
	return strings.TrimPrefix(p.Path(), modulePath+"/")
}

func reexported(pk *types.Package, files []*ast.File, info *types.Info) map[symbol]bool {
	out := map[symbol]bool{}
	add := func(obj types.Object) {
		if obj != nil && internalPath(obj.Pkg()) != "" && obj.Parent() == obj.Pkg().Scope() {
			out[symbol{internalPath(obj.Pkg()), obj.Name()}] = true
		}
	}
	calleeOf := func(e ast.Expr) types.Object {
		for {
			switch x := e.(type) {
			case *ast.CallExpr:
				e = x.Fun
				continue
			case *ast.IndexExpr:
				e = x.X
				continue
			case *ast.IndexListExpr:
				e = x.X
				continue
			case *ast.SelectorExpr:
				return info.Uses[x.Sel]
			case *ast.Ident:
				return info.Uses[x]
			}
			return nil
		}
	}
	for _, f := range files {
		for _, d := range f.Decls {
			switch x := d.(type) {
			case *ast.FuncDecl:
				if x.Recv != nil || !x.Name.IsExported() || x.Body == nil || len(x.Body.List) != 1 {
					continue
				}
				switch st := x.Body.List[0].(type) {
				case *ast.ReturnStmt:
					if len(st.Results) == 1 {
						add(calleeOf(st.Results[0]))
					}
				case *ast.ExprStmt:
					add(calleeOf(st.X))
				}
			case *ast.GenDecl:
				for _, s := range x.Specs {
					switch sp := s.(type) {
					case *ast.TypeSpec:
						if sp.Name.IsExported() && sp.Assign.IsValid() {
							add(calleeOf(sp.Type))
						}
					case *ast.ValueSpec:
						for i, n := range sp.Names {
							if n.IsExported() && i < len(sp.Values) {
								add(calleeOf(sp.Values[i]))
							}
						}
					}
				}
			}
		}
	}
	return out
}

var builtIns = []string{"router", "head", "errors", "portal", "headers"}

var pureHelpers = map[string]bool{
	"internal/route":    true,
	"internal/htmlutil": true,
	"internal/css":      true,
}

type exception struct {
	builtIn string
	prefix  string
	reason  string
}

var exceptions = []exception{
	{"headers", "internal/protocol.", "browser channel: the router bus commands and popstate, until UseScript carries it"},
	{"headers", "internal/runtime.GetBus", "browser channel, until UseScript carries it"},
	{"headers", "internal/runtime.SessionOf", "reads the connected flag; browser channel, until UseScript carries it"},
	{"headers", "internal/runtime.Session.", "methods of the session reached through SessionOf"},
	{"headers", "internal/runtime.CtxOnActor", "actor posting, until UseScript carries the channel"},
	{"headers", "internal/runtime.RunOnActor", "actor posting, until UseScript carries the channel"},
	{"headers", "internal/runtime.Invoke", "actor posting, until UseScript carries the channel"},
	{"headers", "internal/runtime.ReportOn", "framework diagnostics from the channel"},
	{"headers", "internal/runtime.Report", "framework diagnostics from the channel"},
	{"headers", "internal/runtime.Severity", "framework-only severities for those diagnostics"},
	{"headers", "internal/runtime.AttachItem", "attaches the cookie script handle, until UseScript carries it"},
	{"headers", "internal/effectguard.", "the navigation guard is owned by the request channel"},
	{"portal", "internal/work.NewPortal", "portal is a true primitive"},
	{"portal", "internal/work.PortalNode", "portal is a true primitive"},
	{"portal", "internal/work.PortalTarget", "portal is a true primitive"},
}

func exempt(builtIn, name string, used map[int]bool) bool {
	for i, e := range exceptions {
		if e.builtIn == builtIn && strings.HasPrefix(name, e.prefix) {
			used[i] = true
			return true
		}
	}
	return false
}

func typeNamesReexported(pk *types.Package) map[string]bool {
	out := map[string]bool{}
	for _, name := range pk.Scope().Names() {
		if tn, ok := pk.Scope().Lookup(name).(*types.TypeName); ok && tn.Exported() {
			if n, ok := types.Unalias(tn.Type()).(*types.Named); ok && internalPath(n.Obj().Pkg()) != "" {
				out[internalPath(n.Obj().Pkg())+"."+n.Obj().Name()] = true
			}
		}
	}
	return out
}

func receiverName(f *types.Func) (string, bool) {
	recv := f.Type().(*types.Signature).Recv()
	if recv == nil {
		return "", false
	}
	typ := recv.Type()
	for {
		p, ok := typ.(*types.Pointer)
		if !ok {
			break
		}
		typ = p.Elem()
	}
	if n, ok := types.Unalias(typ).(*types.Named); ok {
		return n.Obj().Name(), true
	}
	return "", false
}

func privateUses(info *types.Info, own string, public map[symbol]bool, types_ map[string]bool) []string {
	var out []string
	for _, obj := range info.Uses {
		ip := internalPath(obj.Pkg())
		if ip == "" || obj.Pkg().Path() == own || pureHelpers[ip] {
			continue
		}
		if v, ok := obj.(*types.Var); ok && v.IsField() {
			continue
		}
		if f, ok := obj.(*types.Func); ok {
			if recv, isMethod := receiverName(f); isMethod {
				if !types_[ip+"."+recv] {
					out = append(out, ip+"."+recv+"."+f.Name())
				}
				continue
			}
		}
		if obj.Parent() != obj.Pkg().Scope() {
			continue
		}
		if !public[symbol{ip, obj.Name()}] && !types_[ip+"."+obj.Name()] {
			out = append(out, ip+"."+obj.Name())
		}
	}
	return out
}

func TestBuiltInComponentsUseOnlyWhatPkgReExportsExceptTheNamedExceptions(t *testing.T) {
	fset, imp := sourceImporter()
	pk, files, info := typeCheck(t, fset, imp, modulePath+"/pkg", ".")
	public := reexported(pk, files, info)
	typeNames := typeNamesReexported(pk)
	used := map[int]bool{}
	violations := map[string]bool{}
	for _, dir := range builtIns {
		_, _, binfo := typeCheck(t, fset, imp, modulePath+"/internal/"+dir, "../internal/"+dir)
		for _, name := range privateUses(binfo, modulePath+"/internal/"+dir, public, typeNames) {
			if !exempt(dir, name, used) {
				violations[dir+" uses "+name] = true
			}
		}
	}
	var list []string
	for v := range violations {
		list = append(list, v)
	}
	sort.Strings(list)
	if len(list) > 0 {
		t.Errorf("built-ins use internal symbols that pkg does not re-export; use the public equivalent or name an exception with a reason:\n%s", strings.Join(list, "\n"))
	}
	for i, e := range exceptions {
		if !used[i] {
			t.Errorf("the exception %s %s (%s) is no longer needed; delete it", e.builtIn, e.prefix, e.reason)
		}
	}
}
