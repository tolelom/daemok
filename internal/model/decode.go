package model

import (
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/yaml"
)

type decoder struct {
	diags *Diagnostics
}

func (d *decoder) mapping(n *yaml.Node, what string) bool {
	if n == nil {
		return false
	}
	if n.Kind != yaml.Map {
		d.diags.Errorf(n.Pos, "맵이어야 한다: %s", what)
		return false
	}
	return true
}

func (d *decoder) allow(n *yaml.Node, keys ...string) {
	for _, e := range n.Entries {
		if !slices.Contains(keys, e.Key) {
			d.diags.Errorf(e.Pos, "알 수 없는 키 %q", e.Key)
		}
	}
}

func (d *decoder) scalar(n *yaml.Node, key string) *yaml.Node {
	v := n.Get(key)
	if v == nil {
		return nil
	}
	if v.Kind != yaml.Scalar {
		d.diags.Errorf(v.Pos, "값 하나여야 한다: %q", key)
		return nil
	}
	return v
}

// oneOf는 key 값이 choices 중 하나인지 본다. 키가 없으면 def.
func (d *decoder) oneOf(n *yaml.Node, key, def string, choices ...string) string {
	v := d.scalar(n, key)
	if v == nil || v.IsNull() {
		return def
	}
	if !slices.Contains(choices, v.Value) {
		d.diags.Errorf(v.Pos, "허용하지 않는 값: %s %q (가능한 값: %s)", key, v.Value, strings.Join(choices, ", "))
		return def
	}
	return v.Value
}

func (d *decoder) str(n *yaml.Node, key string) string {
	if v := d.scalar(n, key); v != nil {
		return v.Value
	}
	return ""
}

// need는 key가 있고 null이 아닌지 본다. 오류 위치는 노드의 ID 키(pos)다.
func (d *decoder) need(pos yaml.Pos, n *yaml.Node, key string) bool {
	if v := n.Get(key); v == nil || v.IsNull() {
		d.diags.Errorf(pos, "필요한 키가 없다: %q", key)
		return false
	}
	return true
}

func (d *decoder) ref(n *yaml.Node, key string) Ref {
	v := d.scalar(n, key)
	if v == nil || v.IsNull() {
		return Ref{}
	}
	return Ref{Name: v.Value, Pos: v.Pos}
}

func (d *decoder) refs(n *yaml.Node, key string) []Ref {
	v := n.Get(key)
	if v == nil || v.IsNull() {
		return nil
	}
	if v.Kind != yaml.Seq {
		d.diags.Errorf(v.Pos, "목록이어야 한다: %q", key)
		return nil
	}
	out := []Ref{}
	for _, it := range v.Items {
		if it.Kind != yaml.Scalar {
			d.diags.Errorf(it.Pos, "%q의 항목은 값 하나여야 한다", key)
			continue
		}
		out = append(out, Ref{Name: it.Value, Pos: it.Pos})
	}
	return out
}

func (d *decoder) strs(n *yaml.Node, key string) []string {
	var out []string
	for _, r := range d.refs(n, key) {
		out = append(out, r.Name)
	}
	return out
}

func (d *decoder) intv(n *yaml.Node, key string) (int, bool) {
	v := d.scalar(n, key)
	if v == nil {
		return 0, false
	}
	i, ok := v.Int()
	if !ok {
		d.diags.Errorf(v.Pos, "정수여야 한다: %q", key)
	}
	return i, ok
}

func (d *decoder) boolv(n *yaml.Node, key string, def bool) bool {
	v := d.scalar(n, key)
	if v == nil {
		return def
	}
	b, ok := v.Bool()
	if !ok {
		d.diags.Errorf(v.Pos, "true 또는 false여야 한다: %q", key)
		return def
	}
	return b
}

func (d *decoder) meta(id string, pos yaml.Pos, n *yaml.Node) Meta {
	return Meta{ID: id, Pos: pos, Summary: d.str(n, "summary"), Satisfies: d.refs(n, "satisfies")}
}

func (d *decoder) owned(id string, pos yaml.Pos, n *yaml.Node) Owned {
	o := Owned{Meta: d.meta(id, pos, n), Gen: d.boolv(n, "gen", true)}
	if d.need(pos, n, "in") {
		o.In = d.ref(n, "in")
	}
	return o
}

func (d *decoder) goOpts(n *yaml.Node) GoOpts {
	g := n.Get("go")
	if g == nil || !d.mapping(g, "go") {
		return GoOpts{}
	}
	d.allow(g, "type", "name", "prefix", "register")
	return GoOpts{Type: d.str(g, "type"), Name: d.str(g, "name"), Prefix: d.str(g, "prefix"), Register: d.str(g, "register")}
}

// idMap은 ID를 키로 하는 맵을 돈다. 값이 맵이 아닌 항목은 오류로 남기고 건너뛴다.
func (d *decoder) idMap(n *yaml.Node, what string, fn func(id string, pos yaml.Pos, v *yaml.Node)) {
	if n == nil || n.IsNull() || !d.mapping(n, what) {
		return
	}
	for _, e := range n.Entries {
		if e.Value.Kind != yaml.Map {
			d.diags.Errorf(e.Pos, "맵이어야 한다: %s %q", what, e.Key)
			continue
		}
		fn(e.Key, e.Pos, e.Value)
	}
}
