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

// ref는 이름이 옛 ID인 참조를 바꾼다.
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

// typ은 타입 문자열 안의 옛 ID 토큰을 바꾼다.
func (c *collector) typ(r model.Ref) {
	if r.Name == "" {
		return
	}
	if n := retype(r.Name, c.old, c.new); n != r.Name {
		c.at(r.Pos, r.Name, n)
	}
}

// method는 "계약.메서드" 참조의 계약 부분을 바꾼다.
func (c *collector) method(r model.Ref) {
	if ct, meth, ok := strings.Cut(r.Name, "."); ok && ct == c.old {
		c.at(r.Pos, r.Name, c.new+"."+meth)
	}
}

// retype은 타입 문자열에서 식별자 토큰이 정확히 old인 것만 new로 바꾼다. 경계는 [ ] * , 와 문자열 끝이다.
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

// definition은 kind의 id 노드의 ID 키 위치다.
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

// Occurrences는 kind의 old 노드 정의 키와 그 노드를 가리키는 모든 참조 자리를 준다.
// 참조의 종류는 check의 해석을 따른다.
func Occurrences(m *model.Model, kind, old, new string) ([]edit.Replacement, error) {
	def, ok := definition(m, kind, old)
	if !ok {
		return nil, fmt.Errorf("없는 ID: %s %q", kind, old)
	}
	c := &collector{old: old, new: new}
	c.at(def, old, new)
	switch kind {
	case "requirements":
		for k := range kinds {
			for _, meta := range metas(m, k) {
				c.refs(meta.Satisfies)
			}
		}
		if m.System != nil {
			c.refs(m.System.Satisfies)
		}
	case "actors":
		if m.System != nil {
			for _, it := range m.System.Interfaces {
				c.refs(it.UsedBy)
			}
		}
	case "interfaces":
		for _, e := range m.Executables {
			c.refs(e.Exposes)
		}
		for _, mod := range m.Modules {
			c.refs(mod.Implements)
		}
	case "tiers":
		for _, mod := range m.Modules {
			c.ref(mod.Tier)
		}
		for _, g := range m.Groups {
			c.ref(g.Tier)
		}
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
	case "contracts":
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
	case "handler_kinds":
		for _, g := range m.Groups {
			c.ref(g.Kind)
		}
		for _, f := range m.Funcs {
			c.ref(f.Kind)
		}
	case "event_kinds":
		for _, e := range m.Events {
			c.ref(e.Kind)
		}
	case "events":
		// 이벤트 종류의 key enum 값은 이벤트 ID다.
		for _, e := range m.Events {
			if e.ID != old {
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
	return c.reps, nil
}

// signatures는 enum·타입을 가리키는 타입 문자열과 ref, key다.
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

// deps는 모듈·그룹·멤버를 함께 가리키는 참조다. 셋은 ID가 겹치지 않는다.
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
