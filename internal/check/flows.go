package check

import (
	"regexp"
	"strings"

	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

var snakeName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)

// flows는 단계 이름, commit 위치, 흐름의 모듈과 calls·emits·on의 관계를 본다.
func (c *checker) flows() {
	m := c.m
	for _, f := range m.Flows {
		owner := f.In.Name
		if f.Steps != nil && len(f.Steps) == 0 {
			c.errorf(f.Pos, "단계가 없는 흐름: %s", f.ID)
		}
		seen := map[string]bool{}
		methods := map[string]string{} // Go 메서드 이름 → 처음 쓴 단계·undo 이름
		method := func(name string, pos yaml.Pos) {
			if name == "" {
				return
			}
			goName := model.GoName(name)
			if goName == "Fatal" {
				c.errorf(pos, "Fatal과 겹치는 단계 이름: %s", name)
			}
			if prev, ok := methods[goName]; ok && prev != name {
				c.errorf(pos, "메서드 이름이 겹치는 단계: %s (%s)", name, prev)
			} else if !ok {
				methods[goName] = name
			}
		}
		committed := false
		for _, s := range f.Steps {
			if !snakeName.MatchString(s.ID) {
				c.errorf(s.Pos, "단계 ID는 소문자 snake_case여야 한다: %q", s.ID)
			}
			if seen[s.ID] {
				c.errorf(s.Pos, "흐름 안에서 중복된 단계: %s", s.ID)
			}
			seen[s.ID] = true
			if s.Undo != "" && !snakeName.MatchString(s.Undo) {
				c.errorf(s.UndoPos, "undo는 소문자 snake_case여야 한다: %q", s.Undo)
			}
			method(s.ID, s.Pos)
			method(s.Undo, s.UndoPos)
			switch {
			case s.Commit && s.Undo != "":
				c.errorf(s.UndoPos, "commit 단계에 undo가 있다: %s", s.ID)
			case committed && s.Undo != "":
				c.errorf(s.UndoPos, "커밋 뒤 단계에 undo가 있다: %s", s.ID)
			}
			if s.Commit {
				if committed {
					c.errorf(s.Pos, "commit 단계가 둘 이상이다: %s", s.ID)
				}
				committed = true
			}
			for _, r := range s.Calls {
				ctID, meth, _ := strings.Cut(r.Name, ".")
				ct := m.Contract(ctID)
				if ct == nil || ct.In.Name == owner {
					continue
				}
				if !c.receives(owner, ct.ID) {
					c.errorf(r.Pos, "흐름의 모듈이 받지 않는 계약: %s → %s", owner, ct.ID)
				} else if !c.usesMethod(owner, ct.ID, meth) {
					c.errorf(r.Pos, "받는 계약의 uses에 없는 메서드: %s → %s", owner, r.Name)
				}
			}
			for _, r := range s.Emits {
				if e := m.Event(r.Name); e != nil && !has(e.Publishers, owner) {
					c.errorf(r.Pos, "흐름의 모듈이 발행자가 아닌 이벤트: %s → %s", owner, e.ID)
				}
			}
		}
		if e := m.Event(f.On.Name); e != nil {
			ek := m.EventKind(e.Kind.Name)
			if !has(e.Subscribers, owner) && (ek == nil || !has(ek.Subscribers, owner)) {
				c.errorf(f.On.Pos, "흐름의 모듈이 구독자가 아닌 이벤트: %s → %s", owner, e.ID)
			}
		}
	}
}

// usesMethod는 id가 받은 계약에서 메서드를 쓸 수 있는지다. uses를 생략한 needs가 하나라도 있으면 전부 쓸 수 있다.
func (c *checker) usesMethod(id, contract, method string) bool {
	for _, n := range c.needsOf(id) {
		if n.Contract.Name == contract && (len(n.Uses) == 0 || has(n.Uses, method)) {
			return true
		}
	}
	return false
}

// receives는 id가 needs로 계약을 받는지다.
func (c *checker) receives(id, contract string) bool {
	for _, n := range c.needsOf(id) {
		if n.Contract.Name == contract {
			return true
		}
	}
	return false
}
