package codescan

import (
	"go/ast"
	"go/token"
	"strconv"
	"strings"

	"github.com/tolelom/daemok/internal/yaml"
)

// adHocCall은 즉석 errors.New 호출 하나다. msg는 인자가 문자열 리터럴일 때만 있다.
type adHocCall struct {
	pos    yaml.Pos
	module string
	msg    string
	lit    bool
}

// adHocErrors는 쓸 수 있는 설계 에러가 있는 모듈의 함수 본문에서 errors.New를 부르면 정보로 알린다.
// 상황마다 메시지가 다른 정당한 경우가 섞여 있어 경고(--strict 실패)로 두지 않는다.
// 패키지 수준 var 선언(설계 에러를 손으로 둔 gen: false 선언)과 fmt.Errorf의 감싸기는 보지 않는다.
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

// reportAdHoc은 즉석 errors.New를 메시지와 함께 알린다. 쓸 수 있는 설계 에러와 메시지가 같으면 그 에러를,
// 같은 메시지가 여러 곳이면 그 수를 붙인다. 앞은 바로 바꿔 쓸 후보이고 뒤는 sentinel로 둘 후보다.
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

// enumLiterals는 쓸 수 있는 설계 enum의 값과 같은 문자열 리터럴을 함수 본문의 비교(==, !=),
// switch case, SQL 메서드의 자리표시자 값(placeholderArgs)에서 쓰면 정보로 알린다. enum 상수를 쓰면 값이 바뀔 때 컴파일러가 잡는다.
// 로그 키나 외부 프로토콜 문자열처럼 다른 자리의 같은 문자열은 enum과 상관없을 때가 많아 보지 않는다.
func (s *scanner) enumLiterals(id string, f *ast.File) {
	consts := map[string]string{} // 값 → 상수 이름
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

// sqlMethods는 쿼리 문자열과 자리표시자 값을 함께 받는 GORM·database/sql 메서드다.
var sqlMethods = map[string]bool{
	"Where": true, "Or": true, "Not": true, "Having": true, "Joins": true, "Select": true, "Raw": true, "Exec": true,
	"Find": true, "First": true, "Take": true, "Last": true, "Delete": true,
	"Query": true, "QueryRow": true, "QueryContext": true, "QueryRowContext": true, "ExecContext": true,
}

// placeholderArgs는 SQL 메서드 호출에서 ?가 든 문자열 리터럴 인자 뒤의 인자다.
// db.Where("status = ?", "published")의 값 자리다. IN ?에 넘긴 슬라이스 리터럴은 원소를 준다.
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
