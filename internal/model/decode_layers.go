package model

import "github.com/tolelom/daemok/internal/yaml"

func (mg *merger) requirements(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "requirements", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "text", "status", "summary")
		if !mg.unique("요구사항", id, pos) {
			return
		}
		r := &Requirement{Meta: Meta{ID: id, Pos: pos, Summary: d.str(v, "summary")}}
		if d.need(pos, v, "text") {
			r.Text = d.str(v, "text")
		}
		r.Status = d.str(v, "status")
		if r.Status == "" {
			r.Status = "active"
		}
		switch r.Status {
		case "active", "held", "dropped":
		default:
			d.diags.Errorf(v.Get("status").Pos, "status는 active, held, dropped 중 하나여야 한다")
		}
		mg.m.Requirements = append(mg.m.Requirements, r)
	})
}

func (mg *merger) cpp(n *yaml.Node) {
	if n == nil {
		return
	}
	d := mg.d
	if !d.mapping(n, "cpp") {
		return
	}
	if mg.m.CPP != nil {
		d.diags.Errorf(n.Pos, "cpp는 한 파일에만 있어야 한다 (처음: %s)", mg.m.CPP.Pos)
		return
	}
	d.allow(n, "include_dirs")
	mg.m.CPP = &CPP{IncludeDirs: d.refs(n, "include_dirs"), Pos: n.Pos}
}

func (mg *merger) system(n *yaml.Node) {
	if n == nil {
		return
	}
	d := mg.d
	if !d.mapping(n, "system") {
		return
	}
	if mg.m.System != nil {
		d.diags.Errorf(n.Pos, "system은 한 파일에만 있어야 한다 (처음: %s)", mg.m.System.Pos)
		return
	}
	d.allow(n, "id", "summary", "actors", "interfaces")
	s := &System{Meta: Meta{ID: d.str(n, "id"), Pos: n.Pos, Summary: d.str(n, "summary")}}
	if s.ID == "" {
		d.diags.Errorf(n.Pos, "system에 id가 필요하다")
	}
	d.idMap(n.Get("actors"), "actors", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies")
		if mg.unique("액터", id, pos) {
			s.Actors = append(s.Actors, &Actor{Meta: d.meta(id, pos, v)})
		}
	})
	d.idMap(n.Get("interfaces"), "interfaces", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies", "protocol", "port", "path", "spec", "used_by")
		if !mg.unique("인터페이스", id, pos) {
			return
		}
		it := &Interface{
			Meta:     d.meta(id, pos, v),
			Protocol: d.str(v, "protocol"),
			Path:     d.str(v, "path"),
			Spec:     d.str(v, "spec"),
			UsedBy:   d.refs(v, "used_by"),
		}
		it.Port, _ = d.intv(v, "port")
		s.Interfaces = append(s.Interfaces, it)
	})
	mg.m.System = s
}

func (mg *merger) executables(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "executables", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies", "exposes", "composes")
		if !mg.unique("실행 단위", id, pos) {
			return
		}
		mg.m.Executables = append(mg.m.Executables, &Executable{
			Meta:     d.meta(id, pos, v),
			Exposes:  d.refs(v, "exposes"),
			Composes: d.refs(v, "composes"),
		})
	})
}

func (mg *merger) tiers(n *yaml.Node) {
	if n == nil {
		return
	}
	d := mg.d
	if prev, ok := mg.once["tiers"]; ok {
		d.diags.Errorf(n.Pos, "tiers는 한 파일에만 있어야 한다 (처음: %s)", prev)
		return
	}
	mg.once["tiers"] = n.Pos
	if n.Kind != yaml.Seq {
		d.diags.Errorf(n.Pos, "tiers는 목록이어야 한다")
		return
	}
	for _, it := range n.Items {
		if it.Kind != yaml.Scalar {
			d.diags.Errorf(it.Pos, "tier 이름은 값 하나여야 한다")
			continue
		}
		if mg.unique("tier", it.Value, it.Pos) {
			mg.m.Tiers = append(mg.m.Tiers, Ref{Name: it.Value, Pos: it.Pos})
		}
	}
}

