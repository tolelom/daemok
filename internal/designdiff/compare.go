package designdiff

import (
	"sort"
	"strconv"
	"strings"

	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

type NodeRef struct{ Kind, ID string }

type Rename struct{ Kind, From, To string }

type Modified struct {
	NodeRef
	Lines []string
}

type Report struct {
	Settings []string
	Added    []NodeRef
	Renamed  []Rename
	Modified []Modified
	Removed  []NodeRef
}

func (r *Report) Empty() bool {
	return len(r.Settings)+len(r.Added)+len(r.Renamed)+len(r.Modified)+len(r.Removed) == 0
}

var pairKeys = map[string]string{"steps": "id", "cases": "name"}

var orderedMaps = map[string]bool{"fields": true, "params": true, "data": true}

var pinKeys = map[string]string{
	"modules": "path", "groups": "path", "enums": "go.type",
	"types": "go.name", "contracts": "go.name", "handler_kinds": "go.name", "funcs": "go.name", "errors": "go.name", "consts": "go.name",
}

func pinValue(kind, old string) string {
	switch kind {
	case "modules", "groups", "errors":
		return old
	}
	return model.GoName(old)
}

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
		var added, removed, common []string
		for id := range wn {
			if _, ok := bn[id]; ok {
				common = append(common, id)
			} else {
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
		renamed := map[string]string{}
		for _, p := range pairRenames(kind, bn, wn, removed, added) {
			r.Renamed = append(r.Renamed, Rename{kind, p[0], p[1]})
			renamed[p[1]] = p[0]
		}
		gone := map[string]bool{}
		for _, old := range renamed {
			gone[old] = true
		}
		for _, id := range added {
			if _, ok := renamed[id]; ok {
				common = append(common, id)
			} else {
				r.Added = append(r.Added, NodeRef{kind, id})
			}
		}
		for _, id := range removed {
			if !gone[id] {
				r.Removed = append(r.Removed, NodeRef{kind, id})
			}
		}
		sort.Strings(common)
		for _, id := range common {
			old := id
			if o, ok := renamed[id]; ok {
				old = o
			}
			lines := diffValue("", bn[old].value, wn[id].value)
			if bg, wg := bn[old].group, wn[id].group; bg != wg {
				lines = append(lines, "그룹: "+bg+" → "+wg)
			}
			if len(lines) > 0 {
				r.Modified = append(r.Modified, Modified{NodeRef{kind, id}, lines})
			}
		}
	}
	return r, nil
}

func pairRenames(kind string, bn, wn map[string]node, removed, added []string) [][2]string {
	match := map[string][]string{}
	back := map[string]int{}
	for _, o := range removed {
		ov := bn[o].value
		if norm(ov) == nil {
			continue
		}
		for _, n := range added {
			if bn[o].group == wn[n].group && len(diffValue("", ov, unpinned(kind, o, ov, wn[n].value))) == 0 {
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

func unpinned(kind, old string, ov, nv *yaml.Node) *yaml.Node {
	k, ok := pinKeys[kind]
	if !ok || norm(nv) == nil {
		return nv
	}
	top, sub, nested := strings.Cut(k, ".")
	if !nested {
		if ov.Get(top) == nil && nv.Get(top) != nil && nv.Get(top).Value == pinValue(kind, old) {
			return without(nv, top)
		}
		return nv
	}
	g := nv.Get(top)
	if g == nil || g.Get(sub) == nil || g.Get(sub).Value != pinValue(kind, old) || ov.Get(top).Get(sub) != nil {
		return nv
	}
	out := without(nv, top)
	if rest := without(g, sub); norm(rest) != nil {
		out.Entries = append(out.Entries, yaml.Entry{Key: top, Value: rest})
	}
	return out
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

func norm(n *yaml.Node) *yaml.Node {
	switch {
	case n == nil, n.IsNull():
		return nil
	case n.Kind == yaml.Seq && len(n.Items) == 0, n.Kind == yaml.Map && len(n.Entries) == 0:
		return nil
	}
	return n
}

func typed(v string) bool {
	if v == "" || v == "null" || v == "true" || v == "false" {
		return true
	}
	_, err := strconv.Atoi(v)
	return err == nil
}

func sameScalar(a, b *yaml.Node) bool {
	return a.Value == b.Value && (a.Quoted == b.Quoted || !typed(a.Value))
}

func diffValue(path string, a, b *yaml.Node) []string {
	a, b = norm(a), norm(b)
	label := path
	if label == "" {
		label = "값"
	}
	switch {
	case a == nil && b == nil:
		return nil
	case a == nil:
		return []string{label + " 추가: " + render(b)}
	case b == nil:
		return []string{label + " 삭제: " + render(a)}
	case a.Kind != b.Kind:
		return []string{label + ": " + render(a) + " → " + render(b)}
	}
	switch a.Kind {
	case yaml.Scalar:
		if !sameScalar(a, b) {
			return []string{label + ": " + render(a) + " → " + render(b)}
		}
		return nil
	case yaml.Map:
		var out []string
		seen := map[string]bool{}
		var aKeys, bKeys []string
		for _, e := range a.Entries {
			seen[e.Key] = true
			if b.Get(e.Key) != nil {
				aKeys = append(aKeys, e.Key)
			}
			out = append(out, diffValue(join(path, e.Key), e.Value, b.Get(e.Key))...)
		}
		for _, e := range b.Entries {
			if seen[e.Key] {
				bKeys = append(bKeys, e.Key)
			} else {
				out = append(out, diffValue(join(path, e.Key), nil, e.Value)...)
			}
		}
		if orderedMaps[path[strings.LastIndex(path, ".")+1:]] && strings.Join(aKeys, "\x00") != strings.Join(bKeys, "\x00") {
			out = append([]string{label + ": 순서 바뀜"}, out...)
		}
		return out
	}
	return diffSeq(label, a.Items, b.Items)
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
			plus = append(plus, "+"+render(x))
		}
	}
	for _, x := range a {
		if count[x.Value] > 0 {
			count[x.Value]--
			minus = append(minus, "-"+render(x))
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
	if n.Quoted && (typed(n.Value) || strings.ContainsAny(n.Value, ",:{}[]\"\\\n\r\t")) {
		return strconv.Quote(n.Value)
	}
	return n.Value
}
