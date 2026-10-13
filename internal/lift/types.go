package lift

import (
	"fmt"
	"go/ast"
	"strconv"
	"strings"
)

// designTypes는 모듈에 속한 설계 타입·enum의 Go 이름에서 설계 ID로 가는 표다.
func (c *collector) designTypes(module string) map[string]string {
	out := map[string]string{}
	for _, t := range c.m.Types {
		if t.In.Name == module {
			out[t.GoName()] = t.ID
		}
	}
	for _, e := range c.m.Enums {
		if e.In.Name == module {
			out[e.GoType()] = e.ID
		}
	}
	return out
}

// unitOf는 import 경로가 설계 모듈·멤버 패키지면 그 ID를 준다.
func (c *collector) unitOf(importPath string) (string, bool) {
	rel, ok := strings.CutPrefix(importPath, c.m.GoModule+"/")
	if !ok {
		return "", false
	}
	for _, u := range c.units {
		if u.path == rel {
			return u.id, true
		}
	}
	return "", false
}

// structType은 struct를 타입 노드로 옮긴다. 필드를 모두 설계 타입 문자열로 바꿀 수 있어야 한다.
func (c *collector) structType(module string, ts *ast.TypeSpec, f *ast.File, local map[string]string) {
	name := ts.Name.Name
	if iface, ok := ts.Type.(*ast.InterfaceType); ok && ts.TypeParams == nil {
		c.contract(module, ts, iface, f, local)
		return
	}
	st, ok := ts.Type.(*ast.StructType)
	if !ok || ts.TypeParams != nil {
		c.skip(ts.Name.Pos(), name, "struct, interface, string enum이 아니다")
		return
	}
	var fields []string
	for _, fd := range st.Fields.List {
		if len(fd.Names) == 0 {
			c.skip(ts.Name.Pos(), name, "끼워 넣은 필드가 있다")
			return
		}
		typ, reason := c.fieldType(fd.Type, f, local)
		for _, n := range fd.Names {
			if !n.IsExported() {
				continue
			}
			if reason != "" {
				c.skip(ts.Name.Pos(), name, "필드 "+n.Name+": "+reason)
				return
			}
			id, _, ok := snakeID(n.Name)
			if !ok {
				c.skip(ts.Name.Pos(), name, "필드 이름을 snake_case로 바꿀 수 없다: "+n.Name)
				return
			}
			fields = append(fields, id+": "+typ)
		}
	}
	if len(fields) == 0 {
		c.skip(ts.Name.Pos(), name, "내보내는 필드가 없다")
		return
	}
	id, goName, ok := snakeID(name)
	if !ok {
		c.skip(ts.Name.Pos(), name, "snake_case ID로 바꿀 수 없다")
		return
	}
	out := []string{"fields: { " + strings.Join(fields, ", ") + " }"}
	if goName != "" {
		out = append(out, "go: { name: "+goName+" }")
	}
	c.add("types", module, id, ts.Name, out...)
}

// contract는 interface를 call 계약으로 옮긴다. 메서드는 반환 끝이 error여야 한다(생성 계약이 error를 붙인다).
func (c *collector) contract(module string, ts *ast.TypeSpec, iface *ast.InterfaceType, f *ast.File, local map[string]string) {
	name := ts.Name.Name
	var methods []string
	for _, fd := range iface.Methods.List {
		ft, ok := fd.Type.(*ast.FuncType)
		if len(fd.Names) == 0 || !ok {
			c.skip(ts.Name.Pos(), name, "끼워 넣은 interface가 있다")
			return
		}
		for _, n := range fd.Names {
			m, reason := c.method(n.Name, ft, f, local)
			if reason != "" {
				c.skip(ts.Name.Pos(), name, "메서드 "+n.Name+": "+reason)
				return
			}
			methods = append(methods, m)
		}
	}
	if len(methods) == 0 {
		c.skip(ts.Name.Pos(), name, "메서드가 없다")
		return
	}
	id, goName, ok := snakeID(name)
	if !ok {
		c.skip(ts.Name.Pos(), name, "snake_case ID로 바꿀 수 없다")
		return
	}
	fields := []string{"kind: call", "methods: { " + strings.Join(methods, ", ") + " }"}
	if goName != "" {
		fields = append(fields, "go: { name: "+goName+" }")
	}
	c.add("contracts", module, id, ts.Name, fields...)
}

