package golang

import (
	"fmt"
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

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
			text := fmt.Sprintf("%s는 %s 계약(%s)에서 이 패키지가 쓰는 메서드다.", ct.GoName(), ct.ID, ct.Kind)
			if n.Why != "" {
				text += " " + n.Why
			}
			doc := comment("", text)
			f.add("%s\ntype %s interface {\n%s\n}", doc, ct.GoName(), strings.Join(lines, "\n"))
		}
	}
	return nil
}

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
