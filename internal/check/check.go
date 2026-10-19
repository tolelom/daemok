package check

import (
	"slices"

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
	satisfiers map[string][]satisfier
}

func Run(m *model.Model) model.Diagnostics {
	c := &checker{m: m, satisfiers: map[string][]satisfier{}}
	c.eachMeta(func(meta *model.Meta) {
		for _, r := range meta.Satisfies {
			c.satisfiers[r.Name] = append(c.satisfiers[r.Name], satisfier{meta.ID, r})
		}
	})
	c.refs()
	c.names()
	c.l0()
	c.l2()
	c.l3()
	c.handlers()
	c.events()
	c.l4()
	c.flows()
	c.requirements()
	c.ts()
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
	for _, x := range m.EventKinds {
		fn(&x.Meta)
	}
	for _, x := range m.Events {
		fn(&x.Meta)
	}
	for _, x := range m.Funcs {
		fn(&x.Meta)
	}
	for _, x := range m.Flows {
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

func (c *checker) depsOf(id string) []string { return c.deps(id, c.needTargets) }

func (c *checker) layerDepsOf(id string) []string { return c.deps(id, c.layerTargets) }

func (c *checker) deps(id string, targets func([]model.Need) []model.Ref) []string {
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
	for _, r := range targets(c.needsOf(id)) {
		out = append(out, r.Name)
	}
	return out
}

func (c *checker) ownNeeds(id string) []model.Need {
	m := c.m
	switch {
	case m.Module(id) != nil:
		return m.Module(id).Needs
	case m.Group(id) != nil:
		return m.Group(id).Needs
	case m.Member(id) != nil:
		return m.Member(id).Needs
	}
	return nil
}

func (c *checker) needsOf(id string) []model.Need {
	if mem := c.m.Member(id); mem != nil {
		return append(slices.Clone(mem.Group.Needs), mem.Needs...)
	}
	return c.ownNeeds(id)
}

func (c *checker) layerTargets(ns []model.Need) []model.Ref {
	var kept []model.Need
	for _, n := range ns {
		if ct := c.m.Contract(n.Contract.Name); ct == nil || ct.Kind != "query" {
			kept = append(kept, n)
		}
	}
	return c.needTargets(kept)
}

func (c *checker) needTargets(ns []model.Need) []model.Ref {
	var out []model.Ref
	for _, n := range ns {
		if ct := c.m.Contract(n.Contract.Name); ct != nil && ct.In.Name != "" {
			out = append(out, model.Ref{Name: ct.In.Name, Pos: n.Contract.Pos})
		}
	}
	return out
}

func (c *checker) unitIDs() []string {
	var ids []string
	for _, mod := range c.m.Modules {
		ids = append(ids, mod.ID)
	}
	for _, g := range c.m.Groups {
		ids = append(ids, g.ID)
		for _, mem := range g.Members {
			ids = append(ids, mem.ID)
		}
	}
	return ids
}

func (c *checker) isUnit(id string) bool {
	return c.m.Module(id) != nil || c.m.Group(id) != nil || c.m.Member(id) != nil
}

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
