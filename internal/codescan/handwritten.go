package codescan

import (
	"os"
	"path/filepath"

	"github.com/tolelom/daemok/internal/model"
)

func (s *scanner) dirExists(dir string) bool {
	st, err := os.Stat(filepath.Join(s.root, filepath.FromSlash(dir)))
	return err == nil && st.IsDir()
}

func (s *scanner) handWritten() {
	m := s.m
	need := func(o model.Owned, what string, names ...string) {
		if o.Gen || !s.dirExists(s.ownerDir(o.In.Name)) {
			return
		}
		for _, n := range names {
			if !s.decls[o.In.Name][n] {
				s.diags.Errorf(o.Pos, "손으로 쓴다고 한 선언이 코드에 없다: 모듈 %s, %s %s (%s)", o.In.Name, what, o.ID, n)
			}
		}
	}
	for _, e := range m.Enums {
		names := []string{e.GoType()}
		for _, v := range e.Values {
			names = append(names, e.ConstName(v.Name))
		}
		need(e.Owned, "enum", names...)
	}
	for _, k := range m.Consts {
		need(k.Owned, "상수", k.GoName())
	}
	for _, e := range m.Errors {
		need(e.Owned, "에러", e.GoName())
	}
	for _, t := range m.Types {
		need(t.Owned, "타입", t.GoName())
	}
	for _, ct := range m.Contracts {
		if ct.Kind == "call" {
			need(ct.Owned, "계약", ct.GoName())
		}
	}
	for _, hk := range m.HandlerKinds {
		need(hk.Owned, "핸들러 종류", hk.GoName())
	}
	for _, f := range m.Funcs {
		if f.Free() {
			o := f.Owned
			o.Gen = false
			need(o, "함수", f.GoName())
		}
	}
}

func (s *scanner) ownerDir(id string) string {
	o, _ := s.m.Owner(id)
	return o.Path
}
