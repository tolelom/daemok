package golang

import "github.com/tolelom/daemok/internal/model"

// MethodShape는 계약 메서드를 생성했다면 나왔을 이름과 타입이다.
type MethodShape struct {
	Name    string
	Params  []string
	Results []string
}

// FieldShape는 타입 필드를 생성했다면 나왔을 이름과 타입이다.
type FieldShape struct {
	Name string
	Type string
}

// shapeFile은 소유 패키지 안의 빈 파일이다. 타입식은 그 패키지에서 쓰는 모양이 되고
// 다른 패키지는 별칭 없이 import 경로의 마지막 요소로 한정된다.
func shapeFile(m *model.Model, owner string) (*gen, string, *goFile) {
	g := &gen{m: m, files: map[string]*goFile{}}
	dir := g.ownerPath(owner)
	f := g.file(dir + "/shape.go")
	f.shape = true
	return g, dir, f
}

// ContractShape는 계약의 메서드를 생성 규칙대로 바꾼 것이다. 코드 대조가 gen: false 계약에 쓴다.
func ContractShape(m *model.Model, c *model.Contract) []MethodShape {
	g, dir, f := shapeFile(m, c.In.Name)
	var out []MethodShape
	for _, me := range c.Methods {
		ps, rs := g.methodTypes(me, dir, f)
		out = append(out, MethodShape{model.GoName(me.Name), ps, rs})
	}
	return out
}

// TypeShape는 타입의 필드를 생성 규칙대로 바꾼 것이다. 코드 대조가 gen: false 타입에 쓴다.
func TypeShape(m *model.Model, t *model.Type) []FieldShape {
	g, dir, f := shapeFile(m, t.In.Name)
	var out []FieldShape
	for _, fl := range t.Fields {
		out = append(out, FieldShape{model.GoName(fl.Name), g.fieldType(fl, dir, f)})
	}
	return out
}
