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
		{"in: vm\n    params", "in: nowhere\n    params", `in이 가리키는 모듈이나 멤버가 없다: "nowhere"`},
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
	want(t, ds, model.Error, `max는 상수 ID나 숫자여야 한다: "big"`)
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

func TestL0(t *testing.T) {
	ds := run(t, replace(t, "implements: [rpc]", "implements: []"))
	want(t, ds, model.Error, "구현하는 모듈이 없는 인터페이스: rpc")

	ds = run(t, replace(t, "composes: [core, vm, tx_modules, rpc_mod]", "composes: [core, vm, tx_modules]"))
	want(t, ds, model.Error, "노출하는 인터페이스의 구현 모듈을 조립하지 않는다: 실행 단위 node, 인터페이스 rpc, 모듈 rpc_mod")

	// composes에 없어도 depends_on으로 닿으면 조립된 것이다.
	ds = run(t, replace(t, "composes: [core, vm, tx_modules, rpc_mod]", "composes: [rpc_mod, tx_modules]"))
	wantNone(t, ds, model.Error)
}

func TestTierDirection(t *testing.T) {
	ds := run(t, replace(t, "core:    { tier: domain, depends_on: [crypto]", "core:    { tier: domain, depends_on: [crypto, vm]"))
	d := want(t, ds, model.Error, "상위 tier 의존: core(domain) → vm(execution)")
	if d.Pos.Line == 0 {
		t.Fatal("position")
	}
	// 같은 tier는 허용
	ds = run(t, replace(t, "crypto:  { tier: foundation }", "crypto:  { tier: foundation, depends_on: [hash] }\n  hash:    { tier: foundation }"))
	wantNone(t, ds, model.Error)
	// 멤버 예외도 tier를 따른다
	ds = run(t, replace(t, "asset:   { handles: [mint] }", "asset:   { handles: [mint], exceptions: [{ depends_on: rpc_mod }] }"))
	want(t, ds, model.Error, "상위 tier 의존: asset(execution) → rpc_mod(service)")
}

func TestCycle(t *testing.T) {
	ds := run(t, replace(t, "vm:      { tier: execution, depends_on: [core] }", "vm:      { tier: execution, depends_on: [core, tx_modules] }"))
	want(t, ds, model.Error, "의존 순환: vm → tx_modules → vm")
	if n := ds.Count(model.Error); n != 1 {
		t.Fatalf("one cycle error, got %d:\n%s", n, join(ds))
	}
}

func TestCycleThroughException(t *testing.T) {
	src := replace(t, "asset:   { handles: [mint] }", "asset:   { handles: [mint], exceptions: [{ depends_on: helper }] }")
	src = strings.Replace(src, "  crypto:  { tier: foundation }", "  crypto:  { tier: foundation }\n  helper:  { tier: execution, depends_on: [tx_modules] }", 1)
	ds := run(t, src)
	want(t, ds, model.Error, "의존 순환: helper → tx_modules → helper")
	if n := ds.Count(model.Error); n != 1 {
		t.Fatalf("one cycle error, got %d:\n%s", n, join(ds))
	}
}

func TestSelfDependency(t *testing.T) {
	ds := run(t, replace(t, "depends_on: [crypto]", "depends_on: [crypto, core]"))
	d := want(t, ds, model.Error, "자기 자신에 의존: core")
	if d.Pos.Line == 0 {
		t.Fatal("position")
	}
	ds = run(t, replace(t, "asset:   { handles: [mint] }", "asset:   { handles: [mint], exceptions: [{ depends_on: asset }] }"))
	want(t, ds, model.Error, "자기 자신에 의존: asset")
	// 멤버가 자기 그룹에 의존하면 자기 자신에 의존하는 것이다
	ds = run(t, replace(t, "asset:   { handles: [mint] }", "asset:   { handles: [mint], exceptions: [{ depends_on: tx_modules }] }"))
	want(t, ds, model.Error, "자기 자신에 의존: asset")
}

