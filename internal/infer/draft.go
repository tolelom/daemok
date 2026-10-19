package infer

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

var (
	goIdent   = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)
	snakeName = regexp.MustCompile(`^[a-z][a-z0-9]*(_[a-z0-9]+)*$`)
	version   = regexp.MustCompile(`^v[0-9]+$`)
)

type Exe struct {
	ID, Dir, Summary string
	Composes         []string
}

type Mod struct {
	ID, Path, Summary string
	Tier              int
	DependsOn         []string
}

type Draft struct {
	GoModule, System string
	Exes             []Exe
	Tiers            int
	Mods             []Mod
}

type FanIn struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}

type Report struct {
	Packages        int              `json:"packages"`
	Modules         int              `json:"modules"`
	Executables     int              `json:"executables"`
	Skipped         []Skip           `json:"skipped"`
	FanIn           []FanIn          `json:"fanIn"`
	Cycles          [][]string       `json:"cycles"`
	GroupCandidates []GroupCandidate `json:"groupCandidates"`
	External        []External       `json:"external"`
	Hints           []Hint           `json:"hints"`
	Uses            []Use            `json:"uses,omitempty"`
}

type External struct {
	Module string   `json:"module"`
	Users  []string `json:"users"`
}

type GroupCandidate struct {
	Path       string   `json:"path"`
	Members    []string `json:"members"`
	Common     []string `json:"common"`
	Exceptions []string `json:"exceptions"`
}

type Analysis struct {
	Draft  *Draft
	YAML   []byte
	Report Report
}

func Analyze(root string) (*Analysis, error) {
	module, requires, err := readGoMod(root)
	if err != nil {
		return nil, err
	}
	pkgs, skips, err := collect(root, module)
	if err != nil {
		return nil, fmt.Errorf("코드를 읽을 수 없다: %w", err)
	}
	d, rep := build(module, pkgs, skips)
	rep.External = external(requires, pkgs, d)
	return &Analysis{Draft: d, YAML: Render(d), Report: rep}, nil
}

func dirOfMod(m Mod) string {
	if m.Path != "" {
		return m.Path
	}
	return m.ID
}

func systemID(module string) string {
	els := strings.Split(module, "/")
	last := els[len(els)-1]
	if version.MatchString(last) && len(els) > 1 {
		last = els[len(els)-2]
	}
	if id := snake(last); snakeName.MatchString(id) {
		return id
	}
	return "system"
}

func representable(dir string) bool {
	if dir == "." {
		return false
	}
	for _, el := range strings.Split(dir, "/") {
		if !goIdent.MatchString(el) {
			return false
		}
	}
	return true
}

