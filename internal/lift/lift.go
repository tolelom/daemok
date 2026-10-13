// Package lift는 설계 모듈 패키지에 손으로 쓴 선언 중 설계에 없는 것을 gen: false L3 노드 초안으로 만든다.
// 초안은 기존 설계와 합쳐 검사했을 때 새 오류·경고를 내지 않는 노드만 남긴다.
package lift

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/coverage"
	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/gosrc"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
)

// Node는 초안 노드 하나다. Line은 "ID: { ... }" 한 줄이다.
type Node struct {
	Kind   string `json:"kind"` // errors, consts, enums, types
	Module string `json:"module"`
	ID     string `json:"id"`
	Name   string `json:"name"` // Go 이름
	Line   string `json:"line"`
	Where  string `json:"where"` // 코드 루트 기준 "경로:줄"
}

// Skip은 옮기지 않은 선언과 그 이유다.
type Skip struct {
	Where  string `json:"where"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Result는 초안 YAML과 노드, 옮기지 않은 선언이다. Funcs는 설계에 없는 exported 함수의 수다.
type Result struct {
	YAML    []byte `json:"-"`
	Nodes   []Node `json:"nodes"`
	Skipped []Skip `json:"skipped"`
	Funcs   int    `json:"funcs"`
}

// ErrDesign은 설계를 읽지 못했거나 설계 검사 오류가 있다는 뜻이다.
var ErrDesign = errors.New("설계에 오류가 있어 초안을 만들 수 없다")

// draftName은 검증할 때 초안에 붙이는 설계 파일 이름이다.
const draftName = "daemok-l3-draft.yaml"

var kindOrder = map[string]int{"errors": 0, "consts": 1, "enums": 2, "types": 3}

// Draft는 designDir의 설계와 root의 코드로 초안을 만든다.
func Draft(designDir, root string) (*Result, error) {
	sources, err := readSources(designDir)
	if err != nil {
		return nil, err
	}
	m, diags := model.LoadSources(sources)
	if !diags.HasErrors() {
		diags = append(diags, check.Run(m)...)
	}
	if diags.HasErrors() {
		return nil, ErrDesign
	}
	expected, err := coverage.DesignNames(m)
	if err != nil {
		return nil, err
	}
	c := &collector{m: m, root: root, fset: token.NewFileSet(), res: &Result{}, units: units(m)}
	for _, u := range c.units {
		c.unit(u.id, u.path, expected[u.path])
	}
	nodes, err := validate(sources, designDir, root, c.nodes, c.res)
	if err != nil {
		return nil, err
	}
	c.res.Nodes = nodes
	c.res.YAML = render(nodes)
	sort.Slice(c.res.Skipped, func(i, j int) bool { return c.res.Skipped[i].Where < c.res.Skipped[j].Where })
	return c.res, nil
}

func readSources(designDir string) (map[string][]byte, error) {
	paths, _ := filepath.Glob(filepath.Join(designDir, "*.yaml"))
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: 설계 파일이 없다: %s", ErrDesign, designDir)
	}
	out := map[string][]byte{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		out[p] = b
	}
	return out, nil
}

type unit struct{ id, path string }

// units는 모듈과 그룹 멤버를 ID 순으로 준다.
func units(m *model.Model) []unit {
	var out []unit
	for _, mod := range m.Modules {
		out = append(out, unit{mod.ID, mod.Path})
	}
	for _, g := range m.Groups {
		for _, mem := range g.Members {
			out = append(out, unit{mem.ID, mem.Path()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

type collector struct {
	docs  map[*ast.Ident]*ast.CommentGroup // 선언 이름에서 그 선언의 문서 주석으로
	m     *model.Model
	units []unit
	root  string
	fset  *token.FileSet
	nodes []Node
	res   *Result
}

func (c *collector) where(p token.Pos) string {
	pos := c.fset.Position(p)
	rel, err := filepath.Rel(c.root, pos.Filename)
	if err != nil {
		rel = pos.Filename
	}
	return filepath.ToSlash(rel) + ":" + strconv.Itoa(pos.Line)
}

func (c *collector) skip(p token.Pos, name, reason string) {
	c.res.Skipped = append(c.res.Skipped, Skip{c.where(p), name, reason})
}

func (c *collector) add(kind, module, id string, name *ast.Ident, fields ...string) {
	head := []string{"in: " + module, "gen: false"}
	if q, ok := quote(summary(c.docs[name])); ok && q != `""` {
		head = append(head, "summary: "+q)
	}
	line := id + ": { " + strings.Join(append(head, fields...), ", ") + " }"
	c.nodes = append(c.nodes, Node{Kind: kind, Module: module, ID: id, Name: name.Name, Line: line, Where: c.where(name.Pos())})
}

// unit은 모듈 하나의 패키지에서 설계에 없는 exported 선언을 옮긴다. known은 설계가 만드는 이름이다.
func (c *collector) unit(module, dir string, known map[string]bool) {
	files, _ := gosrc.Files(c.fset, filepath.Join(c.root, filepath.FromSlash(dir)), parser.SkipObjectResolution)
	fresh := func(n *ast.Ident) bool { return n.IsExported() && !known[n.Name] }
	var hand []*ast.File
	for _, f := range files {
		if !generated(f) {
			hand = append(hand, f)
		}
	}
	c.docs = docs(hand)
	enumTypes, enumValues := c.enums(module, hand, fresh)
	// local은 이 패키지에서 설계 ID로 가리킬 수 있는 타입 이름이다. 설계에 있는 타입·enum과 이번에 함께 옮기는 것.
	local := c.designTypes(module)
	// 내장 타입 이름이 되는 ID는 설계 검사가 막으므로 가리키지 않는다.
	refable := func(name string) {
		if id, _, ok := snakeID(name); ok && model.Builtins[id] == "" {
			local[name] = id
		}
	}
	for name := range enumTypes {
		refable(name)
	}
	for _, f := range hand {
		for _, d := range f.Decls {
			if g, ok := d.(*ast.GenDecl); ok && g.Tok == token.TYPE {
				for _, sp := range g.Specs {
					ts := sp.(*ast.TypeSpec)
					if _, isStruct := ts.Type.(*ast.StructType); isStruct && fresh(ts.Name) {
						refable(ts.Name.Name)
					}
				}
			}
		}
	}
	for _, f := range hand {
		for _, d := range f.Decls {
			switch d := d.(type) {
			case *ast.FuncDecl:
				if d.Recv == nil && fresh(d.Name) {
					c.res.Funcs++
				}
			case *ast.GenDecl:
				for _, s := range d.Specs {
					switch sp := s.(type) {
					case *ast.TypeSpec:
						if fresh(sp.Name) && !enumTypes[sp.Name.Name] {
							c.structType(module, sp, f, local)
						}
					case *ast.ValueSpec:
						for i, n := range sp.Names {
							if !fresh(n) || enumValues[n.Name] {
								continue
							}
							if d.Tok == token.VAR {
								c.errorVar(module, n, sp, i)
							} else {
								c.constant(module, n, sp, i)
							}
						}
					}
				}
			}
		}
	}
}

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
	st, ok := ts.Type.(*ast.StructType)
	if !ok || ts.TypeParams != nil {
		c.skip(ts.Name.Pos(), name, "struct나 string enum이 아니다")
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

var builtinGo = map[string]string{
	"string": "string", "int": "int", "uint": "uint", "int64": "int64", "uint64": "uint64",
	"bool": "bool", "float64": "float64", "error": "error", "any": "any",
	"time.Duration": "duration", "time.Time": "time",
}

// fieldType은 필드의 Go 타입식을 설계 타입 문자열로 바꾼다. 바꿀 수 없으면 이유를 준다.
// *, []가 붙으면 YAML 평문이 될 수 없어 따옴표로 감싼다.
func (c *collector) fieldType(e ast.Expr, f *ast.File, local map[string]string) (string, string) {
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
		return "", "map 필드는 설계 문법이 없다"
	default:
		return "", "설계 문법이 없는 타입식이다"
	}
	if prefix == "" {
		return base, ""
	}
	return `"` + prefix + base + `"`, ""
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

// docs는 선언 이름마다 문서 주석이다. 블록 안 선언은 자기 주석, 괄호 없는 단독 선언은 선언 주석이다.
func docs(files []*ast.File) map[*ast.Ident]*ast.CommentGroup {
	out := map[*ast.Ident]*ast.CommentGroup{}
	for _, f := range files {
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok {
				continue
			}
			for _, sp := range g.Specs {
				switch sp := sp.(type) {
				case *ast.TypeSpec:
					out[sp.Name] = specDoc(g, sp.Doc)
				case *ast.ValueSpec:
					for _, n := range sp.Names {
						out[n] = specDoc(g, sp.Doc)
					}
				}
			}
		}
	}
	return out
}

func specDoc(g *ast.GenDecl, doc *ast.CommentGroup) *ast.CommentGroup {
	if doc == nil && !g.Lparen.IsValid() {
		return g.Doc
	}
	return doc
}

// summary는 문서 주석 첫 단락의 첫 문장이다. 160자가 넘으면 요약으로 보지 않는다.
func summary(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	para, _, _ := strings.Cut(doc.Text(), "\n\n")
	text := strings.Join(strings.Fields(para), " ")
	if i := strings.Index(text, ". "); i >= 0 {
		text = text[:i+1]
	}
	if len([]rune(text)) > 160 {
		return ""
	}
	return text
}

func generated(f *ast.File) bool {
	return len(f.Comments) > 0 && f.Comments[0].List[0].Text == golang.Header && f.Comments[0].Pos() < f.Package
}

// errorVar는 var ErrX = errors.New("문자열")를 에러 노드로 옮긴다.
func (c *collector) errorVar(module string, n *ast.Ident, sp *ast.ValueSpec, i int) {
	msg, ok := errorsNew(sp, i)
	if !ok {
		c.skip(n.Pos(), n.Name, "var는 errors.New(문자열) 에러만 옮긴다")
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

// quote는 설계 YAML의 큰따옴표 문자열로 쓴다. 파서가 받는 이스케이프(\" \\ \n \t)만 쓴다.
func quote(s string) (string, bool) {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteRune('\\')
			b.WriteRune(r)
		case r == '\n':
			b.WriteString(`\n`)
		case r == '\t':
			b.WriteString(`\t`)
		case !unicode.IsPrint(r):
			return "", false
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String(), true
}

var snakeName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// snakeID는 Go 이름을 snake_case ID로 바꾼다. GoName(ID)가 원래 이름과 다르면 goName에 원래 이름을 준다.
// 대문자 연속(HTTPPort)은 마지막 대문자 앞에서 끊는다.
func snakeID(name string) (id, goName string, ok bool) {
	rs := []rune(name)
	var parts []string
	start := 0
	for i := 1; i < len(rs); i++ {
		// 숫자 뒤의 대문자는 숫자 앞이 소문자일 때만 끊는다(Sha256Size, P2PPort).
		j := i - 1
		for j > 0 && unicode.IsDigit(rs[j]) {
			j--
		}
		lowerToUpper := unicode.IsUpper(rs[i]) && unicode.IsLower(rs[j])
		acronymEnd := unicode.IsUpper(rs[i]) && unicode.IsUpper(rs[i-1]) && i+1 < len(rs) && unicode.IsLower(rs[i+1])
		if lowerToUpper || acronymEnd {
			parts = append(parts, string(rs[start:i]))
			start = i
		}
	}
	parts = append(parts, string(rs[start:]))
	id = strings.ToLower(strings.Join(parts, "_"))
	if !snakeName.MatchString(id) {
		return "", "", false
	}
	if model.GoName(id) != name {
		goName = name
	}
	return id, goName, true
}

// validate는 초안을 설계에 합쳐 돌리고, 초안 줄을 가리키는 오류·경고가 나온 노드를 빼고 다시 돌린다.
// 초안 밖에 새 오류·경고가 생기면 어느 노드 탓인지 알 수 없어 오류로 끝낸다.
func validate(sources map[string][]byte, designDir, root string, nodes []Node, res *Result) ([]Node, error) {
	sort.SliceStable(nodes, func(i, j int) bool {
		a, b := nodes[i], nodes[j]
		if a.Kind != b.Kind {
			return kindOrder[a.Kind] < kindOrder[b.Kind]
		}
		if a.Module != b.Module {
			return a.Module < b.Module
		}
		return a.ID < b.ID
	})
	base := counts(workspace.RunSources(sources, designDir, root).Diags)
	key := filepath.Join(designDir, draftName)
	for len(nodes) > 0 {
		files := map[string][]byte{key: render(nodes)}
		for k, v := range sources {
			files[k] = v
		}
		r := workspace.RunSources(files, designDir, root)
		lineOf := lineNodes(nodes)
		bad := map[int]string{}
		var outside []string
		for _, d := range r.Diags {
			if d.Level == model.Info {
				continue
			}
			if d.Pos.File == key {
				if i, ok := lineOf[d.Pos.Line]; ok {
					if _, seen := bad[i]; !seen {
						bad[i] = d.Msg
					}
					continue
				}
			}
			outside = append(outside, d.String())
		}
		if len(bad) == 0 {
			if extra := added(base, outside); len(extra) > 0 {
				return nil, fmt.Errorf("초안이 설계의 다른 곳에 진단을 낸다: %s", strings.Join(extra, "; "))
			}
			return nodes, nil
		}
		var kept []Node
		for i, n := range nodes {
			if msg, ok := bad[i]; ok {
				res.Skipped = append(res.Skipped, Skip{n.Where, n.Name, "설계에 더하면 진단이 난다: " + msg})
				continue
			}
			kept = append(kept, n)
		}
		nodes = kept
	}
	return nodes, nil
}

func counts(ds model.Diagnostics) map[string]int {
	out := map[string]int{}
	for _, d := range ds {
		if d.Level != model.Info {
			out[d.String()]++
		}
	}
	return out
}

func added(base map[string]int, now []string) []string {
	seen := map[string]int{}
	var out []string
	for _, s := range now {
		seen[s]++
		if seen[s] > base[s] {
			out = append(out, s)
		}
	}
	return out
}

// header는 초안 파일 머리 주석의 줄 수와 내용이다. lineNodes가 줄 번호를 맞추는 데 쓴다.
const header = "# daemok init --l3가 만든 L3 초안이다. 설계에 없는 손으로 쓴 선언을 gen: false 노드로 옮겼다.\n" +
	"# 새 설계 파일로 저장하거나 기존 설계 파일에 옮겨 붙인다.\n"

func render(nodes []Node) []byte {
	var b bytes.Buffer
	b.WriteString(header)
	kind := ""
	for _, n := range nodes {
		if n.Kind != kind {
			kind = n.Kind
			b.WriteString("\n" + kind + ":\n")
		}
		b.WriteString("  " + n.Line + "\n")
	}
	return b.Bytes()
}

// lineNodes는 render가 쓴 줄 번호(1부터)에서 노드 번호로 가는 표다.
func lineNodes(nodes []Node) map[int]int {
	out := map[int]int{}
	line := strings.Count(header, "\n")
	kind := ""
	for i, n := range nodes {
		if n.Kind != kind {
			kind = n.Kind
			line += 2
		}
		line++
		out[line] = i
	}
	return out
}
