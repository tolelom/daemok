package rename

import (
	"sort"

	"github.com/tolelom/daemok/internal/model"
)

type PinKind int

const (
	PinNone PinKind = iota
	PinGoName
	PinGoType
	PinPath
	PinImpossible
)

var kinds = map[string]PinKind{
	"requirements": PinNone, "actors": PinNone, "interfaces": PinNone, "tiers": PinNone,
	"types": PinGoName, "contracts": PinGoName, "handler_kinds": PinGoName, "funcs": PinGoName, "errors": PinGoName, "consts": PinGoName,
	"enums": PinGoType, "modules": PinPath, "groups": PinPath,
	"events": PinImpossible, "event_kinds": PinImpossible, "flows": PinImpossible, "executables": PinImpossible, "members": PinImpossible,
}

func Kinds() []string {
	var out []string
	for k := range kinds {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func namespace(kind string) []string {
	switch kind {
	case "modules", "groups", "members":
		return []string{"modules", "groups", "members"}
	case "enums", "types":
		return []string{"enums", "types"}
	}
	return []string{kind}
}

func metas(m *model.Model, kind string) []*model.Meta {
	var out []*model.Meta
	switch kind {
	case "requirements":
		for _, x := range m.Requirements {
			out = append(out, &x.Meta)
		}
	case "actors":
		if m.System != nil {
			for _, x := range m.System.Actors {
				out = append(out, &x.Meta)
			}
		}
	case "interfaces":
		if m.System != nil {
			for _, x := range m.System.Interfaces {
				out = append(out, &x.Meta)
			}
		}
	case "executables":
		for _, x := range m.Executables {
			out = append(out, &x.Meta)
		}
	case "modules":
		for _, x := range m.Modules {
			out = append(out, &x.Meta)
		}
	case "groups":
		for _, x := range m.Groups {
			out = append(out, &x.Meta)
		}
	case "members":
		for _, g := range m.Groups {
			for _, x := range g.Members {
				out = append(out, &x.Meta)
			}
		}
	case "enums":
		for _, x := range m.Enums {
			out = append(out, &x.Meta)
		}
	case "consts":
		for _, x := range m.Consts {
			out = append(out, &x.Meta)
		}
	case "errors":
		for _, x := range m.Errors {
			out = append(out, &x.Meta)
		}
	case "types":
		for _, x := range m.Types {
			out = append(out, &x.Meta)
		}
	case "contracts":
		for _, x := range m.Contracts {
			out = append(out, &x.Meta)
		}
	case "handler_kinds":
		for _, x := range m.HandlerKinds {
			out = append(out, &x.Meta)
		}
	case "event_kinds":
		for _, x := range m.EventKinds {
			out = append(out, &x.Meta)
		}
	case "events":
		for _, x := range m.Events {
			out = append(out, &x.Meta)
		}
	case "funcs":
		for _, x := range m.Funcs {
			out = append(out, &x.Meta)
		}
	case "flows":
		for _, x := range m.Flows {
			out = append(out, &x.Meta)
		}
	}
	return out
}

func exists(m *model.Model, kind, id string) bool {
	_, ok := definition(m, kind, id)
	return ok
}
