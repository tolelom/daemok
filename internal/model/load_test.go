package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tolelom/daemok/internal/yaml"
)

func load(t *testing.T, files map[string]string) (*Model, Diagnostics) {
	t.Helper()
	dir := t.TempDir()
	for name, src := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return Load(dir)
}

func loadOne(t *testing.T, src string) (*Model, Diagnostics) {
	t.Helper()
	return load(t, map[string]string{"d.yaml": "version: 0.3\n" + src})
}

func find(ds Diagnostics, level Level, substr string) *Diagnostic {
	for i := range ds {
		if ds[i].Level == level && strings.Contains(ds[i].Msg, substr) {
			return &ds[i]
		}
	}
	return nil
}

func wantDiag(t *testing.T, ds Diagnostics, level Level, substr string) *Diagnostic {
	t.Helper()
	d := find(ds, level, substr)
	if d == nil {
		t.Fatalf("no %v containing %q in:\n%s", level, substr, join(ds))
	}
	return d
}

func wantClean(t *testing.T, ds Diagnostics) {
	t.Helper()
	if len(ds) != 0 {
		t.Fatalf("unexpected diagnostics:\n%s", join(ds))
	}
}

func join(ds Diagnostics) string {
	var b strings.Builder
	for _, d := range ds {
		b.WriteString(d.String())
		b.WriteByte('\n')
	}
	return b.String()
}

func pos(file string, line, col int) yaml.Pos { return yaml.Pos{File: file, Line: line, Col: col} }

func TestNoDesignFiles(t *testing.T) {
	_, ds := Load(t.TempDir())
	d := wantDiag(t, ds, Error, "설계 파일")
	if d.Pos.Line != 0 || !strings.HasSuffix(d.String(), ": 오류: 설계 파일(*.yaml)이 없다") {
		t.Fatalf("%q", d.String())
	}
}

func TestTopLevelScalars(t *testing.T) {
	m, ds := loadOne(t, "go_module: github.com/x/y\nts_out: web/gen.ts\n")
	wantClean(t, ds)
	if m.Version != "0.3" || m.GoModule != "github.com/x/y" || m.TSOut != "web/gen.ts" {
		t.Fatalf("%+v", m)
	}
}

func TestVersionMismatch(t *testing.T) {
	_, ds := load(t, map[string]string{"d.yaml": "version: 0.2\n"})
	wantDiag(t, ds, Error, "0.3")
}

func TestUnknownTopKey(t *testing.T) {
	_, ds := loadOne(t, "modulez: {}\n")
	d := wantDiag(t, ds, Error, `알 수 없는 키 "modulez"`)
	if d.Pos.Line != 2 || d.Pos.Col != 1 {
		t.Fatalf("pos %v", d.Pos)
	}
}

func TestParseErrorBecomesDiagnostic(t *testing.T) {
	_, ds := load(t, map[string]string{"a.yaml": "version: 0.3\n", "b.yaml": "x:\n\ty: 1\n"})
	d := wantDiag(t, ds, Error, "탭")
	if !strings.HasSuffix(d.Pos.File, "b.yaml") || d.Pos.Line != 2 {
		t.Fatalf("pos %v", d.Pos)
	}
}

func TestMergeAcrossFiles(t *testing.T) {
	m, ds := load(t, map[string]string{
		"a.yaml": "version: 0.3\nrequirements:\n  REQ-a: { text: A }\n",
		"b.yaml": "version: 0.3\nrequirements:\n  REQ-b: { text: B, status: held }\n",
	})
	wantClean(t, ds)
	if len(m.Requirements) != 2 || m.Requirements[0].ID != "REQ-a" || m.Requirements[1].Status != "held" {
		t.Fatalf("%+v", m.Requirements)
	}
	if m.Requirements[0].Status != "active" {
		t.Fatal("default status")
	}
}

func TestDuplicateIDAcrossFiles(t *testing.T) {
	_, ds := load(t, map[string]string{
		"a.yaml": "version: 0.3\nrequirements:\n  REQ-a: { text: A }\n",
		"b.yaml": "version: 0.3\nrequirements:\n  REQ-a: { text: B }\n",
	})
	d := wantDiag(t, ds, Error, `중복 ID 요구사항 "REQ-a"`)
	if !strings.Contains(d.Msg, "a.yaml:3:3") || !strings.HasSuffix(d.Pos.File, "b.yaml") {
		t.Fatalf("%v", d)
	}
}

