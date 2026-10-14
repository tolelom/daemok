// Package designdiff는 두 설계 파일 묶음을 노드와 키 단위로 비교해 변경 요약을 만든다.
// 비교는 YAML 값으로 해서 주석, 정렬, 형식, 노드가 든 파일은 변경으로 보지 않는다.
package designdiff

import (
	"fmt"
	"sort"

	"github.com/tolelom/daemok/internal/yaml"
)

// kinds는 노드 종류와 출력 순서다. 설계 파일의 키 순서를 따른다.
var kinds = []string{
	"requirements", "system", "actors", "interfaces", "executables", "modules", "groups", "members",
	"enums", "consts", "errors", "types", "contracts", "handler_kinds", "event_kinds", "events", "funcs", "flows",
}

// settings는 노드가 아닌 최상위 설정 키다.
var settings = []string{"version", "go_module", "ts_out", "tiers"}

type node struct {
	value *yaml.Node
	group string // 멤버의 소속 그룹
	file  string
}

// design은 설계 하나의 노드(종류 → ID → 값)와 설정이다.
type design struct {
	nodes    map[string]map[string]node
	settings map[string]*yaml.Node
}

// collect는 파일들을 이름순으로 읽어 노드를 모은다. 같은 종류·ID가 두 번 나오면 어느 쪽 변경을 볼지 정할 수 없어 오류다.
func collect(files map[string][]byte) (*design, error) {
	d := &design{nodes: map[string]map[string]node{}, settings: map[string]*yaml.Node{}}
	var dup error
	for _, k := range kinds {
		d.nodes[k] = map[string]node{}
	}
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		root, err := yaml.Parse(name, files[name])
		if err != nil {
			return nil, err
		}
		for _, e := range root.Entries {
			switch e.Key {
			case "version", "go_module", "ts_out", "tiers":
				if d.settings[e.Key] == nil {
					d.settings[e.Key] = e.Value
				}
			case "system":
				d.add("system", "", without(e.Value, "actors", "interfaces"), "", name, &dup)
				for _, sub := range []string{"actors", "interfaces"} {
					d.addAll(sub, e.Value.Get(sub), "", name, &dup)
				}
			case "groups":
				if e.Value.Kind != yaml.Map {
					continue
				}
				for _, g := range e.Value.Entries {
					d.add("groups", g.Key, without(g.Value, "members"), "", name, &dup)
					d.addAll("members", g.Value.Get("members"), g.Key, name, &dup)
				}
			default:
				if _, ok := d.nodes[e.Key]; ok {
					d.addAll(e.Key, e.Value, "", name, &dup)
				}
			}
		}
	}
	return d, dup
}

func (d *design) add(kind, id string, v *yaml.Node, group, file string, dup *error) {
	if prev, ok := d.nodes[kind][id]; ok {
		if *dup == nil {
			*dup = fmt.Errorf("같은 ID가 두 곳에 있다: %s %q (%s, %s)", kind, id, prev.file, file)
		}
		return
	}
	d.nodes[kind][id] = node{v, group, file}
}

func (d *design) addAll(kind string, m *yaml.Node, group, file string, dup *error) {
	if m == nil || m.Kind != yaml.Map {
		return
	}
	for _, e := range m.Entries {
		d.add(kind, e.Key, e.Value, group, file, dup)
	}
}

// without은 맵 n에서 keys를 뺀 사본이다. 맵이 아니면 그대로 준다.
func without(n *yaml.Node, keys ...string) *yaml.Node {
	if n == nil || n.Kind != yaml.Map {
		return n
	}
	out := *n
	out.Entries = nil
	for _, e := range n.Entries {
		skip := false
		for _, k := range keys {
			skip = skip || e.Key == k
		}
		if !skip {
			out.Entries = append(out.Entries, e)
		}
	}
	return &out
}
