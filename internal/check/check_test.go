package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/model"
)

// run은 설계 텍스트 하나를 읽어 검사 진단을 준다. 모델 단계 오류가 있으면 실패다.
func run(t *testing.T, src string) model.Diagnostics {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "d.yaml"), []byte("version: 0.3\n"+src), 0o644); err != nil {
		t.Fatal(err)
	}
	m, ds := model.Load(dir)
	if ds.HasErrors() {
		t.Fatalf("model errors:\n%s", join(ds))
	}
	return Run(m)
}

func join(ds model.Diagnostics) string {
	var b strings.Builder
	for _, d := range ds {
		b.WriteString(d.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func want(t *testing.T, ds model.Diagnostics, level model.Level, substr string) model.Diagnostic {
	t.Helper()
	for _, d := range ds {
		if d.Level == level && strings.Contains(d.Msg, substr) {
			return d
		}
	}
	t.Fatalf("no %v containing %q in:\n%s", level, substr, join(ds))
	return model.Diagnostic{}
}

func wantNone(t *testing.T, ds model.Diagnostics, level model.Level) {
	t.Helper()
	if n := ds.Count(level); n != 0 {
		t.Fatalf("want no %v, got %d:\n%s", level, n, join(ds))
	}
}

// base는 모든 검사를 통과하는 작은 설계다. 각 테스트가 한 곳씩 바꿔 위반을 만든다.
const base = `
requirements:
  REQ-a: { text: A }
system:
  id: s
  actors:
    game: {}
  interfaces:
    rpc: { protocol: http, used_by: [game] }
executables:
  node: { exposes: [rpc], composes: [core, vm, tx_modules, rpc_mod] }
tiers: [foundation, domain, execution, service]
modules:
  crypto:  { tier: foundation }
  core:    { tier: domain, depends_on: [crypto], satisfies: [REQ-a] }
  vm:      { tier: execution, depends_on: [core] }
  rpc_mod: { tier: service, depends_on: [core, vm], implements: [rpc] }
groups:
  tx_modules:
    tier: execution
    kind: tx_handler
    depends_on: [core, vm]
    rule: no_sibling_deps
    members:
      economy: { handles: [transfer] }
      asset:   { handles: [mint] }
enums:
  tx_type: { in: core, values: [transfer, mint] }
errors:
  ErrNotFound: { in: core, message: "not found", http: 404 }
  ErrLow:      { in: core, message: "low" }
types:
  payload: { in: core, fields: { to: address, kind: tx_type } }
contracts:
  store:
    in: core
    methods:
      get: { params: { id: uint }, returns: [payload], errors: [ErrNotFound] }
handler_kinds:
  tx_handler:
    in: vm
    params: { payload: payload }
    returns: [error]
    key: tx_type
funcs:
  handle_transfer:
    in: economy
    kind: tx_handler
    handles: transfer
    errors: [ErrLow]
    cases:
      - { name: low, given: { balance: 0 }, error: ErrLow }
      - { name: ok, given: { balance: 9 }, expect: { balance: 8 } }
`

func replace(t *testing.T, old, new string) string {
	t.Helper()
	if !strings.Contains(base, old) {
		t.Fatalf("base does not contain %q", old)
	}
	return strings.Replace(base, old, new, 1)
}

func TestBaseIsClean(t *testing.T) {
	ds := run(t, base)
	if len(ds) != 0 {
		t.Fatalf("base must be clean:\n%s", join(ds))
	}
}

func TestRefErrors(t *testing.T) {
	cases := []struct{ old, new, msg string }{
		{"satisfies: [REQ-a]", "satisfies: [REQ-x]", `없는 요구사항 "REQ-x"`},
		{"used_by: [game]", "used_by: [nobody]", `없는 액터 "nobody"`},
		{"exposes: [rpc]", "exposes: [grpc]", `없는 인터페이스 "grpc"`},
		{"composes: [core, vm, tx_modules, rpc_mod]", "composes: [core, ghost]", `없는 모듈 "ghost"`},
		{"tier: domain,", "tier: cloud,", `없는 tier "cloud"`},
		{"depends_on: [crypto]", "depends_on: [cryptoo]", `없는 모듈 "cryptoo"`},
		{"implements: [rpc]", "implements: [rest]", `없는 인터페이스 "rest"`},
		{"kind: tx_handler\n    depends_on", "kind: nope\n    depends_on", `없는 핸들러 종류 "nope"`},
		{"handles: [mint]", "handles: [burn]", `tx_type에 없는 값 "burn"`},
		{"in: vm\n    params", "in: nowhere\n    params", `"nowhere"가 없다`},
		{"to: address", "to: adress", `알 수 없는 타입 "adress"`},
		{"returns: [payload], errors: [ErrNotFound]", "returns: [payload], errors: [ErrGone]", `없는 에러 "ErrGone"`},
		{"key: tx_type", "key: tx_kind", `없는 enum "tx_kind"`},
		{"handles: transfer\n", "handles: refund\n", `tx_type에 없는 값 "refund"`},
		{"error: ErrLow }", "error: ErrNope }", `없는 에러 "ErrNope"`},
	}
	for _, c := range cases {
		ds := run(t, replace(t, c.old, c.new))
		want(t, ds, model.Error, c.msg)
	}
}

func TestRefFieldRefAndMax(t *testing.T) {
	ds := run(t, replace(t, "kind: tx_type }", "kind: tx_type, owner: { ref: user }, n: { type: uint, max: big } }"))
	want(t, ds, model.Error, `없는 타입 "user"`)
	want(t, ds, model.Error, `max "big"는 상수 ID나 숫자`)
	ds = run(t, replace(t, "kind: tx_type }", "kind: tx_type, n: { type: uint, max: 80 } }"))
	wantNone(t, ds, model.Error)
}

func TestRefOf(t *testing.T) {
	ds := run(t, replace(t, "kind: tx_handler\n    handles: transfer\n", "of: store.put\n"))
	want(t, ds, model.Error, `계약 store에 없는 메서드 "put"`)
	ds = run(t, replace(t, "kind: tx_handler\n    handles: transfer\n", "of: shop.get\n"))
	want(t, ds, model.Error, `없는 계약 "shop"`)
	ds = run(t, replace(t, "kind: tx_handler\n    handles: transfer\n", "of: storeget\n"))
	want(t, ds, model.Error, "계약.메서드 형식")
}

func TestExceptionTargetMustExist(t *testing.T) {
	ds := run(t, replace(t, "asset:   { handles: [mint] }", "asset:   { handles: [mint], exceptions: [{ depends_on: ghost }] }"))
	want(t, ds, model.Error, `없는 모듈 "ghost"`)
}