func TestScalarConflictAcrossFiles(t *testing.T) {
	_, ds := load(t, map[string]string{
		"a.yaml": "version: 0.3\ngo_module: x\n",
		"b.yaml": "version: 0.3\ngo_module: y\n",
	})
	wantDiag(t, ds, Error, `"go_module"가 다른 파일`)
}

func TestRequirementFields(t *testing.T) {
	_, ds := loadOne(t, "requirements:\n  REQ-a: { text: A, status: maybe, extra: 1 }\n  REQ-b: { status: active }\n")
	wantDiag(t, ds, Error, "status는 active, held, dropped")
	wantDiag(t, ds, Error, `알 수 없는 키 "extra"`)
	wantDiag(t, ds, Error, `"text"가 필요하다`)
}

func TestDiagnosticsSort(t *testing.T) {
	ds := Diagnostics{
		{Pos: pos("b", 1, 1)}, {Pos: pos("a", 2, 1)}, {Pos: pos("a", 1, 5)}, {Pos: pos("a", 1, 2)},
	}
	ds.Sort()
	got := ds[0].Pos.String() + " " + ds[1].Pos.String() + " " + ds[2].Pos.String() + " " + ds[3].Pos.String()
	if got != "a:1:2 a:1:5 a:2:1 b:1:1" {
		t.Fatal(got)
	}
}

const l2Sample = `
system:
  id: tolchain
  summary: 블록체인
  actors:
    game_server: { summary: "게임" }
  interfaces:
    json_rpc: { protocol: jsonrpc2-http, port: 8545, used_by: [game_server] }
    events:   { protocol: sse, path: /events, spec: docs/events.yaml }
executables:
  node: { exposes: [json_rpc, events], composes: [core, vm, tx_modules] }
  genkey: { summary: 키 생성, composes: [] }
tiers: [foundation, domain, execution]
modules:
  crypto: { tier: foundation }
  core:   { tier: domain, depends_on: [crypto], path: pkg/core }
  vm:     { tier: execution, depends_on: [core], implements: [json_rpc, events] }
groups:
  tx_modules:
    tier: execution
    kind: tx_handler
    path: vm/modules
    depends_on: [core, vm]
    rule: no_sibling_deps
    members:
      economy: { handles: [transfer], satisfies: [REQ-a] }
      reward:
        handles: [grant_reward]
        exceptions:
          - depends_on: asset
            reason: "스키마 재사용"
`

func TestDecodeL2(t *testing.T) {
	m, ds := loadOne(t, l2Sample)
	wantClean(t, ds)
	if m.System.ID != "tolchain" || len(m.System.Actors) != 1 || len(m.System.Interfaces) != 2 {
		t.Fatalf("system %+v", m.System)
	}
	rpc := m.System.Interfaces[0]
	if rpc.Port != 8545 || rpc.UsedBy[0].Name != "game_server" || m.System.Interfaces[1].Spec != "docs/events.yaml" {
		t.Fatalf("interfaces %+v %+v", rpc, m.System.Interfaces[1])
	}
	if len(m.Executables) != 2 || len(m.Executables[0].Composes) != 3 || m.Executables[1].Composes == nil {
		t.Fatalf("executables %+v", m.Executables)
	}
	if len(m.Tiers) != 3 || m.Tiers[2].Name != "execution" {
		t.Fatalf("tiers %+v", m.Tiers)
	}
	core := m.Modules[1]
	if core.ID != "core" || core.Tier.Name != "domain" || core.Path != "pkg/core" || core.DependsOn[0].Name != "crypto" {
		t.Fatalf("core %+v", core)
	}
	if m.Modules[0].Path != "crypto" {
		t.Fatalf("default path %q", m.Modules[0].Path)
	}
	g := m.Groups[0]
	if g.Kind.Name != "tx_handler" || g.Rule != "no_sibling_deps" || g.Tier.Name != "execution" || len(g.Members) != 2 {
		t.Fatalf("group %+v", g)
	}
	reward := g.Members[1]
	if reward.Group != g || reward.Handles[0].Name != "grant_reward" || len(reward.Exceptions) != 1 {
		t.Fatalf("member %+v", reward)
	}
	if reward.Exceptions[0].DependsOn.Name != "asset" || reward.Exceptions[0].Reason != "스키마 재사용" {
		t.Fatalf("exception %+v", reward.Exceptions[0])
	}
	if g.Members[0].Satisfies[0].Name != "REQ-a" {
		t.Fatal("member satisfies")
	}
}

