package codescan

import (
	"go/ast"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
)

func (s *scanner) collectRefs(id string, f *ast.File) {
	kindImports := map[string]bool{}
	inKindModule := false
	for _, ek := range s.m.EventKinds {
		ip := s.m.ImportPath(ek.In.Name)
		if ek.In.Name == id {
			inKindModule = true
		}
		for _, im := range f.Imports {
			if p, _ := strconv.Unquote(im.Path.Value); p == ip {
				name := filepath.Base(ip)
				if im.Name != nil {
					name = im.Name.Name
				}
				if name == "." {
					inKindModule = true
				}
				kindImports[name] = true
			}
		}
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if pkg, ok := x.X.(*ast.Ident); ok && kindImports[pkg.Name] {
				s.refs[id][x.Sel.Name] = true
			}
		case *ast.Ident:
			if inKindModule {
				s.refs[id][x.Name] = true
			}
		}
		return true
	})
}

func (s *scanner) eventRefs(us []unit) {
	m := s.m
	for _, ek := range m.EventKinds {
		en := m.Enum(ek.Key.Name)
		if en == nil {
			continue
		}
		declared := map[string]bool{}
		for _, r := range ek.Subscribers {
			declared[r.Name] = true
			if s.dirExists(s.ownerDir(r.Name)) && !s.refs[r.Name][en.GoType()+"Values"] {
				s.diags.Errorf(r.Pos, "이벤트 목록 함수를 쓰지 않는다: 모듈 %s, 이벤트 종류 %s (%sValues)", r.Name, ek.ID, en.GoType())
			}
		}
		for _, v := range en.Values {
			e := m.Event(v.Name)
			if e == nil {
				continue
			}
			constName := en.ConstName(v.Name)
			users := map[string]bool{}
			for _, r := range append(slices.Clone(e.Publishers), e.Subscribers...) {
				users[r.Name] = true
				if s.dirExists(s.ownerDir(r.Name)) && !s.refs[r.Name][constName] {
					s.diags.Errorf(r.Pos, "이벤트 상수를 쓰지 않는다: 모듈 %s, 이벤트 %s (%s)", r.Name, e.ID, constName)
				}
			}
			var extra []string
			for _, u := range us {
				if u.id != ek.In.Name && !users[u.id] && !declared[u.id] && s.refs[u.id][constName] {
					extra = append(extra, u.id)
				}
			}
			sort.Strings(extra)
			for _, id := range extra {
				s.diags.Warnf(e.Pos, "발행자나 구독자로 적히지 않은 모듈이 이벤트를 쓴다: 모듈 %s, 이벤트 %s", id, e.ID)
			}
		}
	}
}