func (mg *merger) modules(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "modules", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies", "tier", "path", "role", "depends_on", "implements", "needs")
		if !mg.unique("모듈", id, pos) {
			return
		}
		m := &Module{
			Meta:       d.meta(id, pos, v),
			Path:       d.str(v, "path"),
			Role:       d.oneOf(v, "role", "", "data", "infra", "adapter"),
			DependsOn:  d.refs(v, "depends_on"),
			Implements: d.refs(v, "implements"),
			Needs:      mg.needs(v.Get("needs")),
		}
		if d.need(pos, v, "tier") {
			m.Tier = d.ref(v, "tier")
		}
		if m.Path == "" {
			m.Path = id
		}
		mg.m.Modules = append(mg.m.Modules, m)
	})
}

func (mg *merger) groups(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "groups", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "summary", "satisfies", "tier", "kind", "path", "depends_on", "needs", "rule", "members")
		if !mg.unique("그룹", id, pos) {
			return
		}
		g := &Group{
			Meta:      d.meta(id, pos, v),
			Path:      d.str(v, "path"),
			DependsOn: d.refs(v, "depends_on"),
			Needs:     mg.needs(v.Get("needs")),
			Rule:      d.str(v, "rule"),
		}
		if d.need(pos, v, "tier") {
			g.Tier = d.ref(v, "tier")
		}
		if d.need(pos, v, "kind") {
			g.Kind = d.ref(v, "kind")
		}
		if g.Path == "" {
			g.Path = id
		}
		if g.Rule != "" && g.Rule != "no_sibling_deps" {
			d.diags.Errorf(v.Get("rule").Pos, "rule은 no_sibling_deps만 지원한다")
		}
		if d.need(pos, v, "members") {
			d.idMap(v.Get("members"), "members", func(mid string, mpos yaml.Pos, mv *yaml.Node) {
				d.allow(mv, "summary", "satisfies", "handles", "exceptions", "needs")
				if !mg.unique("멤버", mid, mpos) {
					return
				}
				mem := &Member{Meta: d.meta(mid, mpos, mv), Group: g, Handles: d.refs(mv, "handles")}
				mem.Exceptions = mg.exceptions(mv.Get("exceptions"))
				mem.Needs = mg.needs(mv.Get("needs"))
				g.Members = append(g.Members, mem)
			})
		}
		mg.m.Groups = append(mg.m.Groups, g)
	})
}

func (mg *merger) exceptions(n *yaml.Node) []Exception {
	if n == nil || n.IsNull() {
		return nil
	}
	d := mg.d
	if n.Kind != yaml.Seq {
		d.diags.Errorf(n.Pos, "exceptions는 목록이어야 한다")
		return nil
	}
	var out []Exception
	for _, it := range n.Items {
		if !d.mapping(it, "예외") {
			continue
		}
		d.allow(it, "depends_on", "reason")
		if !d.need(it.Pos, it, "depends_on") {
			continue
		}
		out = append(out, Exception{DependsOn: d.ref(it, "depends_on"), Reason: d.str(it, "reason")})
	}
	return out
}

func (mg *merger) needs(n *yaml.Node) []Need {
	if n == nil || n.IsNull() {
		return nil
	}
	d := mg.d
	if n.Kind != yaml.Seq {
		d.diags.Errorf(n.Pos, "목록이어야 한다: %q", "needs")
		return nil
	}
	var out []Need
	for _, it := range n.Items {
		switch {
		case it.Kind == yaml.Scalar && !it.IsNull():
			out = append(out, Need{Contract: Ref{Name: it.Value, Pos: it.Pos}, Pos: it.Pos})
		case it.Kind == yaml.Map:
			d.allow(it, "contract", "uses", "why")
			if !d.need(it.Pos, it, "contract") {
				continue
			}
			out = append(out, Need{Contract: d.ref(it, "contract"), Uses: d.refs(it, "uses"), Why: d.str(it, "why"), Pos: it.Pos})
		default:
			d.diags.Errorf(it.Pos, "계약 ID나 맵이어야 한다: needs 항목")
		}
	}
	return out
}
