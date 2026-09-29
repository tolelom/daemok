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

// tierDirection은 자기보다 위 tier에 의존하면 오류다. 같은 tier는 허용한다.
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
		check(mod.ID, mod.Tier.Name, c.needTargets(mod.Needs))
	}
	for _, g := range m.Groups {
		check(g.ID, g.Tier.Name, g.DependsOn)
		check(g.ID, g.Tier.Name, c.needTargets(g.Needs))
		for _, mem := range g.Members {
			check(mem.ID, g.Tier.Name, c.needTargets(mem.Needs))
			for _, e := range mem.Exceptions {
				check(mem.ID, g.Tier.Name, []model.Ref{e.DependsOn})
			}
		}
	}
}

// selfDependency는 자기 자신을 가리키는 의존을 오류로 낸다. 멤버가 자기 그룹에 의존하는 것도 포함한다.
// cycles는 그룹으로 접은 뒤 자기 간선을 버리므로 여기서 따로 잡는다.
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

// cycles는 모듈과 그룹의 의존 그래프에서 순환을 찾는다. 순환마다 한 번 보고한다.
// 멤버는 그룹으로 접는다: 멤버를 가리키는 의존은 그 그룹으로, 멤버의 예외는 그룹의 의존으로 본다.
// 그룹에 의존하면 멤버에도 의존할 수 있으므로 멤버 예외를 거치는 순환도 여기서 잡힌다.
// 형제 멤버 사이의 순환은 siblingCycles가 잡는다.
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
	// 형제 멤버로 가는 예외는 그룹 자신으로 접히므로 뺀다.
	edges := func(id string) []string {
		var out []string
		add := func(d string) {
			if d = collapse(d); d != id {
				out = append(out, d)
			}
		}
		for _, d := range c.depsOf(id) {
			add(d)
		}
		if g := m.Group(id); g != nil {
			for _, mem := range g.Members {
				for _, e := range mem.Exceptions {
					add(e.DependsOn.Name)
				}
				for _, r := range c.needTargets(mem.Needs) {
					add(r.Name)
				}
			}
		}
		return out
	}
	c.findCycles(ids, edges)
}

// siblingCycles는 그룹마다 형제 멤버를 가리키는 예외만으로 순환을 찾는다.
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
			for _, r := range c.needTargets(mem.Needs) {
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

// findCycles는 ids 안에서 edges를 따라 DFS로 순환을 찾아 순환마다 한 번 보고한다.
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

// orphans는 어느 실행 단위에서도 닿지 않는 모듈·그룹을 경고한다. 실행 단위가 없으면 건너뛴다.
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

// exceptions는 멤버 예외를 정보로 남기고, 그룹이 이미 의존하는 대상이면 경고한다.
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

// contractRoles는 port·external 계약이 알맞은 role의 모듈에 있는지 본다.
func (c *checker) contractRoles() {
	want := map[string]string{"port": "infra", "external": "adapter"}
	for _, ct := range c.m.Contracts {
		role, ok := want[ct.Kind]
		if !ok {
			continue
		}
		if mod := c.m.Module(ct.In.Name); mod == nil || mod.Role != role {
			c.errorf(ct.Pos, "%s 계약의 모듈은 role이 %s여야 한다: 계약 %s, 모듈 %s", ct.Kind, role, ct.ID, ct.In.Name)
		}
	}
}
