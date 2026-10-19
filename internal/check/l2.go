package check

import (
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

func (c *checker) l2() {
	c.tierDirection()
	c.selfDependency()
	c.contractRoles()
	c.cycles()
	c.siblingCycles()
	c.orphans()
	c.exceptions()
}

func (c *checker) tierDirection() {
	m := c.m
	check := func(fromID, fromTier string, deps []model.Ref) {
		fi := m.TierIndex(fromTier)
		if fi < 0 {
			return
		}
		for _, d := range deps {
			toTier := c.tierOf(d.Name)
			if m.TierIndex(toTier) > fi {
				c.errorf(d.Pos, "상위 tier 의존: %s(%s) → %s(%s)", fromID, fromTier, d.Name, toTier)
			}
		}
	}
	for _, mod := range m.Modules {
		check(mod.ID, mod.Tier.Name, mod.DependsOn)
		check(mod.ID, mod.Tier.Name, c.layerTargets(mod.Needs))
	}
	for _, g := range m.Groups {
		check(g.ID, g.Tier.Name, g.DependsOn)
		check(g.ID, g.Tier.Name, c.layerTargets(g.Needs))
		for _, mem := range g.Members {
			check(mem.ID, g.Tier.Name, c.layerTargets(mem.Needs))
			for _, e := range mem.Exceptions {
				check(mem.ID, g.Tier.Name, []model.Ref{e.DependsOn})
			}
		}
	}
}

func (c *checker) selfDependency() {
	report := func(id string, d model.Ref) {
		c.errorf(d.Pos, "자기 자신에 의존: %s", id)
	}
	for _, mod := range c.m.Modules {
		for _, d := range mod.DependsOn {
			if d.Name == mod.ID {
				report(mod.ID, d)
			}
		}
		for _, d := range c.needTargets(mod.Needs) {
			if d.Name == mod.ID {
				report(mod.ID, d)
			}
		}
	}
	for _, g := range c.m.Groups {
		for _, d := range g.DependsOn {
			if d.Name == g.ID {
				report(g.ID, d)
			}
		}
		for _, mem := range g.Members {
			for _, e := range mem.Exceptions {
				if e.DependsOn.Name == mem.ID || e.DependsOn.Name == g.ID {
					report(mem.ID, e.DependsOn)
				}
			}
			for _, d := range c.needTargets(mem.Needs) {
				if d.Name == mem.ID {
					report(mem.ID, d)
				}
			}
		}
	}
}

func (c *checker) cycles() {
	m := c.m
	var ids []string
	for _, mod := range m.Modules {
		ids = append(ids, mod.ID)
	}
	for _, g := range m.Groups {
		ids = append(ids, g.ID)
	}
	collapse := func(id string) string {
		if mem := m.Member(id); mem != nil {
			return mem.Group.ID
		}
		return id
	}
	edges := func(id string) []string {
		var out []string
		add := func(d string) {
			if d = collapse(d); d != id {
				out = append(out, d)
			}
		}
		for _, d := range c.layerDepsOf(id) {
			add(d)
		}
		if g := m.Group(id); g != nil {
			for _, mem := range g.Members {
				for _, e := range mem.Exceptions {
					add(e.DependsOn.Name)
				}
				for _, r := range c.layerTargets(mem.Needs) {
					add(r.Name)
				}
			}
		}
		return out
	}
	c.findCycles(ids, edges)
}

func (c *checker) siblingCycles() {
	for _, g := range c.m.Groups {
		var ids []string
		for _, mem := range g.Members {
			ids = append(ids, mem.ID)
		}
		edges := func(id string) []string {
			var out []string
			mem := c.m.Member(id)
			var targets []string
			for _, e := range mem.Exceptions {
				targets = append(targets, e.DependsOn.Name)
			}
			for _, r := range c.layerTargets(mem.Needs) {
				targets = append(targets, r.Name)
			}
			for _, d := range targets {
				if d != id && slices.Contains(ids, d) {
					out = append(out, d)
				}
			}
			return out
		}
		c.findCycles(ids, edges)
	}
}

func (c *checker) findCycles(ids []string, edges func(string) []string) {
	const (
		white = iota
		gray
		black
	)
	color := map[string]int{}
	var stack []string
	var visit func(id string)
	visit = func(id string) {
		color[id] = gray
		stack = append(stack, id)
		for _, d := range edges(id) {
			switch color[d] {
			case white:
				if slices.Contains(ids, d) {
					visit(d)
				}
			case gray:
				i := slices.Index(stack, d)
				cyc := append(slices.Clone(stack[i:]), d)
				c.errorf(c.unitPos(id), "의존 순환: %s", strings.Join(cyc, " → "))
			}
		}
		stack = stack[:len(stack)-1]
		color[id] = black
	}
	for _, id := range ids {
		if color[id] == white {
			visit(id)
		}
	}
}

func (c *checker) orphans() {
	m := c.m
	if len(m.Executables) == 0 {
		return
	}
	reach := map[string]bool{}
	for _, e := range m.Executables {
		for id := range c.reachable(e) {
			reach[id] = true
		}
	}
	for _, mod := range m.Modules {
		if !reach[mod.ID] {
			c.warnf(mod.Pos, "어느 실행 단위에도 없는 모듈 %s", mod.ID)
		}
	}
	for _, g := range m.Groups {
		if !reach[g.ID] {
			c.warnf(g.Pos, "어느 실행 단위에도 없는 그룹 %s", g.ID)
		}
	}
}

func (c *checker) exceptions() {
	for _, g := range c.m.Groups {
		for _, mem := range g.Members {
			for _, e := range mem.Exceptions {
				if has(g.DependsOn, e.DependsOn.Name) {
					c.warnf(e.DependsOn.Pos, "그룹이 이미 의존하는 예외: 그룹 %s, 예외 %s", g.ID, e.DependsOn.Name)
					continue
				}
				msg := "그룹 규칙 예외: " + mem.ID + " → " + e.DependsOn.Name
				if e.Reason != "" {
					msg += " (" + e.Reason + ")"
				}
				c.infof(e.DependsOn.Pos, "%s", msg)
			}
		}
	}
}

func (c *checker) contractRoles() {
	want := map[string]string{"port": "infra", "external": "adapter"}
	for _, ct := range c.m.Contracts {
		if o, ok := c.m.Owner(ct.In.Name); ct.Kind == "query" && ok && (o.Path == "query" || strings.HasSuffix(o.Path, "/query")) {
			c.errorf(ct.Pos, "query 계약의 모듈 경로가 /query로 끝난다: %s(%s). 구현이 %s/query에 생긴다", ct.ID, o.Path, o.Path)
		}
		role, ok := want[ct.Kind]
		if !ok {
			continue
		}
		if mod := c.m.Module(ct.In.Name); mod == nil || mod.Role != role {
			c.errorf(ct.Pos, "%s 계약의 모듈은 role이 %s여야 한다: 계약 %s, 모듈 %s", ct.Kind, role, ct.ID, ct.In.Name)
		}
	}
}
