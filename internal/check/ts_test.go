package check

import (
	"testing"

	"github.com/tolelom/daemok/internal/model"
)

const tsBase = "tiers: [a]\nmodules:\n  core: { tier: a }\n"

func TestTSExport(t *testing.T) {
	ok := tsBase + "ts_out: ../web/x.gen.ts\nconsts:\n  a: { in: core, type: int, value: 1, export: [ts] }\n"
	wantNone(t, run(t, ok), model.Error)
	wantNone(t, run(t, ok), model.Warning)

	cases := []struct {
		src   string
		level model.Level
		msg   string
	}{
		{"ts_out: x.gen.ts\nconsts:\n  a: { in: core, type: int, value: 1, export: [py] }\n", model.Error, `알 수 없는 export 대상 "py"`},
		{"ts_out: x.gen.ts\nenums:\n  e: { in: core, values: [v], export: [js] }\n", model.Error, `알 수 없는 export 대상 "js"`},
		{"consts:\n  a: { in: core, type: int, value: 1, export: [ts] }\n", model.Error, "export: [ts]가 있는데 ts_out이 없다"},
		{"ts_out: /abs/x.gen.ts\nconsts:\n  a: { in: core, type: int, value: 1, export: [ts] }\n", model.Error, "ts_out은 코드 루트 기준 상대 경로여야 한다"},
		{"ts_out: x.gen.js\nconsts:\n  a: { in: core, type: int, value: 1, export: [ts] }\n", model.Error, "ts_out은 .ts 파일이어야 한다"},
		{"ts_out: x.gen.ts\nconsts:\n  a: { in: core, type: duration, value: 1s, export: [ts] }\n", model.Error, "duration 상수는 ts로 내보내지 않는다: a"},
		{"ts_out: x.gen.ts\nconsts:\n  post_status: { in: core, type: int, value: 1, export: [ts] }\nenums:\n  post_status: { in: core, values: [v], export: [ts] }\n", model.Error, "TS 이름이 겹친다: POST_STATUS"},
		{"ts_out: x.gen.ts\nenums:\n  user_id: { in: core, values: [v], export: [ts] }\n  user_i_d: { in: core, values: [w], export: [ts] }\n", model.Error, "TS 이름이 겹친다: UserID"},
		{"ts_out: x.gen.ts\nconsts:\n  a: { in: core, type: int, value: 1 }\n", model.Warning, "ts_out이 있는데 export: [ts]인 상수나 enum이 없다"},
	}
	for _, c := range cases {
		want(t, run(t, tsBase+c.src), c.level, c.msg)
	}
}
