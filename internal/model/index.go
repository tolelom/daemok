package model

import "strings"

func (m *Model) Requirement(id string) *Requirement {
	for _, r := range m.Requirements {
		if r.ID == id {
			return r
		}
	}
	return nil
}

func (m *Model) Actor(id string) *Actor {
	if m.System == nil {
		return nil
	}
	for _, a := range m.System.Actors {
		if a.ID == id {
			return a
		}
	}
	return nil
}

func (m *Model) Interface(id string) *Interface {
	if m.System == nil {
		return nil
	}
	for _, i := range m.System.Interfaces {
		if i.ID == id {
			return i
		}
	}
	return nil
}

func (m *Model) Executable(id string) *Executable {
	for _, e := range m.Executables {
		if e.ID == id {
			return e
		}
	}
	return nil
}

func (m *Model) Module(id string) *Module {
	for _, x := range m.Modules {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Group(id string) *Group {
	for _, g := range m.Groups {
		if g.ID == id {
			return g
		}
	}
	return nil
}

func (m *Model) Member(id string) *Member {
	for _, g := range m.Groups {
		for _, mem := range g.Members {
			if mem.ID == id {
				return mem
			}
		}
	}
	return nil
}

func (m *Model) Enum(id string) *Enum {
	for _, x := range m.Enums {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Const(id string) *Const {
	for _, x := range m.Consts {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Error(id string) *ErrorDef {
	for _, x := range m.Errors {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Type(id string) *Type {
	for _, x := range m.Types {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Contract(id string) *Contract {
	for _, x := range m.Contracts {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) HandlerKind(id string) *HandlerKind {
	for _, x := range m.HandlerKinds {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Func(id string) *Func {
	for _, x := range m.Funcs {
		if x.ID == id {
			return x
		}
	}
	return nil
}

// TierIndex는 tiers 목록에서의 순서다. 작을수록 아래 tier. 없으면 -1.
func (m *Model) TierIndex(name string) int {
	for i, t := range m.Tiers {
		if t.Name == name {
			return i
		}
	}
	return -1
}

func (mem *Member) Path() string { return mem.Group.Path + "/" + mem.ID }

// Owner는 L3·L4 노드가 속하는 모듈 또는 그룹 멤버다.
type Owner struct {
	ID     string
	Path   string
	Tier   string
	Module *Module // 모듈이면 채워진다
	Member *Member // 멤버면 채워진다
}

func (m *Model) Owner(id string) (Owner, bool) {
	if mod := m.Module(id); mod != nil {
		return Owner{ID: id, Path: mod.Path, Tier: mod.Tier.Name, Module: mod}, true
	}
	if mem := m.Member(id); mem != nil {
		return Owner{ID: id, Path: mem.Path(), Tier: mem.Group.Tier.Name, Member: mem}, true
	}
	return Owner{}, false
}

func hasRef(rs []Ref, name string) bool {
	for _, r := range rs {
		if r.Name == name {
			return true
		}
	}
	return false
}

// CanDepend는 설계상 from이 to를 import해도 되는지다. from·to는 모듈, 그룹, 멤버 ID.
// 그룹에 의존하면 그 멤버에도 의존할 수 있다.
func (m *Model) CanDepend(from, to string) bool {
	if from == to {
		return true
	}
	var deps []Ref
	switch {
	case m.Module(from) != nil:
		deps = m.Module(from).DependsOn
	case m.Group(from) != nil:
		deps = m.Group(from).DependsOn
	case m.Member(from) != nil:
		mem := m.Member(from)
		deps = mem.Group.DependsOn
		for _, e := range mem.Exceptions {
			if e.DependsOn.Name == to {
				return true
			}
		}
	default:
		return false
	}
	if hasRef(deps, to) {
		return true
	}
	if mem := m.Member(to); mem != nil && hasRef(deps, mem.Group.ID) {
		return true
	}
	return false
}

type TypeKind int

const (
	UnknownType TypeKind = iota
	BuiltinType
	EnumType
	StructType
	ExternalType
)

// Builtins는 설계 타입 이름과 Go 타입의 대응이다.
var Builtins = map[string]string{
	"string": "string", "int": "int", "uint": "uint", "int64": "int64", "uint64": "uint64",
	"bool": "bool", "float64": "float64", "duration": "time.Duration", "time": "time.Time",
	"address": "string", "bytes": "[]byte", "error": "error", "any": "any",
}

// SplitType은 "*[]post"를 ("*[]", "post")로 나눈다.
func SplitType(t string) (prefix, base string) {
	base = t
	for {
		switch {
		case strings.HasPrefix(base, "*"):
			prefix, base = prefix+"*", base[1:]
		case strings.HasPrefix(base, "[]"):
			prefix, base = prefix+"[]", base[2:]
		default:
			return prefix, base
		}
	}
}

// ResolveType은 타입 이름의 종류와 접두사를 뗀 이름을 준다.
// 대문자로 시작하거나 '.'이 있으면 외부 타입으로 본다.
func (m *Model) ResolveType(t string) (TypeKind, string) {
	_, base := SplitType(t)
	switch {
	case Builtins[base] != "":
		return BuiltinType, base
	case m.Enum(base) != nil:
		return EnumType, base
	case m.Type(base) != nil:
		return StructType, base
	case base != "" && (strings.Contains(base, ".") || base[0] >= 'A' && base[0] <= 'Z'):
		return ExternalType, base
	}
	return UnknownType, base
}

func (c *Contract) Method(name string) *Method {
	for _, m := range c.Methods {
		if m.Name == name {
			return m
		}
	}
	return nil
}

func (e *Enum) HasValue(name string) bool { return hasRef(e.Values, name) }

// GoName은 snake_case ID를 CamelCase로 바꾼다. 조각 id는 ID가 된다.
func GoName(id string) string {
	var b strings.Builder
	for _, part := range strings.Split(id, "_") {
		switch part {
		case "":
		case "id":
			b.WriteString("ID")
		default:
			b.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return b.String()
}

func (m *Model) EventKind(id string) *EventKind {
	for _, x := range m.EventKinds {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Event(id string) *Event {
	for _, x := range m.Events {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (m *Model) Flow(id string) *Flow {
	for _, x := range m.Flows {
		if x.ID == id {
			return x
		}
	}
	return nil
}
