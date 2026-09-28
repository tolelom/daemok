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
