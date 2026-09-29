package golang

import (
	"fmt"
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// unitNeeds는 모듈과 멤버가 받는 계약이다. 멤버는 그룹의 needs를 먼저 받는다.
// 같은 계약이 두 번 나오면 uses를 합치고, 한쪽이라도 uses가 비면 메서드 전부다.
func (g *gen) unitNeeds() (ids []string, needs map[string][]model.Need) {
	needs = map[string][]model.Need{}
	add := func(id string, ns []model.Need) {
		for _, n := range ns {
			ct := g.m.Contract(n.Contract.Name)
			if ct == nil || !ct.Gen {
				continue
			}
			if _, ok := needs[id]; !ok {
				ids = append(ids, id)
			}
			i := slices.IndexFunc(needs[id], func(x model.Need) bool { return x.Contract.Name == n.Contract.Name })
			if i < 0 {
				needs[id] = append(needs[id], n)
				continue
			}
			if len(needs[id][i].Uses) == 0 || len(n.Uses) == 0 {
				needs[id][i].Uses = nil
			} else {
				needs[id][i].Uses = append(slices.Clone(needs[id][i].Uses), n.Uses...)
			}
		}
	}
	for _, mod := range g.m.Modules {
		add(mod.ID, mod.Needs)
	}
	for _, grp := range g.m.Groups {
		for _, mem := range grp.Members {
			add(mem.ID, grp.Needs)
			add(mem.ID, mem.Needs)
		}
	}
	return ids, needs
}

func usesMethod(n model.Need, name string) bool {
	if len(n.Uses) == 0 {
		return true
	}
	for _, u := range n.Uses {
		if u.Name == name {
			return true
		}
	}
	return false
}

// needs는 받는 쪽 패키지에 uses의 메서드만 담은 interface를 만든다.
func (g *gen) needs() error {
	ids, needs := g.unitNeeds()
	for _, id := range ids {
		dir := g.ownerPath(id)
		f := g.file(dir + "/needs.gen.go")
		for _, n := range needs[id] {
			ct := g.m.Contract(n.Contract.Name)
			var lines []string
			for _, me := range ct.Methods {
				if usesMethod(n, me.Name) {
					lines = append(lines, g.methodSig(me, dir, f))
				}
			}
			doc := fmt.Sprintf("// %s는 %s 계약(%s)에서 이 패키지가 쓰는 메서드다.", ct.GoName(), ct.ID, ct.Kind)
			if n.Why != "" {
				doc += " " + n.Why
			}
			f.add("%s\ntype %s interface {\n%s\n}", doc, ct.GoName(), strings.Join(lines, "\n"))
		}
	}
	return nil
}

// executableContracts는 실행 단위가 조립하는 받는 쪽·주는 쪽 쌍마다
// 구현 타입이 받는 쪽 interface를 만족하는지 확인하는 줄을 만든다. call 계약은 구현 타입이 정해져 있지 않아 뺀다.
func (g *gen) executableContracts() error {
	ids, needs := g.unitNeeds()
	for _, e := range g.m.Executables {
		units := g.reachable(e)
		var f *goFile
		var lines []string
		for _, id := range ids {
			if !units[id] {
				continue
			}
			for _, n := range needs[id] {
				ct := g.m.Contract(n.Contract.Name)
				if ct.Kind == "call" || !units[ct.In.Name] {
					continue
				}
				if f == nil {
					f = g.mainFile("cmd/" + e.ID + "/contracts.gen.go")
				}
				implDir := g.ownerPath(ct.In.Name)
				if ct.Kind == "query" {
					implDir += "/query"
				}
				consumer := f.use(g.importPath(g.ownerPath(id)))
				impl := f.use(g.importPath(implDir))
				lines = append(lines, fmt.Sprintf("\t_ %s.%s = (*%s.%s)(nil)", consumer, ct.GoName(), impl, ct.GoName()))
			}
		}
		if f != nil {
			f.add("// 구현 타입이 받는 쪽 interface를 만족하지 않으면 여기서 컴파일이 깨진다.\nvar (\n%s\n)", strings.Join(lines, "\n"))
		}
	}
	return nil
}
