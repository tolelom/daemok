package lift

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/workspace"
)

// validate는 초안을 설계에 합쳐 돌리고, 초안 줄을 가리키는 오류·경고가 나온 노드를 빼고 다시 돌린다.
// 초안 밖에 새 오류·경고가 생기면 어느 노드 탓인지 알 수 없어 오류로 끝낸다.
func validate(sources map[string][]byte, designDir, root string, nodes []Node, res *Result) ([]Node, error) {
	sort.SliceStable(nodes, func(i, j int) bool {
		a, b := nodes[i], nodes[j]
		if a.Kind != b.Kind {
			return kindOrder[a.Kind] < kindOrder[b.Kind]
		}
		if a.Module != b.Module {
			return a.Module < b.Module
		}
		return a.ID < b.ID
	})
	base := counts(workspace.RunSources(sources, designDir, root).Diags)
	key := filepath.Join(designDir, draftName)
	// 한 라운드는 진단이 난 노드를 모두 빼므로 라운드 수는 참조 연쇄의 깊이다. YAML 문법 오류처럼 첫 오류에서
	// 멈추는 진단이면 노드를 하나씩만 빼게 되어 상한에서 끝낸다.
	for round := 0; len(nodes) > 0; round++ {
		if round == maxRounds {
			return nil, fmt.Errorf("초안 검증이 %d번 안에 끝나지 않았다", maxRounds)
		}
		files := map[string][]byte{key: render(nodes)}
		for k, v := range sources {
			files[k] = v
		}
		r := workspace.RunSources(files, designDir, root)

		lineOf := lineNodes(nodes)
		bad := map[int]string{}
		var outside []string
		for _, d := range r.Diags {
			if d.Level == model.Info {
				continue
			}
			if d.Pos.File == key {
				if i, ok := lineOf[d.Pos.Line]; ok {
					if _, seen := bad[i]; !seen {
						bad[i] = d.Msg
					}
					continue
				}
			}
			outside = append(outside, d.String())
		}
		if len(bad) == 0 {
			if extra := added(base, outside); len(extra) > 0 {
				return nil, fmt.Errorf("초안이 설계의 다른 곳에 진단을 낸다: %s", strings.Join(extra, "; "))
			}
			return nodes, nil
		}
		var kept []Node
		for i, n := range nodes {
			if msg, ok := bad[i]; ok {
				res.Skipped = append(res.Skipped, Skip{n.Where, n.Name, "설계에 더하면 진단이 난다: " + msg})
				continue
			}
			kept = append(kept, n)
		}
		nodes = kept
	}
	return nodes, nil
}

func counts(ds model.Diagnostics) map[string]int {
	out := map[string]int{}
	for _, d := range ds {
		if d.Level != model.Info {
			out[d.String()]++
		}
	}
	return out
}

func added(base map[string]int, now []string) []string {
	seen := map[string]int{}
	var out []string
	for _, s := range now {
		seen[s]++
		if seen[s] > base[s] {
			out = append(out, s)
		}
	}
	return out
}

const maxRounds = 20

// header는 초안 파일 머리 주석의 줄 수와 내용이다. lineNodes가 줄 번호를 맞추는 데 쓴다.
const header = "# daemok init --l3가 만든 L3 초안이다. 설계에 없는 손으로 쓴 선언을 gen: false 노드로 옮겼다.\n" +
	"# 새 설계 파일로 저장하거나 기존 설계 파일에 옮겨 붙인다.\n"

func render(nodes []Node) []byte {
	var b bytes.Buffer
	b.WriteString(header)
	kind := ""
	for _, n := range nodes {
		if n.Kind != kind {
			kind = n.Kind
			b.WriteString("\n" + kind + ":\n")
		}
		b.WriteString("  " + n.Line + "\n")
	}
	return b.Bytes()
}

// lineNodes는 render가 쓴 줄 번호(1부터)에서 노드 번호로 가는 표다.
func lineNodes(nodes []Node) map[int]int {
	out := map[int]int{}
	line := strings.Count(header, "\n")
	kind := ""
	for i, n := range nodes {
		if n.Kind != kind {
			kind = n.Kind
			line += 2
		}
		line++
		out[line] = i
	}
	return out
}
