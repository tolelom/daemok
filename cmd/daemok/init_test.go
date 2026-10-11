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
