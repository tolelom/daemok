package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestDiffCommand(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 없다")
	}
	root := renameRoot(t)
	for _, args := range [][]string{{"init", "-q"}, {"add", "."}, {"commit", "-q", "-m", "init"}} {
		cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
		cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v %s", err, out)
		}
	}
	var out, errb bytes.Buffer
	if code := run([]string{"diff", "--root", root}, &out, &errb); code != 0 || out.String() != "설계 변경 없음 (HEAD 대비)\n" {
		t.Fatalf("%d %s%s", code, out.String(), errb.String())
	}
	out.Reset()
	if code := run([]string{"rename", "types", "post", "article", "--root", root}, &out, &errb); code != 0 {
		t.Fatal(errb.String())
	}
	out.Reset()
	if code := run([]string{"diff", "--root", root}, &out, &errb); code != 0 || !strings.Contains(out.String(), "- 타입 `post` → `article`") {
		t.Fatalf("%d %s%s", code, out.String(), errb.String())
	}
	errb.Reset()
	if code := run([]string{"diff", "--root", root, "--base", "nope"}, &out, &errb); code != 2 || errb.Len() == 0 {
		t.Fatalf("%d", code)
	}
	if code := run([]string{"diff", "--design", filepath.Join(t.TempDir(), "x")}, &out, &errb); code != 2 {
		t.Fatalf("저장소 밖: %d", code)
	}
}
