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
			var pairs []string
			for _, fl := range e.Data {
				pairs = append(pairs, fmt.Sprintf("%q: d.%s", fl.Name, model.GoName(fl.Name)))
			}
			f.add("// Map은 봉투의 map 본문으로 바꾼다.\nfunc (d %s) Map() map[string]any {\n\treturn map[string]any{%s}\n}", name, strings.Join(pairs, ", "))
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

// eventFrom은 map 본문에서 payload를 읽는 함수를 만든다. 키가 없거나 타입이 다르면 오류다.
// 내장 숫자·문자열·bool과 문자열 목록은 JSON을 거친 값도 받고, 그 밖의 타입은 타입 단언만 한다.
func (g *gen) eventFrom(e *model.Event, name, dir string, f *goFile, helpers eventHelpers) string {
	fmtPkg := f.use("fmt")
	var b strings.Builder
	fmt.Fprintf(&b, "// %sFrom은 봉투의 map 본문에서 %s를 읽는다.\nfunc %sFrom(m map[string]any) (%s, error) {\n\tvar d %s\n", name, name, name, name, name)
	if len(e.Data) > 0 {
		b.WriteString("\tvar err error\n")
	}
	used := false
	for _, fl := range e.Data {
		field := model.GoName(fl.Name)
		kind := g.helperKind(fl)
		if kind != "" {
			if helpers[dir] == nil {
				helpers[dir] = map[string]bool{}
			}
			helpers[dir][kind] = true
			fmt.Fprintf(&b, "\tif d.%s, err = daemok%s(m, %q, %q); err != nil {\n\t\treturn d, err\n\t}\n", field, helperName(kind), e.ID, fl.Name)
			used = true
			continue
		}
		gt := g.fieldType(fl, dir, f)
		fmt.Fprintf(&b, "\tif v, ok := m[%q].(%s); ok {\n\t\td.%s = v\n\t} else {\n\t\treturn d, %s.Errorf(\"%%s: %%s: %%T\", %q, %q, m[%q])\n\t}\n", fl.Name, gt, field, fmtPkg, e.ID, fl.Name, fl.Name)
	}
	if len(e.Data) > 0 && !used {
		b.WriteString("\t_ = err\n")
	}
	b.WriteString("\treturn d, nil\n}")
	return b.String()
}

// fieldType은 struct 필드와 같은 규칙의 Go 타입이다.
func (g *gen) fieldType(fl *model.Field, dir string, f *goFile) string {
	line := g.structFields([]*model.Field{fl}, dir, f, false)[0]
	parts := strings.Fields(line)
	return parts[1]
}

// helperKind는 필드가 변환 도우미를 쓰면 그 종류를 준다.
func (g *gen) helperKind(fl *model.Field) string {
	if fl.RefType.Name != "" || fl.Optional {
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

// numberHelper는 정수 타입이면 float64를 정수일 때만 받는다. 값을 float64로 거쳐 옮기지 않아 큰 정수도 그대로다.
func numberHelper(kind string) string {
	var cases []string
	for _, src := range []string{"int", "int64", "uint64"} {
		if src != kind {
			cases = append(cases, fmt.Sprintf("\tcase %s:\n\t\treturn %s(v), nil", src, kind))
		}
	}
	if kind == "float64" {
		cases = append([]string{"\tcase float64:\n\t\treturn v, nil"}, cases...)
	} else {
		cases = append([]string{fmt.Sprintf("\tcase %s:\n\t\treturn v, nil", kind)}, cases...)
		cases = append(cases, fmt.Sprintf("\tcase float64:\n\t\tif v == float64(%s(v)) {\n\t\t\treturn %s(v), nil\n\t\t}", kind, kind))
	}
	return fmt.Sprintf(`func daemok%s(m map[string]any, ev, key string) (%s, error) {
	switch v := m[key].(type) {
%s
	}
	return 0, fmt.Errorf("%%s: %%s: %%T", ev, key, m[key])
}`, helperName(kind), kind, strings.Join(dedupeCases(cases), "\n"))
}

// dedupeCases는 같은 타입의 case가 두 번 나오지 않게 한다.
func dedupeCases(cases []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range cases {
		head := strings.SplitN(c, ":", 2)[0]
		if seen[head] {
			continue
		}
		seen[head] = true
		out = append(out, c)
	}
	return out
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