func TestDecodeL2Errors(t *testing.T) {
	_, ds := loadOne(t, `
modules:
  core: { summary: x }
  vm: { tier: a, tierr: b }
groups:
  g: { kind: k, rule: whatever, members: { m: { handlez: [] } } }
  h: { members: {} }
`)
	wantDiag(t, ds, Error, `"tier"가 필요하다`)
	wantDiag(t, ds, Error, `알 수 없는 키 "tierr"`)
	wantDiag(t, ds, Error, "rule은 no_sibling_deps")
	wantDiag(t, ds, Error, `알 수 없는 키 "handlez"`)
	wantDiag(t, ds, Error, `"kind"가 필요하다`)
}

func TestSecondSystemAndTiers(t *testing.T) {
	_, ds := load(t, map[string]string{
		"a.yaml": "version: 0.3\nsystem: { id: a }\ntiers: [x]\n",
		"b.yaml": "version: 0.3\nsystem: { id: b }\ntiers: [y]\n",
	})
	wantDiag(t, ds, Error, "system은 한 파일에만")
	wantDiag(t, ds, Error, "tiers는 한 파일에만")
}

func TestExceptionNeedsDependsOn(t *testing.T) {
	_, ds := loadOne(t, "groups:\n  g:\n    kind: k\n    members:\n      m:\n        exceptions:\n          - reason: x\n")
	wantDiag(t, ds, Error, `"depends_on"가 필요하다`)
}

const l3Sample = `
enums:
  tx_type:
    in: core
    go: { type: TxType, prefix: Tx }
    values: [transfer, mint_asset]
    export: [ts]
consts:
  max_block_txs: { in: config, type: int, value: 500, go: { name: DefaultMaxBlockTxs } }
  max_tx_age:    { in: config, type: duration, value: 1h }
  page_size:     { in: httputil, value: 10 }
errors:
  ErrNotFound: { in: core, message: "not found", http: 404 }
  ErrOverflow: { in: core, message: "overflow" }
types:
  transfer_payload:
    in: core
    gen: false
    fields: { to: address, amount: uint64 }
  post:
    in: model
    fields:
      title:  { type: string, max: post_title_max }
      tags:   { ref: tag, many: true }
      series: { ref: series, optional: true }
      note:   { type: string, max: 80 }
contracts:
  post_service:
    in: post
    go: { name: Service }
    methods:
      get_post: { params: { post_id: uint, user_id: "*uint" }, returns: [post], errors: [ErrNotFound] }
      list:     { params: {}, returns: [] }
handler_kinds:
  tx_handler:
    in: vm
    go: { name: Handler, register: Register }
    params: { ctx: "*Context", payload: json.RawMessage }
    returns: [error]
    key: tx_type
`

