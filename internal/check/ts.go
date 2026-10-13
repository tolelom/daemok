package check

import (
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tolelom/daemok/internal/gen/ts"
	"github.com/tolelom/daemok/internal/yaml"
)

// ts는 export 대상, ts_out 형식, TS로 내보낼 수 없는 상수, TS 파일 안의 이름 충돌을 본다.
func (c *checker) ts() {
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
	consts, enums := ts.Exported(m)
	switch {
	case m.TSOut == "" && first != nil:
		c.errorf(*first, "export: [ts]가 있는데 ts_out이 없다")
		return
	case m.TSOut == "":
		return
	case len(consts)+len(enums) == 0:
		c.warnf(m.TSOutPos, "ts_out이 있는데 export: [ts]인 상수나 enum이 없다")
	}
	if filepath.IsAbs(m.TSOut) || path.IsAbs(m.TSOut) {
		c.errorf(m.TSOutPos, "ts_out은 코드 루트 기준 상대 경로여야 한다: %s", m.TSOut)
	}
	if !strings.HasSuffix(m.TSOut, ".ts") {
		c.errorf(m.TSOutPos, "ts_out은 .ts 파일이어야 한다: %s", m.TSOut)
	}
	values, types := map[string]string{}, map[string]string{}
	name := func(space map[string]string, n, source string, pos yaml.Pos) {
		if prev, ok := space[n]; ok {
			c.errorf(pos, "TS 이름이 겹친다: %s (%s, %s)", n, prev, source)
			return
		}
		space[n] = source
	}
	for _, k := range consts {
		if k.Type == "duration" {
			c.errorf(k.Pos, "duration 상수는 ts로 내보내지 않는다: %s", k.ID)
		}
		name(values, ts.ConstName(k.ID), "상수 "+k.ID, k.Pos)
	}
	for _, e := range enums {
		name(values, ts.ConstName(e.ID), "enum "+e.ID, e.Pos)
		name(types, ts.TypeName(e.ID), "enum "+e.ID, e.Pos)
	}
}
