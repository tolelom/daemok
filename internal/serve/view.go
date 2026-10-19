package serve

import (
	"path/filepath"
	"strings"

	"github.com/tolelom/daemok/internal/gen/golang"
	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

type vSystem struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

type vReq struct {
	ID     string `json:"id"`
	Text   string `json:"text"`
	Status string `json:"status"`
}

type vActor struct {
	ID      string `json:"id"`
	Summary string `json:"summary"`
}

type vIface struct {
	ID       string   `json:"id"`
	Protocol string   `json:"protocol"`
	Port     *int     `json:"port"`
	Path     *string  `json:"path"`
	Spec     string   `json:"spec"`
	UsedBy   []string `json:"used_by"`
}

type vExe struct {
	ID       string   `json:"id"`
	Summary  string   `json:"summary"`
	Exposes  []string `json:"exposes"`
	Composes []string `json:"composes"`
}

type vNeed struct {
	Contract string   `json:"contract"`
	Target   string   `json:"target"`
	Uses     []string `json:"uses"`
	Why      string   `json:"why"`
}

type vUnit struct {
	ID         string   `json:"id"`
	Type       string   `json:"type"`
	Tier       string   `json:"tier"`
	Summary    string   `json:"summary"`
	Path       string   `json:"path"`
	Role       string   `json:"role,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	Rule       string   `json:"rule,omitempty"`
	DependsOn  []string `json:"depends_on"`
	Implements []string `json:"implements"`
	Satisfies  []string `json:"satisfies"`
	Members    []string `json:"members"`
	Needs      []vNeed  `json:"needs"`
}

type vException struct {
	To     string `json:"to"`
	Reason string `json:"reason"`
}

type vMember struct {
	ID         string       `json:"id"`
	Type       string       `json:"type"`
	Group      string       `json:"group"`
	Summary    string       `json:"summary"`
	Path       string       `json:"path"`
	Exceptions []vException `json:"exceptions"`
	Handles    []string     `json:"handles"`
	Satisfies  []string     `json:"satisfies"`
	Needs      []vNeed      `json:"needs"`
}

type vGo struct {
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Register string `json:"register,omitempty"`
}

type vEnum struct {
	ID        string   `json:"id"`
	In        string   `json:"in"`
	Values    []string `json:"values"`
	Go        vGo      `json:"go"`
	Export    []string `json:"export"`
	Satisfies []string `json:"satisfies"`
}

type vConst struct {
	ID        string   `json:"id"`
	In        string   `json:"in"`
	Type      string   `json:"type"`
	Value     string   `json:"value"`
	Go        vGo      `json:"go"`
	Export    []string `json:"export"`
	Satisfies []string `json:"satisfies"`
}

type vError struct {
	ID        string   `json:"id"`
	In        string   `json:"in"`
	Message   string   `json:"message"`
	HTTP      *int     `json:"http"`
	Go        vGo      `json:"go"`
	Satisfies []string `json:"satisfies"`
}

type vField struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Ref      string `json:"ref"`
	Many     bool   `json:"many"`
	Optional bool   `json:"optional"`
	Max      string `json:"max"`
	Summary  string `json:"summary"`
}

type vType struct {
	ID        string   `json:"id"`
	In        string   `json:"in"`
	Fields    []vField `json:"fields"`
	Go        vGo      `json:"go"`
	Satisfies []string `json:"satisfies"`
}

type vParam struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type vMethod struct {
	Name      string   `json:"name"`
	Params    []vParam `json:"params"`
	Returns   []string `json:"returns"`
	Errors    []string `json:"errors"`
	HasErrors bool     `json:"hasErrors"`
}

type vContract struct {
	ID        string    `json:"id"`
	In        string    `json:"in"`
	Kind      string    `json:"kind"`
	Go        vGo       `json:"go"`
	Satisfies []string  `json:"satisfies"`
	Methods   []vMethod `json:"methods"`
}

type vHK struct {
	ID      string   `json:"id"`
	In      string   `json:"in"`
	Go      vGo      `json:"go"`
	Params  []vParam `json:"params"`
	Returns []string `json:"returns"`
	Key     string   `json:"key"`
}

type vCase struct {
	Name   string `json:"name"`
	Given  any    `json:"given"`
	Expect any    `json:"expect"`
	Error  string `json:"error"`
}

type vFunc struct {
	ID        string   `json:"id"`
	In        string   `json:"in"`
	Kind      string   `json:"kind"`
	Handles   string   `json:"handles"`
	Of        string   `json:"of"`
	Errors    []string `json:"errors"`
	Satisfies []string `json:"satisfies"`
	Cases     []vCase  `json:"cases"`
}

type vEK struct {
	ID          string   `json:"id"`
	In          string   `json:"in"`
	Key         string   `json:"key"`
	Encoding    string   `json:"encoding"`
	Subscribers []string `json:"subscribers"`
	Satisfies   []string `json:"satisfies"`
}

type vEvent struct {
	ID          string   `json:"id"`
	Kind        string   `json:"kind"`
	In          string   `json:"in"`
	Publishers  []string `json:"publishers"`
	Subscribers []string `json:"subscribers"`
	Data        []vField `json:"data"`
	Satisfies   []string `json:"satisfies"`
}

type vStep struct {
	ID          string   `json:"id"`
	Undo        string   `json:"undo"`
	Commit      bool     `json:"commit"`
	Fatal       bool     `json:"fatal"`
	Calls       []string `json:"calls"`
	Emits       []string `json:"emits"`
	Summary     string   `json:"summary"`
	OnFailUndo  []string `json:"onFailUndo"`
	OnFailFatal bool     `json:"onFailFatal"`
}

type vRelation struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Kind  string `json:"kind"`
	Label string `json:"label"`
	Why   string `json:"why"`
}

type vFlow struct {
	ID        string   `json:"id"`
	In        string   `json:"in"`
	On        string   `json:"on"`
	Summary   string   `json:"summary"`
	Steps     []vStep  `json:"steps"`
	Satisfies []string `json:"satisfies"`
}

type vModel struct {
	GoModule  string      `json:"goModule"`
	System    vSystem     `json:"system"`
	Tiers     []string    `json:"tiers"`
	Reqs      []vReq      `json:"reqs"`
	Actors    []vActor    `json:"actors"`
	Ifaces    []vIface    `json:"ifaces"`
	Exes      []vExe      `json:"exes"`
	Mods      []vUnit     `json:"mods"`
	Groups    []vUnit     `json:"groups"`
	Members   []vMember   `json:"members"`
	Enums     []vEnum     `json:"enums"`
	Consts    []vConst    `json:"consts"`
	Errors    []vError    `json:"errors"`
	Types     []vType     `json:"types"`
	Contracts []vContract `json:"contracts"`
	HKs       []vHK       `json:"hks"`
	Funcs     []vFunc     `json:"funcs"`
	EKs       []vEK       `json:"eks"`
	Events    []vEvent    `json:"events"`
	Flows     []vFlow     `json:"flows"`
}

type vIssue struct {
	Lv     string `json:"lv"`
	Msg    string `json:"msg"`
	Key    string `json:"key"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Origin string `json:"origin"`
	Group  string `json:"group"`
	Abs    string `json:"abs"`
}

type vFile struct {
	Path    string   `json:"path"`
	Lang    string   `json:"lang"`
	Code    string   `json:"code"`
	Sources []string `json:"sources"`
}

func names(rs []model.Ref) []string {
	out := []string{}
	for _, r := range rs {
		out = append(out, r.Name)
	}
	return out
}

func strsOrEmpty(ss []string) []string {
	if ss == nil {
		return []string{}
	}
	return ss
}

func vGoOf(g model.GoOpts) vGo {
	return vGo{Type: g.Type, Name: g.Name, Prefix: g.Prefix, Register: g.Register}
}

func vFields(fs []*model.Field) []vField {
	out := []vField{}
	for _, f := range fs {
		out = append(out, vField{Name: f.Name, Type: f.Type.Name, Ref: f.RefType.Name, Many: f.Many, Optional: f.Optional, Max: f.Max.Name, Summary: f.Summary})
	}
	return out
}

func vParams(ps []*model.Param) []vParam {
	out := []vParam{}
	for _, p := range ps {
		out = append(out, vParam{p.Name, p.Type.Name})
	}
	return out
}

func vNeeds(m *model.Model, ns []model.Need) []vNeed {
	out := []vNeed{}
	for _, n := range ns {
		target := ""
		if ct := m.Contract(n.Contract.Name); ct != nil {
			target = ct.In.Name
		}
		out = append(out, vNeed{Contract: n.Contract.Name, Target: target, Uses: names(n.Uses), Why: n.Why})
	}
	return out
}

func nodeValue(n *yaml.Node) any {
	if n == nil {
		return nil
	}
	if n.Kind == yaml.Map {
		out := map[string]any{}
		for _, e := range n.Entries {
			out[e.Key] = nodeValue(e.Value)
		}
		return out
	}
	if n.IsNull() {
		return nil
	}
	if i, ok := n.Int(); ok {
		return i
	}
	if b, ok := n.Bool(); ok {
		return b
	}
	return n.Value
}

func toView(m *model.Model) *vModel {
	if m == nil {
		return nil
	}
	v := &vModel{GoModule: m.GoModule, Tiers: names(m.Tiers), Reqs: []vReq{}, Actors: []vActor{}, Ifaces: []vIface{}, Exes: []vExe{},
		Mods: []vUnit{}, Groups: []vUnit{}, Members: []vMember{}, Enums: []vEnum{}, Consts: []vConst{}, Errors: []vError{}, Types: []vType{},
		Contracts: []vContract{}, HKs: []vHK{}, Funcs: []vFunc{}, EKs: []vEK{}, Events: []vEvent{}, Flows: []vFlow{}}
	v.System = vSystem{ID: "system"}
	if m.System != nil {
		v.System = vSystem{ID: m.System.ID, Summary: m.System.Summary}
		for _, a := range m.System.Actors {
			v.Actors = append(v.Actors, vActor{a.ID, a.Summary})
		}
		for _, i := range m.System.Interfaces {
			it := vIface{ID: i.ID, Protocol: i.Protocol, Spec: i.Spec, UsedBy: names(i.UsedBy)}
			if i.Port != 0 {
				port := i.Port
				it.Port = &port
			}
			if i.Path != "" {
				p := i.Path
				it.Path = &p
			}
			v.Ifaces = append(v.Ifaces, it)
		}
	}
	for _, r := range m.Requirements {
		v.Reqs = append(v.Reqs, vReq{r.ID, r.Text, r.Status})
	}
	for _, e := range m.Executables {
		v.Exes = append(v.Exes, vExe{e.ID, e.Summary, names(e.Exposes), names(e.Composes)})
	}
	for _, x := range m.Modules {
		v.Mods = append(v.Mods, vUnit{ID: x.ID, Type: "mod", Members: []string{}, Tier: x.Tier.Name, Summary: x.Summary, Path: x.Path, Role: x.Role,
			DependsOn: names(x.DependsOn), Implements: names(x.Implements), Satisfies: names(x.Satisfies), Needs: vNeeds(m, x.Needs)})
	}
	for _, g := range m.Groups {
		u := vUnit{ID: g.ID, Type: "grp", Tier: g.Tier.Name, Summary: g.Summary, Path: g.Path, Kind: g.Kind.Name, Rule: g.Rule,
			DependsOn: names(g.DependsOn), Implements: []string{}, Satisfies: names(g.Satisfies), Members: []string{}, Needs: vNeeds(m, g.Needs)}
		for _, mem := range g.Members {
			u.Members = append(u.Members, mem.ID)
			vm := vMember{ID: mem.ID, Type: "mem", Group: g.ID, Summary: mem.Summary, Path: mem.Path(), Exceptions: []vException{},
				Handles: names(mem.Handles), Satisfies: names(mem.Satisfies), Needs: vNeeds(m, mem.Needs)}
			for _, e := range mem.Exceptions {
				vm.Exceptions = append(vm.Exceptions, vException{e.DependsOn.Name, e.Reason})
			}
			v.Members = append(v.Members, vm)
		}
		v.Groups = append(v.Groups, u)
	}
	for _, e := range m.Enums {
		v.Enums = append(v.Enums, vEnum{e.ID, e.In.Name, names(e.Values), vGoOf(e.Go), strsOrEmpty(e.Export), names(e.Satisfies)})
	}
	for _, c := range m.Consts {
		v.Consts = append(v.Consts, vConst{c.ID, c.In.Name, c.Type, c.Value, vGoOf(c.Go), strsOrEmpty(c.Export), names(c.Satisfies)})
	}
	for _, e := range m.Errors {
		ve := vError{ID: e.ID, In: e.In.Name, Message: e.Message, Go: vGoOf(e.Go), Satisfies: names(e.Satisfies)}
		if e.HTTP != 0 {
			h := e.HTTP
			ve.HTTP = &h
		}
		v.Errors = append(v.Errors, ve)
	}
	for _, t := range m.Types {
		v.Types = append(v.Types, vType{t.ID, t.In.Name, vFields(t.Fields), vGoOf(t.Go), names(t.Satisfies)})
	}
	for _, c := range m.Contracts {
		vc := vContract{ID: c.ID, In: c.In.Name, Kind: c.Kind, Go: vGoOf(c.Go), Satisfies: names(c.Satisfies), Methods: []vMethod{}}
		for _, me := range c.Methods {
			vc.Methods = append(vc.Methods, vMethod{me.Name, vParams(me.Params), names(me.Returns), names(me.Errors), me.HasErrors})
		}
		v.Contracts = append(v.Contracts, vc)
	}
	for _, hk := range m.HandlerKinds {
		v.HKs = append(v.HKs, vHK{hk.ID, hk.In.Name, vGoOf(hk.Go), vParams(hk.Params), names(hk.Returns), hk.Key.Name})
	}
	for _, f := range m.Funcs {
		vf := vFunc{ID: f.ID, In: f.In.Name, Kind: f.Kind.Name, Handles: f.Handles.Name, Of: f.Of.Name, Errors: names(f.Errors), Satisfies: names(f.Satisfies), Cases: []vCase{}}
		for _, c := range f.Cases {
			vc := vCase{Name: c.Name, Given: nodeValue(c.Given), Error: c.Error.Name}
			if c.Expect != nil {
				vc.Expect = nodeValue(c.Expect)
			}
			vf.Cases = append(vf.Cases, vc)
		}
		v.Funcs = append(v.Funcs, vf)
	}
	for _, ek := range m.EventKinds {
		v.EKs = append(v.EKs, vEK{ek.ID, ek.In.Name, ek.Key.Name, ek.Encoding, names(ek.Subscribers), names(ek.Satisfies)})
	}
	for _, e := range m.Events {
		in := ""
		if ek := m.EventKind(e.Kind.Name); ek != nil {
			in = ek.In.Name
		}
		v.Events = append(v.Events, vEvent{e.ID, e.Kind.Name, in, names(e.Publishers), names(e.Subscribers), vFields(e.Data), names(e.Satisfies)})
	}
	for _, f := range m.Flows {
		vf := vFlow{ID: f.ID, In: f.In.Name, On: f.On.Name, Summary: f.Summary, Steps: []vStep{}, Satisfies: names(f.Satisfies)}
		var stack []string
		committed := false
		for _, s := range f.Steps {
			undo := []string{}
			if !committed && !s.Fatal {
				for i := len(stack) - 1; i >= 0; i-- {
					undo = append(undo, stack[i])
				}
			}
			vf.Steps = append(vf.Steps, vStep{ID: s.ID, Undo: s.Undo, Commit: s.Commit, Fatal: s.Fatal, Calls: names(s.Calls),
				Emits: names(s.Emits), Summary: s.Summary, OnFailUndo: undo, OnFailFatal: s.Fatal})
			if !committed && s.Undo != "" && !s.Commit {
				stack = append(stack, s.Undo)
			}
			if s.Commit {
				committed, stack = true, nil
			}
		}
		v.Flows = append(v.Flows, vf)
	}
	return v
}

func levelName(l model.Level) string {
	switch l {
	case model.Error:
		return "err"
	case model.Warning:
		return "warn"
	}
	return "info"
}

func toIssues(m *model.Model, root string, diags, code model.Diagnostics) []vIssue {
	out := []vIssue{}
	var nodes []nodePos
	if m != nil {
		nodes = nodePositions(m)
	}
	fromCode := map[model.Diagnostic]bool{}
	for _, d := range code {
		fromCode[d] = true
	}
	for _, d := range diags {
		key, origin := "", "design"
		if m != nil {
			key, origin = issueKey(m, nodes, root, d)
		}
		if fromCode[d] {
			origin = "code"
		}
		group, _, _ := strings.Cut(d.Msg, ": ")
		abs := d.Pos.File
		if a, err := filepath.Abs(abs); err == nil && abs != "" {
			abs = a
		}
		out = append(out, vIssue{Lv: levelName(d.Level), Msg: d.Msg, Key: key, File: d.Pos.File, Line: d.Pos.Line, Origin: origin, Group: group, Abs: abs})
	}
	return out
}

func fileSources(m *model.Model) map[string][]string {
	src := map[string][]string{}
	add := func(p, key string) { src[p] = append(src[p], key) }
	dir := func(owner string) string { o, _ := m.Owner(owner); return o.Path }
	for _, e := range m.Enums {
		add(dir(e.In.Name)+"/"+e.ID+".gen.go", "enum:"+e.ID)
	}
	for _, c := range m.Consts {
		add(dir(c.In.Name)+"/consts.gen.go", "const:"+c.ID)
	}
	for _, e := range m.Errors {
		add(dir(e.In.Name)+"/errors.gen.go", "err:"+e.ID)
	}
	for _, t := range m.Types {
		add(dir(t.In.Name)+"/types.gen.go", "type:"+t.ID)
	}
	for _, c := range m.Contracts {
		add(dir(c.In.Name)+"/contracts.gen.go", "contract:"+c.ID)
	}
	for _, hk := range m.HandlerKinds {
		add(dir(hk.In.Name)+"/"+hk.ID+".gen.go", "hk:"+hk.ID)
	}
	for _, g := range m.Groups {
		for _, mem := range g.Members {
			add(mem.Path()+"/register.gen.go", "mem:"+mem.ID)
			add(mem.Path()+"/register.gen.go", "hk:"+g.Kind.Name)
			for _, n := range append(g.Needs, mem.Needs...) {
				add(mem.Path()+"/needs.gen.go", "contract:"+n.Contract.Name)
			}
		}
	}
	for _, x := range m.Modules {
		for _, n := range x.Needs {
			add(x.Path+"/needs.gen.go", "contract:"+n.Contract.Name)
		}
	}
	for _, e := range m.Executables {
		add("cmd/"+e.ID+"/modules.gen.go", "exe:"+e.ID)
		add("cmd/"+e.ID+"/contracts.gen.go", "exe:"+e.ID)
	}
	for _, ek := range m.EventKinds {
		p := dir(ek.In.Name) + "/" + ek.ID + ".gen.go"
		add(p, "ek:"+ek.ID)
		add(dir(ek.In.Name)+"/daemok_event.gen.go", "ek:"+ek.ID)
		for _, e := range m.Events {
			if e.Kind.Name == ek.ID {
				add(p, "ev:"+e.ID)
			}
		}
	}
	for _, f := range m.Flows {
		add(dir(f.In.Name)+"/"+f.ID+".gen.go", "flow:"+f.ID)
		add(dir(f.In.Name)+"/daemok_flow.gen.go", "flow:"+f.ID)
	}
	for _, f := range m.Funcs {
		add(dir(f.In.Name)+"/"+f.ID+"_gen_test.go", "func:"+f.ID)
		add(dir(f.In.Name)+"/daemok_case_gen_test.go", "func:"+f.ID)
	}
	return src
}

func toFiles(m *model.Model, files []golang.File) []vFile {
	out := []vFile{}
	if m == nil {
		return out
	}
	src := fileSources(m)
	for _, f := range files {
		out = append(out, vFile{Path: f.Path, Lang: "go", Code: string(f.Content), Sources: strsOrEmpty(src[f.Path])})
	}
	return out
}

func toRelations(m *model.Model) []vRelation {
	out := []vRelation{}
	if m == nil {
		return out
	}
	add := func(from, to, kind, label, why string) {
		if from != "" && to != "" {
			out = append(out, vRelation{from, to, kind, label, why})
		}
	}
	needs := func(from string, ns []model.Need) {
		for _, n := range ns {
			if ct := m.Contract(n.Contract.Name); ct != nil {
				add(from, ct.In.Name, ct.Kind, ct.ID, n.Why)
			}
		}
	}
	for _, x := range m.Modules {
		for _, d := range x.DependsOn {
			add(x.ID, d.Name, "depends", "", "")
		}
		needs(x.ID, x.Needs)
	}
	for _, g := range m.Groups {
		for _, d := range g.DependsOn {
			add(g.ID, d.Name, "depends", "", "")
		}
		needs(g.ID, g.Needs)
		hk := m.HandlerKind(g.Kind.Name)
		for _, mem := range g.Members {
			for _, e := range mem.Exceptions {
				add(mem.ID, e.DependsOn.Name, "exception", e.Reason, e.Reason)
			}
			needs(mem.ID, mem.Needs)
			if hk != nil {
				add(mem.ID, hk.In.Name, "register", hk.ID, "")
			}
		}
	}
	for _, e := range m.Events {
		var subs []model.Ref
		subs = append(subs, e.Subscribers...)
		if ek := m.EventKind(e.Kind.Name); ek != nil {
			subs = append(subs, ek.Subscribers...)
		}
		for _, p := range e.Publishers {
			for _, s := range subs {
				add(p.Name, s.Name, "event", e.ID, "")
			}
		}
	}
	return out
}
