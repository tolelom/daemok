package lift

import (
	"go/ast"
	"sort"
	"strings"
)

// pendingFunc는 모든 모듈의 에러를 모은 뒤 판단할 함수다. 다른 모듈의 초안 에러를 가리킬 수 있다.
type pendingFunc struct {
	module string
	decl   *ast.FuncDecl
	file   *ast.File
}

// errorRef는 모듈과 Go 이름으로 가리킨 에러다.
type errorRef struct{ module, name string }

// funcs는 반환문에서 설계 에러를 가리키는 함수를 일반 함수 노드로 옮기고, 나머지는 수만 센다.
func (c *collector) funcs() {
	ids := map[errorRef]string{}
	for _, e := range c.m.Errors {
		ids[errorRef{e.In.Name, e.GoName()}] = e.ID
	}
	for _, n := range c.nodes {
		if n.Kind == "errors" {
			ids[errorRef{n.Module, n.Name}] = n.ID
		}
	}
	for _, p := range c.pending {
		var errs []string
		seen := map[string]bool{}
		for _, r := range c.returnedRefs(p) {
			if id, ok := ids[r]; ok && !seen[id] {
				seen[id] = true
				errs = append(errs, id)
			}
		}
		if len(errs) == 0 {
			c.res.Funcs++
			continue
		}
		id, goName, ok := snakeID(p.decl.Name.Name)
		if !ok {
			c.skip(p.decl.Name.Pos(), p.decl.Name.Name, "snake_case ID로 바꿀 수 없다")
			continue
		}
		sort.Strings(errs)
		fields := []string{"errors: [" + strings.Join(errs, ", ") + "]"}
		if goName != "" {
			fields = append(fields, "go: { name: "+goName+" }")
		}
		c.add("funcs", p.module, id, p.decl.Name, fields...)
	}
}

// returnedRefs는 함수 본문의 반환식에 나오는 이름이다. 같은 패키지 이름과 설계 모듈 패키지를 한정한 이름을 준다.
// 클로저 안의 return은 그 클로저의 반환이라 보지 않는다.
func (c *collector) returnedRefs(p pendingFunc) []errorRef {
	var out []errorRef
	ast.Inspect(p.decl.Body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.ReturnStmt:
			for _, e := range x.Results {
				ast.Inspect(e, func(n ast.Node) bool {
					switch y := n.(type) {
					case *ast.FuncLit:
						return false
					case *ast.SelectorExpr:
						if pkg, ok := y.X.(*ast.Ident); ok {
							if unit, ok := c.unitOf(importPath(p.file, pkg.Name)); ok {
								out = append(out, errorRef{unit, y.Sel.Name})
							}
							return false
						}
					case *ast.Ident:
						out = append(out, errorRef{p.module, y.Name})
					}
					return true
				})
			}
		}
		return true
	})
	return out
}
