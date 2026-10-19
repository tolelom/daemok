package designdiff

import (
	"fmt"
	"sort"

	"github.com/tolelom/daemok/internal/yaml"
)

var kinds = []string{
	"requirements", "system", "actors", "interfaces", "executables", "modules", "groups", "members",
	"enums", "consts", "errors", "types", "contracts", "handler_kinds", "event_kinds", "events", "funcs", "flows",
}

var settings = []string{"version", "go_module", "ts_out", "cpp", "tiers"}

type node struct {
	value *yaml.Node
	group string
	file  string
}

type design struct {
	nodes    map[string]map[string]node
	settings map[string]*yaml.Node
}

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
			case "version", "go_module", "ts_out", "cpp", "tiers":
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
