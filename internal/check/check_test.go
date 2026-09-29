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
	if err := os.WriteFile(filepath.Join(dir, "d.yaml"), []byte("version: 0.4\n"+src), 0o644); err != nil {
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
  rpc_mod: { tier: service, depends_on: [core, vm], implements: [rpc], needs: [store] }
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
  event_type: { in: core, values: [minted] }
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
event_kinds:
  chain_event: { in: core, key: event_type, subscribers: [rpc_mod] }
events:
  minted: { kind: chain_event, publishers: [asset], subscribers: [vm], data: { to: address } }
flows:
  run_block:
    in: core
    steps:
      - { id: lock, undo: unlock }
      - { id: load, calls: [store.get] }
      - { id: commit, commit: true, fatal: true }
      - { id: unlock }
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
		{name: "relations"},
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

func TestRefErrorsV04(t *testing.T) {
	cases := []struct{ old, new, msg string }{
		{"needs: [store]", "needs: [shop]", `없는 계약 "shop"`},
		{"needs: [store]", "needs: [{ contract: store, uses: [put] }]", `계약 store에 없는 메서드 "put"`},
		{"key: event_type", "key: evt", `없는 enum "evt"`},
		{"subscribers: [rpc_mod]", "subscribers: [nobody]", `없는 모듈 "nobody"`},
		{"kind: chain_event", "kind: chain", `없는 이벤트 종류 "chain"`},
		{"publishers: [asset]", "publishers: [ghost]", `없는 모듈 "ghost"`},
		{"data: { to: address }", "data: { to: adress }", `알 수 없는 타입 "adress"`},
		{"calls: [store.get]", "calls: [store.put]", `계약 store에 없는 메서드 "put"`},
		{"calls: [store.get]", "calls: [storeget]", "계약.메서드 형식"},
		{"in: core\n    steps", "in: nowhere\n    steps", `in이 가리키는 모듈이나 멤버가 없다: "nowhere"`},
		{"{ id: unlock }", "{ id: unlock, emits: [burned] }", `없는 이벤트 "burned"`},
		{"in: core\n    steps", "in: core\n    on: burned\n    steps", `없는 이벤트 "burned"`},
	}
	for _, c := range cases {
		ds := run(t, replace(t, c.old, c.new))
		want(t, ds, model.Error, c.msg)
	}
}

func TestNeedsTierAndCycle(t *testing.T) {
	// crypto(foundation)가 core(domain)의 계약을 받으면 상위 tier 의존
	ds := run(t, replace(t, "crypto:  { tier: foundation }", "crypto:  { tier: foundation, needs: [store] }"))
	want(t, ds, model.Error, "상위 tier 의존: crypto(foundation) → core(domain)")
	// core가 vm의 계약을 받고 vm이 core에 의존하면 순환
	src := replace(t, "core:    { tier: domain, depends_on: [crypto]", "core:    { tier: domain, depends_on: [crypto], needs: [runner]")
	src = strings.Replace(src, "contracts:\n", "contracts:\n  runner: { in: vm, methods: { run: {} } }\n", 1)
	ds = run(t, src)
	want(t, ds, model.Error, "의존 순환: core → vm → core")
}

func TestContractRole(t *testing.T) {
	src := strings.Replace(base, "contracts:\n", "contracts:\n  cache_port: { in: crypto, kind: port, methods: { get: {} } }\n  mailer: { in: crypto, kind: external, methods: { send: {} } }\n", 1)
	ds := run(t, src)
	want(t, ds, model.Error, "port 계약의 모듈은 role이 infra여야 한다: 계약 cache_port, 모듈 crypto")
	want(t, ds, model.Error, "external 계약의 모듈은 role이 adapter여야 한다: 계약 mailer, 모듈 crypto")
	ds = run(t, strings.Replace(src, "crypto:  { tier: foundation }", "crypto:  { tier: foundation, role: infra }", 1))
	for _, d := range ds {
		if strings.Contains(d.Msg, "role이 infra") {
			t.Fatalf("unexpected %s", d)
		}
	}
}

func TestNeedsTypeDependency(t *testing.T) {
	// crypto가 store를 받으면 반환 타입 payload(core)에 의존하지 않으므로 오류
	ds := run(t, replace(t, "crypto:  { tier: foundation }", "crypto:  { tier: foundation, needs: [store] }"))
	d := want(t, ds, model.Error, "받는 계약의 타입이 속한 모듈에 의존하지 않는다: crypto → core.payload (계약 store)")
	if d.Pos.Line != 15 {
		t.Fatalf("위치는 받는 쪽 needs 항목이어야 한다: %v", d.Pos)
	}
	// uses로 고른 메서드의 타입만 본다
	src := strings.Replace(base, "contracts:\n", "contracts:\n  keys: { in: core, methods: { plain: { returns: [int] }, typed: { returns: [payload] } } }\n", 1)
	ds = run(t, strings.Replace(src, "crypto:  { tier: foundation }", "crypto:  { tier: foundation, needs: [{ contract: keys, uses: [plain] }] }", 1))
	for _, d := range ds {
		if strings.Contains(d.Msg, "모듈에 의존하지 않는다") || strings.Contains(d.Msg, "의존하지 않는 모듈의 타입") {
			t.Fatalf("unexpected %s", d)
		}
	}
}

func TestEvents(t *testing.T) {
	ds := run(t, replace(t, "values: [minted]", "values: [minted, burned]"))
	want(t, ds, model.Error, "이벤트가 없는 값: event_type.burned")

	ds = run(t, strings.Replace(base, "events:\n", "events:\n  ghost: { kind: chain_event, publishers: [asset] }\n", 1))
	want(t, ds, model.Error, `event_type에 없는 값 "ghost"`)

	// crypto는 core에 의존하지 않는다
	ds = run(t, replace(t, "publishers: [asset]", "publishers: [asset, crypto]"))
	want(t, ds, model.Error, "이벤트 종류의 모듈에 의존하지 않는다: crypto → core")
	ds = run(t, replace(t, "subscribers: [rpc_mod]", "subscribers: [rpc_mod, crypto]"))
	want(t, ds, model.Error, "이벤트 종류의 모듈에 의존하지 않는다: crypto → core")
	ds = run(t, replace(t, "publishers: [asset]", "publishers: []"))
	want(t, ds, model.Error, "발행자가 없는 이벤트: minted")
	// 이벤트 data의 타입도 이벤트 종류 모듈의 의존을 따른다
	src := strings.Replace(base, "types:\n", "types:\n  job: { in: vm, fields: { n: int } }\n", 1)
	ds = run(t, strings.Replace(src, "data: { to: address }", "data: { to: address, j: job }", 1))
	want(t, ds, model.Error, "의존하지 않는 모듈의 타입을 쓴다: core → vm.job")
}

func TestFlows(t *testing.T) {
	cases := []struct{ old, new, msg string }{
		{"{ id: load, calls", "{ id: Load, calls", `단계 ID는 소문자 snake_case여야 한다: "Load"`},
		{"{ id: unlock }", "{ id: lock }", "흐름 안에서 중복된 단계: lock"},
		{"{ id: unlock }", "{ id: fatal }", "Fatal과 겹치는 단계 이름: fatal"},
		{"{ id: unlock }", "{ id: unlock, commit: true }", "commit 단계가 둘 이상이다: unlock"},
		{"{ id: unlock }", "{ id: unlock, undo: relock }", "커밋 뒤 단계에 undo가 있다: unlock"},
		{"{ id: lock, undo: unlock }", "{ id: lock, undo: Unlock }", `undo는 소문자 snake_case여야 한다: "Unlock"`},
		{"in: core\n    steps", "in: vm\n    steps", "흐름의 모듈이 받지 않는 계약: vm → store"},
		{"{ id: unlock }", "{ id: unlock, emits: [minted] }", "흐름의 모듈이 발행자가 아닌 이벤트: core → minted"},
		{"in: core\n    steps", "in: core\n    on: minted\n    steps", "흐름의 모듈이 구독자가 아닌 이벤트: core → minted"},
	}
	for _, c := range cases {
		ds := run(t, replace(t, c.old, c.new))
		want(t, ds, model.Error, c.msg)
	}
	// rpc_mod는 store를 받고 묶음 전체의 구독자다
	ds := run(t, replace(t, "in: core\n    steps", "in: rpc_mod\n    on: minted\n    steps"))
	wantNone(t, ds, model.Error)
	// vm은 minted의 구독자다(흐름에서 계약 호출을 빼면 통과)
	src := replace(t, "in: core\n    steps", "in: vm\n    on: minted\n    steps")
	ds = run(t, strings.Replace(src, "calls: [store.get]", "calls: []", 1))
	wantNone(t, ds, model.Error)
}

func TestQAFixes(t *testing.T) {
	// 멤버 needs가 만드는 순환
	src := strings.Replace(base, "  crypto:  { tier: foundation }", "  crypto:  { tier: foundation }\n  helper:  { tier: execution, depends_on: [tx_modules] }", 1)
	src = strings.Replace(src, "asset:   { handles: [mint] }", "asset:   { handles: [mint], needs: [hc] }", 1)
	src = strings.Replace(src, "contracts:\n", "contracts:\n  hc: { in: helper, methods: { x: { errors: [] } } }\n", 1)
	want(t, run(t, src), model.Error, "의존 순환: helper → tx_modules → helper")

	// 형제 멤버 needs 순환
	src = strings.Replace(base, "economy: { handles: [transfer] }", "economy: { handles: [transfer], needs: [c_asset] }", 1)
	src = strings.Replace(src, "asset:   { handles: [mint] }", "asset:   { handles: [mint], needs: [c_econ] }", 1)
	src = strings.Replace(src, "contracts:\n", "contracts:\n  c_asset: { in: asset, methods: { x: { errors: [] } } }\n  c_econ: { in: economy, methods: { y: { errors: [] } } }\n", 1)
	want(t, run(t, src), model.Error, "의존 순환: economy → asset → economy")

	// 흐름 calls가 uses 밖의 메서드
	src = strings.Replace(base, "contracts:\n", "contracts:\n  keys: { in: core, methods: { a: { errors: [] }, b: { errors: [] } } }\n", 1)
	src = strings.Replace(src, "needs: [store]", "needs: [store, { contract: keys, uses: [a] }]", 1)
	src = strings.Replace(src, "in: core\n    steps", "in: rpc_mod\n    steps", 1)
	src = strings.Replace(src, "calls: [store.get]", "calls: [store.get, keys.b]", 1)
	want(t, run(t, src), model.Error, "받는 계약의 uses에 없는 메서드: rpc_mod → keys.b")

	// 단계 이름
	want(t, run(t, replace(t, "{ id: unlock }", "{ id: unlock }\n      - { id: x_id }\n      - { id: x_i_d }")), model.Error, "메서드 이름이 겹치는 단계: x_i_d (x_id)")
	want(t, run(t, replace(t, "{ id: unlock }", "{ id: a__b }")), model.Error, `단계 ID는 소문자 snake_case여야 한다: "a__b"`)
	want(t, run(t, replace(t, "{ id: unlock }", "{ id: ab_ }")), model.Error, `단계 ID는 소문자 snake_case여야 한다: "ab_"`)
	want(t, run(t, replace(t, "{ id: commit, commit: true, fatal: true }", "{ id: commit, commit: true, fatal: true, undo: rollback }")), model.Error, "commit 단계에 undo가 있다: commit")
	d := want(t, run(t, replace(t, "{ id: lock, undo: unlock }", "{ id: lock, undo: fatal }")), model.Error, "Fatal과 겹치는 단계 이름: fatal")
	if d.Pos.Col < 20 {
		t.Fatalf("undo 위치여야 한다: %v", d.Pos)
	}
	want(t, run(t, replace(t, "    steps:\n      - { id: lock, undo: unlock }\n      - { id: load, calls: [store.get] }\n      - { id: commit, commit: true, fatal: true }\n      - { id: unlock }\n", "    steps: []\n")), model.Error, "단계가 없는 흐름: run_block")

	// 자기 계약 needs
	want(t, run(t, replace(t, "core:    { tier: domain, depends_on: [crypto]", "core:    { tier: domain, depends_on: [crypto], needs: [store]")), model.Error, "자기 자신에 의존: core")

	// 두 이벤트 종류가 같은 enum
	src = strings.Replace(base, "event_kinds:\n", "event_kinds:\n  other: { in: core, key: event_type }\n", 1)
	want(t, run(t, src), model.Error, "여러 이벤트 종류가 쓰는 enum: event_type (other, chain_event)")

	// 그룹은 발행자·구독자가 아니다
	want(t, run(t, replace(t, "publishers: [asset]", "publishers: [tx_modules]")), model.Error, "모듈이나 그룹 멤버가 아니다: publishers tx_modules")
	want(t, run(t, replace(t, "subscribers: [rpc_mod]", "subscribers: [tx_modules]")), model.Error, "모듈이나 그룹 멤버가 아니다: subscribers tx_modules")

	// 중복 needs, 빈 uses
	want(t, run(t, replace(t, "needs: [store]", "needs: [store, store]")), model.Warning, "같은 계약을 두 번 받는다: rpc_mod → store")
	want(t, run(t, replace(t, "needs: [store]", "needs: [{ contract: store, uses: [] }]")), model.Error, "uses가 비어 있다: store")
}

func TestQualifiedExternalType(t *testing.T) {
	// core가 vm.Thing을 쓰면 vm에 의존해야 한다
	src := strings.Replace(base, "types:\n", "types:\n  wrap: { in: core, fields: { x: \"*vm.Thing\" } }\n", 1)
	want(t, run(t, src), model.Error, "의존하지 않는 모듈의 타입을 쓴다: core → vm.Thing")
	// any는 내장 타입이다
	src = strings.Replace(base, "types:\n", "types:\n  wrap: { in: core, fields: { x: any } }\n", 1)
	wantNone(t, run(t, src), model.Error)
}
