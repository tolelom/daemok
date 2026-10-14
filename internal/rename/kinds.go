// Package rename은 설계 노드 하나의 ID와 그 참조를 설계 파일 전체에서 바꾼다.
// 기본은 코드 이름을 고정 키로 유지하고, 생성 결과와 진단이 그대로인지 확인한 뒤 쓴다.
package rename

import (
	"sort"

	"github.com/tolelom/daemok/internal/model"
)

// PinKind는 코드 이름을 옛 값으로 고정하는 방법이다.
type PinKind int

const (
	PinNone       PinKind = iota // 코드와 무관
	PinGoName                    // go: { name: ... }
	PinGoType                    // go: { type: ... }
	PinPath                      // path: ...
	PinImpossible                // 형식에 코드 이름 키가 없음
)

var kinds = map[string]PinKind{
	"requirements": PinNone, "actors": PinNone, "interfaces": PinNone, "tiers": PinNone,
	"types": PinGoName, "contracts": PinGoName, "handler_kinds": PinGoName, "funcs": PinGoName, "errors": PinGoName, "consts": PinGoName,
	"enums": PinGoType, "modules": PinPath, "groups": PinPath,
	"events": PinImpossible, "event_kinds": PinImpossible, "flows": PinImpossible, "executables": PinImpossible, "members": PinImpossible,
}

// Kinds는 이름을 바꿀 수 있는 종류 이름이다.
func Kinds() []string {
	var out []string
	for k := range kinds {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// namespace는 ID가 서로 겹칠 수 없는 종류 묶음이다. depends_on과 타입 문자열이 묶음을 함께 가리킨다.
func namespace(kind string) []string {
	switch kind {
	case "modules", "groups", "members":
		return []string{"modules", "groups", "members"}
	case "enums", "types":
		return []string{"enums", "types"}
	}
	return []string{kind}
}

// metas는 kind의 노드마다 Meta를 준다. tiers는 Meta가 없어 주지 않는다.
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

// exists는 kind에 id 노드가 있는지다.
func exists(m *model.Model, kind, id string) bool {
	_, ok := definition(m, kind, id)
	return ok
}
