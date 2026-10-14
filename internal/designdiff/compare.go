package designdiff

import (
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/yaml"
)

type NodeRef struct{ Kind, ID string }

type Rename struct{ Kind, From, To string }

// Modified는 바뀐 노드와 키 경로별 변경 줄이다.
type Modified struct {
	NodeRef
	Lines []string
}

// Report는 기준 대비 작업 설계의 변경이다. 각 목록은 종류 순서, 같은 종류 안에서는 ID 순이다.
type Report struct {
	Settings []string
	Added    []NodeRef
	Renamed  []Rename
	Modified []Modified
	Removed  []NodeRef
}

// Empty는 변경이 없는지다.
func (r *Report) Empty() bool {
	return len(r.Settings)+len(r.Added)+len(r.Renamed)+len(r.Modified)+len(r.Removed) == 0
}

// pairKeys는 맵 목록의 원소를 짝지을 키다. 키 경로의 마지막 이름으로 찾는다.
var pairKeys = map[string]string{"steps": "id", "cases": "name"}

// Compare는 기준 설계 파일들과 작업 설계 파일들을 비교한다. 어느 파일이든 YAML로 읽지 못하면 오류다.
func Compare(base, work map[string][]byte) (*Report, error) {
	b, err := collect(base)
	if err != nil {
		return nil, err
	}
	w, err := collect(work)
	if err != nil {
		return nil, err
	}
	r := &Report{}
	for _, k := range settings {
		r.Settings = append(r.Settings, diffValue(k, b.settings[k], w.settings[k])...)
	}
	for _, kind := range kinds {
		bn, wn := b.nodes[kind], w.nodes[kind]
		var added, removed []string
		for id := range wn {
			if _, ok := bn[id]; !ok {
				added = append(added, id)
			}
		}
		for id := range bn {
			if _, ok := wn[id]; !ok {
				removed = append(removed, id)
			}
		}
		sort.Strings(added)
		sort.Strings(removed)
		renamed := pairRenames(bn, wn, removed, added)
		for _, rn := range renamed {
			r.Renamed = append(r.Renamed, Rename{kind, rn[0], rn[1]})
		}
		for _, id := range added {
			if !isRenamed(renamed, 1, id) {
				r.Added = append(r.Added, NodeRef{kind, id})
			}
		}
		for _, id := range removed {
			if !isRenamed(renamed, 0, id) {
				r.Removed = append(r.Removed, NodeRef{kind, id})
			}
		}
		var ids []string
		for id := range wn {
			if _, ok := bn[id]; ok {
				ids = append(ids, id)
			}
		}
		sort.Strings(ids)
		for _, id := range ids {
			lines := diffValue("", bn[id].value, wn[id].value)
			if bg, wg := bn[id].group, wn[id].group; bg != wg {
				lines = append(lines, "그룹: "+bg+" → "+wg)
			}
			if len(lines) > 0 {
				r.Modified = append(r.Modified, Modified{NodeRef{kind, id}, lines})
			}
		}
	}
	return r, nil
}

// pairRenames는 삭제·추가 노드 가운데 고정 키를 뺀 값이 같은 짝이 서로 하나뿐인 것을 [옛, 새]로 준다.
func pairRenames(bn, wn map[string]node, removed, added []string) [][2]string {
	match := map[string][]string{}
	back := map[string]int{}
	for _, o := range removed {
		for _, n := range added {
			if bn[o].group == wn[n].group && len(diffValue("", unpinned(bn[o].value), unpinned(wn[n].value))) == 0 {
				match[o] = append(match[o], n)
				back[n]++
			}
		}
	}
	var out [][2]string
	for _, o := range removed {
		if len(match[o]) == 1 && back[match[o][0]] == 1 {
			out = append(out, [2]string{o, match[o][0]})
		}
	}
	return out
}

func isRenamed(pairs [][2]string, side int, id string) bool {
	for _, p := range pairs {
		if p[side] == id {
			return true
		}
	}
	return false
}

