package check

import "github.com/tolelom/daemok/internal/model"

// l0은 인터페이스마다 구현 모듈이 있고, 노출하는 실행 단위가 그 모듈을 조립하는지 본다.
func (c *checker) l0() {
	m := c.m
	if m.System == nil {
		return
	}
	implementers := map[string][]string{}
	for _, mod := range m.Modules {
		for _, r := range mod.Implements {
			implementers[r.Name] = append(implementers[r.Name], mod.ID)
		}
	}
	for _, it := range m.System.Interfaces {
		if len(implementers[it.ID]) == 0 {
			c.errorf(it.Pos, "인터페이스 %s를 구현하는 모듈이 없다", it.ID)
		}
	}
	for _, e := range m.Executables {
		reach := c.reachable(e)
		for _, r := range e.Exposes {
			for _, impl := range implementers[r.Name] {
				if !reach[impl] {
					c.errorf(r.Pos, "실행 단위 %s가 %s를 노출하지만 구현 모듈 %s를 조립하지 않는다", e.ID, r.Name, impl)
				}
			}
		}
	}
}

// reachable은 composes에서 의존을 따라 닿는 모듈·그룹·멤버 ID 집합이다. 그룹은 멤버를 포함한다.
func (c *checker) reachable(e *model.Executable) map[string]bool {
	seen := map[string]bool{}
	var visit func(id string)
	visit = func(id string) {
		if seen[id] {
			return
		}
		seen[id] = true
		for _, d := range c.depsOf(id) {
			visit(d)
		}
		if g := c.m.Group(id); g != nil {
			for _, mem := range g.Members {
				visit(mem.ID)
			}
		}
	}
	for _, r := range e.Composes {
		visit(r.Name)
	}
	return seen
}
