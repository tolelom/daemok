package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeRoot는 design/ 아래 cleanDesign을 둔 코드 루트를 만든다.
// 테스트의 현재 디렉토리(cmd/daemok)에는 design/이 없으므로, 기본값이 현재 디렉토리 기준이면 설계를 찾지 못한다.
func writeRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "design"), 0o755)
	os.WriteFile(filepath.Join(root, "design", "d.yaml"), []byte(cleanDesign), 0o644)
	return root
}

func TestRootOnlyUsesRootDesign(t *testing.T) {
	for _, args := range [][]string{{"check"}, {"gen", "--dry-run"}, {"coverage"}} {
		root := writeRoot(t)
		code, out, errOut := runCLI(t, append(args, "--root", root)...)
		if code != 0 || strings.Contains(out, "설계 파일(*.yaml)이 없다") {
			t.Errorf("%v: exit %d\n%s%s", args, code, out, errOut)
		}
	}
}

func TestMCPRootOnlyUsesRootDesign(t *testing.T) {
	root := writeRoot(t)
	var out, errb bytes.Buffer
	in := strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"design","arguments":{}}}` + "\n")
	if code := runMCP([]string{"--root", root}, in, &out, &errb); code != 0 {
		t.Fatalf("%d %s", code, errb.String())
	}
	if !strings.Contains(out.String(), "core: { tier: a }") {
		t.Fatal(out.String())
	}
}

func TestDesignDirDefault(t *testing.T) {
	if got := designDir("", "/p"); got != filepath.Join("/p", "design") {
		t.Errorf("빈 --design: %q", got)
	}
	if got := designDir("x/d", "/p"); got != "x/d" {
		t.Errorf("준 --design: %q", got)
	}
}

func TestExtraArgsRejected(t *testing.T) {
	root := writeRoot(t)
	for _, cmd := range []string{"check", "gen", "coverage", "init", "serve", "mcp"} {
		code, _, errOut := runCLI(t, cmd, "--root", root, "extra", "--strict")
		if code != 2 || !strings.Contains(errOut, "extra") {
			t.Errorf("%s: exit %d, err %q", cmd, code, errOut)
		}
	}
}

// gen은 아직 없는 루트에 처음 생성할 수 있어야 하므로 check만 막는다.
func TestMissingRootIsFatal(t *testing.T) {
	root := writeRoot(t)
	missing := filepath.Join(root, "nope")
	code, out, _ := runCLI(t, "check", "--design", filepath.Join(root, "design"), "--root", missing)
	if code != 2 || !strings.Contains(out, "코드 루트가 없다") {
		t.Errorf("exit %d\n%s", code, out)
	}
}
