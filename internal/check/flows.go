package check

import (
	"regexp"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

var snakeName = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)

// flows는 단계 이름, commit 위치, 흐름의 모듈과 calls·emits·on의 관계를 본다.
func (c *checker) flows() {
	m := c.m
	for _, f := range m.Flows {
		owner := f.In.Name
		seen := map[string]bool{}
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
			for _, name := range []string{s.ID, s.Undo} {
				if name != "" && model.GoName(name) == "Fatal" {
					c.errorf(s.Pos, "Fatal과 겹치는 단계 이름: %s", name)
				}
			}
			if s.Commit {
				if committed {
					c.errorf(s.Pos, "commit 단계가 둘 이상이다: %s", s.ID)
				}
				committed = true
			}
			if committed && s.Undo != "" {
				c.errorf(s.UndoPos, "커밋 뒤 단계에 undo가 있다: %s", s.ID)
			}
			for _, r := range s.Calls {
				ctID, _, _ := strings.Cut(r.Name, ".")
				if ct := m.Contract(ctID); ct != nil && ct.In.Name != owner && !c.receives(owner, ct.ID) {
					c.errorf(r.Pos, "흐름의 모듈이 받지 않는 계약: %s → %s", owner, ct.ID)
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

// receives는 id가 needs로 계약을 받는지다.
func (c *checker) receives(id, contract string) bool {
	for _, n := range c.needsOf(id) {
		if n.Contract.Name == contract {
			return true
		}
	}
	return false
}
