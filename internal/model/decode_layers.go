package model

import "github.com/tolelom/daemok/internal/yaml"

func (mg *merger) requirements(n *yaml.Node) {
	d := mg.d
	d.idMap(n, "requirements", func(id string, pos yaml.Pos, v *yaml.Node) {
		d.allow(v, "text", "status", "summary")
		if !mg.unique("요구사항", id, pos) {
			return
		}
		r := &Requirement{Meta: Meta{ID: id, Pos: pos, Summary: d.str(v, "summary")}}
		if d.need(v, "text") {
			r.Text = d.str(v, "text")
		}
		r.Status = d.str(v, "status")
		if r.Status == "" {
			r.Status = "active"
		}
		switch r.Status {
		case "active", "held", "dropped":
		default:
			d.diags.Errorf(v.Get("status").Pos, "status는 active, held, dropped 중 하나여야 한다")
		}
		mg.m.Requirements = append(mg.m.Requirements, r)
	})
}

func (mg *merger) system(n *yaml.Node)      {}
func (mg *merger) executables(n *yaml.Node) {}
func (mg *merger) tiers(n *yaml.Node)       {}
func (mg *merger) modules(n *yaml.Node)     {}
func (mg *merger) groups(n *yaml.Node)      {}
