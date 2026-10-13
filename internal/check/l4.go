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
				c.errorf(f.In.Pos, "핸들러 함수는 그룹 멤버에 있어야 한다: %s", f.ID)
			case owner.Member.Group.Kind.Name != f.Kind.Name:
				c.errorf(f.Kind.Pos, "함수와 그룹의 핸들러 종류가 다르다: 함수 %s(%s), 멤버 %s(%s)", f.ID, f.Kind.Name, owner.ID, owner.Member.Group.Kind.Name)
			case f.Handles.Name != "" && !has(owner.Member.Handles, f.Handles.Name):
				c.errorf(f.Handles.Pos, "멤버 handles에 없는 값: 멤버 %s, 값 %s", owner.ID, f.Handles.Name)
			}
		}
		if ok && f.Of.Name != "" {
			ct, _, _ := strings.Cut(f.Of.Name, ".")
			if contract := m.Contract(ct); contract != nil && contract.In.Name != f.In.Name {
				c.errorf(f.Of.Pos, "계약과 함수의 위치가 다르다: 계약 %s(%s), 함수 %s(%s)", ct, contract.In.Name, f.ID, f.In.Name)
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
				c.errorf(cs.Error.Pos, "함수 errors에 없는 에러: 케이스 %q, 에러 %s, 함수 %s", cs.Name, e, f.ID)
			}
		}
		for _, r := range f.Errors {
			if f.Gen && !covered[r.Name] { // gen: false면 케이스 테스트를 만들지 않는다
				c.warnf(r.Pos, "확인하는 케이스가 없는 에러: 함수 %s, 에러 %s", f.ID, r.Name)
			}
		}
	}
}
