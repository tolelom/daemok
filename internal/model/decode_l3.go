package model

import "github.com/tolelom/daemok/internal/yaml"

func (mg *merger) enums(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "enums", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "values", "go", "export")
		if !mg.unique("enum", id, pos) {
			return
		}
		e := &Enum{Owned: d.owned(id, pos, v), Go: d.goOpts(v), Export: d.strs(v, "export")}
		if d.need(pos, v, "values") {
			e.Values = d.refs(v, "values")
		}
		mg.m.Enums = append(mg.m.Enums, e)
	})
}

func (mg *merger) consts(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "consts", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "type", "value", "go", "export")
		if !mg.unique("상수", id, pos) {
			return
		}
		c := &Const{Owned: d.owned(id, pos, v), Type: d.str(v, "type"), Go: d.goOpts(v), Export: d.strs(v, "export")}
		if c.Type == "" {
			c.Type = "int"
		}
		switch c.Type {
		case "int", "string", "duration":
		default:
			d.diags.Errorf(v.Get("type").Pos, "type은 int, string, duration 중 하나여야 한다")
		}
		if d.need(pos, v, "value") {
			if val := d.scalar(v, "value"); val != nil {
				c.Value, c.ValuePos = val.Value, val.Pos
				if _, ok := val.Int(); c.Type == "int" && !ok {
					d.diags.Errorf(val.Pos, "value는 정수여야 한다")
				}
			}
		}
		mg.m.Consts = append(mg.m.Consts, c)
	})
}

func (mg *merger) errors(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "errors", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "message", "http", "go")
		if !mg.unique("에러", id, pos) {
			return
		}
		e := &ErrorDef{Owned: d.owned(id, pos, v), Go: d.goOpts(v)}
		if d.need(pos, v, "message") {
			e.Message = d.str(v, "message")
		}
		e.HTTP, _ = d.intv(v, "http")
		mg.m.Errors = append(mg.m.Errors, e)
	})
}

func (mg *merger) types(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "types", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "fields", "go")
		if !mg.unique("타입", id, pos) {
			return
		}
		t := &Type{Owned: d.owned(id, pos, v), Go: d.goOpts(v)}
		if d.need(pos, v, "fields") {
			t.Fields = mg.fields(v.Get("fields"))
		}
		mg.m.Types = append(mg.m.Types, t)
	})
}

func (mg *merger) fields(n *yaml.Node) []*Field {
	d := mg.d
	if !d.mapping(n, "fields") {
		return nil
	}
	var out []*Field
	for _, e := range n.Entries {
		f := &Field{Name: e.Key, Pos: e.Pos}
		switch e.Value.Kind {
		case yaml.Scalar:
			if e.Value.IsNull() {
				d.diags.Errorf(e.Pos, "필드 %q에 타입이 필요하다", e.Key)
				continue
			}
			f.Type = Ref{Name: e.Value.Value, Pos: e.Value.Pos}
		case yaml.Map:
			v := e.Value
			d.allow(v, "type", "ref", "many", "optional", "max")
			f.Type, f.RefType = d.ref(v, "type"), d.ref(v, "ref")
			switch {
			case f.Type.Name != "" && f.RefType.Name != "":
				d.diags.Errorf(e.Pos, "필드 %q에는 type과 ref 중 하나만 쓴다", e.Key)
			case f.Type.Name == "" && f.RefType.Name == "":
				d.diags.Errorf(e.Pos, "필드 %q에 type이나 ref가 필요하다", e.Key)
			}
			f.Many, f.Optional = d.boolv(v, "many", false), d.boolv(v, "optional", false)
			f.Max = d.ref(v, "max")
		default:
			d.diags.Errorf(e.Pos, "타입 이름이나 맵이어야 한다: 필드 %q", e.Key)
			continue
		}
		out = append(out, f)
	}
	return out
}

func (mg *merger) params(n *yaml.Node) []*Param {
	d := mg.d
	p := n.Get("params")
	if p == nil || p.IsNull() || !d.mapping(p, "params") {
		return nil
	}
	var out []*Param
	for _, e := range p.Entries {
		if e.Value.Kind != yaml.Scalar {
			d.diags.Errorf(e.Pos, "타입 이름이어야 한다: 파라미터 %q", e.Key)
			continue
		}
		if e.Value.IsNull() {
			d.diags.Errorf(e.Pos, "파라미터 %q에 타입이 필요하다", e.Key)
			continue
		}
		out = append(out, &Param{Name: e.Key, Pos: e.Pos, Type: Ref{Name: e.Value.Value, Pos: e.Value.Pos}})
	}
	return out
}

func (mg *merger) contracts(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "contracts", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "kind", "go", "methods")
		if !mg.unique("계약", id, pos) {
			return
		}
		c := &Contract{Owned: d.owned(id, pos, v), Kind: d.oneOf(v, "kind", "call", "call", "query", "port", "external"), Go: d.goOpts(v)}
		if d.need(pos, v, "methods") {
			d.idMap(v.Get("methods"), "methods", func(name string, mpos yaml.Pos, mv *yaml.Node) {
				d.allow(mv, "params", "returns", "errors")
				c.Methods = append(c.Methods, &Method{
					Name:      name,
					Pos:       mpos,
					Params:    mg.params(mv),
					Returns:   d.refs(mv, "returns"),
					Errors:    d.refs(mv, "errors"),
					HasErrors: mv.Get("errors") != nil,
				})
			})
		}
		mg.m.Contracts = append(mg.m.Contracts, c)
	})
}

func (mg *merger) handlerKinds(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "handler_kinds", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "params", "returns", "key", "go")
		if !mg.unique("핸들러 종류", id, pos) {
			return
		}
		hk := &HandlerKind{Owned: d.owned(id, pos, v), Params: mg.params(v), Returns: d.refs(v, "returns"), Go: d.goOpts(v)}
		if d.need(pos, v, "key") {
			hk.Key = d.ref(v, "key")
		}
		mg.m.HandlerKinds = append(mg.m.HandlerKinds, hk)
	})
}

func (mg *merger) eventKinds(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "event_kinds", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "in", "gen", "summary", "satisfies", "key", "encoding", "subscribers")
		if !mg.unique("이벤트 종류", id, pos) {
			return
		}
		ek := &EventKind{Owned: d.owned(id, pos, v), Encoding: d.oneOf(v, "encoding", "map", "map", "json"), Subscribers: d.refs(v, "subscribers")}
		if d.need(pos, v, "key") {
			ek.Key = d.ref(v, "key")
		}
		mg.m.EventKinds = append(mg.m.EventKinds, ek)
	})
}

func (mg *merger) events(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "events", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies", "kind", "publishers", "subscribers", "data")
		if !mg.unique("이벤트", id, pos) {
			return
		}
		e := &Event{Meta: d.meta(id, pos, v), Subscribers: d.refs(v, "subscribers")}
		if d.need(pos, v, "kind") {
			e.Kind = d.ref(v, "kind")
		}
		if d.need(pos, v, "publishers") {
			e.Publishers = d.refs(v, "publishers")
		}
		if dv := v.Get("data"); dv != nil && !dv.IsNull() {
			e.Data = mg.fields(dv)
		}
		mg.m.Events = append(mg.m.Events, e)
	})
}