func build(module string, pkgs []*pkg, skips []Skip) (*Draft, Report) {
	d := &Draft{GoModule: module, System: systemID(module)}
	var libDirs, exeDirs []string
	byDir := map[string]*pkg{}
	for _, p := range pkgs {
		byDir[p.dir] = p
		switch {
		case p.main:
			exeDirs = append(exeDirs, p.dir)
		case p.dir == ".":
			skips = append(skips, Skip{".", "루트 패키지는 설계 형식으로 표현할 수 없다"})
		case !representable(p.dir):
			skips = append(skips, Skip{p.dir, "Go 패키지 경로로 쓸 수 없는 디렉토리 이름"})
		default:
			libDirs = append(libDirs, p.dir)
		}
	}
	libIDs := assignIDs(libDirs, d.System)
	exeIDs := assignIDs(exeDirs, d.System)

	comp, comps := components(libDirs, func(dir string) []string {
		var out []string
		for _, dep := range byDir[dir].imports {
			if _, ok := libIDs[dep]; ok {
				out = append(out, dep)
			}
		}
		return out
	})
	compTier := map[int]int{}
	var level func(dir string) int
	level = func(dir string) int {
		c := comp[dir]
		if t, ok := compTier[c]; ok {
			return t
		}
		t := 0
		for _, member := range comps[c] {
			for _, dep := range byDir[member].imports {
				if _, ok := libIDs[dep]; ok && comp[dep] != c {
					if l := level(dep) + 1; l > t {
						t = l
					}
				}
			}
		}
		compTier[c] = t
		return t
	}
	internalDeps := func(p *pkg) []string {
		var out []string
		for _, dep := range p.imports {
			if id, ok := libIDs[dep]; ok {
				out = append(out, id)
			}
		}
		sort.Strings(out)
		return out
	}
	fan := map[string]int{}
	for _, dir := range libDirs {
		p := byDir[dir]
		m := Mod{ID: libIDs[dir], Summary: p.doc, Tier: level(dir), DependsOn: internalDeps(p)}
		if m.ID != dir {
			m.Path = dir
		}
		for _, dep := range m.DependsOn {
			fan[dep]++
		}
		if m.Tier+1 > d.Tiers {
			d.Tiers = m.Tier + 1
		}
		d.Mods = append(d.Mods, m)
	}
	for _, dir := range exeDirs {
		id := exeIDs[dir]
		if !snakeName.MatchString(id) {
			skips = append(skips, Skip{dir, "실행 단위 ID로 쓸 수 없는 디렉토리 이름"})
			continue
		}
		p := byDir[dir]
		e := Exe{ID: id, Dir: dir, Summary: p.doc, Composes: internalDeps(p)}
		for _, dep := range e.Composes {
			fan[dep]++
		}
		d.Exes = append(d.Exes, e)
	}
	sort.Slice(d.Mods, func(i, j int) bool {
		if d.Mods[i].Tier != d.Mods[j].Tier {
			return d.Mods[i].Tier < d.Mods[j].Tier
		}
		return d.Mods[i].ID < d.Mods[j].ID
	})
	sort.Slice(d.Exes, func(i, j int) bool { return d.Exes[i].ID < d.Exes[j].ID })

	rep := Report{Packages: len(pkgs), Modules: len(d.Mods), Executables: len(d.Exes)}
	for _, members := range comps {
		if len(members) < 2 {
			continue
		}
		var ids []string
		for _, dir := range members {
			ids = append(ids, libIDs[dir])
		}
		sort.Strings(ids)
		rep.Cycles = append(rep.Cycles, ids)
	}
	sort.Slice(rep.Cycles, func(i, j int) bool { return rep.Cycles[i][0] < rep.Cycles[j][0] })
	rep.GroupCandidates = groupCandidates(libDirs, libIDs, d.Mods)
	for _, dir := range libDirs {
		for _, h := range byDir[dir].hints {
			h.Module = libIDs[dir]
			rep.Hints = append(rep.Hints, h)
		}
	}
	sort.Slice(skips, func(i, j int) bool { return skips[i].Path < skips[j].Path })
	rep.Skipped = skips
	for id, n := range fan {
		rep.FanIn = append(rep.FanIn, FanIn{id, n})
	}
	sort.Slice(rep.FanIn, func(i, j int) bool {
		if rep.FanIn[i].Count != rep.FanIn[j].Count {
			return rep.FanIn[i].Count > rep.FanIn[j].Count
		}
		return rep.FanIn[i].ID < rep.FanIn[j].ID
	})
	ids := map[string]string{}
	for _, m := range d.Mods {
		ids[dirOfMod(m)] = m.ID
	}
	for _, e := range d.Exes {
		ids[e.Dir] = e.ID
	}
	rep.Uses = uses(byDir, ids, libIDs)
	if len(rep.FanIn) > 5 {
		rep.FanIn = rep.FanIn[:5]
	}
	return d, rep
}

func assignIDs(dirs []string, rootID string) map[string]string {
	dirs = append([]string(nil), dirs...)
	sort.Strings(dirs)
	parts := func(dir string) []string { return strings.Split(dir, "/") }
	cand := func(dir string, n int) string {
		if dir == "." {
			return rootID
		}
		els := parts(dir)
		if n > len(els) {
			n = len(els)
		}
		return snake(strings.Join(els[len(els)-n:], "_"))
	}
	level := map[string]int{}
	for _, d := range dirs {
		level[d] = 1
	}
	for {
		byID := map[string][]string{}
		for _, d := range dirs {
			c := cand(d, level[d])
			byID[c] = append(byID[c], d)
		}
		changed := false
		for _, ds := range byID {
			if len(ds) < 2 {
				continue
			}
			for _, d := range ds {
				if d != "." && level[d] < len(parts(d)) {
					level[d]++
					changed = true
				}
			}
		}
		if !changed {
			break
		}
	}
	ids := map[string]string{}
	used := map[string]int{}
	for _, d := range dirs {
		id := cand(d, level[d])
		used[id]++
		if used[id] > 1 {
			id = fmt.Sprintf("%s_%d", id, used[id])
		}
		ids[d] = id
	}
	return ids
}

