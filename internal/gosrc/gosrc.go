package gosrc

import (
	"go/ast"
	"go/build/constraint"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

type ParseError struct {
	Path string
	Err  error
}

func Files(fset *token.FileSet, dir string, mode parser.Mode) ([]*ast.File, []ParseError) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil
	}
	var files []*ast.File
	var errs []ParseError
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		p := filepath.Join(dir, name)
		f, err := parser.ParseFile(fset, p, nil, mode|parser.ParseComments)
		if err != nil {
			errs = append(errs, ParseError{p, err})
			continue
		}
		if ignored(f) {
			continue
		}
		files = append(files, f)
	}
	return files, errs
}

func ImportsOnly(fset *token.FileSet, path string) (*ast.File, error) {
	f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly|parser.ParseComments)
	if err != nil || ignored(f) {
		return nil, err
	}
	return f, nil
}

func TopNames(f *ast.File) []string {
	var out []string
	add := func(n string) {
		if n != "_" && n != "init" && n != "main" {
			out = append(out, n)
		}
	}
	for _, d := range f.Decls {
		switch x := d.(type) {
		case *ast.FuncDecl:
			if x.Recv == nil {
				add(x.Name.Name)
			}
		case *ast.GenDecl:
			for _, s := range x.Specs {
				switch sp := s.(type) {
				case *ast.TypeSpec:
					add(sp.Name.Name)
				case *ast.ValueSpec:
					for _, n := range sp.Names {
						add(n.Name)
					}
				}
			}
		}
	}
	return out
}

func SkipDir(name string) bool {
	return strings.HasPrefix(name, ".") || name == "vendor" || name == "testdata" || name == "node_modules"
}

func SkipPackageDir(name string) bool {
	return SkipDir(name) || strings.HasPrefix(name, "_")
}

func ignored(f *ast.File) bool {
	for _, cg := range f.Comments {
		if cg.Pos() >= f.Package {
			break
		}
		for _, c := range cg.List {
			if !constraint.IsGoBuild(c.Text) {
				continue
			}
			expr, err := constraint.Parse(c.Text)
			if err != nil {
				return false
			}
			return !satisfiable(expr)
		}
	}
	return false
}

func satisfiable(expr constraint.Expr) bool {
	var tags []string
	seen := map[string]bool{}
	var walk func(e constraint.Expr)
	walk = func(e constraint.Expr) {
		switch x := e.(type) {
		case *constraint.AndExpr:
			walk(x.X)
			walk(x.Y)
		case *constraint.OrExpr:
			walk(x.X)
			walk(x.Y)
		case *constraint.NotExpr:
			walk(x.X)
		case *constraint.TagExpr:
			if x.Tag != "ignore" && !seen[x.Tag] {
				seen[x.Tag] = true
				tags = append(tags, x.Tag)
			}
		}
	}
	walk(expr)
	if len(tags) > 12 {
		return true
	}
	for mask := 0; mask < 1<<len(tags); mask++ {
		ok := expr.Eval(func(tag string) bool {
			for i, t := range tags {
				if t == tag {
					return mask&(1<<i) != 0
				}
			}
			return false
		})
		if ok {
			return true
		}
	}
	return false
}
