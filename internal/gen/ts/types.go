package ts

import (
	"fmt"
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

// 설계 내장 타입의 TS 타입. 없는 내장 타입(duration, error)은 옮기지 않는다.
var builtin = map[string]string{
	"string": "string", "address": "string", "time": "string",
	"int": "number", "uint": "number", "int64": "number", "uint64": "number", "float64": "number",
	"bool": "boolean", "any": "unknown", "json.RawMessage": "unknown",
	"bytes": "string | null",
}

// FieldType은 설계 필드의 TS 타입이다. Go에서 nil이 될 수 있는 값(포인터, 슬라이스, bytes, map)에 | null을 붙인다.
// optional 필드는 생성 코드가 omitempty로 키를 빼므로 TS에서 키를 선택으로 쓰고(optional이 true),
// 타입은 키가 있을 때의 값이다. omitempty는 nil 포인터와 빈 슬라이스·bytes를 빼므로 그 null은 붙이지 않는다.
// 옮길 수 없는 타입이거나 내보내지 않는 enum·타입을 가리키면 오류다.
func FieldType(m *model.Model, f *model.Field) (ts string, optional bool, err error) {
	t := f.Type.Name
	switch {
	case f.RefType.Name != "" && f.Many:
		t = "[]" + f.RefType.Name
	case f.RefType.Name != "":
		t = "*" + f.RefType.Name
	case f.Many:
		t = "[]" + t
	}
	if !f.Optional {
		ts, err = typeString(m, t)
		return ts, false, err
	}
	switch {
	case strings.HasPrefix(t, "[]") || t == "bytes":
		ts, err = typeString(m, t)
		return strings.TrimSuffix(ts, orNull), true, err
	case strings.HasPrefix(t, "*"):
		ts, err = typeString(m, t[1:])
	default:
		ts, err = typeString(m, t) // 생성 코드가 포인터를 붙인다
	}
	return ts, true, err
}

const orNull = " | null"

func typeString(m *model.Model, t string) (string, error) {
	switch {
	case strings.HasPrefix(t, "*"):
		inner, err := typeString(m, t[1:])
		if err != nil || strings.HasSuffix(inner, orNull) {
			return inner, err
		}
		return inner + orNull, nil
	case strings.HasPrefix(t, "[]"):
		inner, err := typeString(m, t[2:])
		if strings.Contains(inner, " | ") {
			inner = "(" + inner + ")"
		}
		return inner + "[]" + orNull, err
	case strings.HasPrefix(t, "map["):
		_, value, ok := model.MapParts(t)
		if !ok {
			return "", fmt.Errorf("TS로 옮길 수 없는 타입 %s", t)
		}
		inner, err := typeString(m, value)
		return "{ [key: string]: " + inner + " }" + orNull, err
	}
	if s, ok := builtin[t]; ok {
		return s, nil
	}
	if e := m.Enum(t); e != nil {
		if !slices.Contains(e.Export, "ts") {
			return "", fmt.Errorf("ts로 내보내지 않는 enum %s", t)
		}
		return TypeName(t), nil
	}
	if n := m.Type(t); n != nil {
		if !slices.Contains(n.Export, "ts") {
			return "", fmt.Errorf("ts로 내보내지 않는 타입 %s", t)
		}
		return TypeName(t), nil
	}
	return "", fmt.Errorf("TS로 옮길 수 없는 타입 %s", t)
}