func snake(s string) string {
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		if unicode.IsUpper(r) && i > 0 {
			prev := rs[i-1]
			nextLower := i+1 < len(rs) && unicode.IsLower(rs[i+1])
			if unicode.IsLower(prev) || unicode.IsDigit(prev) || (unicode.IsUpper(prev) && nextLower) {
				b.WriteByte('_')
			}
		}
		if r == '-' || r == '.' {
			r = '_'
		}
		b.WriteRune(unicode.ToLower(r))
	}
	out := b.String()
	for strings.Contains(out, "__") {
		out = strings.ReplaceAll(out, "__", "_")
	}
	return strings.Trim(out, "_")
}

func components(nodes []string, edges func(string) []string) (map[string]int, [][]string) {
	index := map[string]int{}
	low := map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	comp := map[string]int{}
	var comps [][]string
	var visit func(v string)
	visit = func(v string) {
		index[v] = len(index)
		low[v] = index[v]
		stack = append(stack, v)
		onStack[v] = true
		for _, w := range edges(v) {
			if _, seen := index[w]; !seen {
				visit(w)
				low[v] = min(low[v], low[w])
			} else if onStack[w] {
				low[v] = min(low[v], index[w])
			}
		}
		if low[v] == index[v] {
			var members []string
			for {
				w := stack[len(stack)-1]
				stack = stack[:len(stack)-1]
				onStack[w] = false
				comp[w] = len(comps)
				members = append(members, w)
				if w == v {
					break
				}
			}
			comps = append(comps, members)
		}
	}
	for _, v := range nodes {
		if _, seen := index[v]; !seen {
			visit(v)
		}
	}
	return comp, comps
}

func groupCandidates(libDirs []string, libIDs map[string]string, mods []Mod) []GroupCandidate {
	deps := map[string][]string{}
	for _, m := range mods {
		deps[m.ID] = m.DependsOn
	}
	byParent := map[string][]string{}
	for _, dir := range libDirs {
		if parent := path.Dir(dir); parent != "." {
			byParent[parent] = append(byParent[parent], libIDs[dir])
		}
	}
	var out []GroupCandidate
	for parent, members := range byParent {
		if len(members) < 3 {
			continue
		}
		sort.Strings(members)
		sibling := map[string]bool{}
		for _, id := range members {
			sibling[id] = true
		}
		count := map[string]int{}
		var exceptions []string
		for _, id := range members {
			for _, dep := range deps[id] {
				if sibling[dep] {
					exceptions = append(exceptions, id+" → "+dep)
				} else {
					count[dep]++
				}
			}
		}
		var common []string
		for dep, n := range count {
			if n == len(members) {
				common = append(common, dep)
			}
		}
		if len(common) == 0 {
			continue
		}
		sort.Strings(common)
		sort.Strings(exceptions)
		out = append(out, GroupCandidate{Path: parent, Members: members, Common: common, Exceptions: exceptions})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

func external(requires []string, pkgs []*pkg, d *Draft) []External {
	if len(requires) == 0 {
		return nil
	}
	idOf := map[string]string{}
	for _, m := range d.Mods {
		dir := m.Path
		if dir == "" {
			dir = m.ID
		}
		idOf[dir] = m.ID
	}
	for _, e := range d.Exes {
		idOf[e.Dir] = e.ID
	}
	users := map[string]map[string]bool{}
	for _, p := range pkgs {
		id, ok := idOf[p.dir]
		if !ok {
			continue
		}
		for _, ip := range p.ext {
			best := ""
			for _, r := range requires {
				if (ip == r || strings.HasPrefix(ip, r+"/")) && len(r) > len(best) {
					best = r
				}
			}
			if best != "" {
				if users[best] == nil {
					users[best] = map[string]bool{}
				}
				users[best][id] = true
			}
		}
	}
	out := make([]External, 0, len(requires))
	for _, r := range requires {
		ids := []string{}
		for id := range users[r] {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		out = append(out, External{Module: r, Users: ids})
	}
	return out
}
