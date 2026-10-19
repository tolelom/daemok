package codescan

import (
	"path"
	"slices"
	"strconv"

	"github.com/tolelom/daemok/internal/model"
)

func (s *scanner) needsOf(id string) []model.Need {
	if mem := s.m.Member(id); mem != nil {
		return append(slices.Clone(mem.Group.Needs), mem.Needs...)
	}
	if mod := s.m.Module(id); mod != nil {
		return mod.Needs
	}
	return nil
}

func (s *scanner) receivesFrom(id, target string) bool {
	for _, n := range s.needsOf(id) {
		if ct := s.m.Contract(n.Contract.Name); ct != nil && ct.In.Name == target {
			return true
		}
	}
	return false
}

func (s *scanner) queryPackages() {
	done := map[string]bool{}
	for _, ct := range s.m.Contracts {
		if ct.Kind != "query" || done[ct.In.Name] {
			continue
		}
		done[ct.In.Name] = true
		o, ok := s.m.Owner(ct.In.Name)
		if !ok {
			continue
		}
		dir := o.Path + "/query"
		for _, f := range s.goFiles(dir) {
			for _, im := range f.Imports {
				ip, _ := strconv.Unquote(im.Path.Value)
				target, ok := s.units[ip]
				if !ok {
					continue
				}
				if mod := s.m.Module(target); mod == nil || mod.Role != "data" {
					s.diags.Errorf(s.pos(im.Pos()), "query 패키지가 data가 아닌 모듈을 import한다: %s → %s", dir, target)
				}
			}
		}
	}
}

func (s *scanner) unusedDeps(us []unit) {
	for _, mod := range s.m.Modules {
		if !s.hasCode(mod.Path) {
			continue
		}
		for _, d := range mod.DependsOn {
			targets := []string{d.Name}
			if g := s.m.Group(d.Name); g != nil {
				targets = nil
				for _, mem := range g.Members {
					targets = append(targets, mem.ID)
				}
			}
			used := false
			for _, t := range targets {
				used = used || s.imports[mod.ID][t]
			}
			if !used {
				verb := "import"
				if s.m.CPP != nil && !s.dirExists(mod.Path) {
					verb = "include"
				}
				s.diags.Infof(d.Pos, "설계상 의존하지만 %s하지 않는다: %s → %s", verb, mod.ID, d.Name)
			}
		}
	}
}

func (s *scanner) hasCode(dir string) bool {
	if s.dirExists(dir) {
		return true
	}
	if s.m.CPP != nil {
		for _, inc := range s.m.CPP.IncludeDirs {
			if s.dirExists(path.Join(inc.Name, dir)) {
				return true
			}
		}
	}
	return false
}
