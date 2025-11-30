package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := flag.String("root", ".", "root directory to process")
	flag.Parse()

	var files []string
	filepath.WalkDir(*root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "vendor" || name == ".gocache" || name == "tools" {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})

	fset := token.NewFileSet()
	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		file, err := parser.ParseFile(fset, path, src, parser.ParseComments)
		if err != nil {
			continue
		}

		// Strip all comments (doc and inline) from the file AST so the printer omits them.
		ast.Inspect(file, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.File:
				v.Doc = nil
				v.Comments = nil
			case *ast.GenDecl:
				v.Doc = nil
			case *ast.FuncDecl:
				v.Doc = nil
			case *ast.ImportSpec:
				v.Doc, v.Comment = nil, nil
			case *ast.ValueSpec:
				v.Doc, v.Comment = nil, nil
			case *ast.TypeSpec:
				v.Doc, v.Comment = nil, nil
			case *ast.Field:
				v.Doc, v.Comment = nil, nil
			}
			return true
		})

		var out strings.Builder
		cfg := &printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
		if err := cfg.Fprint(&out, fset, file); err != nil {
			fmt.Fprintf(os.Stderr, "print error for %s: %v\n", path, err)
			continue
		}
		_ = os.WriteFile(path, []byte(out.String()), 0644)
	}
}