func TestSiblingCycle(t *testing.T) {
	src := replace(t, "economy: { handles: [transfer] }", "economy: { handles: [transfer], exceptions: [{ depends_on: asset }] }")
	src = strings.Replace(src, "asset:   { handles: [mint] }", "asset:   { handles: [mint], exceptions: [{ depends_on: economy }] }", 1)
	ds := run(t, src)
	want(t, ds, model.Error, "의존 순환: economy → asset → economy")
	if n := ds.Count(model.Error); n != 1 {
		t.Fatalf("one cycle error, got %d:\n%s", n, join(ds))
	}
	// 한 방향 형제 예외는 순환이 아니다
	ds = run(t, replace(t, "economy: { handles: [transfer] }", "economy: { handles: [transfer], exceptions: [{ depends_on: asset }] }"))
	wantNone(t, ds, model.Error)
}

func TestOrphanModule(t *testing.T) {
	src := strings.Replace(base, "  crypto:  { tier: foundation }", "  crypto:  { tier: foundation }\n  lonely:  { tier: foundation }", 1)
	ds := run(t, src)
	want(t, ds, model.Warning, "어느 실행 단위에도 없는 모듈 lonely")
	// 실행 단위가 없으면 검사하지 않는다
	ds = run(t, strings.Replace(src, "executables:\n  node: { exposes: [rpc], composes: [core, vm, tx_modules, rpc_mod] }\n", "", 1))
	wantNone(t, ds, model.Warning)
}

func TestGroupExceptions(t *testing.T) {
	ds := run(t, replace(t, "economy: { handles: [transfer] }", "economy: { handles: [transfer], exceptions: [{ depends_on: asset, reason: \"검증 재사용\" }, { depends_on: core }] }"))
	want(t, ds, model.Info, "그룹 규칙 예외: economy → asset (검증 재사용)")
	want(t, ds, model.Warning, "그룹이 이미 의존하는 예외: 그룹 tx_modules, 예외 core")
}

func TestTypeDependency(t *testing.T) {
	// crypto(foundation)의 타입이 core의 enum을 쓰면 crypto는 core에 의존하지 않으므로 오류
	src := strings.Replace(base, "types:\n", "types:\n  sig: { in: crypto, fields: { kind: tx_type } }\n", 1)
	ds := run(t, src)
	want(t, ds, model.Error, "의존하지 않는 모듈의 타입을 쓴다: crypto → core.tx_type")
	// 멤버는 그룹 의존을 따른다
	src = strings.Replace(base, "types:\n", "types:\n  memo: { in: economy, fields: { p: payload } }\n", 1)
	wantNone(t, run(t, src), model.Error)
	// handler_kind의 key enum도 의존 대상이다
	src = replace(t, "vm:      { tier: execution, depends_on: [core] }", "vm:      { tier: execution, depends_on: [] }")
	src = strings.Replace(src, "params: { payload: payload }", "params: {}", 1)
	ds = run(t, src)
	want(t, ds, model.Error, "의존하지 않는 모듈의 타입을 쓴다: vm → core.tx_type")
}

func TestErrorHTTP(t *testing.T) {
	ds := run(t, replace(t, `http: 404 }`, `http: 400 }`))
	want(t, ds, model.Warning, "이름이 뜻하는 상태와 http가 다르다: ErrNotFound (이름 404, http 400)")
	ds = run(t, replace(t, `ErrLow:      { in: core, message: "low" }`, `ErrLow:      { in: core, message: "low" }
  ErrUnauthorized: { in: core, message: "u", http: 403 }
  ErrForbidden:    { in: core, message: "f", http: 403 }`))
	want(t, ds, model.Warning, "이름이 뜻하는 상태와 http가 다르다: ErrUnauthorized (이름 401, http 403)")
	if strings.Contains(join(ds), "ErrForbidden") {
		t.Fatal("ErrForbidden 403 is consistent")
	}
}

func TestContractWithoutErrors(t *testing.T) {
	ds := run(t, replace(t, "returns: [payload], errors: [ErrNotFound] }", "returns: [payload] }"))
	want(t, ds, model.Warning, "계약 store의 메서드 get에 errors가 없다")
	ds = run(t, replace(t, "returns: [payload], errors: [ErrNotFound] }", "returns: [payload], errors: [] }"))
	wantNone(t, ds, model.Warning)
}

