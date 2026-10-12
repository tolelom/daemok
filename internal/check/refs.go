package check

import (
	"strconv"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// refs는 모든 참조 필드가 존재하는, 받을 수 있는 종류의 대상을 가리키는지 본다.
func (c *checker) refs() {
	m := c.m
	c.eachMeta(func(meta *model.Meta) {
		for _, r := range meta.Satisfies {
			if m.Requirement(r.Name) == nil {
				c.errorf(r.Pos, "없는 요구사항 %q", r.Name)
			}
		}
	})
	if m.System != nil {
		for _, it := range m.System.Interfaces {
			for _, r := range it.UsedBy {
				if m.Actor(r.Name) == nil {
					c.errorf(r.Pos, "없는 액터 %q", r.Name)
				}
			}
		}
	}
	for _, e := range m.Executables {
		for _, r := range e.Exposes {
			c.iface(r)
		}
		for _, r := range e.Composes {
			c.dep(r)
		}
	}
	for _, mod := range m.Modules {
		c.tier(mod.Tier)
		for _, r := range mod.DependsOn {
			c.dep(r)
		}
		for _, r := range mod.Implements {
			c.iface(r)
		}
		c.needRefs(mod.ID, mod.Needs)
	}
	for _, g := range m.Groups {
		c.tier(g.Tier)
		if g.Kind.Name != "" && m.HandlerKind(g.Kind.Name) == nil {
			c.errorf(g.Kind.Pos, "없는 핸들러 종류 %q", g.Kind.Name)
		}
		for _, r := range g.DependsOn {
			c.dep(r)
		}
		c.needRefs(g.ID, g.Needs)
		for _, mem := range g.Members {
			c.needRefs(mem.ID, mem.Needs)
			if g.Kind.Name == "" && len(mem.Handles) > 0 {
				c.errorf(mem.Handles[0].Pos, "핸들러 종류가 없는 그룹의 멤버에 handles가 있다: %s", mem.ID)
			}
			for _, r := range mem.Handles {
				c.handleValue(g.Kind, r)
			}
			for _, e := range mem.Exceptions {
				c.dep(e.DependsOn)
			}
		}
	}
	for _, x := range m.Enums {
		c.in(x.In)
	}
	for _, x := range m.Consts {
		c.in(x.In)
	}
	for _, x := range m.Errors {
		c.in(x.In)
	}
	for _, t := range m.Types {
		c.in(t.In)
		c.fieldRefs(t.Fields)
	}
	for _, ct := range m.Contracts {
		c.in(ct.In)
		for _, me := range ct.Methods {
			c.sig(me.Params, me.Returns)
			for _, r := range me.Errors {
				c.errRef(r)
			}
		}
	}
	for _, hk := range m.HandlerKinds {
		c.in(hk.In)
		c.sig(hk.Params, hk.Returns)
		if hk.Key.Name != "" && m.Enum(hk.Key.Name) == nil {
			c.errorf(hk.Key.Pos, "없는 enum %q", hk.Key.Name)
		}
	}
	for _, f := range m.Funcs {
		c.in(f.In)
		if f.Kind.Name != "" {
			if m.HandlerKind(f.Kind.Name) == nil {
				c.errorf(f.Kind.Pos, "없는 핸들러 종류 %q", f.Kind.Name)
			} else if f.Handles.Name != "" {
				c.handleValue(f.Kind, f.Handles)
			}
		}
		if f.Of.Name != "" {
			c.methodRef(f.Of, "of")
		}
		for _, r := range f.Errors {
			c.errRef(r)
		}
		for _, cs := range f.Cases {
			if cs.Error.Name != "" && cs.Error.Name != "invalid" {
				c.errRef(cs.Error)
			}
		}
	}
	for _, ek := range m.EventKinds {
		c.in(ek.In)
		if ek.Key.Name != "" && m.Enum(ek.Key.Name) == nil {
			c.errorf(ek.Key.Pos, "없는 enum %q", ek.Key.Name)
		}
		for _, r := range ek.Subscribers {
			c.pubSub(r, "subscribers")
		}
	}
	for _, e := range m.Events {
		if e.Kind.Name != "" && m.EventKind(e.Kind.Name) == nil {
			c.errorf(e.Kind.Pos, "없는 이벤트 종류 %q", e.Kind.Name)
		}
		for _, r := range e.Publishers {
			c.pubSub(r, "publishers")
		}
		for _, r := range e.Subscribers {
			c.pubSub(r, "subscribers")
		}
		c.fieldRefs(e.Data)
	}
	for _, f := range m.Flows {
		c.in(f.In)
		if f.On.Name != "" {
			c.eventRef(f.On)
		}
		for _, s := range f.Steps {
			for _, r := range s.Calls {
				c.methodRef(r, "calls")
			}
			for _, r := range s.Emits {
				c.eventRef(r)
			}
		}
	}
}

func (c *checker) fieldRefs(fields []*model.Field) {
	m := c.m
	for _, f := range fields {
		if f.Type.Name != "" {
			c.typeRef(f.Type)
		}
		if f.RefType.Name != "" && m.Type(f.RefType.Name) == nil {
			c.errorf(f.RefType.Pos, "없는 타입 %q", f.RefType.Name)
		}
		if f.Max.Name != "" {
			if _, err := strconv.Atoi(f.Max.Name); err != nil && m.Const(f.Max.Name) == nil {
				c.errorf(f.Max.Pos, "max는 상수 ID나 숫자여야 한다: %q", f.Max.Name)
			}
		}
	}
}

// pubSub은 발행자·구독자가 모듈이나 그룹 멤버인지 본다.
func (c *checker) pubSub(r model.Ref, field string) {
	if c.m.Group(r.Name) != nil {
		c.errorf(r.Pos, "모듈이나 그룹 멤버가 아니다: %s %s", field, r.Name)
		return
	}
	c.dep(r)
}

func (c *checker) needRefs(owner string, ns []model.Need) {
	seen := map[string]bool{}
	for _, n := range ns {
		if seen[n.Contract.Name] {
			c.warnf(n.Contract.Pos, "같은 계약을 두 번 받는다: %s → %s", owner, n.Contract.Name)
		}
		seen[n.Contract.Name] = true
		if n.Uses != nil && len(n.Uses) == 0 {
			c.errorf(n.Pos, "uses가 비어 있다: %s", n.Contract.Name)
		}
		ct := c.m.Contract(n.Contract.Name)
		if ct == nil {
			c.errorf(n.Contract.Pos, "없는 계약 %q", n.Contract.Name)
			continue
		}
		for _, u := range n.Uses {
			if ct.Method(u.Name) == nil {
				c.errorf(u.Pos, "계약 %s에 없는 메서드 %q", ct.ID, u.Name)
			}
		}
	}
}

func (c *checker) eventRef(r model.Ref) {
	if c.m.Event(r.Name) == nil {
		c.errorf(r.Pos, "없는 이벤트 %q", r.Name)
	}
}

func (c *checker) iface(r model.Ref) {
	if c.m.Interface(r.Name) == nil {
		c.errorf(r.Pos, "없는 인터페이스 %q", r.Name)
	}
}

func (c *checker) dep(r model.Ref) {
	m := c.m
	if m.Module(r.Name) == nil && m.Group(r.Name) == nil && m.Member(r.Name) == nil {
		c.errorf(r.Pos, "없는 모듈 %q", r.Name)
	}
}

func (c *checker) tier(r model.Ref) {
	if r.Name != "" && c.m.TierIndex(r.Name) < 0 {
		c.errorf(r.Pos, "없는 tier %q", r.Name)
	}
}

func (c *checker) in(r model.Ref) {
	if r.Name == "" {
		return
	}
	if _, ok := c.m.Owner(r.Name); !ok {
		c.errorf(r.Pos, "in이 가리키는 모듈이나 멤버가 없다: %q", r.Name)
	}
}

func (c *checker) typeRef(r model.Ref) {
	if k, base := c.m.ResolveType(r.Name); k == model.UnknownType {
		c.errorf(r.Pos, "알 수 없는 타입 %q", base)
	}
}

func (c *checker) sig(params []*model.Param, returns []model.Ref) {
	for _, p := range params {
		c.typeRef(p.Type)
	}
	for _, r := range returns {
		c.typeRef(r)
	}
}

func (c *checker) errRef(r model.Ref) {
	if c.m.Error(r.Name) == nil {
		c.errorf(r.Pos, "없는 에러 %q", r.Name)
	}
}

// handleValue는 handles 값이 핸들러 종류의 키 enum에 있는지 본다.
// 핸들러 종류나 enum이 없으면 다른 곳에서 이미 오류이므로 조용히 넘긴다.
func (c *checker) handleValue(kind model.Ref, r model.Ref) {
	hk := c.m.HandlerKind(kind.Name)
	if hk == nil {
		return
	}
	en := c.m.Enum(hk.Key.Name)
	if en == nil {
		return
	}
	if !en.HasValue(r.Name) {
		c.errorf(r.Pos, "%s에 없는 값 %q", en.ID, r.Name)
	}
}

func (c *checker) methodRef(r model.Ref, field string) {
	ct, meth, ok := strings.Cut(r.Name, ".")
	if !ok {
		c.errorf(r.Pos, "계약.메서드 형식이어야 한다: %s %q", field, r.Name)
		return
	}
	contract := c.m.Contract(ct)
	if contract == nil {
		c.errorf(r.Pos, "없는 계약 %q", ct)
		return
	}
	if contract.Method(meth) == nil {
		c.errorf(r.Pos, "계약 %s에 없는 메서드 %q", ct, meth)
	}
}
