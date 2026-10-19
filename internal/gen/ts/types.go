package ts

import (
	"fmt"
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/model"
)

var builtin = map[string]string{
	"string": "string", "address": "string", "time": "string",
	"int": "number", "uint": "number", "int64": "number", "uint64": "number", "float64": "number",
	"bool": "boolean", "any": "unknown", "json.RawMessage": "unknown",
	"bytes": "string | null",
}

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
		ts, err = typeString(m, t)
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
