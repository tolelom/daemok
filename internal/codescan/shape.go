package codescan

import (
	"go/ast"
	"path"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

type typeSpec struct {
	spec *ast.TypeSpec
	file *ast.File
}

func collectTypeSpecs(f *ast.File, into map[string]typeSpec) {
	for _, d := range f.Decls {
		g, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, sp := range g.Specs {
			if ts, ok := sp.(*ast.TypeSpec); ok {
				into[ts.Name.Name] = typeSpec{ts, f}
			}
		}
	}
}

// matchName은 설계 이름과 코드 이름을 대조하는 키다. 생성 이름은 AvatarUrl, 손으로 쓴 이름은 AvatarURL일 수 있다.
func matchName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "_", ""))
}

// typeExpr는 코드의 타입식을 생성 코드와 같은 모양으로 쓴다. 설계 모듈 패키지는 import 경로의 마지막 요소로,
// 그 밖의 패키지는 코드에 쓴 이름 그대로 한정한다.
// 생성 코드에 나오지 않는 타입식(함수, 채널, 배열 길이, 가변 인자, 제네릭 등)이면 false다.
func typeExpr(e ast.Expr, f *ast.File, module string) (string, bool) {
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name, true
	case *ast.SelectorExpr:
		pkg, ok := x.X.(*ast.Ident)
		if !ok {
			return "", false
		}
		return importBase(f, pkg.Name, module) + "." + x.Sel.Name, true
	case *ast.StarExpr:
		s, ok := typeExpr(x.X, f, module)
		return "*" + s, ok
	case *ast.ArrayType:
		if x.Len != nil {
			return "", false
		}
		s, ok := typeExpr(x.Elt, f, module)
		return "[]" + s, ok
	case *ast.MapType:
		k, ok1 := typeExpr(x.Key, f, module)
		v, ok2 := typeExpr(x.Value, f, module)
		return "map[" + k + "]" + v, ok1 && ok2
	case *ast.InterfaceType:
		if x.Methods == nil || len(x.Methods.List) == 0 {
			return "any", true
		}
	case *ast.ParenExpr:
		return typeExpr(x.X, f, module)
	}
	return "", false
}

// importBase는 파일에서 패키지를 가리키는 이름이 설계 모듈(go_module 안) 패키지나 생성기가 쓰는
// 표준 라이브러리면 그 import 경로의 마지막 요소로 바꾼다. 생성기가 그 이름으로 한정하기 때문이다. 별칭 없는 import의 패키지 이름은
// 경로의 마지막 요소로 본다. go_module 밖 패키지는 설계에 코드와 같은 이름으로 적으므로 그대로 둔다.
func importBase(f *ast.File, name, module string) string {
	for _, im := range f.Imports {
		p, _ := strconv.Unquote(im.Path.Value)
		if p != module && !strings.HasPrefix(p, module+"/") && !golang.StdImport(p) {
			continue
		}
		local := path.Base(p)
		if im.Name != nil {
			local = im.Name.Name
		}
		if local == name {
			return path.Base(p)
		}
	}
	return name
}

// fieldTypes는 파라미터·반환 목록을 타입 하나씩 편다. a, b int는 int 두 개다.
func fieldTypes(fl *ast.FieldList, f *ast.File, module string) ([]string, bool) {
	var out []string
	if fl == nil {
		return out, true
	}
	for _, fd := range fl.List {
		t, ok := typeExpr(fd.Type, f, module)
		if !ok {
			return nil, false
		}
		for i := 0; i < max(1, len(fd.Names)); i++ {
			out = append(out, t)
		}
	}
	return out, true
}

func signature(params, results []string) string {
	return "(" + strings.Join(params, ", ") + ") (" + strings.Join(results, ", ") + ")"
}

func sameList(a, b []string) bool {
	return strings.Join(a, "\x00") == strings.Join(b, "\x00") && len(a) == len(b)
}

