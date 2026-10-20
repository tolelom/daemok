// Package coverage는 패키지마다 최상위 선언 중 설계 노드에 대응하는 것의 수를 센다.
// 이 비율은 설계 형식을 넓힐지 정하는 근거다.
package coverage

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"path/filepath"
	"sort"

	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/gosrc"
	"github.com/tolelom/daemok/internal/model"
)

type Row struct {
	Package      string
	Decls        int
	Matched      int
	Funcs        int
	FuncsMatched int
}

func Count(m *model.Model, root string) ([]Row, []gosrc.ParseError, error) {
	expected, err := DesignNames(m)
	if err != nil {
		return nil, nil, err
	}
	var rows []Row
	var bad []gosrc.ParseError
	fset := token.NewFileSet()
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if p != root && gosrc.SkipPackageDir(d.Name()) {
			return filepath.SkipDir
		}
		files, errs := gosrc.Files(fset, p, parser.SkipObjectResolution)
		bad = append(bad, errs...)
		if len(files) == 0 {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		row := Row{Package: rel}
		for _, f := range files {
			funcs := map[string]bool{}
			for _, d := range f.Decls {
				if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil {
					funcs[fd.Name.Name] = true
				}
			}
			for _, name := range gosrc.TopNames(f) {
				row.Decls++
				if funcs[name] {
					row.Funcs++
				}
				if expected[rel][name] {
					row.Matched++
					if funcs[name] {
						row.FuncsMatched++
					}
				}
			}
		}
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Package < rows[j].Package })
	return rows, bad, nil
}

func DesignNames(m *model.Model) (map[string]map[string]bool, error) {
	var gens []*bool
	for _, x := range m.Enums {
		gens = append(gens, &x.Gen)
	}
	for _, x := range m.Consts {
		gens = append(gens, &x.Gen)
	}
	for _, x := range m.Errors {
		gens = append(gens, &x.Gen)
	}
	for _, x := range m.Types {
		gens = append(gens, &x.Gen)
	}
	for _, x := range m.Contracts {
		gens = append(gens, &x.Gen)
	}
	for _, x := range m.HandlerKinds {
		gens = append(gens, &x.Gen)
	}
	for _, x := range m.EventKinds {
		gens = append(gens, &x.Gen)
	}
	for _, x := range m.Funcs {
		gens = append(gens, &x.Gen)
	}
	for _, x := range m.Flows {
		gens = append(gens, &x.Gen)
	}
	saved := make([]bool, len(gens))
	for i, g := range gens {
		saved[i], *g = *g, true
	}
	files, err := golang.Generate(m)
	for i, g := range gens {
		*g = saved[i]
	}
	if err != nil {
		return nil, fmt.Errorf("생성할 수 없다: %w", err)
	}
	out := map[string]map[string]bool{}
	fset := token.NewFileSet()
	for _, gf := range files {
		f, err := parser.ParseFile(fset, gf.Path, gf.Content, parser.SkipObjectResolution)
		if err != nil {
			return nil, fmt.Errorf("생성 코드를 읽을 수 없다: %s: %w", gf.Path, err)
		}
		dir := path.Dir(gf.Path)
		if out[dir] == nil {
			out[dir] = map[string]bool{}
		}
		for _, name := range gosrc.TopNames(f) {
			out[dir][name] = true
		}
		for _, name := range handlerRefs(f) {
			out[dir][name] = true
		}
	}
	for _, fn := range m.Funcs {
		if o, ok := m.Owner(fn.In.Name); ok && fn.Free() {
			if out[o.Path] == nil {
				out[o.Path] = map[string]bool{}
			}
			out[o.Path][fn.GoName()] = true
		}
	}
	return out, nil
}

func handlerRefs(f *ast.File) []string {
	var out []string
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok || g.Tok != token.VAR {
			continue
		}
		for _, s := range g.Specs {
			sp, ok := s.(*ast.ValueSpec)
			if !ok || len(sp.Names) != 1 || sp.Names[0].Name != "_" || len(sp.Values) != 1 {
				continue
			}
			if id, ok := sp.Values[0].(*ast.Ident); ok {
				out = append(out, id.Name)
			}
		}
	}
	return out
}
