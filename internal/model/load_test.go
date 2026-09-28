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
