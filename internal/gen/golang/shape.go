package golang

import "github.com/tolelom/daemok/internal/model"

type MethodShape struct {
	Name    string
	Params  []string
	Results []string
}

type FieldShape struct {
	Name string
	Type string
}

func shapeFile(m *model.Model, owner string) (*gen, string, *goFile) {
	g := &gen{m: m, files: map[string]*goFile{}}
	dir := g.ownerPath(owner)
	f := g.file(dir + "/shape.go")
	f.shape = true
	return g, dir, f
}

func ContractShape(m *model.Model, c *model.Contract) []MethodShape {
	g, dir, f := shapeFile(m, c.In.Name)
	var out []MethodShape
	for _, me := range c.Methods {
		ps, rs := g.methodTypes(me, dir, f)
		out = append(out, MethodShape{model.GoName(me.Name), ps, rs})
	}
	return out
}

func TypeShape(m *model.Model, t *model.Type) []FieldShape {
	g, dir, f := shapeFile(m, t.In.Name)
	var out []FieldShape
	for _, fl := range t.Fields {
		out = append(out, FieldShape{model.GoName(fl.Name), g.fieldType(fl, dir, f)})
	}
	return out
}
