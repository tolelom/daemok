package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCoverageCommand(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "design"), 0o755)
	os.WriteFile(filepath.Join(root, "design", "d.yaml"), []byte("version: 0.5\ngo_module: example.com/c\ntiers: [a]\nmodules:\n  core: { tier: a }\nconsts:\n  limit: { in: core, gen: false, type: int, value: 1, go: { name: Limit } }\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "core"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "c.go"), []byte("package core\n\nconst Limit = 1\n\nfunc other() {}\n\nfunc Exported() {}\n"), 0o644)
	var out, errb bytes.Buffer
	code := run([]string{"coverage", "--design", filepath.Join(root, "design"), "--root", root}, &out, &errb)
	if code != 0 || !strings.Contains(out.String(), "| `core` | 3 | 1 | 33% | 100% |") || !strings.Contains(out.String(), "| 합계 | 3 | 1 | 33% | 100% |") || !strings.Contains(out.String(), "함수 제외: 선언 1개 중 1개") {
		t.Fatalf("%d\n%s%s", code, out.String(), errb.String())
	}
}

func TestCoverageReportsParseErrors(t *testing.T) {
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "design"), 0o755)
	os.WriteFile(filepath.Join(root, "design", "d.yaml"), []byte("version: 0.5\ntiers: [a]\nmodules:\n  core: { tier: a }\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "core"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "c.go"), []byte("package core\n\nfunc ok() {}\n"), 0o644)
	os.WriteFile(filepath.Join(root, "core", "bad.go"), []byte("package core\n\nfunc broken( {\n"), 0o644)
	var out, errb bytes.Buffer
	code := run([]string{"coverage", "--root", root}, &out, &errb)
	if code != 1 || !strings.Contains(errb.String(), "bad.go") {
		t.Fatalf("%d\n%s%s", code, out.String(), errb.String())
	}
}