func TestDecodeL3(t *testing.T) {
	m, ds := loadOne(t, l3Sample)
	wantClean(t, ds)
	e := m.Enums[0]
	if e.ID != "tx_type" || e.In.Name != "core" || !e.Gen || e.Go.Prefix != "Tx" || len(e.Values) != 2 || e.Export[0] != "ts" {
		t.Fatalf("enum %+v", e)
	}
	if m.Consts[0].Go.Name != "DefaultMaxBlockTxs" || m.Consts[0].Value != "500" || m.Consts[1].Type != "duration" || m.Consts[2].Type != "int" {
		t.Fatalf("consts %+v %+v %+v", m.Consts[0], m.Consts[1], m.Consts[2])
	}
	if m.Errors[0].HTTP != 404 || m.Errors[1].HTTP != 0 || m.Errors[1].Message != "overflow" {
		t.Fatalf("errors %+v %+v", m.Errors[0], m.Errors[1])
	}
	tp := m.Types[0]
	if tp.Gen || len(tp.Fields) != 2 || tp.Fields[1].Type.Name != "uint64" || tp.Fields[1].RefType.Name != "" {
		t.Fatalf("type %+v", tp)
	}
	post := m.Types[1]
	if post.Fields[0].Max.Name != "post_title_max" || !post.Fields[1].Many || post.Fields[1].RefType.Name != "tag" || !post.Fields[2].Optional || post.Fields[3].Max.Name != "80" {
		t.Fatalf("post fields %+v", post.Fields)
	}
	c := m.Contracts[0]
	if c.Go.Name != "Service" || len(c.Methods) != 2 || c.Methods[0].Params[1].Type.Name != "*uint" || !c.Methods[0].HasErrors || c.Methods[1].HasErrors {
		t.Fatalf("contract %+v", c)
	}
	hk := m.HandlerKinds[0]
	if hk.Key.Name != "tx_type" || hk.Go.Register != "Register" || hk.Params[0].Type.Name != "*Context" || hk.Returns[0].Name != "error" {
		t.Fatalf("handler kind %+v", hk)
	}
}

func TestDecodeL3Errors(t *testing.T) {
	_, ds := loadOne(t, `
enums:
  e: { values: [a] }
consts:
  c: { in: m, type: float, value: 1 }
  d: { in: m, value: abc }
errors:
  E: { in: m }
types:
  t: { in: m, fields: { a: { type: x, ref: y }, b: { many: true } } }
contracts:
  k: { in: m }
handler_kinds:
  h: { in: m, params: {} }
`)
	wantDiag(t, ds, Error, `"in"가 필요하다`)
	wantDiag(t, ds, Error, "type은 int, string, duration")
	wantDiag(t, ds, Error, "value는 정수여야 한다")
	wantDiag(t, ds, Error, `"message"가 필요하다`)
	wantDiag(t, ds, Error, "type과 ref 중 하나만")
	wantDiag(t, ds, Error, "type이나 ref가 필요하다")
	wantDiag(t, ds, Error, `"methods"가 필요하다`)
	wantDiag(t, ds, Error, `"key"가 필요하다`)
}

func TestDecodeL4(t *testing.T) {
	m, ds := loadOne(t, `
funcs:
  handle_transfer:
    in: economy
    kind: tx_handler
    handles: transfer
    errors: [ErrInsufficientBalance]
    cases:
      - { name: "금액 0", given: { amount: 0 }, error: invalid }
      - { name: "정상", given: { balance: 10, req: { title: "새 제목" } }, expect: { from: 7 } }
  update_post:
    in: post
    of: post_service.update_post
    errors: []
    cases: []
`)
	wantClean(t, ds)
	f := m.Funcs[0]
	if f.Kind.Name != "tx_handler" || f.Handles.Name != "transfer" || len(f.Cases) != 2 {
		t.Fatalf("%+v", f)
	}
	if f.Cases[0].Error.Name != "invalid" || f.Cases[0].Expect != nil || f.Cases[0].Given.Get("amount").Value != "0" {
		t.Fatalf("case0 %+v", f.Cases[0])
	}
	if f.Cases[1].Error.Name != "" || f.Cases[1].Expect.Get("from").Value != "7" {
		t.Fatalf("case1 %+v", f.Cases[1])
	}
	g := m.Funcs[1]
	if g.Of.Name != "post_service.update_post" || g.Errors == nil || len(g.Errors) != 0 {
		t.Fatalf("%+v", g)
	}
}

func TestDecodeL4Errors(t *testing.T) {
	_, ds := loadOne(t, `
funcs:
  a: { in: m, kind: k, handles: h, of: c.m }
  b: { in: m }
  c: { in: m, kind: k }
  d:
    in: m
    of: c.m
    cases:
      - { given: {} }
      - { name: x, given: {}, expect: {}, error: E }
      - { name: y, given: {} }
      - { name: z, given: [1], error: E }
      - { name: w, given: { a: [1] }, error: E }
`)
	wantDiag(t, ds, Error, "kind+handles와 of 중 하나만")
	wantDiag(t, ds, Error, "kind+handles 또는 of가 필요하다")
	wantDiag(t, ds, Error, `"handles"가 필요하다`)
	wantDiag(t, ds, Error, `"name"가 필요하다`)
	wantDiag(t, ds, Error, "expect와 error 중 하나만")
	wantDiag(t, ds, Error, "expect나 error가 필요하다")
	wantDiag(t, ds, Error, "given은 맵이어야 한다")
	wantDiag(t, ds, Error, "given의 값은 값 하나 또는 맵이어야 한다")
}

