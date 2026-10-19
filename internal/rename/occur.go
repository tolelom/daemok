package rename

import (
	"fmt"
	"strings"

	"github.com/tolelom/daemok/internal/edit"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

type collector struct {
	old, new string
	reps     []edit.Replacement
}

func (c *collector) at(pos yaml.Pos, old, new string) {
	c.reps = append(c.reps, edit.Replacement{Pos: pos, Old: old, New: new})
}

func (c *collector) ref(r model.Ref) {
	if r.Name == c.old {
		c.at(r.Pos, c.old, c.new)
	}
}

func (c *collector) refs(rs []model.Ref) {
	for _, r := range rs {
		c.ref(r)
	}
}

func (c *collector) typ(r model.Ref) {
	if r.Name == "" {
		return
	}
	if n := retype(r.Name, c.old, c.new); n != r.Name {
		c.at(r.Pos, r.Name, n)
	}
}

func (c *collector) method(r model.Ref) {
	if ct, meth, ok := strings.Cut(r.Name, "."); ok && ct == c.old {
		c.at(r.Pos, r.Name, c.new+"."+meth)
	}
}

func retype(s, old, new string) string {
	var b strings.Builder
	tok := 0
	flush := func(end int) {
		if s[tok:end] == old {
			b.WriteString(new)
		} else {
			b.WriteString(s[tok:end])
		}
	}
	for i := 0; i < len(s); i++ {
		if strings.IndexByte("[]*,", s[i]) >= 0 {
			flush(i)
			b.WriteByte(s[i])
			tok = i + 1
		}
	}
	flush(len(s))
	return b.String()
}

func definition(m *model.Model, kind, id string) (yaml.Pos, bool) {
	if kind == "tiers" {
		for _, t := range m.Tiers {
			if t.Name == id {
				return t.Pos, true
			}
		}
		return yaml.Pos{}, false
	}
	for _, meta := range metas(m, kind) {
		if meta.ID == id {
			return meta.Pos, true
		}
	}
	return yaml.Pos{}, false
}

func Occurrences(m *model.Model, kind, old, new string) ([]edit.Replacement, error) {
	def, ok := definition(m, kind, old)
	if !ok {
		return nil, fmt.Errorf("없는 ID: %s %q", kind, old)
	}
	c := &collector{old: old, new: new}
	c.at(def, old, new)
	references(m, c, kind)
	return c.reps, nil
}

func references(m *model.Model, c *collector, kind string) {
	switch kind {
	case "requirements":
		requirementRefs(m, c)
	case "actors":
		actorRefs(m, c)
	case "interfaces":
		interfaceRefs(m, c)
	case "tiers":
		tierRefs(m, c)
	case "modules", "groups", "members":
		deps(m, c)
		if kind != "groups" {
			eachOwned(m, func(o *model.Owned) { c.ref(o.In) })
		}
	case "enums", "types":
		signatures(m, c, kind)
	case "consts":
		eachField(m, func(f *model.Field) { c.ref(f.Max) })
	case "errors":
		errorRefs(m, c)
	case "contracts":
		contractRefs(m, c)
	case "handler_kinds":
		handlerKindRefs(m, c)
	case "event_kinds":
		for _, e := range m.Events {
			c.ref(e.Kind)
		}
	case "events":
		eventRefs(m, c)
	}
}

func requirementRefs(m *model.Model, c *collector) {
	for k := range kinds {
		for _, meta := range metas(m, k) {
			c.refs(meta.Satisfies)
		}
	}
	if m.System != nil {
		c.refs(m.System.Satisfies)
	}
}

func actorRefs(m *model.Model, c *collector) {
	if m.System != nil {
		for _, it := range m.System.Interfaces {
			c.refs(it.UsedBy)
		}
	}
}

func interfaceRefs(m *model.Model, c *collector) {
	for _, e := range m.Executables {
		c.refs(e.Exposes)
	}
	for _, mod := range m.Modules {
		c.refs(mod.Implements)
	}
}

func tierRefs(m *model.Model, c *collector) {
	for _, mod := range m.Modules {
		c.ref(mod.Tier)
	}
	for _, g := range m.Groups {
		c.ref(g.Tier)
	}
}

func errorRefs(m *model.Model, c *collector) {
	for _, ct := range m.Contracts {
		for _, me := range ct.Methods {
			c.refs(me.Errors)
		}
	}
	for _, f := range m.Funcs {
		c.refs(f.Errors)
		for _, cs := range f.Cases {
			if cs.Error.Name != "invalid" {
				c.ref(cs.Error)
			}
		}
	}
}

func contractRefs(m *model.Model, c *collector) {
	for _, mod := range m.Modules {
		needs(mod.Needs, c)
	}
	for _, g := range m.Groups {
		needs(g.Needs, c)
		for _, mem := range g.Members {
			needs(mem.Needs, c)
		}
	}
	for _, f := range m.Funcs {
		c.method(f.Of)
	}
	for _, fl := range m.Flows {
		for _, s := range fl.Steps {
			for _, r := range s.Calls {
				c.method(r)
			}
		}
	}
}

func handlerKindRefs(m *model.Model, c *collector) {
	for _, g := range m.Groups {
		c.ref(g.Kind)
	}
	for _, f := range m.Funcs {
		c.ref(f.Kind)
	}
}

func eventRefs(m *model.Model, c *collector) {
	for _, e := range m.Events {
		if e.ID != c.old {
			continue
		}
		if ek := m.EventKind(e.Kind.Name); ek != nil {
			if en := m.Enum(ek.Key.Name); en != nil {
				c.refs(en.Values)
			}
		}
	}
	for _, fl := range m.Flows {
		c.ref(fl.On)
		for _, s := range fl.Steps {
			c.refs(s.Emits)
		}
	}
}

func signatures(m *model.Model, c *collector, kind string) {
	eachField(m, func(f *model.Field) {
		c.typ(f.Type)
		if kind == "types" {
			c.ref(f.RefType)
		}
	})
	params := func(ps []*model.Param, returns []model.Ref) {
		for _, p := range ps {
			c.typ(p.Type)
		}
		for _, r := range returns {
			c.typ(r)
		}
	}
	for _, ct := range m.Contracts {
		for _, me := range ct.Methods {
			params(me.Params, me.Returns)
		}
	}
	for _, hk := range m.HandlerKinds {
		params(hk.Params, hk.Returns)
		if kind == "enums" {
			c.ref(hk.Key)
		}
	}
	if kind == "enums" {
		for _, ek := range m.EventKinds {
			c.ref(ek.Key)
		}
	}
}

func needs(ns []model.Need, c *collector) {
	for _, n := range ns {
		c.ref(n.Contract)
	}
}

func deps(m *model.Model, c *collector) {
	for _, e := range m.Executables {
		c.refs(e.Composes)
	}
	for _, mod := range m.Modules {
		c.refs(mod.DependsOn)
	}
	for _, g := range m.Groups {
		c.refs(g.DependsOn)
		for _, mem := range g.Members {
			for _, x := range mem.Exceptions {
				c.ref(x.DependsOn)
			}
		}
	}
	for _, ek := range m.EventKinds {
		c.refs(ek.Subscribers)
	}
	for _, e := range m.Events {
		c.refs(e.Publishers)
		c.refs(e.Subscribers)
	}
}

func eachOwned(m *model.Model, fn func(*model.Owned)) {
	for _, x := range m.Enums {
		fn(&x.Owned)
	}
	for _, x := range m.Consts {
		fn(&x.Owned)
	}
	for _, x := range m.Errors {
		fn(&x.Owned)
	}
	for _, x := range m.Types {
		fn(&x.Owned)
	}
	for _, x := range m.Contracts {
		fn(&x.Owned)
	}
	for _, x := range m.HandlerKinds {
		fn(&x.Owned)
	}
	for _, x := range m.EventKinds {
		fn(&x.Owned)
	}
	for _, x := range m.Funcs {
		fn(&x.Owned)
	}
	for _, x := range m.Flows {
		fn(&x.Owned)
	}
}

func eachField(m *model.Model, fn func(*model.Field)) {
	for _, t := range m.Types {
		for _, f := range t.Fields {
			fn(f)
		}
	}
	for _, e := range m.Events {
		for _, f := range e.Data {
			fn(f)
		}
	}
}
