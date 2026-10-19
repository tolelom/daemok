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

type Executable struct {
	ID, Dir, Summary string
	Composes         []string
}

type Module struct {
	ID, Path, Summary string
	Tier              int
	DependsOn         []string
}

type Draft struct {
	GoModule, System string
	Executables      []Executable
	Tiers            int
	Modules          []Module
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

func dirOfModule(m Module) string {
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
	byDir, libDirs, exeDirs, skips := classifyPackages(pkgs, skips)
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
	d.Modules = modules(libDirs, byDir, libIDs, componentTiers(libDirs, byDir, libIDs, comp, comps))
	d.Tiers = tierCount(d.Modules)
	d.Executables, skips = executables(exeDirs, byDir, exeIDs, libIDs, skips)
	sortDraft(d)

	rep := Report{Packages: len(pkgs), Modules: len(d.Modules), Executables: len(d.Executables)}
	rep.Cycles = cycles(comps, libIDs)
	rep.GroupCandidates = groupCandidates(libDirs, libIDs, d.Modules)
	rep.Hints = moduleHints(libDirs, byDir, libIDs)
	sort.Slice(skips, func(i, j int) bool { return skips[i].Path < skips[j].Path })
	rep.Skipped = skips
	rep.FanIn = topFanIn(d, 5)
	rep.Uses = uses(byDir, draftIDs(d), libIDs)
	return d, rep
}

func classifyPackages(pkgs []*pkg, skips []Skip) (map[string]*pkg, []string, []string, []Skip) {
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
	return byDir, libDirs, exeDirs, skips
}

func componentTiers(libDirs []string, byDir map[string]*pkg, libIDs map[string]string, comp map[string]int, comps [][]string) map[string]int {
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
	tiers := map[string]int{}
	for _, dir := range libDirs {
		tiers[dir] = level(dir)
	}
	return tiers
}

func internalDeps(p *pkg, libIDs map[string]string) []string {
	var out []string
	for _, dep := range p.imports {
		if id, ok := libIDs[dep]; ok {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func modules(libDirs []string, byDir map[string]*pkg, libIDs map[string]string, tiers map[string]int) []Module {
	var mods []Module
	for _, dir := range libDirs {
		p := byDir[dir]
		m := Module{ID: libIDs[dir], Summary: p.doc, Tier: tiers[dir], DependsOn: internalDeps(p, libIDs)}
		if m.ID != dir {
			m.Path = dir
		}
		mods = append(mods, m)
	}
	return mods
}

func tierCount(mods []Module) int {
	n := 0
	for _, m := range mods {
		if m.Tier+1 > n {
			n = m.Tier + 1
		}
	}
	return n
}

func executables(exeDirs []string, byDir map[string]*pkg, exeIDs, libIDs map[string]string, skips []Skip) ([]Executable, []Skip) {
	var exes []Executable
	for _, dir := range exeDirs {
		id := exeIDs[dir]
		if !snakeName.MatchString(id) {
			skips = append(skips, Skip{dir, "실행 단위 ID로 쓸 수 없는 디렉토리 이름"})
			continue
		}
		p := byDir[dir]
		exes = append(exes, Executable{ID: id, Dir: dir, Summary: p.doc, Composes: internalDeps(p, libIDs)})
	}
	return exes, skips
}

func sortDraft(d *Draft) {
	sort.Slice(d.Modules, func(i, j int) bool {
		if d.Modules[i].Tier != d.Modules[j].Tier {
			return d.Modules[i].Tier < d.Modules[j].Tier
		}
		return d.Modules[i].ID < d.Modules[j].ID
	})
	sort.Slice(d.Executables, func(i, j int) bool { return d.Executables[i].ID < d.Executables[j].ID })
}

func cycles(comps [][]string, libIDs map[string]string) [][]string {
	var out [][]string
	for _, members := range comps {
		if len(members) < 2 {
			continue
		}
		var ids []string
		for _, dir := range members {
			ids = append(ids, libIDs[dir])
		}
		sort.Strings(ids)
		out = append(out, ids)
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	return out
}

func moduleHints(libDirs []string, byDir map[string]*pkg, libIDs map[string]string) []Hint {
	var out []Hint
	for _, dir := range libDirs {
		for _, h := range byDir[dir].hints {
			h.Module = libIDs[dir]
			out = append(out, h)
		}
	}
	return out
}

func topFanIn(d *Draft, limit int) []FanIn {
	fan := map[string]int{}
	for _, m := range d.Modules {
		for _, dep := range m.DependsOn {
			fan[dep]++
		}
	}
	for _, e := range d.Executables {
		for _, dep := range e.Composes {
			fan[dep]++
		}
	}
	var out []FanIn
	for id, n := range fan {
		out = append(out, FanIn{id, n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out
}

func draftIDs(d *Draft) map[string]string {
	ids := map[string]string{}
	for _, m := range d.Modules {
		ids[dirOfModule(m)] = m.ID
	}
	for _, e := range d.Executables {
		ids[e.Dir] = e.ID
	}
	return ids
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

func groupCandidates(libDirs []string, libIDs map[string]string, mods []Module) []GroupCandidate {
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
	for _, m := range d.Modules {
		dir := m.Path
		if dir == "" {
			dir = m.ID
		}
		idOf[dir] = m.ID
	}
	for _, e := range d.Executables {
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
