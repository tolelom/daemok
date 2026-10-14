package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const renameDesign = "version: 0.5\ngo_module: example.com/r\ntiers: [a, b]\nmodules:\n  core: { tier: a }\n  web: { tier: b, depends_on: [core] }\ntypes:\n  post: { in: core, fields: { title: string } }\n"

func renameRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "design"), 0o755)
	os.WriteFile(filepath.Join(root, "design", "d.yaml"), []byte(renameDesign), 0o644)
	return root
}

func TestRenameCommand(t *testing.T) {
	root := renameRoot(t)
	var out, errb bytes.Buffer
	code := run([]string{"rename", "types", "post", "article", "--root", root}, &out, &errb)
	if code != 0 || !strings.Contains(out.String(), "고정: go: { name: Post }") || !strings.Contains(out.String(), "d.yaml") {
		t.Fatalf("%d\n%s%s", code, out.String(), errb.String())
	}
	b, _ := os.ReadFile(filepath.Join(root, "design", "d.yaml"))
	if !strings.Contains(string(b), "article: { in: core, fields: { title: string }, go: { name: Post } }") {
		t.Fatalf("%s", b)
	}
}

func TestRenameCommandFlagsFirst(t *testing.T) {
	root := renameRoot(t)
	var out, errb bytes.Buffer
	code := run([]string{"rename", "--root", root, "--code", "modules", "core", "base"}, &out, &errb)
	if code != 0 || !strings.Contains(out.String(), "옮길 디렉토리: core → base") {
		t.Fatalf("%d\n%s%s", code, out.String(), errb.String())
	}
}

func TestRenameCommandRefuses(t *testing.T) {
	root := renameRoot(t)
	var out, errb bytes.Buffer
	if code := run([]string{"rename", "bogus", "a", "b", "--root", root}, &out, &errb); code != 2 {
		t.Fatalf("알 수 없는 종류: %d", code)
	}
	if code := run([]string{"rename", "types", "post", "--root", root}, &out, &errb); code != 2 {
		t.Fatalf("인자 둘: %d", code)
	}
	errb.Reset()
	if code := run([]string{"rename", "modules", "core", "web", "--root", root}, &out, &errb); code != 1 || errb.Len() == 0 {
		t.Fatalf("%d %s", code, errb.String())
	}
	b, _ := os.ReadFile(filepath.Join(root, "design", "d.yaml"))
	if string(b) != renameDesign {
		t.Fatal("거부했는데 파일이 바뀌었다")
	}
}
