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
	"strings"

	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/gosrc"
	"github.com/tolelom/daemok/internal/model"
)

// Row는 패키지 하나의 결과다. Pkg는 코드 루트 기준 슬래시 경로이고 루트는 "."다.
type Row struct {
	Pkg     string
	Decls   int
	Matched int
}

// Count는 root 아래 패키지마다 최상위 선언 수와 그중 설계가 만드는 이름의 수를 준다.
// 설계가 만드는 이름은 모든 L3·L4 노드를 gen: true로 보고 생성한 코드에서 얻는다. 모델은 바꾸지 않는다.
// 파싱하지 못한 Go 파일은 세지 않고 함께 돌려준다.
func Count(m *model.Model, root string) ([]Row, []gosrc.ParseError, error) {
	expected, err := designNames(m)
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
		if p != root && (gosrc.SkipDir(d.Name()) || strings.HasPrefix(d.Name(), "_")) {
			return filepath.SkipDir
		}
		files, errs := gosrc.Files(fset, p, parser.SkipObjectResolution)
		bad = append(bad, errs...)
		if len(files) == 0 {
			return nil
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		row := Row{Pkg: rel}
		for _, f := range files {
			for _, name := range gosrc.TopNames(f) {
				row.Decls++
				if expected[rel][name] {
					row.Matched++
				}
			}
		}
		rows = append(rows, row)
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Pkg < rows[j].Pkg })
	return rows, bad, nil
}

// designNames는 패키지 디렉토리마다 설계가 만드는 최상위 이름이다. 등록 파일이 가리키는 핸들러 함수도 넣는다.
func designNames(m *model.Model) (map[string]map[string]bool, error) {
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
	return out, nil
}

// handlerRefs는 등록 파일의 `_ 패키지.종류 = 함수` 줄이 가리키는 손으로 쓴 함수 이름이다.
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
