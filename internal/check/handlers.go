package check

import (
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// handlers는 핸들러 종류의 키 enum 값마다 처리하는 멤버가 정확히 하나인지 본다.
func (c *checker) handlers() {
	m := c.m
	for _, hk := range m.HandlerKinds {
		en := m.Enum(hk.Key.Name)
		if en == nil {
			continue
		}
		type handler struct {
			member string
			ref    model.Ref
		}
		byValue := map[string][]handler{}
		for _, g := range m.Groups {
			if g.Kind.Name != hk.ID {
				continue
			}
			for _, mem := range g.Members {
				for _, r := range mem.Handles {
					byValue[r.Name] = append(byValue[r.Name], handler{mem.ID, r})
				}
			}
		}
		for _, v := range en.Values {
			hs := byValue[v.Name]
			switch {
			case len(hs) == 0:
				c.errorf(v.Pos, "처리하는 멤버가 없는 값: %s.%s", en.ID, v.Name)
			case len(hs) > 1:
				var names []string
				for _, h := range hs {
					names = append(names, h.member)
				}
				c.errorf(hs[1].ref.Pos, "여러 멤버가 처리하는 값: %s.%s (%s)", en.ID, v.Name, strings.Join(names, ", "))
			}
		}
	}
}
