package pkg

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func isHookCall(call *ast.CallExpr) bool {
	var name string
	switch f := call.Fun.(type) {
	case *ast.SelectorExpr:
		name = f.Sel.Name
	case *ast.Ident:
		name = f.Name
	case *ast.IndexExpr:
		return isHookCall(&ast.CallExpr{Fun: f.X})
	case *ast.IndexListExpr:
		return isHookCall(&ast.CallExpr{Fun: f.X})
	}
	return strings.HasPrefix(name, "Use") && len(name) > 3 && name[3] >= 'A' && name[3] <= 'Z' || name == "Provide"
}

func conditionalHooks(body *ast.BlockStmt, fset *token.FileSet) []string {
	var found []string
	var walk func(n ast.Node, conditional bool)
	walk = func(n ast.Node, conditional bool) {
		ast.Inspect(n, func(c ast.Node) bool {
			switch x := c.(type) {
			case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.SwitchStmt, *ast.TypeSwitchStmt, *ast.SelectStmt:
				if c != n {
					walk(c, true)
					return false
				}
			case *ast.FuncLit:
				if c != n {
					walk(x.Body, true)
					return false
				}
			case *ast.CallExpr:
				if conditional && isHookCall(x) {
					found = append(found, fset.Position(x.Pos()).String())
				}
			}
			return true
		})
	}
	walk(body, false)
	return found
}

func TestBuiltInComponentsCallHooksOnlyAtTheTopOfTheirBody(t *testing.T) {
	var checked int
	for _, dir := range []string{"router", "errors", "head", "headers", "portal"} {
		fset := token.NewFileSet()
		pkgs, err := parser.ParseDir(fset, filepath.Join("..", "internal", dir), func(fi fs.FileInfo) bool { return !strings.HasSuffix(fi.Name(), "_test.go") }, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, p := range pkgs {
			for _, f := range p.Files {
				ast.Inspect(f, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok || len(call.Args) != 1 {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || (sel.Sel.Name != "PropsComponent" && sel.Sel.Name != "Component") {
						return true
					}
					lit, ok := call.Args[0].(*ast.FuncLit)
					if !ok {
						return true
					}
					checked++
					for _, where := range conditionalHooks(lit.Body, fset) {
						t.Errorf("a hook is called conditionally or inside a closure at %s", where)
					}
					return true
				})
			}
		}
	}
	if checked < 8 {
		t.Fatalf("only %d built-in components were scanned", checked)
	}
}
