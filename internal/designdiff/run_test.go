package designdiff

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestRun(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 없다")
	}
	repo := t.TempDir()
	design := filepath.Join(repo, "design")
	os.MkdirAll(design, 0o755)
	if _, err := Run(design, "HEAD"); err == nil {
		t.Fatal("저장소 밖을 받았다")
	}
	git(t, repo, "init", "-q")
	os.WriteFile(filepath.Join(repo, "README"), []byte("x"), 0o644)
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "zero")
	os.WriteFile(filepath.Join(design, "d.yaml"), []byte("modules:\n  web: { tier: b }\n"), 0o644)
	out, err := Run(design, "HEAD")
	if err != nil || !strings.Contains(out, "### 추가\n- 모듈 `web`") {
		t.Fatalf("기준에 설계가 없으면 모두 추가: %v\n%s", err, out)
	}
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "one")
	os.WriteFile(filepath.Join(design, "d.yaml"), []byte("modules:\n  web: { tier: c }\n"), 0o644)
	out, err = Run(design, "HEAD")
	if err != nil || !strings.Contains(out, "- 모듈 `web`\n  - tier: b → c") {
		t.Fatalf("%v\n%s", err, out)
	}
	if out, err = Run(design, "HEAD~1"); err != nil || !strings.Contains(out, "(HEAD~1 대비)") {
		t.Fatalf("%v\n%s", err, out)
	}
	if _, err := Run(design, "nope"); err == nil {
		t.Fatal("없는 ref를 받았다")
	}
}

func TestRunMissingDesignDir(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git이 없다")
	}
	repo := t.TempDir()
	git(t, repo, "init", "-q")
	os.WriteFile(filepath.Join(repo, "README"), []byte("x"), 0o644)
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "zero")
	if _, err := Run(filepath.Join(repo, "design"), "HEAD"); err == nil || !strings.Contains(err.Error(), "설계 디렉토리") {
		t.Fatalf("%v", err)
	}
	glob := filepath.Join(repo, "설계 [1]")
	os.MkdirAll(glob, 0o755)
	os.WriteFile(filepath.Join(glob, "d.yaml"), []byte("modules:\n  a: {}\n"), 0o644)
	git(t, repo, "add", ".")
	git(t, repo, "commit", "-q", "-m", "one")
	if out, err := Run(glob, "HEAD"); err != nil || out != "설계 변경 없음 (HEAD 대비)\n" {
		t.Fatalf("%v %s", err, out)
	}
}
