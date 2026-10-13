// Package lift는 설계 모듈 패키지에 손으로 쓴 선언 중 설계에 없는 것을 gen: false L3 노드 초안으로 만든다.
// 초안은 기존 설계와 합쳐 검사했을 때 새 오류·경고를 내지 않는 노드만 남긴다.
package lift

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/tolelom/daemok/internal/check"
	"github.com/tolelom/daemok/internal/coverage"
	"github.com/tolelom/daemok/internal/model"
)

// Node는 초안 노드 하나다. Line은 "ID: { ... }" 한 줄이다.
type Node struct {
	Kind   string `json:"kind"` // errors, consts, enums, types
	Module string `json:"module"`
	ID     string `json:"id"`
	Name   string `json:"name"` // Go 이름
	Line   string `json:"line"`
	Where  string `json:"where"` // 코드 루트 기준 "경로:줄"
	fields []string
}

func (n *Node) render() { n.Line = n.ID + ": { " + strings.Join(n.fields, ", ") + " }" }

// Skip은 옮기지 않은 선언과 그 이유다.
type Skip struct {
	Where  string `json:"where"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

// Result는 초안 YAML과 노드, 옮기지 않은 선언이다. Funcs는 설계에 없는 exported 함수의 수다.
type Result struct {
	YAML    []byte `json:"-"`
	Nodes   []Node `json:"nodes"`
	Skipped []Skip `json:"skipped"`
	Funcs   int    `json:"funcs"`
	// Coverage는 초안을 더하기 전과 뒤에 설계에 대응하는 최상위 선언 수다(daemok coverage의 합계).
	Coverage Coverage `json:"coverage"`
}

// Coverage는 최상위 선언 수와, 초안 전후의 설계 대응 선언 수다.
type Coverage struct {
	Decls  int `json:"decls"`
	Before int `json:"before"`
	After  int `json:"after"`
}

// ErrDesign은 설계를 읽지 못했거나 설계 검사 오류가 있다는 뜻이다.
var ErrDesign = errors.New("설계에 오류가 있어 초안을 만들 수 없다")

// draftName은 검증할 때 초안에 붙이는 설계 파일 이름이다.
const draftName = "daemok-l3-draft.yaml"

var kindOrder = map[string]int{"errors": 0, "consts": 1, "enums": 2, "types": 3, "contracts": 4}

// Draft는 designDir의 설계와 root의 코드로 초안을 만든다.
func Draft(designDir, root string) (*Result, error) {
	sources, err := readSources(designDir)
	if err != nil {
		return nil, err
	}
	m, diags := model.LoadSources(sources)
	if !diags.HasErrors() {
		diags = append(diags, check.Run(m)...)
	}
	if diags.HasErrors() {
		return nil, ErrDesign
	}
	expected, err := coverage.DesignNames(m)
	if err != nil {
		return nil, err
	}
	c := &collector{m: m, root: root, fset: token.NewFileSet(), res: &Result{}, units: units(m)}
	for _, u := range c.units {
		c.unit(u.id, u.path, expected[u.path])
	}
	nodes, err := validate(sources, designDir, root, dedupe(c.nodes, c.res), c.res)
	if err != nil {
		return nil, err
	}
	c.res.Nodes = nodes
	c.res.YAML = render(nodes)
	if c.res.Coverage, err = coverageDelta(m, sources, designDir, root, c.res.YAML); err != nil {
		return nil, err
	}
	sort.Slice(c.res.Skipped, func(i, j int) bool { return c.res.Skipped[i].Where < c.res.Skipped[j].Where })
	return c.res, nil
}

// coverageDelta는 설계만으로 센 coverage와 초안을 더해 센 coverage다.
func coverageDelta(m *model.Model, sources map[string][]byte, designDir, root string, yaml []byte) (Coverage, error) {
	total := func(m *model.Model) (decls, matched int, err error) {
		rows, _, err := coverage.Count(m, root)
		for _, r := range rows {
			decls += r.Decls
			matched += r.Matched
		}
		return decls, matched, err
	}
	var cov Coverage
	var err error
	if cov.Decls, cov.Before, err = total(m); err != nil {
		return cov, err
	}
	files := map[string][]byte{filepath.Join(designDir, draftName): yaml}
	for k, v := range sources {
		files[k] = v
	}
	withDraft, _ := model.LoadSources(files)
	_, cov.After, err = total(withDraft)
	return cov, err
}

func readSources(designDir string) (map[string][]byte, error) {
	paths, _ := filepath.Glob(filepath.Join(designDir, "*.yaml"))
	if len(paths) == 0 {
		return nil, fmt.Errorf("%w: 설계 파일이 없다: %s", ErrDesign, designDir)
	}
	out := map[string][]byte{}
	for _, p := range paths {
		b, err := os.ReadFile(p)
		if err != nil {
			return nil, err
		}
		out[p] = b
	}
	return out, nil
}

type unit struct{ id, path string }

// units는 모듈과 그룹 멤버를 ID 순으로 준다.
func units(m *model.Model) []unit {
	var out []unit
	for _, mod := range m.Modules {
		out = append(out, unit{mod.ID, mod.Path})
	}
	for _, g := range m.Groups {
		for _, mem := range g.Members {
			out = append(out, unit{mem.ID, mem.Path()})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

type collector struct {
	docs  map[*ast.Ident]*ast.CommentGroup // 선언 이름에서 그 선언의 문서 주석으로
	m     *model.Model
	units []unit
	root  string
	fset  *token.FileSet
	nodes []Node
	res   *Result
}

func (c *collector) where(p token.Pos) string {
	pos := c.fset.Position(p)
	rel, err := filepath.Rel(c.root, pos.Filename)
	if err != nil {
		rel = pos.Filename
	}
	return filepath.ToSlash(rel) + ":" + strconv.Itoa(pos.Line)
}

func (c *collector) skip(p token.Pos, name, reason string) {
	c.res.Skipped = append(c.res.Skipped, Skip{c.where(p), name, reason})
}

func (c *collector) add(kind, module, id string, name *ast.Ident, fields ...string) {
	head := []string{"in: " + module, "gen: false"}
	if q, ok := quote(summary(c.docs[name])); ok && q != `""` {
		head = append(head, "summary: "+q)
	}
	n := Node{Kind: kind, Module: module, ID: id, Name: name.Name, Where: c.where(name.Pos()), fields: append(head, fields...)}
	n.render()
	c.nodes = append(c.nodes, n)
}

// platformVariants는 한 모듈에 같은 Go 이름의 같은 종류 선언이 여러 번 있으면(build 조건이 다른 플랫폼 파일)
// 내용이 모두 같을 때 하나만 남기고, 다르면 모두 뺀다. 한 플랫폼의 값을 설계에 적으면 다른 플랫폼에서 틀린다.
func platformVariants(nodes []Node, res *Result) []Node {
	key := func(n Node) string { return n.Kind + " " + n.Module + " " + n.Name }
	body := func(n Node) string {
		var fs []string
		for _, f := range n.fields {
			if !strings.HasPrefix(f, "summary: ") {
				fs = append(fs, f)
			}
		}
		return strings.Join(fs, ", ")
	}
	first := map[string]Node{}
	differs := map[string]bool{}
	for _, n := range nodes {
		if f, ok := first[key(n)]; !ok {
			first[key(n)] = n
		} else if body(f) != body(n) {
			differs[key(n)] = true
		}
	}
	var out []Node
	emitted := map[string]bool{}
	for _, n := range nodes {
		k := key(n)
		if emitted[k] {
			continue
		}
		emitted[k] = true
		if differs[k] {
			res.Skipped = append(res.Skipped, Skip{n.Where, n.Name, "플랫폼 파일마다 선언이 다르다"})
			continue
		}
		out = append(out, n)
	}
	return out
}

// dedupe는 초안 안에서 같은 종류의 ID가 겹치면(모듈마다 있는 Config, Service) 모듈 ID를 앞에 붙이고
// go.name으로 Go 이름을 지킨다. 에러는 ID가 Go 이름이고 enum은 go 블록이 달라 바꾸지 않는다.
// 그래도 겹치는 노드는 첫 것만 남긴다. 초안 YAML에 같은 키가 두 번 나오면 설계를 읽을 수 없다.
func dedupe(nodes []Node, res *Result) []Node {
	nodes = platformVariants(nodes, res)
	seen := map[string]int{}
	for _, n := range nodes {
		seen[n.Kind+" "+n.ID]++
	}
	for i := range nodes {
		n := &nodes[i]
		if seen[n.Kind+" "+n.ID] < 2 || n.Kind == "errors" || n.Kind == "enums" {
			continue
		}
		n.ID = n.Module + "_" + n.ID
		if !slices.ContainsFunc(n.fields, func(f string) bool { return strings.HasPrefix(f, "go: ") }) {
			n.fields = append(n.fields, "go: { name: "+n.Name+" }")
		}
		n.render()
	}
	kept := map[string]bool{}
	var out []Node
	for _, n := range nodes {
		key := n.Kind + " " + n.ID
		if kept[key] {
			res.Skipped = append(res.Skipped, Skip{n.Where, n.Name, "초안 안에서 ID가 겹친다: " + n.ID})
			continue
		}
		kept[key] = true
		out = append(out, n)
	}
	return out
}
