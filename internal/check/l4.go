package check

import "strings"

// l4는 함수 계약이 소속 멤버·계약과 맞고, 케이스의 에러가 errors와 맞는지 본다.
func (c *checker) l4() {
	m := c.m
	for _, f := range m.Funcs {
		owner, ok := m.Owner(f.In.Name)
		if ok && f.Kind.Name != "" {
			switch {
			case owner.Member == nil:
				c.errorf(f.In.Pos, "핸들러 함수 %s는 그룹 멤버에 있어야 한다", f.ID)
			case owner.Member.Group.Kind.Name != f.Kind.Name:
				c.errorf(f.Kind.Pos, "멤버 %s의 그룹은 %s 종류인데 함수 %s는 %s다", owner.ID, owner.Member.Group.Kind.Name, f.ID, f.Kind.Name)
			case f.Handles.Name != "" && !has(owner.Member.Handles, f.Handles.Name):
				c.errorf(f.Handles.Pos, "멤버 %s의 handles에 %s가 없다", owner.ID, f.Handles.Name)
			}
		}
		if ok && f.Of.Name != "" {
			ct, _, _ := strings.Cut(f.Of.Name, ".")
			if contract := m.Contract(ct); contract != nil && contract.In.Name != f.In.Name {
				c.errorf(f.Of.Pos, "계약 %s는 %s에 있는데 함수 %s는 %s에 있다", ct, contract.In.Name, f.ID, f.In.Name)
			}
		}
		covered := map[string]bool{}
		for _, cs := range f.Cases {
			e := cs.Error.Name
			if e == "" || e == "invalid" {
				continue
			}
			covered[e] = true
			if !has(f.Errors, e) {
				c.errorf(cs.Error.Pos, "케이스 %q의 에러 %s가 함수 %s의 errors에 없다", cs.Name, e, f.ID)
			}
		}
		for _, r := range f.Errors {
			if !covered[r.Name] {
				c.warnf(r.Pos, "함수 %s의 에러 %s를 확인하는 케이스가 없다", f.ID, r.Name)
			}
		}
	}
}
