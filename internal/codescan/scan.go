// Package codescan은 코드의 import와 이벤트 상수 참조를 설계와 대조한다.
package codescan

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/gosrc"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

type scanner struct {
	m         *model.Model
	root      string
	fset      *token.FileSet
	diags     model.Diagnostics
	units     map[string]string
	imports   map[string]map[string]bool
	refs      map[string]map[string]bool
	decls     map[string]map[string]bool
	typeSpecs map[string]map[string]typeSpec
	adHoc     []adHocCall
}

type unit struct {
	id, dir string
}

func Scan(m *model.Model, root string) model.Diagnostics {
	s := &scanner{
		m: m, root: root, fset: token.NewFileSet(),
		units: map[string]string{}, imports: map[string]map[string]bool{}, refs: map[string]map[string]bool{},
		decls: map[string]map[string]bool{}, typeSpecs: map[string]map[string]typeSpec{},
	}
	us := s.unitList()
	for _, u := range us {
		s.units[m.GoModule+"/"+u.dir] = u.id
	}
	for _, u := range us {
		s.scanUnit(u)
	}
	if m.CPP != nil {
		s.cppScan(us)
	}
	s.reportAdHoc()
	s.queryPackages()
	s.unusedDeps(us)
	s.eventRefs(us)
	s.handWritten()
	s.shapes()
	s.diags.Sort()
	return s.diags
}

func (s *scanner) unitList() []unit {
	var out []unit
	for _, mod := range s.m.Modules {
		out = append(out, unit{mod.ID, mod.Path})
	}
	for _, g := range s.m.Groups {
		for _, mem := range g.Members {
			out = append(out, unit{mem.ID, mem.Path()})
		}
	}
	return out
}

func (s *scanner) goFiles(dir string) []*ast.File {
	files, errs := gosrc.Files(s.fset, filepath.Join(s.root, filepath.FromSlash(dir)), parser.SkipObjectResolution)
	for _, e := range errs {
		s.diags.Errorf(yaml.Pos{File: e.Path}, "Go 파일을 읽을 수 없다: %v", e.Err)
	}
	return files
}

func (s *scanner) pos(p token.Pos) yaml.Pos {
	q := s.fset.Position(p)
	return yaml.Pos{File: q.Filename, Line: q.Line, Column: q.Column}
}

func (s *scanner) scanUnit(u unit) {
	s.imports[u.id] = map[string]bool{}
	s.refs[u.id] = map[string]bool{}
	s.decls[u.id] = map[string]bool{}
	s.typeSpecs[u.id] = map[string]typeSpec{}
	unmanaged := map[string]bool{}
	for _, f := range s.goFiles(u.dir) {
		for _, name := range gosrc.TopNames(f) {
			s.decls[u.id][name] = true
		}
		collectTypeSpecs(f, s.typeSpecs[u.id])
		for _, im := range f.Imports {
			ip, _ := strconv.Unquote(im.Path.Value)
			if ip != s.m.GoModule && !strings.HasPrefix(ip, s.m.GoModule+"/") {
				continue
			}
			target, ok := s.units[ip]
			if !ok {
				if !unmanaged[ip] {
					unmanaged[ip] = true
					s.diags.Infof(s.pos(im.Pos()), "관리하지 않는 패키지를 import한다: %s → %s", u.id, ip)
				}
				continue
			}
			if target == u.id {
				continue
			}
			s.depend(u.id, target, s.pos(im.Pos()), "import")
		}
		if !golang.IsDaemokGeneratedFile(f) {
			s.collectRefs(u.id, f)
			s.adHocErrors(u.id, f)
			s.enumLiterals(u.id, f)
		}
	}
}

func (s *scanner) depend(id, target string, pos yaml.Pos, verb string) {
	s.imports[id][target] = true
	if s.m.CanDepend(id, target) {
		return
	}
	if s.receivesFrom(id, target) {
		s.diags.Errorf(pos, "needs로만 이어진 모듈을 직접 %s한다: %s → %s", verb, id, target)
	} else {
		s.diags.Errorf(pos, "설계에 없는 의존을 %s한다: %s → %s", verb, id, target)
	}
}
