package infer

import (
	"go/ast"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

// Hint는 손으로 쓴 선언 중 설계의 L3 노드로 올릴 만한 것이다. 초안에는 넣지 않고 gen: false 노드 후보로 알린다.
type Hint struct {
	Kind   string   `json:"kind"` // 에러, enum
	Module string   `json:"module"`
	Type   string   `json:"type,omitempty"` // enum의 Go 타입
	Names  []string `json:"names"`
	Where  string   `json:"where"` // 코드 루트 기준 "경로:줄"
}

// fileHints는 파일 하나에서 에러 변수 묶음과 enum처럼 쓰는 상수 블록을 찾는다.
// types는 같은 패키지에 선언된 타입 이름이다.
func fileHints(fset *token.FileSet, root string, f *ast.File, types map[string]bool) []Hint {
	where := func(p token.Pos) string {
		pos := fset.Position(p)
		rel, err := filepath.Rel(root, pos.Filename)
		if err != nil {
			rel = pos.Filename
		}
		return filepath.ToSlash(rel) + ":" + strconv.Itoa(pos.Line)
	}
	var out []Hint
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		switch g.Tok {
		case token.VAR:
			var h *Hint
			for _, s := range g.Specs {
				sp := s.(*ast.ValueSpec)
				if len(sp.Names) != 1 || len(sp.Values) != 1 || !strings.HasPrefix(sp.Names[0].Name, "Err") || !isErrorCtor(sp.Values[0]) {
					continue
				}
				if h == nil {
					h = &Hint{Kind: "에러", Where: where(sp.Names[0].Pos())}
				}
				h.Names = append(h.Names, sp.Names[0].Name)
			}
			if h != nil {
				out = append(out, *h)
			}
		case token.CONST:
			if !g.Lparen.IsValid() || len(g.Specs) < 2 {
				continue
			}
			first := g.Specs[0].(*ast.ValueSpec)
			typ, ok := first.Type.(*ast.Ident)
			if !ok || !types[typ.Name] {
				continue
			}
			h := Hint{Kind: "enum", Type: typ.Name, Where: where(first.Names[0].Pos())}
			for _, s := range g.Specs {
				sp := s.(*ast.ValueSpec)
				// 같은 타입을 적었거나 iota처럼 타입과 값을 생략한 줄만 값으로 본다.
				t, _ := sp.Type.(*ast.Ident)
				if !(t != nil && t.Name == typ.Name) && !(sp.Type == nil && len(sp.Values) == 0) {
					h.Names = nil
					break
				}
				for _, n := range sp.Names {
					h.Names = append(h.Names, n.Name)
				}
			}
			if len(h.Names) >= 2 {
				out = append(out, h)
			}
		}
	}
	return out
}

func isErrorCtor(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && ((pkg.Name == "errors" && sel.Sel.Name == "New") || (pkg.Name == "fmt" && sel.Sel.Name == "Errorf"))
}

// typeNames는 파일들의 최상위 타입 이름이다.
func typeNames(files []*ast.File) map[string]bool {
	out := map[string]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.TYPE {
				for _, s := range g.Specs {
					out[s.(*ast.TypeSpec).Name.Name] = true
				}
			}
		}
	}
	return out
}
