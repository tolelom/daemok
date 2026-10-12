package golang

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// numberKinds는 JSON을 거친 값(float64)과 Go에서 직접 넣은 정수를 모두 받는 변환 도우미의 대상이다.
var numberKinds = []string{"int", "int64", "uint", "uint64", "float64"}

// eventHelpers는 모듈 디렉토리별로 쓰인 변환 도우미다.
type eventHelpers map[string]map[string]bool

func (g *gen) events() error {
	helpers := eventHelpers{}
	for _, ek := range g.m.EventKinds {
		if !ek.Gen {
			continue
		}
		dir := g.ownerPath(ek.In.Name)
		f := g.file(dir + "/" + ek.ID + ".gen.go")
		for _, e := range g.kindEvents(ek) {
			name := model.GoName(e.ID) + "Data"
			f.add("// %s는 이벤트 %s의 본문이다.\ntype %s struct {\n%s\n}", name, e.ID, name, strings.Join(g.structFields(e.Data, dir, f, false), "\n"))
			if ek.Encoding != "map" {
				continue
			}
			f.add("%s", g.eventMap(e, name, dir, f))
			f.add("%s", g.eventFrom(e, name, dir, f, helpers))
		}
	}
	dirs := make([]string, 0, len(helpers))
	for d := range helpers {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		f := g.file(dir + "/daemok_event.gen.go")
		f.use("fmt")
		used := helpers[dir]
		for _, k := range []string{"string", "bool"} {
			if used[k] {
				f.add("%s", scalarHelper(k))
			}
		}
		for _, k := range numberKinds {
			if used[k] {
				if k != "float64" {
					f.use("math")
				}
				f.add("%s", numberHelper(k))
			}
		}
		if used["[]string"] {
			f.add("%s", stringsHelper)
		}
	}
	return nil
}

// kindEvents는 이벤트 종류의 이벤트를 key enum 값 순서로 준다.
func (g *gen) kindEvents(ek *model.EventKind) []*model.Event {
	var out []*model.Event
	en := g.m.Enum(ek.Key.Name)
	if en == nil {
		return out
	}
	for _, v := range en.Values {
		if e := g.m.Event(v.Name); e != nil && e.Kind.Name == ek.ID {
			out = append(out, e)
		}
	}
	return out
}

// optionalValue는 optional 필드를 포인터로 두고 map에는 값으로 넣는지다.
func (g *gen) optionalValue(fl *model.Field, dir string, f *goFile) bool {
	return fl.Optional && fl.RefType.Name == "" && !fl.Many && !strings.HasPrefix(g.goType(fl.Type.Name, dir, f), "*")
}

// eventMap은 map 본문을 만드는 메서드다. optional 필드는 값이 있을 때만 역참조해서 넣어,
// map에는 JSON을 거친 것과 같은 모양의 값만 들어간다.
func (g *gen) eventMap(e *model.Event, name, dir string, f *goFile) string {
	var pairs, opts []string
	for _, fl := range e.Data {
		field := model.GoName(fl.Name)
		if g.optionalValue(fl, dir, f) {
			opts = append(opts, fmt.Sprintf("\tif d.%s != nil {\n\t\tm[%q] = *d.%s\n\t}", field, fl.Name, field))
			continue
		}
		if fl.Optional {
			// ref·many·포인터 타입의 optional은 nil이 아닐 때만 넣는다
			opts = append(opts, fmt.Sprintf("\tif d.%s != nil {\n\t\tm[%q] = d.%s\n\t}", field, fl.Name, field))
			continue
		}
		pairs = append(pairs, fmt.Sprintf("%q: d.%s", fl.Name, field))
	}
	if len(opts) == 0 {
		return fmt.Sprintf("// Map은 봉투의 map 본문으로 바꾼다.\nfunc (d %s) Map() map[string]any {\n\treturn map[string]any{%s}\n}", name, strings.Join(pairs, ", "))
	}
	return fmt.Sprintf("// Map은 봉투의 map 본문으로 바꾼다. optional 필드는 값이 있을 때만 넣는다.\nfunc (d %s) Map() map[string]any {\n\tm := map[string]any{%s}\n%s\n\treturn m\n}", name, strings.Join(pairs, ", "), strings.Join(opts, "\n"))
}