// shapes는 gen: false인 call 계약과 타입의 코드 선언이 생성했다면 나왔을 모양과 같은지 본다.
// 선언이 없는 경우는 handWritten이 알린다. 타입이 아닌 선언(함수, 변수, 상수)이면 오류다.
func (s *scanner) shapes() {
	for _, ct := range s.m.Contracts {
		if ct.Gen || ct.Kind != "call" {
			continue
		}
		ts, ok := s.typeSpecs[ct.In.Name][ct.GoName()]
		switch {
		case ok && s.uncomparable(ts, "계약", ct.ID, ct.Pos):
		case ok:
			s.contractShape(ct, ts)
		case s.decls[ct.In.Name][ct.GoName()]:
			s.diags.Errorf(ct.Pos, "손으로 쓴 계약이 interface가 아니다: 계약 %s (%s)", ct.ID, ct.GoName())
		}
	}
	for _, t := range s.m.Types {
		if t.Gen {
			continue
		}
		ts, ok := s.typeSpecs[t.In.Name][t.GoName()]
		switch {
		case ok && s.uncomparable(ts, "타입", t.ID, t.Pos):
		case ok:
			s.typeShape(t, ts)
		case s.decls[t.In.Name][t.GoName()]:
			s.diags.Errorf(t.Pos, "손으로 쓴 타입이 struct가 아니다: 타입 %s (%s)", t.ID, t.GoName())
		}
	}
}

// jsonTag는 struct 필드 태그의 json 부분을 `json:"값"`으로 준다. 없으면 "태그 없음"이다.
func jsonTag(tag *ast.BasicLit) string {
	if tag != nil {
		raw, _ := strconv.Unquote(tag.Value)
		if v, ok := reflect.StructTag(raw).Lookup("json"); ok {
			return `json:"` + v + `"`
		}
	}
	return "태그 없음"
}

// uncomparable은 타입 별칭과 제네릭 선언이면 정보로 알리고 true를 준다. 생성 코드에 없는 모양이다.
func (s *scanner) uncomparable(ts typeSpec, what, id string, pos yaml.Pos) bool {
	if ts.spec.Assign == 0 && ts.spec.TypeParams == nil {
		return false
	}
	s.diags.Infof(pos, "대조할 수 없는 타입식이라 건너뛴다: %s %s (%s)", what, id, ts.spec.Name.Name)
	return true
}

func (s *scanner) contractShape(ct *model.Contract, ts typeSpec) {
	iface, ok := ts.spec.Type.(*ast.InterfaceType)
	if !ok {
		s.diags.Errorf(ct.Pos, "손으로 쓴 계약이 interface가 아니다: 계약 %s (%s)", ct.ID, ct.GoName())
		return
	}
	type method struct {
		name            string
		params, results []string
		ok              bool
	}
	code := map[string]*method{}
	var order []string
	embedded := false
	for _, fd := range iface.Methods.List {
		ft, isFunc := fd.Type.(*ast.FuncType)
		if len(fd.Names) == 0 || !isFunc {
			embedded = true
			continue
		}
		ps, ok1 := fieldTypes(ft.Params, ts.file, s.m.GoModule)
		rs, ok2 := fieldTypes(ft.Results, ts.file, s.m.GoModule)
		for _, n := range fd.Names {
			code[matchName(n.Name)] = &method{n.Name, ps, rs, ok1 && ok2}
			order = append(order, matchName(n.Name))
		}
	}
	seen := map[string]bool{}
	for i, want := range golang.ContractShape(s.m, ct) {
		key := matchName(want.Name)
		seen[key] = true
		got, ok := code[key]
		pos := ct.Methods[i].Pos
		switch {
		case !ok:
			s.diags.Errorf(pos, "손으로 쓴 계약에 설계 메서드가 없다: 계약 %s, %s", ct.ID, want.Name)
		case !got.ok:
			s.diags.Infof(pos, "대조할 수 없는 타입식이라 건너뛴다: 계약 %s, %s", ct.ID, got.name)
		case !sameList(want.Params, got.params) || !sameList(want.Results, got.results):
			s.diags.Errorf(pos, "손으로 쓴 계약 메서드의 시그니처가 설계와 다르다: 계약 %s, %s: 설계 %s, 코드 %s",
				ct.ID, got.name, signature(want.Params, want.Results), signature(got.params, got.results))
		}
	}
	if embedded {
		s.diags.Infof(s.pos(ts.spec.Pos()), "끼워 넣은 interface가 있어 코드에만 있는 메서드는 보지 않는다: 계약 %s", ct.ID)
		return
	}
	var extra []string
	for _, key := range order {
		if !seen[key] {
			extra = append(extra, code[key].name)
		}
	}
	if len(extra) > 0 {
		s.diags.Warnf(s.pos(ts.spec.Pos()), "설계에 없는 메서드가 코드 계약에 있다: 계약 %s (%s)", ct.ID, strings.Join(extra, ", "))
	}
}

