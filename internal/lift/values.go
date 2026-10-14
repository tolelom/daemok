package lift

import (
	"go/ast"
	"go/token"
	"regexp"
	"strconv"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// enums는 string 타입과 그 타입의 문자열 리터럴 상수를 enum 노드로 옮긴다.
// 옮겼든 못 옮겼든 다룬 타입과 값 상수 이름을 돌려줘 다른 종류로 다시 알리지 않게 한다.
func (c *collector) enums(module string, files []*ast.File, fresh func(*ast.Ident) bool) (types, values map[string]bool) {
	types, values = map[string]bool{}, map[string]bool{}
	var order []*ast.TypeSpec
	for _, f := range files {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.TYPE {
				continue
			}
			for _, s := range g.Specs {
				ts := s.(*ast.TypeSpec)
				if under, ok := ts.Type.(*ast.Ident); ok && under.Name == "string" && ts.Assign == 0 && ts.TypeParams == nil && fresh(ts.Name) {
					types[ts.Name.Name] = true
					order = append(order, ts)
				}
			}
		}
	}
	type value struct {
		name *ast.Ident
		lit  string
		ok   bool
	}
	consts := map[string][]value{}
	for _, f := range files {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			for _, s := range g.Specs {
				sp := s.(*ast.ValueSpec)
				typ, _ := sp.Type.(*ast.Ident)
				if typ == nil || !types[typ.Name] {
					continue
				}
				for i, n := range sp.Names {
					values[n.Name] = true
					v := value{name: n}
					if len(sp.Values) == len(sp.Names) {
						v.lit, v.ok = stringLit(sp.Values[i])
					}
					consts[typ.Name] = append(consts[typ.Name], v)
				}
			}
		}
	}
	for _, ts := range order {
		name := ts.Name.Name
		vs := consts[name]
		if len(vs) < 2 {
			c.skip(ts.Name.Pos(), name, "string 타입이지만 값 상수가 2개 미만이다")
			continue
		}
		prefix, lits, reason := "", []string{}, ""
		for i, v := range vs {
			if !v.ok || !enumValue.MatchString(v.lit) {
				reason = "값 상수가 단순한 문자열 리터럴이 아니다: " + v.name.Name
				break
			}
			suffix := model.GoName(v.lit)
			p, found := strings.CutSuffix(v.name.Name, suffix)
			if !found || (i > 0 && p != prefix) {
				reason = "상수 이름이 같은 접두사 + 값 이름이 아니다: " + v.name.Name
				break
			}
			prefix = p
			lits = append(lits, v.lit)
		}
		if reason == "" && prefix == "" {
			// go.prefix를 비우면 타입 이름이 접두사가 되므로 설계로 적을 수 없다.
			reason = "접두사 없는 enum 상수는 설계로 적을 수 없다"
		}
		if reason != "" {
			c.skip(ts.Name.Pos(), name, reason)
			continue
		}
		id, goType, ok := snakeID(name)
		if !ok {
			c.skip(ts.Name.Pos(), name, "snake_case ID로 바꿀 수 없다")
			continue
		}
		var goOpts []string
		if goType != "" {
			goOpts = append(goOpts, "type: "+goType)
		}
		if prefix != name {
			goOpts = append(goOpts, "prefix: "+prefix)
		}
		fields := []string{"values: [" + strings.Join(lits, ", ") + "]"}
		if len(goOpts) > 0 {
			fields = append(fields, "go: { "+strings.Join(goOpts, ", ")+" }")
		}
		c.add("enums", module, id, ts.Name, fields...)
	}
	return types, values
}

// enumValue는 YAML 평문으로 그대로 쓸 수 있는 enum 값이다. 설계 검사의 값 문자 규칙보다 좁다.
var enumValue = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// errorVar는 var ErrX = errors.New("문자열")를 에러 노드로 옮긴다.
func (c *collector) errorVar(module string, n *ast.Ident, sp *ast.ValueSpec, i int) {
	msg, ok := errorsNew(sp, i)
	switch {
	case !ok && makesError(sp, i):
		c.skip(n.Pos(), n.Name, "에러는 errors.New(문자열 리터럴)만 옮긴다")
		return
	case !ok:
		c.skip(n.Pos(), n.Name, "에러가 아닌 var는 대응하는 설계 노드가 없다")
		return
	}
	q, ok := quote(msg)
	if !ok {
		c.skip(n.Pos(), n.Name, "메시지에 설계 문자열로 쓸 수 없는 문자가 있다")
		return
	}
	c.add("errors", module, n.Name, n, "message: "+q)
}

