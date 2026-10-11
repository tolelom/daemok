package infer

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestHints(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/h\n"), 0o644)
	os.MkdirAll(filepath.Join(root, "core"), 0o755)
	os.WriteFile(filepath.Join(root, "core", "core.go"), []byte(`package core

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound = errors.New("not found")
	ErrBad      = fmt.Errorf("bad")
	notErr      = errors.New("x")
)

type TxType string

const (
	TxTransfer TxType = "transfer"
	TxMint     TxType = "mint"
)

type Level int

const (
	LevelLow Level = iota
	LevelHigh
)

const Single TxType = "single"

const (
	A = 1
	B = 2
)
`), 0o644)
	a, err := Analyze(root)
	if err != nil {
		t.Fatal(err)
	}
	want := []Hint{
		{Kind: "에러", Module: "core", Names: []string{"ErrNotFound", "ErrBad"}, Where: "core/core.go:9"},
		{Kind: "enum", Module: "core", Names: []string{"TxTransfer", "TxMint"}, Type: "TxType", Where: "core/core.go:17"},
		{Kind: "enum", Module: "core", Names: []string{"LevelLow", "LevelHigh"}, Type: "Level", Where: "core/core.go:24"},
	}
	if !reflect.DeepEqual(a.Report.Hints, want) {
		t.Fatalf("%+v", a.Report.Hints)
	}
}