func (s *scanner) typeShape(t *model.Type, ts typeSpec) {
	st, ok := ts.spec.Type.(*ast.StructType)
	if !ok {
		s.diags.Errorf(t.Pos, "손으로 쓴 타입이 struct가 아니다: 타입 %s (%s)", t.ID, t.GoName())
		return
	}
	type field struct {
		name, typ, json string // json은 태그의 json 값. 태그가 없으면 "태그 없음"
		ok              bool
	}
	code := map[string]*field{}
	var order []string
	embedded := false
	for _, fd := range st.Fields.List {
		if len(fd.Names) == 0 {
			embedded = true
			continue
		}
		typ, ok := typeExpr(fd.Type, ts.file, s.m.GoModule)
		for _, n := range fd.Names {
			if !n.IsExported() {
				continue
			}
			code[matchName(n.Name)] = &field{n.Name, typ, jsonTag(fd.Tag), ok}
			order = append(order, matchName(n.Name))
		}
	}
	toTS := slices.Contains(t.Export, "ts")
	seen := map[string]bool{}
	var unknown []string
	for i, want := range golang.TypeShape(s.m, t) {
		key := matchName(want.Name)
		seen[key] = true
		got, ok := code[key]
		pos := t.Fields[i].Pos
		switch {
		case !ok && embedded:
			unknown = append(unknown, want.Name)
		case !ok:
			s.diags.Errorf(pos, "손으로 쓴 타입에 설계 필드가 없다: 타입 %s, %s", t.ID, want.Name)
		case !got.ok:
			s.diags.Infof(pos, "대조할 수 없는 타입식이라 건너뛴다: 타입 %s, %s", t.ID, got.name)
		case want.Type != got.typ:
			s.diags.Errorf(pos, "손으로 쓴 타입 필드의 타입이 설계와 다르다: 타입 %s, %s: 설계 %s, 코드 %s", t.ID, got.name, want.Type, got.typ)
		}
		if wantTag := golang.JSONTag(t.Fields[i]); ok && toTS && got.json != wantTag {
			s.diags.Errorf(pos, "손으로 쓴 타입 필드의 json 태그가 설계와 다르다: 타입 %s, %s: 설계 %s, 코드 %s", t.ID, got.name, wantTag, got.json)
		}
	}
	if len(unknown) > 0 {
		s.diags.Infof(t.Pos, "끼워 넣은 필드가 있어 코드에 없는 설계 필드를 확정하지 않는다: 타입 %s (%s)", t.ID, strings.Join(unknown, ", "))
	}
	var extra []string
	for _, key := range order {
		if !seen[key] {
			extra = append(extra, code[key].name)
		}
	}
	if len(extra) > 0 {
		s.diags.Warnf(s.pos(ts.spec.Pos()), "설계에 없는 필드가 코드 타입에 있다: 타입 %s (%s)", t.ID, strings.Join(extra, ", "))
	}
}
