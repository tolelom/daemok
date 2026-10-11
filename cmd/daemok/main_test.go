package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCLI(t *testing.T, args ...string) (code int, out, errOut string) {
	t.Helper()
	var o, e bytes.Buffer
	code = run(args, &o, &e)
	return code, o.String(), e.String()
}

func TestNoArgsPrintsUsage(t *testing.T) {
	code, _, errOut := runCLI(t)
	if code != 2 {
		t.Fatalf("exit %d, want 2", code)
	}
	if !strings.Contains(errOut, "daemok check") {
		t.Fatalf("usage missing: %q", errOut)
	}
}

func TestUnknownCommand(t *testing.T) {
	code, _, errOut := runCLI(t, "frobnicate")
	if code != 2 || !strings.Contains(errOut, "frobnicate") {
		t.Fatalf("exit %d, err %q", code, errOut)
	}
}

func writeDesign(t *testing.T, src string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "design")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "d.yaml"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const cleanDesign = `version: 0.4
tiers: [a]
modules:
  core: { tier: a }
`

func TestCheckClean(t *testing.T) {
	dir := writeDesign(t, cleanDesign)
	code, out, _ := runCLI(t, "check", "--design", dir)
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	if strings.TrimSpace(out) != "오류 0, 경고 0, 정보 0" {
		t.Fatalf("out %q", out)
	}
}

func TestCheckMissingDir(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "nope")
	code, out, _ := runCLI(t, "check", "--design", missing)
	if code != 2 || !strings.Contains(out, missing) || !strings.Contains(out, "설계 파일(*.yaml)이 없다") {
		t.Fatalf("exit %d out %q", code, out)
	}
}

func TestCheckModelErrorExits2(t *testing.T) {
	dir := writeDesign(t, "version: 0.4\nmodulez: {}\n")
	code, out, _ := runCLI(t, "check", "--design", dir)
	if code != 2 || !strings.Contains(out, "d.yaml:2:1: 오류: 알 수 없는 키 \"modulez\"") {
		t.Fatalf("exit %d out %q", code, out)
	}
}

func TestCheckDesignErrorExits1(t *testing.T) {
	dir := writeDesign(t, "version: 0.4\ntiers: [a, b]\nmodules:\n  low:  { tier: a, depends_on: [high] }\n  high: { tier: b }\n")
	code, out, _ := runCLI(t, "check", "--design", dir)
	if code != 1 || !strings.Contains(out, "d.yaml:4:33: 오류: 상위 tier 의존: low(a) → high(b)") || !strings.Contains(out, "오류 1, 경고 0, 정보 0") {
		t.Fatalf("exit %d out %q", code, out)
	}
}

func TestCheckStrict(t *testing.T) {
	dir := writeDesign(t, "version: 0.4\nrequirements:\n  REQ-a: { text: A }\n")
	code, _, _ := runCLI(t, "check", "--design", dir)
	if code != 0 {
		t.Fatalf("warning only: exit %d", code)
	}
	code, out, _ := runCLI(t, "check", "--design", dir, "--strict")
	if code != 1 || !strings.Contains(out, "경고: 충족하는 설계 요소가 없는 요구사항: REQ-a") {
		t.Fatalf("strict: exit %d out %q", code, out)
	}
}

func TestCheckOutputSorted(t *testing.T) {
	dir := writeDesign(t, "version: 0.4\nrequirements:\n  REQ-b: { text: B }\n  REQ-a: { text: A }\n")
	_, out, _ := runCLI(t, "check", "--design", dir, "--strict")
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 || !strings.Contains(lines[0], ":3:3:") || !strings.Contains(lines[1], ":4:3:") {
		t.Fatalf("%q", out)
	}
}

func TestCheckBadFlag(t *testing.T) {
	code, _, errOut := runCLI(t, "check", "--bogus")
	if code != 2 || !strings.Contains(errOut, "bogus") {
		t.Fatalf("exit %d err %q", code, errOut)
	}
}

const genDesign = `version: 0.4
go_module: example.com/x
tiers: [a]
modules:
  core: { tier: a }
enums:
  kind: { in: core, values: [x, y] }
`

