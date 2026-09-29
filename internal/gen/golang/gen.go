// Package golang은 모델에서 Go 생성 파일을 만든다. 디스크는 읽거나 쓰지 않는다.
package golang

import (
	"fmt"
	"go/token"
	"regexp"
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// Generate는 설계 검사를 통과한 모델에서 생성 파일을 경로순으로 만든다.
func Generate(m *model.Model) ([]File, error) {
	g := &gen{m: m, files: map[string]*goFile{}}
	steps := []func() error{
		g.enums, g.consts, g.errors, g.types, g.contracts, g.handlerKinds, g.registrations,
		g.executableModules, g.needs, g.executableContracts, g.events, g.flows, g.cases,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return nil, err
		}
	}
	paths := make([]string, 0, len(g.files))
	for p := range g.files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	out := make([]File, 0, len(paths))
	for _, p := range paths {
		f, err := g.files[p].render()
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, nil
}

// byDir는 노드를 소속 디렉토리별로 처음 나온 순서를 지켜 묶는다.
type byDir[T any] struct {
	dirs  []string
	nodes map[string][]T
}

func (b *byDir[T]) add(dir string, n T) {
	if b.nodes == nil {
		b.nodes = map[string][]T{}
	}
	if _, ok := b.nodes[dir]; !ok {
		b.dirs = append(b.dirs, dir)
	}
	b.nodes[dir] = append(b.nodes[dir], n)
}

func (g *gen) enums() error {
	for _, e := range g.m.Enums {
		if !e.Gen {
			continue
		}
		f := g.file(g.ownerPath(e.In.Name) + "/" + e.ID + ".gen.go")
		tn := e.GoType()
		var lines, names []string
		for _, v := range e.Values {
			n := e.ConstName(v.Name)
			names = append(names, n)
			lines = append(lines, fmt.Sprintf("\t%s %s = %q", n, tn, v.Name))
		}
		f.add("type %s string", tn)
		f.add("const (\n%s\n)", strings.Join(lines, "\n"))
		if len(names) == 0 {
			f.add("// Valid는 값이 설계에 있는지 본다.\nfunc (v %s) Valid() bool { return false }", tn)
		} else {
			f.add("// Valid는 값이 설계에 있는지 본다.\nfunc (v %s) Valid() bool {\n\tswitch v {\n\tcase %s:\n\t\treturn true\n\t}\n\treturn false\n}", tn, strings.Join(names, ", "))
		}
		f.add("// %sValues는 설계에 적힌 순서의 값 전부다.\nfunc %sValues() []%s {\n\treturn []%s{%s}\n}", tn, tn, tn, tn, strings.Join(names, ", "))
	}
	return nil
}

var durationValue = regexp.MustCompile(`^(\d+)(ms|s|m|h)$`)

func (g *gen) consts() error {
	var b byDir[*model.Const]
	for _, c := range g.m.Consts {
		if c.Gen {
			b.add(g.ownerPath(c.In.Name), c)
		}
	}
	units := map[string]string{"ms": "Millisecond", "s": "Second", "m": "Minute", "h": "Hour"}
	for _, dir := range b.dirs {
		f := g.file(dir + "/consts.gen.go")
		var lines []string
		for _, c := range b.nodes[dir] {
			name := c.Go.Name
			if name == "" {
				name = model.GoName(c.ID)
			}
			val := c.Value
			switch c.Type {
			case "string":
				val = fmt.Sprintf("%q", c.Value)
			case "duration":
				mm := durationValue.FindStringSubmatch(c.Value)
				if mm == nil {
					return fmt.Errorf("%s: duration 값의 형식이 아니다: 상수 %s %q", c.ValuePos, c.ID, c.Value)
				}
				val = mm[1] + " * " + f.use("time") + "." + units[mm[2]]
			}
			lines = append(lines, fmt.Sprintf("\t%s = %s", name, val))
		}
		f.add("const (\n%s\n)", strings.Join(lines, "\n"))
	}
	return nil
}

func (g *gen) errors() error {
	var b byDir[*model.ErrorDef]
	for _, e := range g.m.Errors {
		if e.Gen {
			b.add(g.ownerPath(e.In.Name), e)
		}
	}
	for _, dir := range b.dirs {
		f := g.file(dir + "/errors.gen.go")
		pkg := f.use("errors")
		var lines, cases []string
		for _, e := range b.nodes[dir] {
			lines = append(lines, fmt.Sprintf("\t%s = %s.New(%q)", e.ID, pkg, e.Message))
			if e.HTTP != 0 {
				cases = append(cases, fmt.Sprintf("\tcase %s.Is(err, %s):\n\t\treturn %d", pkg, e.ID, e.HTTP))
			}
		}
		f.add("var (\n%s\n)", strings.Join(lines, "\n"))
		if len(cases) > 0 {
			f.add("// HTTPStatus는 설계에 적힌 에러별 HTTP 상태를 돌려준다. 설계에 없는 에러는 500이다.\nfunc HTTPStatus(err error) int {\n\tswitch {\n%s\n\t}\n\treturn 500\n}", strings.Join(cases, "\n"))
		}
	}
	return nil
}

func (g *gen) types() error {
	var b byDir[*model.Type]
	for _, t := range g.m.Types {
		if t.Gen {
			b.add(g.ownerPath(t.In.Name), t)
		}
	}
	for _, dir := range b.dirs {
		f := g.file(dir + "/types.gen.go")
		for _, t := range b.nodes[dir] {
			f.add("type %s struct {\n%s\n}", typeName(t), strings.Join(g.structFields(t.Fields, dir, f, true), "\n"))
		}
	}
	return nil
}

// structFields는 설계 필드를 struct 필드 줄로 바꾼다. optional이면 omitempty를 붙인다.
func (g *gen) structFields(fields []*model.Field, dir string, f *goFile, withMax bool) []string {
	var lines []string
	for _, fl := range fields {
		var gt string
		if fl.RefType.Name != "" {
			base := g.goType(fl.RefType.Name, dir, f)
			if fl.Many {
				gt = "[]" + base
			} else {
				gt = "*" + base
			}
		} else {
			gt = g.goType(fl.Type.Name, dir, f)
			if fl.Many {
				gt = "[]" + gt
			}
			if fl.Optional && !strings.HasPrefix(gt, "*") && !strings.HasPrefix(gt, "[]") {
				gt = "*" + gt
			}
		}
		tag := fl.Name
		if fl.Optional {
			tag += ",omitempty"
		}
		line := fmt.Sprintf("\t%s %s `json:%q`", model.GoName(fl.Name), gt, tag)
		if withMax && fl.Max.Name != "" {
			line += " // 최대 " + fl.Max.Name
		}
		lines = append(lines, line)
	}
	return lines
}

func paramName(name string) string {
	n := lowerFirst(model.GoName(name))
	if token.IsKeyword(n) {
		n += "_"
	}
	return n
}

// methodSig는 계약 메서드의 시그니처 줄이다. 반환 끝에 error가 붙는다.
func (g *gen) methodSig(me *model.Method, dir string, f *goFile) string {
	var ps []string
	for _, p := range me.Params {
		ps = append(ps, paramName(p.Name)+" "+g.argType(p.Type.Name, dir, f))
	}
	var rs []string
	for _, r := range me.Returns {
		rs = append(rs, g.argType(r.Name, dir, f))
	}
	rs = append(rs, "error")
	ret := " " + rs[0]
	if len(rs) > 1 {
		ret = " (" + strings.Join(rs, ", ") + ")"
	}
	line := fmt.Sprintf("\t%s(%s)%s", model.GoName(me.Name), strings.Join(ps, ", "), ret)
	if len(me.Errors) > 0 {
		var names []string
		for _, e := range me.Errors {
			names = append(names, e.Name)
		}
		line += " // " + strings.Join(names, ", ")
	}
	return line
}

func (g *gen) contracts() error {
	var b byDir[*model.Contract]
	for _, c := range g.m.Contracts {
		if c.Gen && c.Kind == "call" {
			b.add(g.ownerPath(c.In.Name), c)
		}
	}
	for _, dir := range b.dirs {
		f := g.file(dir + "/contracts.gen.go")
		for _, c := range b.nodes[dir] {
			var lines []string
			for _, me := range c.Methods {
				lines = append(lines, g.methodSig(me, dir, f))
			}
			f.add("type %s interface {\n%s\n}", contractName(c), strings.Join(lines, "\n"))
		}
	}
	return nil
}

func handlerName(hk *model.HandlerKind) string {
	if hk.Go.Name != "" {
		return hk.Go.Name
	}
	return model.GoName(hk.ID)
}

func returnList(rs []string) string {
	switch len(rs) {
	case 0:
		return ""
	case 1:
		return " " + rs[0]
	}
	return " (" + strings.Join(rs, ", ") + ")"
}

func (g *gen) handlerKinds() error {
	for _, hk := range g.m.HandlerKinds {
		if !hk.Gen {
			continue
		}
		dir := g.ownerPath(hk.In.Name)
		f := g.file(dir + "/" + hk.ID + ".gen.go")
		var ps, rs []string
		for _, p := range hk.Params {
			ps = append(ps, paramName(p.Name)+" "+g.goType(p.Type.Name, dir, f))
		}
		for _, r := range hk.Returns {
			rs = append(rs, g.goType(r.Name, dir, f))
		}
		f.add("type %s func(%s)%s", handlerName(hk), strings.Join(ps, ", "), returnList(rs))
	}
	return nil
}

// handlerFunc는 멤버가 값 v를 처리하는 함수 이름이다. L4 함수 노드가 있으면 그 ID를 쓴다.
func (g *gen) handlerFunc(member, kind, v string) string {
	for _, fn := range g.m.Funcs {
		if fn.In.Name == member && fn.Kind.Name == kind && fn.Handles.Name == v {
			return lowerFirst(model.GoName(fn.ID))
		}
	}
	return "handle" + model.GoName(v)
}

// registrations는 그룹 멤버마다 핸들러 등록 파일을 만든다. 핸들러 종류가 gen: false여도 만든다.
func (g *gen) registrations() error {
	for _, grp := range g.m.Groups {
		hk := g.m.HandlerKind(grp.Kind.Name)
		if hk == nil {
			continue
		}
		en := g.m.Enum(hk.Key.Name)
		for _, mem := range grp.Members {
			if len(mem.Handles) == 0 {
				continue
			}
			dir := mem.Path()
			f := g.file(dir + "/register.gen.go")
			hp := f.use(g.importPath(g.ownerPath(hk.In.Name)))
			register := hk.Go.Register
			if register == "" {
				register = "Register"
			}
			var calls, checks []string
			for _, v := range mem.Handles {
				key := fmt.Sprintf("%q", v.Name)
				if en != nil {
					key = g.qualify(en.In.Name, en.ConstName(v.Name), dir, f)
				}
				fn := g.handlerFunc(mem.ID, hk.ID, v.Name)
				calls = append(calls, fmt.Sprintf("\t%s.%s(%s, %s)", hp, register, key, fn))
				checks = append(checks, fmt.Sprintf("\t_ %s.%s = %s", hp, handlerName(hk), fn))
			}
			f.add("func init() {\n%s\n}", strings.Join(calls, "\n"))
			f.add("// 본문은 손으로 쓰는 파일에 있다. 시그니처가 다르면 여기서 컴파일이 깨진다.\nvar (\n%s\n)", strings.Join(checks, "\n"))
		}
	}
	return nil
}

// executableModules는 실행 단위가 composes에 적은 그룹의 멤버를 blank import한다.
func (g *gen) executableModules() error {
	for _, e := range g.m.Executables {
		var dirs []string
		for _, r := range e.Composes {
			grp := g.m.Group(r.Name)
			if grp == nil || g.m.HandlerKind(grp.Kind.Name) == nil {
				continue
			}
			for _, mem := range grp.Members {
				dirs = append(dirs, mem.Path())
			}
		}
		if len(dirs) == 0 {
			continue
		}
		f := g.file("cmd/" + e.ID + "/modules.gen.go")
		for _, d := range dirs {
			f.blank(g.importPath(d))
		}
		f.add("// 그룹 멤버는 init()에서 스스로 등록한다. 목록은 설계의 composes에서 나온다.")
	}
	return nil
}
