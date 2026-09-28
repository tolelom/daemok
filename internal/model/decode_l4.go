package model

import "github.com/tolelom/daemok/internal/yaml"

func (mg *merger) funcs(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "funcs", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "kind", "handles", "of", "errors", "cases")
		if !mg.unique("함수", id, pos) {
			return
		}
		f := &Func{Owned: d.owned(id, pos, v), Kind: d.ref(v, "kind"), Handles: d.ref(v, "handles"), Of: d.ref(v, "of"), Errors: d.refs(v, "errors")}
		hasKind, hasOf := f.Kind.Name != "" || f.Handles.Name != "", f.Of.Name != ""
		switch {
		case hasKind && hasOf:
			d.diags.Errorf(pos, "함수 %q에는 kind+handles와 of 중 하나만 쓴다", id)
		case !hasKind && !hasOf:
			d.diags.Errorf(pos, "함수 %q에 kind+handles 또는 of가 필요하다", id)
		case hasKind:
			d.need(v, "kind")
			d.need(v, "handles")
		}
		f.Cases = mg.cases(v.Get("cases"))
		mg.m.Funcs = append(mg.m.Funcs, f)
	})
}

func (mg *merger) cases(n *yaml.Node) []*Case {
	if n == nil || n.IsNull() {
		return nil
	}
	d := mg.d
	if n.Kind != yaml.Seq {
		d.diags.Errorf(n.Pos, "cases는 목록이어야 한다")
		return nil
	}
	var out []*Case
	for _, it := range n.Items {
		if !d.mapping(it, "케이스") {
			continue
		}
		d.allow(it, "name", "given", "expect", "error")
		c := &Case{Pos: it.Pos, Error: d.ref(it, "error")}
		if d.need(it, "name") {
			c.Name = d.str(it, "name")
		}
		if d.need(it, "given") {
			c.Given = mg.caseMap(it.Get("given"), "given")
		}
		if e := it.Get("expect"); e != nil {
			c.Expect = mg.caseMap(e, "expect")
		}
		switch {
		case c.Expect != nil && c.Error.Name != "":
			d.diags.Errorf(it.Pos, "케이스에는 expect와 error 중 하나만 쓴다")
		case c.Expect == nil && c.Error.Name == "":
			d.diags.Errorf(it.Pos, "케이스에 expect나 error가 필요하다")
		}
		out = append(out, c)
	}
	return out
}

// caseMap은 given·expect가 스칼라 또는 맵만 값으로 갖는 맵인지 확인한다.
func (mg *merger) caseMap(n *yaml.Node, what string) *yaml.Node {
	d := mg.d
	if n.Kind != yaml.Map {
		d.diags.Errorf(n.Pos, "%s은 맵이어야 한다", what)
		return nil
	}
	for _, e := range n.Entries {
		switch e.Value.Kind {
		case yaml.Scalar:
		case yaml.Map:
			mg.caseMap(e.Value, what)
		default:
			d.diags.Errorf(e.Value.Pos, "%s의 값은 값 하나 또는 맵이어야 한다", what)
		}
	}
	return n
}
