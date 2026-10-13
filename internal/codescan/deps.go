package codescan

import (
	"slices"
	"strconv"

	"github.com/tolelom/daemok/internal/model"
)

// needsOf는 id가 받는 계약이다. 멤버는 그룹의 needs도 받는다.
func (s *scanner) needsOf(id string) []model.Need {
	if mem := s.m.Member(id); mem != nil {
		return append(slices.Clone(mem.Group.Needs), mem.Needs...)
	}
	if mod := s.m.Module(id); mod != nil {
		return mod.Needs
	}
	return nil
}

// receivesFrom은 id가 target 모듈의 계약을 needs로 받는지다.
func (s *scanner) receivesFrom(id, target string) bool {
	for _, n := range s.needsOf(id) {
		if ct := s.m.Contract(n.Contract.Name); ct != nil && ct.In.Name == target {
			return true
		}
	}
	return false
}

// queryPackages는 query 계약을 가진 모듈의 query 패키지가 data 모듈 말고 다른 설계 모듈을 import하는지 본다.
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

// unusedDeps는 모듈의 depends_on 대상을 import하지 않으면 정보로 알린다. 그룹 대상은 멤버 하나를 import하면 된다.
// 코드 디렉토리가 없는 모듈은 보지 않는다.
func (s *scanner) unusedDeps(us []unit) {
	for _, mod := range s.m.Modules {
		if !s.dirExists(mod.Path) {
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
				s.diags.Infof(d.Pos, "설계상 의존하지만 import하지 않는다: %s → %s", mod.ID, d.Name)
			}
		}
	}
}
