package check

import (
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/gen/ts"
	"github.com/tolelom/daemok/internal/yaml"
)

func (c *checker) ts() {
	first := c.exportTargets()
	x := ts.Exported(c.m)
	if !c.tsOut(first, x.Len()) {
		return
	}
	c.tsNames(x)
}

func (c *checker) exportTargets() *yaml.Pos {
	m := c.m
	exports := func(id string, pos yaml.Pos, export []string) {
		for _, x := range export {
			if x != "ts" {
				c.errorf(pos, "알 수 없는 export 대상 %q (ts): %s", x, id)
			}
		}
	}
	var first *yaml.Pos
	for _, k := range m.Consts {
		exports(k.ID, k.Pos, k.Export)
		if slices.Contains(k.Export, "ts") && first == nil {
			first = &k.Pos
		}
	}
	for _, e := range m.Enums {
		exports(e.ID, e.Pos, e.Export)
		if slices.Contains(e.Export, "ts") && first == nil {
			first = &e.Pos
		}
	}
	for _, t := range m.Types {
		exports(t.ID, t.Pos, t.Export)
		if slices.Contains(t.Export, "ts") && first == nil {
			first = &t.Pos
		}
	}
	return first
}

func (c *checker) tsOut(firstExport *yaml.Pos, exported int) bool {
	m := c.m
	switch {
	case m.TSOut == "" && firstExport != nil:
		c.errorf(*firstExport, "export: [ts]가 있는데 ts_out이 없다")
		return false
	case m.TSOut == "":
		return false
	case exported == 0:
		c.warnf(m.TSOutPos, "ts_out이 있는데 export: [ts]인 상수, enum, 타입이 없다")
	}
	if filepath.IsAbs(m.TSOut) || path.IsAbs(m.TSOut) {
		c.errorf(m.TSOutPos, "ts_out은 코드 루트 기준 상대 경로여야 한다: %s", m.TSOut)
	}
	if !strings.HasSuffix(m.TSOut, ".ts") {
		c.errorf(m.TSOutPos, "ts_out은 .ts 파일이어야 한다: %s", m.TSOut)
	}
	return true
}

func (c *checker) tsNames(x ts.Nodes) {
	values, types := map[string]string{}, map[string]string{}
	name := func(space map[string]string, n, source string, pos yaml.Pos) {
		if prev, ok := space[n]; ok {
			c.errorf(pos, "TS 이름이 겹친다: %s (%s, %s)", n, prev, source)
			return
		}
		space[n] = source
	}
	for _, k := range x.Consts {
		if k.Type == "duration" {
			c.errorf(k.Pos, "duration 상수는 ts로 내보내지 않는다: %s", k.ID)
		}
		name(values, ts.ConstName(k.ID), "상수 "+k.ID, k.Pos)
	}
	for _, e := range x.Enums {
		name(values, ts.ConstName(e.ID), "enum "+e.ID, e.Pos)
		name(types, ts.TypeName(e.ID), "enum "+e.ID, e.Pos)
	}
	for _, t := range x.Types {
		for _, f := range t.Fields {
			if _, _, err := ts.FieldType(c.m, f); err != nil {
				c.errorf(f.Pos, "타입 %s 필드 %s: %v", t.ID, f.Name, err)
			}
		}
		name(types, ts.TypeName(t.ID), "타입 "+t.ID, t.Pos)
	}
}
