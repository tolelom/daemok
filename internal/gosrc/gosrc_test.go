package gosrc

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, dir, name, src string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestFiles(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "package p\n")
	write(t, dir, "a_test.go", "package p\n")
	write(t, dir, "linux.go", "//go:build linux\n\npackage p\n")
	write(t, dir, "tool.go", "//go:build ignore\n\npackage main\n")
	write(t, dir, "broken.go", "package p\nfunc (\n")
	write(t, dir, "note.txt", "x")
	os.Mkdir(filepath.Join(dir, "sub.go"), 0o755)
	files, errs := Files(token.NewFileSet(), dir, parser.SkipObjectResolution)
	var names []string
	for _, f := range files {
		names = append(names, f.Name.Name)
	}
	if len(files) != 2 {
		t.Fatalf("a.go와 linux.go만 읽어야 한다: %d개 %v", len(files), names)
	}
	if len(errs) != 1 || filepath.Base(errs[0].Path) != "broken.go" {
		t.Fatalf("errs %v", errs)
	}
}

func TestFilesImportsOnlyKeepsDoc(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "a.go", "// Package p는 예제다.\npackage p\n\nimport \"fmt\"\n\nfunc broken( {\n")
	files, errs := Files(token.NewFileSet(), dir, parser.ImportsOnly)
	if len(errs) != 0 || len(files) != 1 {
		t.Fatalf("files %d errs %v", len(files), errs)
	}
	if files[0].Doc == nil || len(files[0].Imports) != 1 {
		t.Fatal("ImportsOnly에서도 doc 주석과 import가 있어야 한다")
	}
}

func TestFilesMissingDir(t *testing.T) {
	files, errs := Files(token.NewFileSet(), filepath.Join(t.TempDir(), "none"), 0)
	if files != nil || errs != nil {
		t.Fatal("없는 디렉토리는 비어 있다")
	}
}

func TestSkipDir(t *testing.T) {
	for name, want := range map[string]bool{".git": true, "vendor": true, "testdata": true, "node_modules": true, "core": false, "_x": false} {
		if SkipDir(name) != want {
			t.Errorf("%s: %v", name, !want)
		}
	}
}
