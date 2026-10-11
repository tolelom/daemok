package infer

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestWrite(t *testing.T) {
	a := &Analysis{Draft: &Draft{System: "x"}, YAML: []byte("version: 0.4\n")}
	dir := filepath.Join(t.TempDir(), "design")
	p, err := Write(dir, a)
	if err != nil || p != filepath.Join(dir, "x.yaml") {
		t.Fatalf("%q %v", p, err)
	}
	if b, _ := os.ReadFile(p); !bytes.Equal(b, a.YAML) {
		t.Fatal(string(b))
	}
	if !HasDesignFiles(dir) {
		t.Fatal("HasDesignFiles")
	}
	if _, err := Write(dir, &Analysis{Draft: &Draft{System: "y"}, YAML: []byte("z")}); !errors.Is(err, ErrDesignExists) {
		t.Fatalf("%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "y.yaml")); err == nil {
		t.Fatal("설계 파일이 있으면 쓰지 않는다")
	}
}

// 모델은 *.yaml만 읽으므로 .yml 파일은 설계 파일로 세지 않는다.
func TestWriteIgnoresYml(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "x.yml"), []byte("keep"), 0o644)
	if _, err := Write(dir, &Analysis{Draft: &Draft{System: "x"}, YAML: []byte("new")}); err != nil {
		t.Fatal(err)
	}
}
