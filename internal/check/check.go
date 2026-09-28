// Package check는 모델만 보고 설계 검사를 한다. 코드는 읽지 않는다.
package check

import (
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

type satisfier struct {
	id  string
	ref model.Ref
}

type checker struct {
	m          *model.Model
	diags      model.Diagnostics
	satisfiers map[string][]satisfier // 요구사항 ID → 충족한다고 적은 요소
}

// Run은 설계 검사 전부를 돌려 위치순으로 정렬한 진단을 준다.
func Run(m *model.Model) model.Diagnostics {
	c := &checker{m: m, satisfiers: map[string][]satisfier{}}
	c.eachMeta(func(meta *model.Meta) {
		for _, r := range meta.Satisfies {
			c.satisfiers[r.Name] = append(c.satisfiers[r.Name], satisfier{meta.ID, r})
		}
	})
	c.refs()
	c.l0()
	c.l2()
	c.l3()
	c.handlers()
	c.l4()
	c.requirements()
	c.diags.Sort()
	return c.diags
}

func (c *checker) errorf(pos yaml.Pos, format string, args ...any) {
	c.diags.Errorf(pos, format, args...)
}

func (c *checker) warnf(pos yaml.Pos, format string, args ...any) {
	c.diags.Warnf(pos, format, args...)
}

func (c *checker) infof(pos yaml.Pos, format string, args ...any) {
	c.diags.Infof(pos, format, args...)
}

// eachMeta는 satisfies를 가질 수 있는 모든 노드를 돈다.
func (c *checker) eachMeta(fn func(*model.Meta)) {
	m := c.m
	if m.System != nil {
		for _, a := range m.System.Actors {
			fn(&a.Meta)
		}
		for _, i := range m.System.Interfaces {
			fn(&i.Meta)
		}
	}
	for _, e := range m.Executables {
		fn(&e.Meta)
	}
	for _, x := range m.Modules {
		fn(&x.Meta)
	}
	for _, g := range m.Groups {
		fn(&g.Meta)
		for _, mem := range g.Members {
			fn(&mem.Meta)
		}
	}
	for _, x := range m.Enums {
		fn(&x.Meta)
	}
	for _, x := range m.Consts {
		fn(&x.Meta)
	}
	for _, x := range m.Errors {
		fn(&x.Meta)
	}
	for _, x := range m.Types {
		fn(&x.Meta)
	}
	for _, x := range m.Contracts {
		fn(&x.Meta)
	}
	for _, x := range m.HandlerKinds {
		fn(&x.Meta)
	}
	for _, x := range m.Funcs {
		fn(&x.Meta)
	}
}

func has(rs []model.Ref, name string) bool {
	for _, r := range rs {
		if r.Name == name {
			return true
		}
	}
	return false
}

// depsOf는 id가 설계상 의존하는 ID 목록이다. 멤버는 그룹의 의존과 자기 예외를 따른다.
func (c *checker) depsOf(id string) []string {
	m := c.m
	var out []string
	add := func(rs []model.Ref) {
		for _, r := range rs {
			out = append(out, r.Name)
		}
	}
	switch {
	case m.Module(id) != nil:
		add(m.Module(id).DependsOn)
	case m.Group(id) != nil:
		add(m.Group(id).DependsOn)
	case m.Member(id) != nil:
		mem := m.Member(id)
		add(mem.Group.DependsOn)
		for _, e := range mem.Exceptions {
			out = append(out, e.DependsOn.Name)
		}
	}
	return out
}

// unitPos는 모듈·그룹·멤버 ID의 정의 위치다.
func (c *checker) unitPos(id string) yaml.Pos {
	m := c.m
	switch {
	case m.Module(id) != nil:
		return m.Module(id).Pos
	case m.Group(id) != nil:
		return m.Group(id).Pos
	case m.Member(id) != nil:
		return m.Member(id).Pos
	}
	return yaml.Pos{}
}

func (c *checker) tierOf(id string) string {
	m := c.m
	switch {
	case m.Module(id) != nil:
		return m.Module(id).Tier.Name
	case m.Group(id) != nil:
		return m.Group(id).Tier.Name
	case m.Member(id) != nil:
		return m.Member(id).Group.Tier.Name
	}
	return ""
}
