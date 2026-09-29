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
	}
	for _, g := range m.Groups {
		c.tier(g.Tier)
		if g.Kind.Name != "" && m.HandlerKind(g.Kind.Name) == nil {
			c.errorf(g.Kind.Pos, "없는 핸들러 종류 %q", g.Kind.Name)
		}
		for _, r := range g.DependsOn {
			c.dep(r)
		}
		for _, mem := range g.Members {
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
		for _, f := range t.Fields {
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
			c.ofRef(f.Of)
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

func (c *checker) ofRef(r model.Ref) {
	ct, meth, ok := strings.Cut(r.Name, ".")
	if !ok {
		c.errorf(r.Pos, "of는 계약.메서드 형식이어야 한다")
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
