package serve

import (
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

// nodePos는 화면 키와 그 노드의 설계 파일 위치다.
type nodePos struct {
	key string
	pos yaml.Pos
}

// nodePositions는 모든 노드의 키와 위치를 파일·줄·열 순으로 준다.
func nodePositions(m *model.Model) []nodePos {
	var out []nodePos
	add := func(kind string, meta model.Meta) { out = append(out, nodePos{kind + ":" + meta.ID, meta.Pos}) }
	for _, r := range m.Requirements {
		add("req", r.Meta)
	}
	if m.System != nil {
		add("sys", m.System.Meta)
		for _, a := range m.System.Actors {
			add("actor", a.Meta)
		}
		for _, i := range m.System.Interfaces {
			add("iface", i.Meta)
		}
	}
	for _, e := range m.Executables {
		add("exe", e.Meta)
	}
	for _, x := range m.Modules {
		add("mod", x.Meta)
	}
	for _, g := range m.Groups {
		add("grp", g.Meta)
		for _, mem := range g.Members {
			add("mem", mem.Meta)
		}
	}
	for _, x := range m.Enums {
		add("enum", x.Meta)
	}
	for _, x := range m.Consts {
		add("const", x.Meta)
	}
	for _, x := range m.Errors {
		add("err", x.Meta)
	}
	for _, x := range m.Types {
		add("type", x.Meta)
	}
	for _, x := range m.Contracts {
		add("contract", x.Meta)
	}
	for _, x := range m.HandlerKinds {
		add("hk", x.Meta)
	}
	for _, x := range m.EventKinds {
		add("ek", x.Meta)
	}
	for _, x := range m.Events {
		add("ev", x.Meta)
	}
	for _, x := range m.Funcs {
		add("func", x.Meta)
	}
	for _, x := range m.Flows {
		add("flow", x.Meta)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].pos.Less(out[j].pos) })
	return out
}

// issueKey는 진단이 가리키는 화면 노드 키다.
// 설계 파일 진단이면 같은 파일에서 진단 위치 이전의 마지막 노드, Go 파일 진단이면 경로 접두사가 가장 긴 모듈·멤버다.
func issueKey(m *model.Model, nodes []nodePos, root string, d model.Diagnostic) (key, origin string) {
	if strings.HasSuffix(d.Pos.File, ".go") {
		rel, err := filepath.Rel(root, d.Pos.File)
		if err != nil {
			return "", "code"
		}
		dir := path.Dir(filepath.ToSlash(rel))
		best := -1
		consider := func(p, k string) {
			if (dir == p || strings.HasPrefix(dir, p+"/")) && len(p) > best {
				best, key = len(p), k
			}
		}
		for _, x := range m.Modules {
			consider(x.Path, "mod:"+x.ID)
		}
		for _, g := range m.Groups {
			for _, mem := range g.Members {
				consider(mem.Path(), "mem:"+mem.ID)
			}
		}
		return key, "code"
	}
	for _, n := range nodes {
		if n.pos.File != d.Pos.File {
			continue
		}
		if d.Pos.Less(n.pos) {
			break
		}
		key = n.key
	}
	return key, "design"
}
