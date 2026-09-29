package main

import (
	"bytes"
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
