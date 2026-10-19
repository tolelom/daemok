package codescan

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/tolelom/daemok/internal/yaml"
)

type adHocCall struct {
	pos    yaml.Pos
	module string
	msg    string
	lit    bool
}

func (s *scanner) adHocErrors(id string, f *ast.File) {
	visible := false
	for _, e := range s.m.Errors {
		if e.In.Name == id || s.m.CanDepend(id, e.In.Name) {
			visible = true
			break
		}
	}
	if !visible {
		return
	}
	name := ""
	for _, im := range f.Imports {
		if p, _ := strconv.Unquote(im.Path.Value); p == "errors" {
			name = "errors"
			if im.Name != nil {
				name = im.Name.Name
			}
		}
	}
	if name == "" || name == "_" || name == "." {
		return
	}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "New" {
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == name {
					c := adHocCall{pos: s.pos(call.Pos()), module: id}
					if len(call.Args) == 1 {
						if b, ok := call.Args[0].(*ast.BasicLit); ok && b.Kind == token.STRING {
							c.msg, _ = strconv.Unquote(b.Value)
							c.lit = true
						}
					}
					s.adHoc = append(s.adHoc, c)
				}
			}
			return true
		})
	}
}

func (s *scanner) reportAdHoc() {
	same := map[string]int{}
	for _, c := range s.adHoc {
		if c.lit {
			same[c.msg]++
		}
	}
	for _, c := range s.adHoc {
		if !c.lit {
			s.diags.Infof(c.pos, "설계 에러가 있는데 즉석 errors.New를 쓴다: 모듈 %s", c.module)
			continue
		}
		note := ""
		for _, e := range s.m.Errors {
			if e.Message == c.msg && (e.In.Name == c.module || s.m.CanDepend(c.module, e.In.Name)) {
				note = " (설계 에러 " + e.GoName() + "와 메시지가 같다)"
				break
			}
		}
		if note == "" && same[c.msg] > 1 {
			note = " (같은 메시지 " + strconv.Itoa(same[c.msg]) + "곳)"
		}
		s.diags.Infof(c.pos, "설계 에러가 있는데 즉석 errors.New를 쓴다: 모듈 %s, %s%s", c.module, strconv.Quote(c.msg), note)
	}
}

func (s *scanner) enumLiterals(id string, f *ast.File) {
	consts := map[string]string{}
	for _, en := range s.m.Enums {
		if en.In.Name != id && !s.m.CanDepend(id, en.In.Name) {
			continue
		}
		for _, v := range en.Values {
			if _, dup := consts[v.Name]; !dup {
				consts[v.Name] = en.ConstName(v.Name)
			}
		}
	}
	if len(consts) == 0 {
		return
	}
	for _, d := range f.Decls {
		fn, ok := d.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		var compared []ast.Expr
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BinaryExpr:
				if x.Op == token.EQL || x.Op == token.NEQ {
					compared = append(compared, x.X, x.Y)
				}
			case *ast.CaseClause:
				compared = append(compared, x.List...)
			case *ast.CallExpr:
				compared = append(compared, placeholderArgs(x)...)
			}
			return true
		})
		for _, e := range compared {
			lit, ok := ast.Unparen(e).(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				continue
			}
			v, err := strconv.Unquote(lit.Value)
			if c, hit := consts[v]; err == nil && hit {
				s.diags.Infof(s.pos(lit.Pos()), "enum 상수 대신 문자열 리터럴을 쓴다: 모듈 %s, %q (%s)", id, v, c)
			}
		}
	}
}

var sqlMethods = map[string]bool{
	"Where": true, "Or": true, "Not": true, "Having": true, "Joins": true, "Select": true, "Raw": true, "Exec": true,
	"Find": true, "First": true, "Take": true, "Last": true, "Delete": true,
	"Query": true, "QueryRow": true, "QueryContext": true, "QueryRowContext": true, "ExecContext": true,
}

func placeholderArgs(call *ast.CallExpr) []ast.Expr {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || !sqlMethods[sel.Sel.Name] {
		return nil
	}
	for i, a := range call.Args {
		lit, ok := ast.Unparen(a).(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			continue
		}
		q, err := strconv.Unquote(lit.Value)
		if err != nil || !strings.Contains(q, "?") {
			continue
		}
		var out []ast.Expr
		for _, v := range call.Args[i+1:] {
			if cl, ok := ast.Unparen(v).(*ast.CompositeLit); ok {
				out = append(out, cl.Elts...)
			} else {
				out = append(out, v)
			}
		}
		return out
	}
	return nil
}