// unpinned는 이름 바꾸기가 더하는 고정 키(path, go.name, go.type)를 뺀 노드 값이다.
func unpinned(n *yaml.Node) *yaml.Node {
	out := without(n, "path", "go")
	if g := n.Get("go"); g != nil {
		if rest := without(g, "name", "type"); rest.Kind != yaml.Map || len(rest.Entries) > 0 {
			out.Entries = append(out.Entries, yaml.Entry{Key: "go", Value: rest})
		}
	}
	return out
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// diffValue는 path의 두 값의 차이를 줄로 준다. 같으면 nil이다.
func diffValue(path string, a, b *yaml.Node) []string {
	switch {
	case a == nil && b == nil:
		return nil
	case a == nil:
		return []string{path + " 추가: " + render(b)}
	case b == nil:
		return []string{path + " 삭제: " + render(a)}
	case a.Kind != b.Kind:
		return []string{path + ": " + render(a) + " → " + render(b)}
	}
	switch a.Kind {
	case yaml.Scalar:
		if a.Value != b.Value {
			return []string{path + ": " + render(a) + " → " + render(b)}
		}
		return nil
	case yaml.Map:
		var out []string
		seen := map[string]bool{}
		for _, e := range a.Entries {
			seen[e.Key] = true
			out = append(out, diffValue(join(path, e.Key), e.Value, b.Get(e.Key))...)
		}
		for _, e := range b.Entries {
			if !seen[e.Key] {
				out = append(out, diffValue(join(path, e.Key), nil, e.Value)...)
			}
		}
		return out
	}
	return diffSeq(path, a.Items, b.Items)
}

func diffSeq(path string, a, b []*yaml.Node) []string {
	if allKind(a, yaml.Scalar) && allKind(b, yaml.Scalar) {
		return diffScalars(path, a, b)
	}
	last := path[strings.LastIndex(path, ".")+1:]
	if key, ok := pairKeys[last]; ok && keyed(a, key) && keyed(b, key) {
		return diffPaired(path, key, a, b)
	}
	if len(a) == len(b) {
		same := true
		for i := range a {
			same = same && len(diffValue("", a[i], b[i])) == 0
		}
		if same {
			return nil
		}
	}
	return []string{path + ": 값 바뀜"}
}

func diffScalars(path string, a, b []*yaml.Node) []string {
	count := map[string]int{}
	for _, x := range a {
		count[x.Value]++
	}
	var plus, minus []string
	for _, x := range b {
		if count[x.Value] > 0 {
			count[x.Value]--
		} else {
			plus = append(plus, "+"+x.Value)
		}
	}
	for _, x := range a {
		if count[x.Value] > 0 {
			count[x.Value]--
			minus = append(minus, "-"+x.Value)
		}
	}
	if len(plus)+len(minus) > 0 {
		return []string{path + ": " + strings.Join(append(plus, minus...), " ")}
	}
	for i := range a {
		if a[i].Value != b[i].Value {
			return []string{path + ": 순서 바뀜"}
		}
	}
	return nil
}

func diffPaired(path, key string, a, b []*yaml.Node) []string {
	at := map[string]*yaml.Node{}
	var aOrder []string
	for _, x := range a {
		k := x.Get(key).Value
		at[k] = x
		aOrder = append(aOrder, k)
	}
	inB := map[string]bool{}
	var out, common []string
	for _, x := range b {
		k := x.Get(key).Value
		inB[k] = true
		p := path + "[" + k + "]"
		if old, ok := at[k]; ok {
			common = append(common, k)
			out = append(out, diffValue(p, old, x)...)
		} else {
			out = append(out, p+" 추가: "+render(x))
		}
	}
	var commonA []string
	for _, k := range aOrder {
		if inB[k] {
			commonA = append(commonA, k)
		} else {
			out = append(out, path+"["+k+"] 삭제: "+render(at[k]))
		}
	}
	if strings.Join(commonA, "\x00") != strings.Join(common, "\x00") {
		out = append([]string{path + ": 순서 바뀜"}, out...)
	}
	return out
}

func allKind(ns []*yaml.Node, k yaml.Kind) bool {
	for _, n := range ns {
		if n.Kind != k {
			return false
		}
	}
	return true
}

// keyed는 모든 원소가 key를 스칼라로 갖고 그 값이 겹치지 않는지다.
func keyed(ns []*yaml.Node, key string) bool {
	seen := map[string]bool{}
	for _, n := range ns {
		v := n.Get(key)
		if v == nil || v.Kind != yaml.Scalar || seen[v.Value] {
			return false
		}
		seen[v.Value] = true
	}
	return true
}

// render는 값을 한 줄로 쓴다.
func render(n *yaml.Node) string {
	switch n.Kind {
	case yaml.Map:
		var parts []string
		for _, e := range n.Entries {
			parts = append(parts, e.Key+": "+render(e.Value))
		}
		if len(parts) == 0 {
			return "{}"
		}
		return "{ " + strings.Join(parts, ", ") + " }"
	case yaml.Seq:
		var parts []string
		for _, it := range n.Items {
			parts = append(parts, render(it))
		}
		return "[" + strings.Join(parts, ", ") + "]"
	}
	if n.Quoted && (n.Value == "" || strings.ContainsAny(n.Value, ",:{}[]")) {
		return `"` + n.Value + `"`
	}
	return n.Value
}