func errorsNew(sp *ast.ValueSpec, i int) (string, bool) {
	if len(sp.Values) != len(sp.Names) || sp.Type != nil {
		return "", false
	}
	call, ok := sp.Values[i].(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	pkg, ok := sel.X.(*ast.Ident)
	if !ok || pkg.Name != "errors" || sel.Sel.Name != "New" {
		return "", false
	}
	return stringLit(call.Args[0])
}

// makesError는 var 값이 errors 패키지 함수나 fmt.Errorf 호출인지 본다.
func makesError(sp *ast.ValueSpec, i int) bool {
	if len(sp.Values) != len(sp.Names) {
		return false
	}
	call, ok := sp.Values[i].(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := sel.X.(*ast.Ident)
	return ok && (pkg.Name == "errors" || pkg.Name == "fmt" && sel.Sel.Name == "Errorf")
}

var durationUnits = map[string]string{"Millisecond": "ms", "Second": "s", "Minute": "m", "Hour": "h"}

// constant는 정수·문자열 리터럴과 N * time.Unit 상수를 상수 노드로 옮긴다.
func (c *collector) constant(module string, n *ast.Ident, sp *ast.ValueSpec, i int) {
	typ, value, ok := constValue(sp, i)
	if !ok {
		c.skip(n.Pos(), n.Name, "정수·문자열 리터럴이나 N * time.Unit 상수가 아니다")
		return
	}
	id, goName, ok := snakeID(n.Name)
	if !ok {
		c.skip(n.Pos(), n.Name, "snake_case ID로 바꿀 수 없다")
		return
	}
	fields := []string{"type: " + typ, "value: " + value}
	if goName != "" {
		fields = append(fields, "go: { name: "+goName+" }")
	}
	c.add("consts", module, id, n, fields...)
}

func constValue(sp *ast.ValueSpec, i int) (typ, value string, ok bool) {
	if len(sp.Values) != len(sp.Names) {
		return "", "", false
	}
	declared := typeName(sp.Type)
	expr := sp.Values[i]
	sign := ""
	if u, ok := expr.(*ast.UnaryExpr); ok && u.Op == token.SUB {
		if lit, ok := u.X.(*ast.BasicLit); ok && lit.Kind == token.INT {
			expr, sign = lit, "-"
		}
	}
	switch v := expr.(type) {
	case *ast.BasicLit:
		switch {
		case v.Kind == token.INT && (declared == "" || declared == "int"):
			// Go 리터럴 규칙(0x, 0o, 앞자리 0의 8진수, 밑줄)으로 읽어 10진수로 쓴다.
			n, err := strconv.ParseInt(sign+v.Value, 0, 0)
			if err != nil {
				return "", "", false
			}
			return "int", strconv.FormatInt(n, 10), true
		case v.Kind == token.STRING && sign == "" && (declared == "" || declared == "string"):
			s, ok := stringLit(v)
			if !ok {
				return "", "", false
			}
			q, ok := quote(s)
			return "string", q, ok
		}
	case *ast.BinaryExpr:
		if v.Op != token.MUL || (declared != "" && declared != "time.Duration") {
			return "", "", false
		}
		num, unit := v.X, v.Y
		if _, isLit := num.(*ast.BasicLit); !isLit {
			num, unit = unit, num
		}
		lit, ok := num.(*ast.BasicLit)
		if !ok || lit.Kind != token.INT {
			return "", "", false
		}
		n, err := strconv.ParseInt(lit.Value, 0, 0)
		if err != nil || n < 0 {
			return "", "", false
		}
		if sel, ok := unit.(*ast.SelectorExpr); ok && typeName(sel) != "" {
			if pkg, _ := sel.X.(*ast.Ident); pkg != nil && pkg.Name == "time" && durationUnits[sel.Sel.Name] != "" {
				return "duration", strconv.FormatInt(n, 10) + durationUnits[sel.Sel.Name], true
			}
		}
	}
	return "", "", false
}

// typeName은 타입 식이 이름이나 패키지.이름이면 그 글자를 준다.
func typeName(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.SelectorExpr:
		if x, ok := t.X.(*ast.Ident); ok {
			return x.Name + "." + t.Sel.Name
		}
	}
	return ""
}

func stringLit(e ast.Expr) (string, bool) {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(lit.Value)
	return s, err == nil
}