func TestHandlerCoverage(t *testing.T) {
	ds := run(t, replace(t, "economy: { handles: [transfer] }", "economy: { handles: [] }"))
	want(t, ds, model.Error, "처리하는 멤버가 없는 값: tx_type.transfer")
	ds = run(t, replace(t, "asset:   { handles: [mint] }", "asset:   { handles: [mint, transfer] }"))
	d := want(t, ds, model.Error, "여러 멤버가 처리하는 값: tx_type.transfer (economy, asset)")
	if d.Pos.Line == 0 {
		t.Fatal("position")
	}
}

func TestL4Handler(t *testing.T) {
	ds := run(t, replace(t, "economy: { handles: [transfer] }", "economy: { handles: [] }"))
	want(t, ds, model.Error, "멤버 handles에 없는 값: 멤버 economy, 값 transfer")

	ds = run(t, replace(t, "in: economy\n    kind: tx_handler", "in: core\n    kind: tx_handler"))
	want(t, ds, model.Error, "핸들러 함수는 그룹 멤버에 있어야 한다: handle_transfer")
}

func TestL4Of(t *testing.T) {
	src := replace(t, "in: economy\n    kind: tx_handler\n    handles: transfer\n", "in: vm\n    of: store.get\n")
	ds := run(t, src)
	want(t, ds, model.Error, "계약과 함수의 위치가 다르다: 계약 store(core), 함수 handle_transfer(vm)")
}

func TestL4CaseErrors(t *testing.T) {
	ds := run(t, replace(t, "error: ErrLow }", "error: ErrNotFound }"))
	want(t, ds, model.Error, `함수 errors에 없는 에러: 케이스 "low", 에러 ErrNotFound, 함수 handle_transfer`)
	want(t, ds, model.Warning, "확인하는 케이스가 없는 에러: 함수 handle_transfer, 에러 ErrLow")
	// invalid는 errors에 없어도 된다
	ds = run(t, replace(t, "error: ErrLow }", "error: invalid }"))
	wantNone(t, ds, model.Error)
}

func TestRequirements(t *testing.T) {
	src := strings.Replace(base, "  REQ-a: { text: A }\n", "  REQ-a: { text: A }\n  REQ-b: { text: B }\n  REQ-c: { text: C, status: held }\n", 1)
	src = strings.Replace(src, "vm:      { tier: execution, depends_on: [core] }", "vm:      { tier: execution, depends_on: [core], satisfies: [REQ-c] }", 1)
	ds := run(t, src)
	want(t, ds, model.Warning, "충족하는 설계 요소가 없는 요구사항: REQ-b")
	d := want(t, ds, model.Warning, "held 요구사항에 연결된 설계 요소: vm → REQ-c")
	if d.Pos.Line == 0 {
		t.Fatal("position")
	}
	if strings.Contains(join(ds), "REQ-a") {
		t.Fatal("REQ-a is satisfied")
	}
}

func TestSamples(t *testing.T) {
	cases := []struct {
		name     string
		errors   []string
		warnings []string
		infos    []string
	}{
		{
			name:     "tolchain",
			warnings: []string{"held 요구사항에 연결된 설계 요소: random → REQ-random", "확인하는 케이스가 없는 에러: 함수 handle_transfer, 에러 ErrBalanceOverflow"},
			infos:    []string{"그룹 규칙 예외: reward → asset (보상 지급 시 자산 스키마 검증 재사용)"},
		},
		{
			name:     "tolelog",
			errors:   []string{"상위 tier 의존: post(service) → dto(transport)"},
			warnings: []string{"이름이 뜻하는 상태와 http가 다르다: ErrUnauthorized (이름 401, http 403)", "충족하는 설계 요소가 없는 요구사항: REQ-oauth"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m, ds := model.Load(filepath.Join("..", "model", "testdata", c.name, "design"))
			if len(ds) != 0 {
				t.Fatalf("load:\n%s", join(ds))
			}
			ds = Run(m)
			for level, msgs := range map[model.Level][]string{model.Error: c.errors, model.Warning: c.warnings, model.Info: c.infos} {
				if got := ds.Count(level); got != len(msgs) {
					t.Errorf("%v: want %d, got %d\n%s", level, len(msgs), got, join(ds))
				}
				for _, msg := range msgs {
					want(t, ds, level, msg)
				}
			}
		})
	}
}
