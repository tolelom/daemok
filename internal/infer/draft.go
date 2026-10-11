package infer

import (
	"fmt"
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

// Exe는 초안의 실행 단위다.
type Exe struct {
	ID, Dir, Summary string
	Composes         []string
}

// Mod는 초안의 모듈이다. Path는 ID와 다를 때만 있다.
type Mod struct {
	ID, Path, Summary string
	Tier              int
	DependsOn         []string
}

// Draft는 코드에서 추정한 L2 설계다. tier 이름은 tier0부터 tier(Tiers-1)까지다.
type Draft struct {
	GoModule, System string
	Exes             []Exe
	Tiers            int
	Mods             []Mod
}

// FanIn은 모듈을 import하는 모듈·실행 단위의 수다.
type FanIn struct {
	ID    string `json:"id"`
	Count int    `json:"count"`
}

// Report는 분석 모드 화면과 daemok init이 보여 주는 메모다.
type Report struct {
	Packages    int     `json:"packages"`
	Modules     int     `json:"modules"`
	Executables int     `json:"executables"`
	Skipped     []Skip  `json:"skipped"`
	FanIn       []FanIn `json:"fanIn"`
}

// Analysis는 Analyze의 결과다.
type Analysis struct {
	Draft  *Draft
	YAML   []byte
	Report Report
}

// Analyze는 root의 go.mod와 Go 패키지에서 초안을 만든다. 파일을 쓰지 않는다.
func Analyze(root string) (*Analysis, error) {
	module, err := modulePath(root)
	if err != nil {
		return nil, err
	}
	pkgs, skips, err := collect(root, module)
	if err != nil {
		return nil, fmt.Errorf("코드를 읽을 수 없다: %w", err)
	}
	d, rep := build(module, pkgs, skips)
	return &Analysis{Draft: d, YAML: Render(d), Report: rep}, nil
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

	tier := map[string]int{}
	var level func(dir string) int
	level = func(dir string) int {
		if t, ok := tier[dir]; ok {
			return t
		}
		tier[dir] = 0
		t := 0
		for _, dep := range byDir[dir].imports {
			if _, ok := libIDs[dep]; ok {
				if l := level(dep) + 1; l > t {
					t = l
				}
			}
		}
		tier[dir] = t
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
	if len(rep.FanIn) > 5 {
		rep.FanIn = rep.FanIn[:5]
	}
	return d, rep
}

// assignIDs는 디렉토리마다 ID를 정한다. 마지막 경로 요소부터 쓰고, 겹치면 겹친 것끼리
// 경로 요소를 하나씩 앞으로 더 붙인다. 경로 전체로도 겹치면 디렉토리 순서대로 _2, _3을 붙인다.
// 루트(".")는 rootID다.
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

// snake는 이름을 소문자 snake_case로 바꾼다. 소문자나 숫자 뒤의 대문자 앞, 대문자 연속 뒤에
// 소문자가 오면 그 연속의 마지막 대문자 앞에 _를 넣는다. -와 .은 _가 된다.
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
