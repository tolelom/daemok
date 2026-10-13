package infer

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
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
	// 이름 사용(name, decls, refs)은 TestAnalyzeUses가 본다.
	for _, p := range pkgs {
		got[p.dir] = pkg{dir: p.dir, main: p.main, imports: p.imports, ext: p.ext, doc: p.doc, hints: p.hints}
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

func TestCollectSkipsUnreadableDir(t *testing.T) {
	root := t.TempDir()
	os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/u\n"), 0o644)
	for _, d := range []string{"ok", "locked"} {
		os.MkdirAll(filepath.Join(root, d), 0o755)
		os.WriteFile(filepath.Join(root, d, "x.go"), []byte("package "+d+"\n"), 0o644)
	}
	lock(t, filepath.Join(root, "locked"))
	pkgs, skips, err := collect(root, "example.com/u")
	if err != nil {
		t.Fatalf("폴더 하나 때문에 전체가 실패했다: %v", err)
	}
	if len(pkgs) != 1 || pkgs[0].dir != "ok" {
		t.Fatalf("%+v", pkgs)
	}
	if len(skips) != 1 || skips[0].Path != "locked" || !strings.HasPrefix(skips[0].Reason, "읽을 수 없다") {
		t.Fatalf("%+v", skips)
	}
}

// lock은 dir을 읽을 수 없게 하고 테스트가 끝나면 되돌린다.
func lock(t *testing.T, dir string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		if out, err := exec.Command("icacls", dir, "/deny", "*S-1-1-0:(RX)").CombinedOutput(); err != nil {
			t.Skipf("icacls: %v %s", err, out)
		}
		t.Cleanup(func() { exec.Command("icacls", dir, "/remove:d", "*S-1-1-0").Run() })
	} else {
		os.Chmod(dir, 0)
		t.Cleanup(func() { os.Chmod(dir, 0o755) })
	}
	if _, err := os.ReadDir(dir); err == nil {
		t.Skip("이 환경에서는 디렉토리를 읽을 수 없게 만들 수 없다")
	}
}