func TestGenWritesThenCheckPasses(t *testing.T) {
	dir := writeDesign(t, genDesign)
	root := filepath.Dir(dir)
	code, out, _ := runCLI(t, "check", "--design", dir, "--root", root)
	if code != 1 || !strings.Contains(out, "생성 파일이 없다: core/kind.gen.go") {
		t.Fatalf("check before gen: exit %d\n%s", code, out)
	}
	code, out, _ = runCLI(t, "gen", "--design", dir, "--root", root, "--dry-run")
	if code != 0 || !strings.Contains(out, "쓰기: core/kind.gen.go") {
		t.Fatalf("dry-run: exit %d\n%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(root, "core", "kind.gen.go")); !os.IsNotExist(err) {
		t.Fatal("dry-run이 파일을 썼다")
	}
	code, out, _ = runCLI(t, "gen", "--design", dir, "--root", root)
	if code != 0 || !strings.Contains(out, "쓴 파일 1, 지운 파일 0") {
		t.Fatalf("gen: exit %d\n%s", code, out)
	}
	code, out, _ = runCLI(t, "check", "--design", dir, "--root", root)
	if code != 0 {
		t.Fatalf("check after gen: exit %d\n%s", code, out)
	}
	// 설계에서 enum을 지우면 gen이 지운다
	if err := os.WriteFile(filepath.Join(dir, "d.yaml"), []byte("version: 0.4\ngo_module: example.com/x\ntiers: [a]\nmodules:\n  core: { tier: a }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, _ = runCLI(t, "gen", "--design", dir, "--root", root)
	if code != 0 || !strings.Contains(out, "지우기: core/kind.gen.go") {
		t.Fatalf("gen delete: exit %d\n%s", code, out)
	}
}

func TestGenRefusesHandWrittenFile(t *testing.T) {
	dir := writeDesign(t, genDesign)
	root := filepath.Dir(dir)
	hand := filepath.Join(root, "core", "kind.gen.go")
	os.MkdirAll(filepath.Dir(hand), 0o755)
	os.WriteFile(hand, []byte("package core\n"), 0o644)
	for _, args := range [][]string{{"gen", "--design", dir, "--root", root, "--dry-run"}, {"gen", "--design", dir, "--root", root}} {
		code, out, _ := runCLI(t, args...)
		if code != 1 || !strings.Contains(out, "거부: core/kind.gen.go") {
			t.Fatalf("%v: exit %d\n%s", args, code, out)
		}
	}
	if src, _ := os.ReadFile(hand); string(src) != "package core\n" {
		t.Fatal("손으로 쓴 파일을 덮어썼다")
	}
}

func TestGenStopsOnDesignError(t *testing.T) {
	dir := writeDesign(t, "version: 0.4\ntiers: [a, b]\nmodules:\n  low:  { tier: a, depends_on: [high] }\n  high: { tier: b }\n")
	code, out, _ := runCLI(t, "gen", "--design", dir, "--root", filepath.Dir(dir))
	if code != 2 || !strings.Contains(out, "상위 tier 의존") {
		t.Fatalf("exit %d\n%s", code, out)
	}
}

func TestCheckJSON(t *testing.T) {
	dir := writeDesign(t, cleanDesign+"\n  orphan_extra: { tier: x }\n")
	code, out, _ := runCLI(t, "check", "--design", dir, "--format", "json")
	var got struct {
		Diagnostics []struct {
			Level string `json:"level"`
			Msg   string `json:"msg"`
			File  string `json:"file"`
			Line  int    `json:"line"`
			Group string `json:"group"`
		} `json:"diagnostics"`
		Counts map[string]int `json:"counts"`
	}
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("JSON이 아니다: %v\n%s", err, out)
	}
	if code != 1 || got.Counts["error"] == 0 || len(got.Diagnostics) == 0 {
		t.Fatalf("%d %+v", code, got)
	}
	d := got.Diagnostics[0]
	if d.Level != "error" || d.File == "" || d.Line == 0 || d.Group == "" || strings.Contains(d.Group, ":") {
		t.Fatalf("%+v", d)
	}
}

func TestCheckUnknownFormat(t *testing.T) {
	dir := writeDesign(t, cleanDesign)
	if code, _, errOut := runCLI(t, "check", "--design", dir, "--format", "xml"); code != 2 || !strings.Contains(errOut, "format") {
		t.Fatalf("%d %s", code, errOut)
	}
}
