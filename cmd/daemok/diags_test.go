package main

import (
	"bytes"
	"testing"

	"github.com/tolelom/daemok/internal/model"
	"github.com/tolelom/daemok/internal/yaml"
)

func TestPrintDiagsGroups(t *testing.T) {
	var ds model.Diagnostics
	for _, p := range []string{"a.go", "b.go", "c.go"} {
		ds.Errorf(yaml.Pos{File: p}, "생성 파일이 없다: %s (daemok gen)", p)
	}
	ds.Warnf(yaml.Pos{File: "d.yaml", Line: 3}, "고아 모듈: x")
	ds.Errorf(yaml.Pos{File: "e.go"}, "생성 파일이 없다: %s (daemok gen)", "e.go")
	var b bytes.Buffer
	printDiags(&b, ds)
	want := `오류: 생성 파일이 없다 4개
  a.go: a.go (daemok gen)
  b.go: b.go (daemok gen)
  c.go: c.go (daemok gen)
  e.go: e.go (daemok gen)
d.yaml:3:0: 경고: 고아 모듈: x
오류 4, 경고 1, 정보 0
`
	if b.String() != want {
		t.Fatalf("got\n%s", b.String())
	}
}
