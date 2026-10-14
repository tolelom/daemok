package lift

import (
	"go/ast"
	"slices"
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// pendingFunc는 모든 모듈의 에러를 모은 뒤 판단할 함수다. 다른 모듈의 초안 에러를 가리킬 수 있다.
type pendingFunc struct {
	module string
	decl   *ast.FuncDecl
	file   *ast.File
}

// errorRef는 모듈과 Go 이름으로 가리킨 에러다.
type errorRef struct{ module, name string }

// funcs는 반환문에서 설계 에러나 초안 에러를 가리키는 함수를 일반 함수 노드로 옮기고, 나머지는 수만 센다.
// 에러는 모듈과 Go 이름으로 들고 있다가 dedupe 뒤 resolveFuncErrors가 ID로 푼다.
// 에러 ID는 모듈을 붙이지 않아, 미리 풀면 빠진 초안 에러 대신 같은 ID의 다른 모듈 에러를 가리킨다.
func (c *collector) funcs() {
	known := map[errorRef]bool{}
	for _, e := range c.m.Errors {
		known[errorRef{e.In.Name, e.GoName()}] = true
	}
	for _, n := range c.nodes {
		if n.Kind == "errors" {
			known[errorRef{n.Module, n.Name}] = true
		}
	}
	for _, p := range c.pending {
		var refs []errorRef
		seen := map[errorRef]bool{}
		for _, r := range c.returnedRefs(p) {
			if known[r] && !seen[r] {
				seen[r] = true
				refs = append(refs, r)
			}
		}
		if len(refs) == 0 {
			c.res.Funcs++
			continue
		}
		id, goName, ok := snakeID(p.decl.Name.Name)
		if !ok {
			c.skip(p.decl.Name.Pos(), p.decl.Name.Name, "snake_case ID로 바꿀 수 없다")
			continue
		}
		var fields []string
		if goName != "" {
			fields = append(fields, "go: { name: "+goName+" }")
		}
		c.add("funcs", p.module, id, p.decl.Name, fields...)
		c.nodes[len(c.nodes)-1].errRefs = refs
	}
}

// resolveFuncErrors는 funcs 노드의 에러를 설계 에러와 dedupe에서 남은 초안 에러의 ID로 푼다.
// 설계의 다른 에러와 ID가 겹치는 초안 에러는 검증에서 빠지므로 풀지 않는다.
// 반환하는 에러를 하나라도 풀지 못하면 에러 계약이 모자라므로 함수를 뺀다.
func resolveFuncErrors(nodes []Node, m *model.Model, res *Result) []Node {
	ids := map[errorRef]string{}
	designIDs := map[string]bool{}
	for _, e := range m.Errors {
		ids[errorRef{e.In.Name, e.GoName()}] = e.ID
		designIDs[e.ID] = true
	}
	for _, n := range nodes {
		if n.Kind == "errors" && !designIDs[n.ID] {
			ids[errorRef{n.Module, n.Name}] = n.ID
		}
	}
	var out []Node
	for _, n := range nodes {
		if n.Kind != "funcs" {
			out = append(out, n)
			continue
		}
		var errs, missing []string
		for _, r := range n.errRefs {
			if id, ok := ids[r]; ok {
				errs = append(errs, id)
			} else {
				missing = append(missing, r.module+"."+r.name)
			}
		}
		if len(missing) > 0 {
			res.Skipped = append(res.Skipped, Skip{n.Where, n.Name, "반환하는 에러가 초안에 남지 않았다: " + strings.Join(missing, ", ")})
			continue
		}
		sort.Strings(errs)
		errs = slices.Compact(errs)
		at := slices.IndexFunc(n.fields, func(f string) bool { return strings.HasPrefix(f, "go: ") })
		if at < 0 {
			at = len(n.fields)
		}
		n.fields = slices.Insert(n.fields, at, "errors: ["+strings.Join(errs, ", ")+"]")
		n.render()
		out = append(out, n)
	}
	return out
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
