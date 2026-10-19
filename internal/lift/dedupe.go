package lift

import (
	"regexp"
	"slices"
	"strings"
)

func platformVariants(nodes []Node, res *Result) []Node {
	key := func(n Node) string { return n.Kind + " " + n.Module + " " + n.Name }
	body := func(n Node) string {
		var fs []string
		for _, f := range n.fields {
			if !strings.HasPrefix(f, "summary: ") {
				fs = append(fs, f)
			}
		}
		var refs []string
		for _, r := range n.errRefs {
			refs = append(refs, "returns "+r.module+"."+r.name)
		}
		slices.Sort(refs)
		return strings.Join(append(fs, refs...), ", ")
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

var qualified = regexp.MustCompile(`(^|[^A-Za-z0-9_.])([a-z][a-z0-9_]*)\.([A-Z][A-Za-z0-9_]*)`)

func relink(nodes []Node) []Node {
	ids := map[string]string{}
	for _, n := range nodes {
		if n.Kind == "types" || n.Kind == "enums" {
			ids[n.Module+"."+n.Name] = n.ID
		}
	}
	for i := range nodes {
		n := &nodes[i]
		changed := false
		for j, f := range n.fields {
			if literalField(f) {
				continue
			}
			g := qualified.ReplaceAllStringFunc(f, func(m string) string {
				sub := qualified.FindStringSubmatch(m)
				if id, ok := ids[sub[2]+"."+sub[3]]; ok {
					return sub[1] + id
				}
				return m
			})
			if g != f {
				n.fields[j], changed = g, true
			}
		}
		if changed {
			n.render()
		}
	}
	return nodes
}

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

func literalField(f string) bool {
	for _, k := range []string{"summary: ", "message: ", "value: "} {
		if strings.HasPrefix(f, k) {
			return true
		}
	}
	return false
}
