package check

import "github.com/tolelom/daemok/internal/model"

// events는 key enum 값마다 이벤트가 있는지, 발행자·구독자가 이벤트 종류의 모듈에 의존하는지 본다.
func (c *checker) events() {
	m := c.m
	for _, ek := range m.EventKinds {
		if en := m.Enum(ek.Key.Name); en != nil {
			for _, v := range en.Values {
				if e := m.Event(v.Name); e == nil || e.Kind.Name != ek.ID {
					c.errorf(v.Pos, "이벤트가 없는 값: %s.%s", en.ID, v.Name)
				}
			}
		}
		for _, r := range ek.Subscribers {
			c.dependsOnKind(r, ek.In.Name)
		}
	}
	for _, e := range m.Events {
		ek := m.EventKind(e.Kind.Name)
		if ek == nil {
			continue
		}
		if en := m.Enum(ek.Key.Name); en != nil && !en.HasValue(e.ID) {
			c.errorf(e.Pos, "%s에 없는 값 %q", en.ID, e.ID)
		}
		if len(e.Publishers) == 0 {
			c.errorf(e.Pos, "발행자가 없는 이벤트: %s", e.ID)
		}
		for _, r := range e.Publishers {
			c.dependsOnKind(r, ek.In.Name)
		}
		for _, r := range e.Subscribers {
			c.dependsOnKind(r, ek.In.Name)
		}
		for _, f := range e.Data {
			c.useType(ek.In.Name, f.Type)
			c.useType(ek.In.Name, f.RefType)
		}
	}
}

// dependsOnKind는 발행자·구독자가 이벤트 종류의 모듈에 의존하는지 본다.
// 없는 모듈은 참조 검사가 이미 잡으므로 넘긴다.
func (c *checker) dependsOnKind(r model.Ref, kindModule string) {
	if kindModule == "" || !c.isUnit(r.Name) {
		return
	}
	if !c.m.CanDepend(r.Name, kindModule) {
		c.errorf(r.Pos, "이벤트 종류의 모듈에 의존하지 않는다: %s → %s", r.Name, kindModule)
	}
}