// eventFrom은 map 본문에서 payload를 읽는 함수를 만든다. 키가 없거나 타입이 다르면 오류다.
// optional 필드는 키가 없거나 nil이면 건너뛴다. 내장 숫자·문자열·bool, 문자열 목록, 설계 enum은
// JSON을 거친 값도 받고, 그 밖의 타입은 타입 단언만 한다.
func (g *gen) eventFrom(e *model.Event, name, dir string, f *goFile, helpers eventHelpers) string {
	var b strings.Builder
	fmt.Fprintf(&b, "// %sFrom은 봉투의 map 본문에서 %s를 읽는다.\nfunc %sFrom(m map[string]any) (%s, error) {\n\tvar d %s\n", name, name, name, name, name)
	for _, fl := range e.Data {
		field := "d." + model.GoName(fl.Name)
		if g.optionalValue(fl, dir, f) {
			fmt.Fprintf(&b, "\tif raw, present := m[%q]; present && raw != nil {\n\t\tvar x %s\n%s\t\t%s = &x\n\t}\n",
				fl.Name, g.goType(fl.Type.Name, dir, f), indent(g.assign(e, fl, "x", dir, f, helpers)), field)
			continue
		}
		if fl.Optional {
			fmt.Fprintf(&b, "\tif raw, present := m[%q]; present && raw != nil {\n%s\t}\n", fl.Name, indent(g.assign(e, fl, field, dir, f, helpers)))
			continue
		}
		b.WriteString(g.assign(e, fl, field, dir, f, helpers))
	}
	b.WriteString("\treturn d, nil\n}")
	return b.String()
}

func indent(s string) string {
	lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
	for i, l := range lines {
		lines[i] = "\t" + l
	}
	return strings.Join(lines, "\n") + "\n"
}

// assign은 필드 하나를 target에 읽어 넣는 문장이다. 실패하면 d와 오류를 돌려준다.
func (g *gen) assign(e *model.Event, fl *model.Field, target, dir string, f *goFile, helpers eventHelpers) string {
	fmtPkg := f.use("fmt")
	fail := fmt.Sprintf("return d, %s.Errorf(\"%%s: %%s: %%T\", %q, %q, m[%q])", fmtPkg, e.ID, fl.Name, fl.Name)
	if kind := g.helperKind(fl); kind != "" {
		if helpers[dir] == nil {
			helpers[dir] = map[string]bool{}
		}
		helpers[dir][kind] = true
		return fmt.Sprintf("\tif v, err := daemok%s(m, %q, %q); err != nil {\n\t\treturn d, err\n\t} else {\n\t\t%s = v\n\t}\n", helperName(kind), e.ID, fl.Name, target)
	}
	if kind, base := g.m.ResolveType(fl.Type.Name); kind == model.EnumType && fl.RefType.Name == "" && !fl.Many {
		if prefix, _ := model.SplitType(fl.Type.Name); prefix == "" {
			en := g.m.Enum(base)
			gt := g.qualify(en.In.Name, en.GoType(), dir, f)
			fromString := fmt.Sprintf("\t\t%s = %s(v)", target, gt)
			if en.Gen {
				// 생성한 enum이면 Valid()로 설계에 없는 값을 거른다
				fromString = fmt.Sprintf("\t\tif !%s(v).Valid() {\n\t\t\t%s\n\t\t}\n%s", gt, fail, fromString)
			}
			return fmt.Sprintf("\tswitch v := m[%q].(type) {\n\tcase %s:\n\t\t%s = v\n\tcase string:\n%s\n\tdefault:\n\t\t%s\n\t}\n", fl.Name, gt, target, fromString, fail)
		}
	}
	gt := g.fieldType(fl, dir, f)
	if target == "x" {
		gt = strings.TrimPrefix(gt, "*")
	}
	return fmt.Sprintf("\tif v, ok := m[%q].(%s); ok {\n\t\t%s = v\n\t} else {\n\t\t%s\n\t}\n", fl.Name, gt, target, fail)
}

