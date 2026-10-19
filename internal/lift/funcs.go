package lift

import (
	"go/ast"
	"slices"
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

type pendingFunc struct {
	module string
	decl   *ast.FuncDecl
	file   *ast.File
}

type errorRef struct{ module, name string }

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
