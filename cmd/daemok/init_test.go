package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "core"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "c.go"), []byte("package core\n"), 0o644)
	var out, errb bytes.Buffer
	if code := run([]string{"init", "--root", root}, &out, &errb); code != 0 {
		t.Fatalf("%d %s %s", code, out.String(), errb.String())
	}
	if _, err := os.Stat(filepath.Join(root, "design", "demo.yaml")); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "패키지 1, 모듈 1, 실행 단위 0") {
		t.Fatal(out.String())
	}
	out.Reset()
	if code := run([]string{"check", "--design", filepath.Join(root, "design"), "--root", root}, &out, &errb); code != 0 {
		t.Fatalf("초안의 check: %d\n%s", code, out.String())
	}
	if code := run([]string{"init", "--root", root}, &out, &errb); code != 1 {
		t.Fatalf("이미 설계 파일이 있으면 1: %d", code)
	}
}

func TestInitNotModule(t *testing.T) {
	var out, errb bytes.Buffer
	if code := run([]string{"init", "--root", t.TempDir()}, &out, &errb); code != 2 {
		t.Fatalf("%d", code)
	}
	if !strings.Contains(errb.String(), "Go 모듈이 아니다") {
		t.Fatal(errb.String())
	}
}

func TestInitDryRun(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "core"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "c.go"), []byte("package core\n"), 0o644)
	var out, errb bytes.Buffer
	if code := run([]string{"init", "--root", root, "--dry-run"}, &out, &errb); code != 0 {
		t.Fatalf("%d %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "core: { tier: tier0 }") || !strings.Contains(out.String(), "패키지 1, 모듈 1") {
		t.Fatal(out.String())
	}
	if _, err := os.Stat(filepath.Join(root, "design")); err == nil {
		t.Fatal("--dry-run은 쓰지 않는다")
	}
}

// --l3는 설계가 있는 저장소에서 L3 초안을 stdout에 낸다. 저장하면 check가 통과하고 coverage가 오른다.
func TestInitL3(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "core"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "c.go"), []byte("package core\n\nimport \"errors\"\n\nvar ErrGone = errors.New(\"gone\")\n\nconst Limit = 3\n\nconst Computed = Limit * 2\n\nfunc Do() {}\n"), 0o644)
	var out, errb bytes.Buffer
	if code := run([]string{"init", "--root", root}, &out, &errb); code != 0 {
		t.Fatalf("%d %s", code, errb.String())
	}
	total := func() string {
		out.Reset()
		run([]string{"coverage", "--root", root}, &out, &errb)
		return out.String()[strings.LastIndex(out.String(), "| 합계"):]
	}
	before := total()
	out.Reset()
	errb.Reset()
	if code := run([]string{"init", "--root", root, "--l3"}, &out, &errb); code != 0 {
		t.Fatalf("%d %s", code, errb.String())
	}
	if !strings.Contains(out.String(), `ErrGone: { in: core, gen: false, message: "gone" }`) || !strings.Contains(out.String(), "limit: { in: core, gen: false, type: int, value: 3 }") {
		t.Fatal(out.String())
	}
	if !strings.Contains(errb.String(), "Computed") || !strings.Contains(errb.String(), "함수 1개") {
		t.Fatal(errb.String())
	}
	if err := os.WriteFile(filepath.Join(root, "design", "l3.yaml"), out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if code := run([]string{"check", "--root", root, "--strict"}, &out, &errb); code != 0 {
		t.Fatalf("초안을 더한 check: %d\n%s", code, out.String())
	}
	if after := total(); after == before {
		t.Fatalf("coverage가 그대로다: %s", after)
	}
}

func TestInitL3WithoutDesign(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/demo\n"), 0o644)
	var out, errb bytes.Buffer
	if code := run([]string{"init", "--root", root, "--l3"}, &out, &errb); code != 2 {
		t.Fatalf("%d %s", code, errb.String())
	}
}
