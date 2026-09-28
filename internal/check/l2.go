package check

import (
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

func (c *checker) l2() {
	c.tierDirection()
	c.cycles()
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
	}
	for _, g := range m.Groups {
		check(g.ID, g.Tier.Name, g.DependsOn)
		for _, mem := range g.Members {
			for _, e := range mem.Exceptions {
				check(mem.ID, g.Tier.Name, []model.Ref{e.DependsOn})
			}
		}
	}
}

// cycles는 모듈과 그룹의 의존 그래프에서 순환을 찾는다. 순환마다 한 번 보고한다.
// 멤버는 그룹으로 접는다: 멤버를 가리키는 의존은 그 그룹으로, 멤버의 예외는 그룹의 의존으로 본다.
// 그룹에 의존하면 멤버에도 의존할 수 있으므로 멤버 예외를 거치는 순환도 여기서 잡힌다.
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
			}
		}
		return out
	}
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
					c.warnf(e.DependsOn.Pos, "예외 %s는 그룹 %s가 이미 의존한다", e.DependsOn.Name, g.ID)
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