func TestIndexAndOwner(t *testing.T) {
	m, ds := loadOne(t, l2Sample+`
enums:
  tx_type: { in: core, values: [transfer] }
types:
  payload: { in: core, fields: { to: address } }
`)
	wantClean(t, ds)
	if m.Module("core") == nil || m.Module("nope") != nil || m.Member("reward") == nil || m.Group("tx_modules") == nil {
		t.Fatal("lookup")
	}
	if m.TierIndex("execution") != 2 || m.TierIndex("x") != -1 {
		t.Fatal("tier index")
	}
	o, ok := m.Owner("reward")
	if !ok || o.Path != "vm/modules/reward" || o.Tier != "execution" || o.Member == nil {
		t.Fatalf("owner %+v", o)
	}
	o, _ = m.Owner("core")
	if o.Path != "pkg/core" || o.Tier != "domain" || o.Module == nil {
		t.Fatalf("owner %+v", o)
	}
	if _, ok := m.Owner("tx_modules"); ok {
		t.Fatal("group is not an owner")
	}
	cases := []struct {
		from, to string
		want     bool
	}{
		{"core", "core", true}, {"core", "crypto", true}, {"crypto", "core", false},
		{"reward", "core", true}, {"reward", "asset", true}, {"economy", "asset", false},
		{"vm", "economy", false}, {"tx_modules", "vm", true},
	}
	for _, c := range cases {
		if got := m.CanDepend(c.from, c.to); got != c.want {
			t.Errorf("CanDepend(%s, %s) = %v", c.from, c.to, got)
		}
	}
	if k, base := m.ResolveType("*uint"); k != BuiltinType || base != "uint" {
		t.Fatal("builtin")
	}
	if k, _ := m.ResolveType("[]tx_type"); k != EnumType {
		t.Fatal("enum")
	}
	if k, _ := m.ResolveType("payload"); k != StructType {
		t.Fatal("struct")
	}
	if k, _ := m.ResolveType("json.RawMessage"); k != ExternalType {
		t.Fatal("external dotted")
	}
	if k, _ := m.ResolveType("*Context"); k != ExternalType {
		t.Fatal("external upper")
	}
	if k, _ := m.ResolveType("nothing"); k != UnknownType {
		t.Fatal("unknown")
	}
}

func TestMemberViaGroupDependency(t *testing.T) {
	m, ds := loadOne(t, strings.Replace(l2Sample, "vm:     { tier: execution, depends_on: [core]", "vm:     { tier: execution, depends_on: [core, tx_modules]", 1))
	wantClean(t, ds)
	if !m.CanDepend("vm", "economy") {
		t.Fatal("depending on a group allows its members")
	}
}

func TestNamespaceCollisions(t *testing.T) {
	_, ds := loadOne(t, `
modules:
  events: { tier: a }
  economy: { tier: a }
groups:
  g: { kind: k, members: { economy: {} } }
system:
  id: s
  interfaces:
    events: {}
enums:
  post: { in: m, values: [a] }
types:
  post: { in: m, fields: {} }
`)
	wantDiag(t, ds, Error, `멤버 "economy"가 모듈`)
	wantDiag(t, ds, Error, `타입 "post"가 enum`)
	if find(ds, Error, `"events"`) != nil {
		t.Fatal("module and interface may share an ID")
	}
}

func TestSamplesLoadClean(t *testing.T) {
	for _, s := range []string{"tolchain", "tolelog"} {
		m, ds := Load(filepath.Join("testdata", s, "design"))
		if len(ds) != 0 {
			t.Fatalf("%s:\n%s", s, join(ds))
		}
		if m.GoModule == "" || len(m.Modules) == 0 {
			t.Fatalf("%s: empty model", s)
		}
	}
}