// method는 메서드 하나를 "이름: { params, returns, errors: [] }"로 쓴다. 이름 없는 파라미터는 arg1부터 붙인다.
func (c *collector) method(name string, ft *ast.FuncType, f *ast.File, local map[string]string) (string, string) {
	id, _, ok := snakeID(name)
	if !ok {
		return "", "snake_case 이름으로 바꿀 수 없다"
	}
	var results []ast.Expr
	if ft.Results != nil {
		for _, r := range ft.Results.List {
			for range max(1, len(r.Names)) {
				results = append(results, r.Type)
			}
		}
	}
	if len(results) == 0 || typeName(results[len(results)-1]) != "error" {
		return "", "반환 끝이 error가 아니다"
	}
	var params []string
	arg := 0
	for _, p := range ft.Params.List {
		if _, ok := p.Type.(*ast.Ellipsis); ok {
			return "", "가변 인자는 설계 문법이 없다"
		}
		typ, reason := c.fieldType(p.Type, f, local)
		if reason != "" {
			return "", reason
		}
		names := p.Names
		if len(names) == 0 {
			arg++
			names = []*ast.Ident{{Name: fmt.Sprintf("arg%d", arg)}}
		}
		for _, n := range names {
			pid, _, ok := snakeID(n.Name)
			if !ok || n.Name == "_" {
				arg++
				pid = fmt.Sprintf("arg%d", arg)
			}
			params = append(params, pid+": "+typ)
		}
	}
	var returns []string
	for _, r := range results[:len(results)-1] {
		typ, reason := c.fieldType(r, f, local)
		if reason != "" {
			return "", reason
		}
		returns = append(returns, typ)
	}
	var parts []string
	if len(params) > 0 {
		parts = append(parts, "params: { "+strings.Join(params, ", ")+" }")
	}
	if len(returns) > 0 {
		parts = append(parts, "returns: ["+strings.Join(returns, ", ")+"]")
	}
	parts = append(parts, "errors: []")
	return id + ": { " + strings.Join(parts, ", ") + " }", ""
}

var builtinGo = map[string]string{
	"string": "string", "int": "int", "uint": "uint", "int64": "int64", "uint64": "uint64",
	"bool": "bool", "float64": "float64", "error": "error", "any": "any",
	"time.Duration": "duration", "time.Time": "time",
}

// fieldType은 필드의 Go 타입식을 설계 타입 문자열로 바꾼다. 바꿀 수 없으면 이유를 준다.
// *, [가 들어가면 YAML 평문이 될 수 없어 따옴표로 감싼다.
func (c *collector) fieldType(e ast.Expr, f *ast.File, local map[string]string) (string, string) {
	t, reason := c.typeString(e, f, local)
	if reason == "" && strings.ContainsAny(t, "*[") {
		t = `"` + t + `"`
	}
	return t, reason
}

// typeString은 fieldType의 따옴표 없는 설계 타입 문자열이다.
func (c *collector) typeString(e ast.Expr, f *ast.File, local map[string]string) (string, string) {
	prefix := ""
	for {
		switch x := e.(type) {
		case *ast.StarExpr:
			prefix, e = prefix+"*", x.X
			continue
		case *ast.ArrayType:
			if x.Len != nil {
				return "", "배열 길이가 있는 타입은 설계 문법이 없다"
			}
			if id, ok := x.Elt.(*ast.Ident); ok && id.Name == "byte" {
				e = &ast.Ident{Name: "bytes"}
				break
			}
			prefix, e = prefix+"[]", x.Elt
			continue
		case *ast.ParenExpr:
			e = x.X
			continue
		}
		break
	}
	var base string
	switch x := e.(type) {
	case *ast.Ident:
		switch {
		case x.Name == "bytes":
			base = "bytes"
		case builtinGo[x.Name] != "":
			base = builtinGo[x.Name]
		case local[x.Name] != "":
			base = local[x.Name]
		case x.IsExported():
			base = x.Name
		default:
			return "", "설계 내장 타입이 아니다: " + x.Name
		}
	case *ast.SelectorExpr:
		pkg, ok := x.X.(*ast.Ident)
		if !ok {
			return "", "타입식을 읽을 수 없다"
		}
		qualified := pkg.Name + "." + x.Sel.Name
		if b := builtinGo[qualified]; b != "" && importPath(f, pkg.Name) == "time" {
			base = b
			break
		}
		if unit, ok := c.unitOf(importPath(f, pkg.Name)); ok {
			if id := c.designTypes(unit)[x.Sel.Name]; id != "" {
				base = id
			} else {
				base = unit + "." + x.Sel.Name
			}
			break
		}
		base = qualified
	case *ast.InterfaceType:
		if x.Methods != nil && len(x.Methods.List) > 0 {
			return "", "메서드가 있는 interface 필드는 설계 문법이 없다"
		}
		base = "any"
	case *ast.MapType:
		key, reason := c.typeString(x.Key, f, local)
		if reason != "" {
			return "", reason
		}
		value, reason := c.typeString(x.Value, f, local)
		if reason != "" {
			return "", reason
		}
		base = "map[" + key + "]" + value
	default:
		return "", "설계 문법이 없는 타입식이다"
	}
	return prefix + base, ""
}

// importPath는 파일에서 패키지 이름이 가리키는 import 경로다. 별칭이 없으면 경로의 마지막 요소로 맞춘다.
func importPath(f *ast.File, name string) string {
	for _, im := range f.Imports {
		p, _ := strconv.Unquote(im.Path.Value)
		local := p[strings.LastIndex(p, "/")+1:]
		if im.Name != nil {
			local = im.Name.Name
		}
		if local == name {
			return p
		}
	}
	return ""
}
