package lift

import (
	"regexp"
	"slices"
	"strings"
)

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

// qualified는 필드 타입 문자열 안의 "모듈.Go이름"이다. 앞 글자가 이름의 일부면 맞추지 않는다.
var qualified = regexp.MustCompile(`(^|[^A-Za-z0-9_.])([a-z][a-z0-9_]*)\.([A-Z][A-Za-z0-9_]*)`)

// relink는 다른 모듈 패키지의 타입을 코드 이름(core.Block)으로 가리킨 곳을, 그 타입·enum이 같은 초안에
// 남아 있으면 ID로 바꾼다. 수집할 때는 다른 모듈의 어떤 선언이 초안에 남을지 몰라 코드 이름으로 둔다.
// 요약 문장은 바꾸지 않는다.
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
			if strings.HasPrefix(f, "summary: ") {
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
