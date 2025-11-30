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

	hasDirective := func(cg *ast.CommentGroup) bool {
		if cg == nil {
			return false
		}
		for _, c := range cg.List {
			text := strings.TrimSpace(c.Text)
			if strings.HasPrefix(text, "//go:") || strings.HasPrefix(text, "/*go:") || strings.HasPrefix(text, "// +build") {
				return true
			}
		}
		return false
	}

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

		// Strip all comments except Go directives (//go:*, // +build) to keep build/embed/generate hints.
		ast.Inspect(file, func(n ast.Node) bool {
			switch v := n.(type) {
			case *ast.File:
				if !hasDirective(v.Doc) {
					v.Doc = nil
				}
			case *ast.GenDecl:
				if !hasDirective(v.Doc) {
					v.Doc = nil
				}
			case *ast.FuncDecl:
				if !hasDirective(v.Doc) {
					v.Doc = nil
				}
			case *ast.ImportSpec:
				if !hasDirective(v.Doc) {
					v.Doc = nil
				}
				if !hasDirective(v.Comment) {
					v.Comment = nil
				}
			case *ast.ValueSpec:
				if !hasDirective(v.Doc) {
					v.Doc = nil
				}
				if !hasDirective(v.Comment) {
					v.Comment = nil
				}
			case *ast.TypeSpec:
				if !hasDirective(v.Doc) {
					v.Doc = nil
				}
				if !hasDirective(v.Comment) {
					v.Comment = nil
				}
			case *ast.Field:
				if !hasDirective(v.Doc) {
					v.Doc = nil
				}
				if !hasDirective(v.Comment) {
					v.Comment = nil
				}
			}
			return true
		})

		// Keep only directive comment groups; drop everything else.
		if len(file.Comments) > 0 {
			var kept []*ast.CommentGroup
			for _, cg := range file.Comments {
				if hasDirective(cg) {
					kept = append(kept, cg)
				}
			}
			file.Comments = kept
		}

		var out strings.Builder
		cfg := &printer.Config{Mode: printer.UseSpaces | printer.TabIndent, Tabwidth: 8}
		if err := cfg.Fprint(&out, fset, file); err != nil {
			fmt.Fprintf(os.Stderr, "print error for %s: %v\n", path, err)
			continue
		}
		_ = os.WriteFile(path, []byte(out.String()), 0644)
	}
}