// helperKind는 필드가 변환 도우미를 쓰면 그 종류를 준다. optional 여부는 호출하는 쪽이 다룬다.
func (g *gen) helperKind(fl *model.Field) string {
	if fl.RefType.Name != "" {
		return ""
	}
	prefix, base := model.SplitType(fl.Type.Name)
	if prefix != "" {
		return ""
	}
	kind, _ := g.m.ResolveType(fl.Type.Name)
	if kind != model.BuiltinType {
		return ""
	}
	goName := model.Builtins[base]
	if fl.Many {
		if goName == "string" {
			return "[]string"
		}
		return ""
	}
	switch goName {
	case "string", "bool":
		return goName
	}
	for _, k := range numberKinds {
		if goName == k {
			return k
		}
	}
	return ""
}

func helperName(kind string) string {
	if kind == "[]string" {
		return "Strings"
	}
	return strings.ToUpper(kind[:1]) + kind[1:]
}

func scalarHelper(kind string) string {
	zero := `""`
	if kind == "bool" {
		zero = "false"
	}
	return fmt.Sprintf(`func daemok%s(m map[string]any, ev, key string) (%s, error) {
	v, ok := m[key].(%s)
	if !ok {
		return %s, fmt.Errorf("%%s: %%s: %%T", ev, key, m[key])
	}
	return v, nil
}`, helperName(kind), kind, kind, zero)
}

// numberHelper는 JSON을 거친 float64와 Go에서 넣은 정수를 받는다.
// 부호·범위를 넘거나 정수가 아닌 값은 오류다. float64를 거쳐 옮기지 않아 큰 정수도 그대로다.
func numberHelper(kind string) string {
	signed := kind == "int" || kind == "int64"
	var cases []string
	for _, src := range numberKinds {
		if src == kind {
			cases = append(cases, fmt.Sprintf("\tcase %s:\n\t\treturn v, nil", kind))
			continue
		}
		cond := ""
		switch {
		case kind == "float64":
		case src == "float64" && signed:
			cond = "v == math.Trunc(v) && v >= -9.223372036854775808e18 && v < 9.223372036854775808e18"
		case src == "float64":
			cond = "v == math.Trunc(v) && v >= 0 && v < 1.8446744073709552e19"
		case signed && (src == "uint" || src == "uint64"):
			cond = "v <= math.MaxInt64"
			if kind == "int" {
				cond = "v <= math.MaxInt"
			}
		case kind == "int" && src == "int64":
			cond = "v >= math.MinInt && v <= math.MaxInt"
		case !signed && (src == "int" || src == "int64"):
			cond = "v >= 0"
		case kind == "uint" && src == "uint64":
			cond = "v <= math.MaxUint"
		}
		if cond == "" {
			cases = append(cases, fmt.Sprintf("\tcase %s:\n\t\treturn %s(v), nil", src, kind))
		} else {
			cases = append(cases, fmt.Sprintf("\tcase %s:\n\t\tif %s {\n\t\t\treturn %s(v), nil\n\t\t}", src, cond, kind))
		}
	}
	return fmt.Sprintf(`func daemok%s(m map[string]any, ev, key string) (%s, error) {
	switch v := m[key].(type) {
%s
	}
	return 0, fmt.Errorf("%%s: %%s: %%T %%v", ev, key, m[key], m[key])
}`, helperName(kind), kind, strings.Join(cases, "\n"))
}

const stringsHelper = `func daemokStrings(m map[string]any, ev, key string) ([]string, error) {
	switch v := m[key].(type) {
	case []string:
		return v, nil
	case []any:
		out := make([]string, len(v))
		for i, x := range v {
			s, ok := x.(string)
			if !ok {
				return nil, fmt.Errorf("%s: %s[%d]: %T", ev, key, i, x)
			}
			out[i] = s
		}
		return out, nil
	}
	return nil, fmt.Errorf("%s: %s: %T", ev, key, m[key])
}`
