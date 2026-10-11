package infer

import (
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestModulePath(t *testing.T) {
	if p, err := modulePath(filepath.Join("testdata", "layers")); err != nil || p != "example.com/layers" {
		t.Fatalf("%q %v", p, err)
	}
	if _, err := modulePath(filepath.Join("testdata", "nomod")); !errors.Is(err, ErrNotModule) {
		t.Fatalf("%v", err)
	}
}

func TestCollect(t *testing.T) {
	pkgs, skips, err := collect(filepath.Join("testdata", "layers"), "example.com/layers")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]pkg{}
	for _, p := range pkgs {
		got[p.dir] = *p
	}
	want := map[string]pkg{
		"api":           {dir: "api", imports: []string{"bad-name", "core", "util"}},
		"bad-name":      {dir: "bad-name"},
		"certGen":       {dir: "certGen", imports: []string{"internal/util"}},
		"cmd/app":       {dir: "cmd/app", main: true, imports: []string{"api"}, doc: "app은 예제 실행 파일이다."},
		"core":          {dir: "core", imports: []string{"util"}, doc: "Package core는 도메인 타입이다."},
		"internal/util": {dir: "internal/util"},
		"util":          {dir: "util"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got  %+v\nwant %+v", got, want)
	}
	if !reflect.DeepEqual(skips, []Skip{{"nested", "중첩 Go 모듈"}}) {
		t.Fatalf("skips %v", skips)
	}
}

func TestCollectRootPackage(t *testing.T) {
	pkgs, _, err := collect(filepath.Join("testdata", "rootmain"), "example.com/tool-kit/v2")
	if err != nil {
		t.Fatal(err)
	}
	if len(pkgs) != 2 || pkgs[0].dir != "." || !pkgs[0].main || !reflect.DeepEqual(pkgs[0].imports, []string{"lib"}) {
		t.Fatalf("%+v", pkgs[0])
	}
}
